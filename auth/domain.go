package auth

import (
	"crypto/subtle"
	"log/slog"
	"net"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/Nouveau-Printemps/den"
	"nouveauprintemps.org/atmail/utils"
)

type Crypto struct {
	PGPPubKey     *string `toml:"pgp_pub_key"`
	PGPPubKeyFile *string `toml:"pgp_pub_key_file"`
}

func (c Crypto) GetKey() *string {
	if c.PGPPubKey != nil {
		return c.PGPPubKey
	} else if c.PGPPubKeyFile != nil {
		b, err := os.ReadFile(*c.PGPPubKeyFile)
		if err != nil {
			panic(err)
		}
		return new(string(b))
	}
	return nil
}

type Admin struct {
	User   string `toml:"user"`
	Folder string `toml:"folder"`
	Crypto
}

type Config struct {
	Admin                     Admin          `toml:"admin"`
	PlusSubaddressing         bool           `toml:"plus_subaddressing"`
	CreateFolderSubaddressing bool           `toml:"create_folder_subaddressing"`
	ATProto                   *ATProtoConfig `toml:"atproto"`
	Static                    *StaticConfig  `toml:"static"`
	CatchAll                  *CatchAll      `toml:"catch_all"`
	Den                       *Den           `toml:"den"`
}

type ATProtoConfig struct {
	PDS          string `json:"pds"`
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

type StaticConfig struct {
	Users       map[string]StaticUser `toml:"users"`
	SystemUsers map[string]SystemUser `toml:"system_users"`
	Groups      map[string]Group      `toml:"groups"`
}

type StaticUser struct {
	// Bcrypt password
	Password string `toml:"password"`
	Crypto
}

type SystemUser struct {
	utils.ListenConfig
}

type Group struct {
	Members           []string `toml:"members"`
	AuthorizedSenders []string `toml:"authorized_senders"`
}

type CatchAll struct {
	User string `toml:"user"`
	// bcrypt Password of user
	Password string `toml:"password"`
	Crypto
}

type Den struct {
	Address string `toml:"address"`
	client  *den.Client
}

func (cfg *Config) VerifyUser(conn net.Conn, domain, username, password string) bool {
	var realPass string
	switch {
	case cfg.CatchAll != nil:
		if subtle.ConstantTimeCompare([]byte(cfg.CatchAll.User+"@"+domain), []byte(username)) != 1 {
			return false
		}
		realPass = cfg.CatchAll.Password
	case cfg.Static != nil:
		u, d, _ := strings.Cut(username, "@")
		if d != domain {
			return false
		}
		user, ok := cfg.Static.Users[u]
		if !ok {
			u, ok := cfg.Static.SystemUsers[u]
			return ok && u.From(conn)
		}
		realPass = user.Password
	case cfg.Den != nil:
		resp, err := cfg.Den.client.Authn(username, password)
		if err != nil {
			slog.Error("verifying user with den", "error", err, "username", username)
		}
		return resp != nil
	}
	ok, err := detectPasswordAlgorithm(realPass).Compare(realPass, password)
	if err != nil {
		time.Sleep(1 * time.Second)
	}
	return ok
}

type UserData struct {
	Username string
	Folder   string
	Address  string
	Key      *string
	Group    *Group
}

var AdminEmails = []string{"root", "postmaster", "operator", "mail-daemon", "daemon", "uucp", "abuse", "security"}

func (cfg *Config) Exists(from, username string) *UserData {
	if slices.Contains(AdminEmails, username) {
		return &UserData{Username: username}
	}
	var data UserData
	switch {
	case cfg.CatchAll != nil:
		data.Username = cfg.CatchAll.User
		if cfg.CreateFolderSubaddressing && username != cfg.CatchAll.User {
			data.Folder = username
		}
		data.Key = cfg.CatchAll.Crypto.GetKey()
	case cfg.Static != nil:
		var subaddress string
		if cfg.PlusSubaddressing {
			username, subaddress, _ = strings.Cut(username, "+")
		}
		u, ok := cfg.Static.Users[username]
		if !ok {
			g, ok := cfg.Static.Groups[username]
			if !ok || (len(g.AuthorizedSenders) != 0 && slices.Contains(g.AuthorizedSenders, from)) {
				return nil
			}
			data.Group = &g
			return &data
		}
		data.Username = username
		if cfg.CreateFolderSubaddressing && ok {
			data.Folder = subaddress
		}
		data.Key = u.Crypto.GetKey()
	case cfg.Den != nil:
		var subaddress string
		if cfg.PlusSubaddressing {
			username, subaddress, _ = strings.Cut(username, "+")
		}
		u, err := cfg.Den.client.Exists(username)
		if err != nil {
			slog.Error("checking if user exists", "user", username, "error", err)
		}
		if u == nil {
			return nil
		}
		if cfg.CreateFolderSubaddressing {
			data.Folder = subaddress
		}
		data.Key = u.Key
	}
	return &data
}
