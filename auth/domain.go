package auth

import (
	"crypto/subtle"
	"net"
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

func (cfg *Config) VerifyUser(ip net.IP, domain, username, password string) bool {
	var realPass string
	if cfg.CatchAll != nil {
		if subtle.ConstantTimeCompare([]byte(cfg.CatchAll.User+"@"+domain), []byte(username)) != 1 {
			return false
		}
		realPass = cfg.CatchAll.Password
	} else if cfg.Static != nil {
		u, d, _ := strings.Cut(username, "@")
		if d != domain {
			return false
		}
		user, ok := cfg.Static.Users[u]
		if !ok {
			return false
		}
		if user.LocalOnly {
			return ip.IsLoopback() || ip.IsPrivate()
		}
		realPass = user.Password
	}
	return bcrypt.CompareHashAndPassword([]byte(realPass), []byte(password)) == nil
}

type UserData struct {
	Username string
	Folder   string
	Key      *string
}

func (cfg *Config) Exists(username string) *UserData {
	var data UserData
	var crypto Crypto
	if cfg.CatchAll != nil {
		if !cfg.CreateFolderSubaddressing {
			username = ""
		}
		data.Username = cfg.CatchAll.User
		if username != cfg.CatchAll.User {
			data.Folder = username
		}
		crypto = cfg.CatchAll.Crypto
	} else if cfg.Static != nil {
		var subaddress string
		if cfg.PlusSubaddressing {
			username, subaddress, _ = strings.Cut(username, "+")
		}
		if !cfg.CreateFolderSubaddressing {
			subaddress = ""
		}
		u, ok := cfg.Static.Users[username]
		if !ok {
			return nil
		}
		data.Username = username
		data.Folder = subaddress
		crypto = u.Crypto
	}
	if crypto.PGPPubKey != nil {
		data.Key = crypto.PGPPubKey
	} else if crypto.PGPPubKeyFile != nil {
		b, err := os.ReadFile(*crypto.PGPPubKeyFile)
		if err != nil {
			panic(err)
		}
		data.Key = new(string(b))
	}
	return &data
}
