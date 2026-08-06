package auth

import (
	"crypto/subtle"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	PlusSubaddressing         bool           `toml:"plus_subaddressing"`
	CreateFolderSubaddressing bool           `toml:"create_folder_subaddressing"`
	ATProto                   *ATProtoConfig `toml:"atproto"`
	Static                    *StaticConfig  `toml:"static"`
	CatchAll                  *CatchAll      `toml:"catch_all"`
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

func (cfg *Config) VerifyUser(domain, username, password string) bool {
	var realPass string
	if cfg.CatchAll != nil {
		if subtle.ConstantTimeCompare([]byte(cfg.CatchAll.User+"@"+domain), []byte(username)) != 1 {
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

func (cfg *Config) Exists(username string) (exists bool, user string, subaddress string) {
	if cfg.CatchAll != nil {
		if !cfg.CreateFolderSubaddressing {
			username = ""
		}
		return true, cfg.CatchAll.User, username
	}
	if cfg.Static != nil {
		var subaddress string
		if cfg.PlusSubaddressing {
			username, subaddress, _ = strings.Cut(username, "+")
		}
		if !cfg.CreateFolderSubaddressing {
			subaddress = ""
		}
		_, ok := cfg.Static.Users[username]
		return ok, username, subaddress
	}
	return
}
