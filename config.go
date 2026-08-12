package main

import (
	_ "embed"
	"log/slog"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"nouveauprintemps.org/atmail/auth"
	"nouveauprintemps.org/atmail/utils"
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

type SmtpConfig struct {
	utils.ListenConfig
	AllowInsecureAuth bool   `toml:"allow_insecure_auth"`
	MaxMailSize       uint32 `toml:"max_mail_size"`
	ConcurrentSender  uint8  `toml:"concurrent_sender"`
}

type ImapConfig struct {
	utils.ListenConfig
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
	for domain, d := range cfg.Domains {
		l := slog.With("domain", domain)
		if d.ATProto == nil && d.Static == nil && d.CatchAll == nil {
			l.Error("decoding config: one auth configuration must be enabled per domain")
			os.Exit(2)
		}
		if d.Admin.User == "" {
			l.Warn("decoding config: admin not set")
		}
		if (d.ATProto != nil && d.Static != nil) ||
			(d.ATProto != nil && d.CatchAll != nil) ||
			(d.CatchAll != nil && d.Static != nil) {
			l.Error("decoding config: only one auth configuration must be enabled per domain")
			os.Exit(2)
		}
		if d.CatchAll != nil && d.CatchAll.PGPPubKey != nil && d.CatchAll.PGPPubKeyFile != nil {
			l.Error("decoding config: only one of pgp_pub_key and pgp_pub_key_file can be enabled")
			os.Exit(2)
		}
		if d.Static != nil {
			if d.Static.Users == nil {
				d.Static.Users = make(map[string]auth.StaticUser)
			}
			if d.Static.SystemUsers == nil {
				d.Static.SystemUsers = make(map[string]auth.SystemUser)
			}
			for u, v := range d.Static.Users {
				if slices.Contains(auth.AdminEmails, u) {
					l.Error("this username is reserved", "user", u)
					os.Exit(2)
				}
				if _, ok := d.Static.SystemUsers[u]; ok {
					l.Error("user is defined twice", "user", u)
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
