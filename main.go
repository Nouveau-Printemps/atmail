package main

import (
	"database/sql"
	_ "embed"
	"log/slog"

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
	database, err := sql.Open("sqlite3", "debug.db")
	if err != nil {
		panic(err)
	}
	_, err = database.Exec(migrations)
	if err != nil {
		panic(err)
	}
	bck := relay.Backend{
		Domains: []string{"foo"},
		Storage: storage.New(index.New(database), "data"),
	}
	srv := smtp.NewServer(&bck)
	srv.Addr = ":8080"
	srv.AllowInsecureAuth = true
	srv.MaxMessageBytes = 1 << 10
	srv.Domain = "foo"
	//srv.ReadTimeout = 10 * time.Second
	//srv.WriteTimeout = 10 * time.Second
	slog.Info("starting...")
	srv.ListenAndServe()
}
