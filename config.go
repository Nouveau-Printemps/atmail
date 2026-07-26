package main

import (
	_ "embed"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pires/go-proxyproto"
	"nouveauprintemps.org/atmail/auth"
)

type Config struct {
	ListenAddr  string                 `toml:"listen"`
	UseProxyV2  bool                   `toml:"use_proxy_v2"`
	DB          string                 `toml:"db"`
	Directory   string                 `toml:"directory"`
	MainDomain  string                 `toml:"main_domain"`
	MaxMailSize uint32                 `toml:"max_mail_size"`
	Domains     map[string]auth.Config `toml:"domains"`
}

const DefaultConfigPath = "/etc/atmail/config.toml"

//go:embed default.toml
var defaultConfig []byte

func ParseConfig(path string) (Config, error) {
	var cfg Config
	data, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("config file not found, writing the default config", "path", path)
			err := os.WriteFile(path, defaultConfig, 0o640)
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
	cfg.MaxMailSize *= 1024
	return cfg, nil
}

func (cfg *Config) Listen() (net.Listener, error) {
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
