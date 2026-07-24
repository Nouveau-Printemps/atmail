package main

import (
	"time"

	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/relay"
)

func main() {
	bck := relay.Backend{
		Domains: []string{"foo"},
	}
	srv := smtp.NewServer(&bck)
	srv.Addr = ":8080"
	srv.AllowInsecureAuth = true
	srv.MaxMessageBytes = 2 << 10
	srv.Domain = "foo"
	srv.ReadTimeout = 10 * time.Second
	srv.WriteTimeout = 10 * time.Second
	srv.ListenAndServe()
}
