package auth

import (
	"crypto/subtle"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	// ATProto related
	ATProtoPDS          string `toml:"atproto_pds"`
	ATProtoClientID     string `toml:"atproto_client_id"`
	ATProtoClientSecret string `toml:"atproto_client_secret"`

	// StaticUsers maps username to bcrypt password
	StaticUsers map[string]string `toml:"static_users"`

	// CatchAll mail to a specific address
	CatchAll         string `toml:"catch_all"`
	CatchAllPassword string `toml:"catch_all_password"`
}

func (cfg *Config) VerifyUser(username, password string) bool {
	var realPass string
	if cfg.CatchAllPassword != "" {
		if subtle.ConstantTimeCompare([]byte(cfg.CatchAll), []byte(username)) != 1 {
			return false
		}
		realPass = cfg.CatchAllPassword
	} else {
		var ok bool
		realPass, ok = cfg.StaticUsers[username]
		if !ok {
			return false
		}
	}
	return bcrypt.CompareHashAndPassword([]byte(realPass), []byte(password)) == nil
}
