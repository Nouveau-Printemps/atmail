package auth

import (
	"crypto/subtle"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	ATProto  *ATProtoConfig `toml:"atproto"`
	Static   *StaticConfig  `toml:"static"`
	CatchAll *CatchAll      `toml:"catch_all"`
}

type ATProtoConfig struct {
	PDS          string `json:"pds"`
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

type StaticConfig struct {
	// Users maps username to bcrypt password
	Users map[string]string `toml:"users"`
}

type CatchAll struct {
	User string `toml:"user"`
	// bcrypt Password of user
	Password string `toml:"password"`
}

func (cfg *Config) VerifyUser(username, password string) bool {
	var realPass string
	if cfg.CatchAll != nil {
		if subtle.ConstantTimeCompare([]byte(cfg.CatchAll.User), []byte(username)) != 1 {
			return false
		}
		realPass = cfg.CatchAll.Password
	} else if cfg.Static != nil {
		var ok bool
		realPass, ok = cfg.Static.Users[username]
		if !ok {
			return false
		}
	}
	return bcrypt.CompareHashAndPassword([]byte(realPass), []byte(password)) == nil
}
