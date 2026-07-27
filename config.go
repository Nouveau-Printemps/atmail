package main

import (
	_ "embed"
	"log/slog"
	"net"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pires/go-proxyproto"
	"nouveauprintemps.org/atmail/auth"
)

type Config struct {
	DB         string                 `toml:"db"`
	Directory  string                 `toml:"directory"`
	MainDomain string                 `toml:"main_domain"`
	Smtp       SmtpConfig             `toml:"smtp"`
	Imap       ImapConfig             `toml:"imap"`
	Domains    map[string]auth.Config `toml:"domains"`
}

type ListenConfig struct {
	ListenAddr string `toml:"listen"`
	UseProxyV2 bool   `toml:"use_proxy_v2"`
}

type SmtpConfig struct {
	ListenConfig
	MaxMailSize uint32 `toml:"max_mail_size"`
}

type ImapConfig struct {
	ListenConfig
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
				panic(err)
			}
			os.Exit(1)
		}
		return cfg, err
	}
	for _, k := range data.Undecoded() {
		slog.Warn("decoding config: configuration key not decoded", "key", k)
	}
	for d, k := range cfg.Domains {
		if k.ATProto == nil && k.Static == nil && k.CatchAll == nil {
			slog.Error("decoding config: one auth configuration must be enabled per domain", "domain", d)
			os.Exit(2)
		}
		if (k.ATProto != nil && k.Static != nil) ||
			(k.ATProto != nil && k.CatchAll != nil) ||
			(k.CatchAll != nil && k.Static != nil) {
			slog.Error(
				"decoding config: only one auth configuration must be enabled per domain",
				"domain", d,
			)
			os.Exit(2)
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
	return l, nil
}
