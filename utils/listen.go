package utils

import (
	"errors"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"

	"github.com/pires/go-proxyproto"
)

type ListenConfig struct {
	ListenAddr       string `toml:"listen"`
	UseProxyV2       bool   `toml:"use_proxy_v2"`
	SocketGroup      any    `toml:"socket_group"`
	SocketPermission uint   `toml:"socket_permission"`
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

type MultiListener struct {
	ls []net.Listener
	ch chan lRes
}

type lRes struct {
	conn net.Conn
	err  error
}

func NewMultiListener(ls []net.Listener) MultiListener {
	var l MultiListener
	l.ls = ls
	l.ch = make(chan lRes)
	for _, sub := range ls {
		go func() {
			for {
				conn, err := sub.Accept()
				l.ch <- lRes{conn, err}
			}
		}()
	}
	return l
}

func (l MultiListener) Addr() net.Addr {
	return l.ls[0].Addr()
}

func (l MultiListener) Close() error {
	var err error
	for _, sub := range l.ls {
		e := sub.Close()
		if e == nil {
			continue
		}
		if err != nil {
			err = errors.Join(err, e)
		} else {
			err = e
		}
	}
	return err
}

func (l MultiListener) Accept() (net.Conn, error) {
	v := <-l.ch
	return v.conn, v.err
}
