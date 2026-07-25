package main

import (
	"context"
	"database/sql"
	_ "embed"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emersion/go-smtp"
	_ "github.com/mattn/go-sqlite3"
	"nouveauprintemps.org/atmail/relay"
	"nouveauprintemps.org/atmail/storage"
	"nouveauprintemps.org/atmail/storage/index"
)

//go:generate go tool sqlc generate

//go:embed storage/index/schema.sql
var migrations string

func main() {
	slog.Info("starting...")
	database, err := sql.Open("sqlite3", "debug.db?_journal=WAL")
	if err != nil {
		panic(err)
	}
	_, err = database.Exec(migrations)
	if err != nil {
		panic(err)
	}
	st := storage.New(index.New(database), "data")

	bck := relay.Backend{
		Domains: []string{"foo"},
		Storage: st,
	}
	srv := smtp.NewServer(&bck)
	srv.Addr = ":8080"
	srv.AllowInsecureAuth = true
	srv.MaxMessageBytes = 1 << 10
	srv.Domain = "foo"
	srv.ReadTimeout = 10 * time.Second
	srv.WriteTimeout = 10 * time.Second

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGINT)
	defer cancel()

	err = storage.Cache.Sync(ctx, st.Index)
	if err != nil {
		panic(err)
	}
	defer func() {
		err = storage.Cache.Save(context.TODO(), st.Index)
		if err != nil {
			panic(err)
		}
	}()

	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	errc := make(chan error, 1)
	go func() {
		errc <- srv.Serve(l)
	}()
	slog.Info("started")
	select {
	case <-ctx.Done():
	case err = <-errc:
		panic(err)
	}
	slog.Info("exiting")
}
