package main

import (
	"flag"
	"io"
	"os"

	"nouveauprintemps.org/atmail"
	"nouveauprintemps.org/atmail/auth"
)

func passwd(cfg *atmail.Config, args []string) {
	set := flag.NewFlagSet("passwd", flag.ExitOnError)
	def := "argon2id"
	if cfg.Password.Default != nil {
		def = *cfg.Password.Default
	}
	algo := set.String("algo", def, "algorithm to use")
	_ = set.Parse(args)
	if len(set.Args()) > 0 {
		println("Invalid passwd usage.")
		println("passwd [-algo <algo>]")
		os.Exit(1)
	}
	var pass auth.Password
	switch *algo {
	case "argon2id", "argon2", "argon":
		pass = auth.DefaultArgon2id
	case "scrypt":
		pass = auth.DefaultScrypt
	case "bcrypt":
		pass = auth.DefaultBcrypt
	default:
		println("Invalid passwd usage.")
		println("Available algorithms: argon2id, scrypt and bcrypt.")
		os.Exit(1)
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	h, err := pass.Hash(string(b))
	if err != nil {
		panic(err)
	}
	os.Stdout.Write([]byte(h))
}
