package main

import (
	"database/sql"
	_ "embed"
	"log/slog"

	"github.com/emersion/go-smtp"
	_ "github.com/mattn/go-sqlite3"
	"nouveauprintemps.org/atmail/db"
	"nouveauprintemps.org/atmail/relay"
)

//go:generate go tool sqlc generate

//go:embed db/schema.sql
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
		Queries: db.New(database),
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
