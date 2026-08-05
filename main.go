package main

import (
	"context"
	_ "embed"
	"flag"
	"log/slog"
	"log/syslog"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-smtp"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nyttikord/logos"
	"nouveauprintemps.org/atmail/auth"
	"nouveauprintemps.org/atmail/display"
	"nouveauprintemps.org/atmail/relay"
	"nouveauprintemps.org/atmail/storage"
	"nouveauprintemps.org/atmail/utils"
)

//go:generate go tool sqlc generate

//go:embed storage/store/schema/emails.sql
var emailsMigrations string

//go:embed storage/store/schema/mailbox.sql
var mailboxMigrations string

var (
	configPath = DefaultConfigPath
	verbose    = false
	toSyslog   = false
	dev        = false
)

func init() {
	flag.StringVar(&configPath, "config", configPath, "sets the config path")
	flag.BoolVar(&verbose, "v", verbose, "increase verbosity")
	flag.BoolVar(&toSyslog, "syslog", toSyslog, "log to syslog instead of stderr")
	flag.BoolVar(&dev, "dev", dev, "enable dev mode (insecure by definition)")
}

func main() {
	flag.Parse()
	lvl := slog.LevelInfo
	if verbose || dev {
		lvl = slog.LevelDebug
	}
	var lg *logos.Logos
	var err error
	if toSyslog {
		lg, err = logos.NewSyslog("atmail", syslog.LOG_MAIL, &logos.Options{Level: lvl})
		if err != nil {
			panic(err)
		}
	} else {
		lg = logos.NewColor(os.Stderr, &logos.Options{Level: lvl})
	}
	slog.SetDefault(slog.New(lg))
	slog.Info("starting...")
	cfg, err := ParseConfig(configPath)
	if err != nil {
		slog.Error("parsing config", "error", err)
		os.Exit(2)
	}
	slog.Debug("config parsed", "path", configPath)
	for d, v := range cfg.Domains {
		if v.CatchAll != nil {
			err = os.Mkdir(path.Join(cfg.Directory, v.CatchAll.User+"@"+d), 0o750)
			if err != nil && !os.IsExist(err) {
				slog.Error("creating folders", "domain", d, "user", v.CatchAll.User, "error", err)
				os.Exit(3)
			}
		} else if v.Static != nil {
			for u := range v.Static.Users {
				err = os.Mkdir(path.Join(cfg.Directory, u+"@"+d), 0o750)
				if err != nil && !os.IsExist(err) {
					slog.Error("creating folders", "domain", d, "user", u, "error", err)
					os.Exit(3)
				}
			}
		}
	}
	slog.Debug("users' folders created")
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt, os.Kill, syscall.SIGINT,
	)
	defer cancel()

	rl := auth.NewRateLimiter()

	bck := relay.Backend{
		Domains:     cfg.Domains,
		Queue:       relay.NewQueue(),
		LocalName:   cfg.MainDomain,
		Context:     utils.WithLogger(ctx, slog.With("module", "smtp")),
		RateLimiter: rl,
	}
	smtpSrv := smtp.NewServer(&bck)
	smtpSrv.AllowInsecureAuth = dev
	smtpSrv.MaxMessageBytes = int64(cfg.Smtp.MaxMailSize)
	smtpSrv.Domain = cfg.MainDomain
	smtpSrv.ReadTimeout = 10 * time.Second
	smtpSrv.WriteTimeout = 10 * time.Second
	defer smtpSrv.Close()

	d := &display.Backend{
		Context:     utils.WithLogger(ctx, slog.With("module", "imap")),
		Domains:     cfg.Domains,
		MaxMailSize: cfg.Smtp.MaxMailSize,
		RateLimiter: rl,
	}
	imapSrv := imapserver.New(d.Options(dev))
	defer imapSrv.Close()
	bck.OnReceive = func(user string, id int64) {
		boxes, ok := d.GetUserBoxes(user)
		if !ok {
			slog.Warn("user not found", "user", user)
			return
		}
		boxes["INBOX"].WriteNewMessages(1)
	}

	storage.Cache.Path = cfg.Directory
	storage.Cache.Migrations = emailsMigrations + mailboxMigrations
	defer func() {
		err = storage.Cache.Close(context.TODO())
		if err != nil {
			slog.Error("saving cache", "error", err)
		}
	}()

	smtpL, err := cfg.Smtp.Listen()
	if err != nil {
		slog.Error("listening for smtp", "error", err)
		os.Exit(4)
	}
	defer smtpL.Close()
	imapL, err := cfg.Imap.Listen()
	if err != nil {
		slog.Error("listening for imap", "error", err)
		os.Exit(4)
	}
	defer imapL.Close()

	errc := make(chan error, 2)
	go func() {
		errc <- smtpSrv.Serve(&auth.LimiterListener{Listener: smtpL, Limiter: rl})
	}()
	go func() {
		errc <- imapSrv.Serve(&auth.LimiterListener{Listener: imapL, Limiter: rl})
	}()
	go bck.Queue.Loop(utils.WithLogger(ctx, slog.With("module", "smtp-queue")), &bck)

	slog.Info("started")
	select {
	case <-ctx.Done():
	case err = <-errc:
		slog.Error("handling requests", "error", err)
		os.Exit(5)
	}
	slog.Info("exiting")
}
