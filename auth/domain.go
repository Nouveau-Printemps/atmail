package auth

import (
	"crypto/subtle"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Crypto struct {
	PGPPubKey     *string `toml:"pgp_pub_key"`
	PGPPubKeyFile *string `toml:"pgp_pub_key_file"`
}

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
	Users map[string]StaticUser `toml:"users"`
}

type StaticUser struct {
	// Bcrypt password
	Password  string `toml:"password"`
	LocalOnly bool   `toml:"local_only"`
	Crypto
}

type CatchAll struct {
	User string `toml:"user"`
	// bcrypt Password of user
	Password string `toml:"password"`
	Crypto
}

var LocalOnlyAccountKey = "local only"

func (cfg *Config) VerifyUser(domain, username, password string) (bool, *string) {
	var realPass string
	var crypto Crypto
	if cfg.CatchAll != nil {
		if subtle.ConstantTimeCompare([]byte(cfg.CatchAll.User+"@"+domain), []byte(username)) != 1 {
			return false, nil
		}
		realPass = cfg.CatchAll.Password
		crypto = cfg.CatchAll.Crypto
	} else if cfg.Static != nil {
		user, ok := cfg.Static.Users[username]
		if !ok {
			return false, nil
		}
		if user.LocalOnly {
			return false, &LocalOnlyAccountKey
		}
		realPass = user.Password
		crypto = user.Crypto
	}
	var key *string
	if crypto.PGPPubKey != nil {
		key = crypto.PGPPubKey
	} else if crypto.PGPPubKeyFile != nil {
		b, err := os.ReadFile(*crypto.PGPPubKeyFile)
		if err != nil {
			panic(err)
		}
		key = new(string(b))
	}
	return bcrypt.CompareHashAndPassword([]byte(realPass), []byte(password)) == nil, key
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
