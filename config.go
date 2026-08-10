package main

import (
	_ "embed"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/user"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pires/go-proxyproto"
	"nouveauprintemps.org/atmail/auth"
)

type Config struct {
	DB            string                 `toml:"db"`
	Directory     string                 `toml:"directory"`
	MainDomain    string                 `toml:"main_domain"`
	AdminEmail    string                 `toml:"admin_email"`
	DomainsFolder *string                `toml:"domains_folder"`
	Rspamd        *RspamdConfig          `toml:"rspamd"`
	Smtp          SmtpConfig             `toml:"smtp"`
	Imap          ImapConfig             `toml:"imap"`
	Domains       map[string]auth.Config `toml:"domains"`
}

type ListenConfig struct {
	ListenAddr       string `toml:"listen"`
	UseProxyV2       bool   `toml:"use_proxy_v2"`
	SocketGroup      any    `toml:"socket_group"`
	SocketPermission uint   `toml:"socket_permission"`
}

type SmtpConfig struct {
	ListenConfig
	AllowInsecureAuth bool   `toml:"allow_insecure_auth"`
	MaxMailSize       uint32 `toml:"max_mail_size"`
}

type ImapConfig struct {
	ListenConfig
	AllowInsecureAuth bool `toml:"allow_insecure_auth"`
}

type RspamdConfig struct {
	Address string `toml:"address"`
}

const DefaultConfigPath = "/etc/atmail/config.toml"

//go:embed default.toml
var defaultConfig []byte

func ParseConfig(p string) (Config, error) {
	var cfg Config
	data, err := toml.DecodeFile(p, &cfg)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("config file not found, writing the default config", "path", p)
			err := os.WriteFile(p, defaultConfig, 0o640)
			if err != nil {
				slog.Error("writing config file", "error", err)
			}
			os.Exit(1)
		}
		return cfg, err
	}
	for _, k := range data.Undecoded() {
		slog.Warn("decoding config: configuration key not decoded", "key", k)
	}
	if cfg.DomainsFolder != nil {
		entries, err := os.ReadDir(*cfg.DomainsFolder)
		if err != nil {
			return cfg, err
		}
		if cfg.Domains == nil {
			cfg.Domains = make(map[string]auth.Config, len(entries))
		}
		for _, entry := range entries {
			d, rest, ok := strings.Cut(entry.Name(), ".toml")
			if !ok || entry.IsDir() || len(rest) != 0 {
				continue
			}
			var auth auth.Config
			data, err := toml.DecodeFile(path.Join(*cfg.DomainsFolder, entry.Name()), &auth)
			if err != nil {
				return cfg, err
			}
			for _, k := range data.Undecoded() {
				slog.Warn("decoding config: configuration key not decoded", "key", k, "file", entry.Name())
			}
			cfg.Domains[d] = auth
		}
	}
	for d, k := range cfg.Domains {
		l := slog.With("domain", d)
		if k.ATProto == nil && k.Static == nil && k.CatchAll == nil {
			l.Error("decoding config: one auth configuration must be enabled per domain")
			os.Exit(2)
		}
		if k.Admin.User == "" {
			l.Warn("decoding config: admin not set")
		}
		if (k.ATProto != nil && k.Static != nil) ||
			(k.ATProto != nil && k.CatchAll != nil) ||
			(k.CatchAll != nil && k.Static != nil) {
			l.Error("decoding config: only one auth configuration must be enabled per domain")
			os.Exit(2)
		}
		if k.CatchAll != nil && k.CatchAll.PGPPubKey != nil && k.CatchAll.PGPPubKeyFile != nil {
			l.Error("decoding config: only one of pgp_pub_key and pgp_pub_key_file can be enabled")
			os.Exit(2)
		}
		if k.Static != nil {
			for u, v := range k.Static.Users {
				if slices.Contains(auth.AdminEmails, u) {
					l.Error("this username is reserved", "user", u)
					os.Exit(2)
				}
				if v.PGPPubKey != nil && v.PGPPubKeyFile != nil {
					l.Error(
						"decoding config: only one of pgp_pub_key and pgp_pub_key_file can be enabled",
						"user", u,
					)
					os.Exit(2)
				}
			}
		}
	}
	cfg.Smtp.MaxMailSize *= 1024
	if !strings.HasPrefix(cfg.Directory, "/") {
		base, err := os.Getwd()
		if err != nil {
			return cfg, err
		}
		cfg.Directory = path.Join(base, cfg.Directory)
	}
	return cfg, nil
}

func (cfg ListenConfig) Listen() (net.Listener, error) {
	kind := "tcp"
	if strings.ContainsAny(cfg.ListenAddr, "/") {
		kind = "unix"
	}
	l, err := net.Listen(kind, cfg.ListenAddr)
	if err != nil {
		return nil, err
	}
	if cfg.UseProxyV2 {
		l = &proxyproto.Listener{Listener: l}
	}
	defer func() {
		if err != nil {
			l.Close()
		}
	}()
	if kind == "unix" {
		if cfg.SocketPermission > 0 {
			err = os.Chmod(cfg.ListenAddr, os.FileMode(cfg.SocketPermission))
			if err != nil {
				return nil, err
			}
		}
		if cfg.SocketGroup != nil {
			var gid int
			switch v := cfg.SocketGroup.(type) {
			case int64:
				if gid < 0 {
					err = errors.New("invalid socket group: must be an uint")
					return nil, err
				}
				gid = int(v)
			case string:
				group, err := user.LookupGroup(v)
				if err != nil {
					return nil, err
				}
				gid, _ = strconv.Atoi(group.Gid)
			default:
				err = errors.New("invalid socket group type: must be an uint or a string")
				return nil, err
			}
			err = os.Chown(cfg.ListenAddr, -1, gid)
			if err != nil {
				return nil, err
			}
		}
	}
	return l, nil
}
