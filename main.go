package main

import (
	"context"
	_ "embed"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-smtp"
	_ "github.com/mattn/go-sqlite3"
	"nouveauprintemps.org/atmail/display"
	"nouveauprintemps.org/atmail/relay"
	"nouveauprintemps.org/atmail/storage"
)

//go:generate go tool sqlc generate

//go:embed storage/index/schema/emails.sql
var emailsMigrations string

//go:embed storage/index/schema/mailbox.sql
var mailboxMigrations string

var (
	configPath = DefaultConfigPath
)

func init() {
	flag.StringVar(&configPath, "config", configPath, "sets the config path")
}

func main() {
	slog.Info("starting...")
	flag.Parse()
	cfg, err := ParseConfig(configPath)
	if err != nil {
		panic(err)
	}
	slog.Debug("config parsed", "path", configPath)
	for d, v := range cfg.Domains {
		if v.CatchAll != nil {
			err = os.Mkdir(path.Join(cfg.Directory, v.CatchAll.User+"@"+d), 0o750)
			if err != nil && !os.IsExist(err) {
				panic(err)
			}
		} else if v.Static != nil {
			for u := range v.Static.Users {
				err = os.Mkdir(path.Join(cfg.Directory, u+"@"+d), 0o750)
				if err != nil && !os.IsExist(err) {
					panic(err)
				}
			}
		}
	}
	slog.Debug("users' folders created")
	bck := relay.Backend{
		Domains: cfg.Domains,
	}
	smtpSrv := smtp.NewServer(&bck)
	smtpSrv.AllowInsecureAuth = true
	smtpSrv.MaxMessageBytes = int64(cfg.Smtp.MaxMailSize)
	smtpSrv.Domain = cfg.MainDomain
	smtpSrv.ReadTimeout = 10 * time.Second
	smtpSrv.WriteTimeout = 10 * time.Second

	d := &display.Backend{
		Domains: cfg.Domains,
	}

	imapSrv := imapserver.New(d.Options(slog.Default()))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGINT)
	defer cancel()

	storage.Cache.Path = cfg.Directory
	storage.Cache.Migrations = emailsMigrations + mailboxMigrations
	defer func() {
		err = storage.Cache.Close(context.TODO())
		if err != nil {
			panic(err)
		}
	}()

	smtpL, err := cfg.Smtp.Listen()
	if err != nil {
		panic(err)
	}
	defer smtpL.Close()
	imapL, err := cfg.Imap.Listen()
	if err != nil {
		panic(err)
	}
	defer imapL.Close()

	errc := make(chan error, 1)
	go func() {
		errc <- smtpSrv.Serve(smtpL)
	}()
	go func() {
		errc <- imapSrv.Serve(imapL)
	}()

	slog.Info("started")
	select {
	case <-ctx.Done():
	case err = <-errc:
		panic(err)
	}
	slog.Info("exiting")
}
