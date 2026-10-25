package main

import (
	"flag"
	"os"
	"strings"

	"nouveauprintemps.org/atmail"
)

var (
	configPath = atmail.DefaultConfigPath
	verbose    = false
)

var Version = "dev"

func init() {
	flag.StringVar(&configPath, "config", configPath, "sets the config path")
	flag.BoolVar(&verbose, "v", verbose, "increase verbosity")
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		usage()
		return
	}
	if args[0] == "passwd" {
		passwd(args[1:])
		return
	}
	cfg, err := atmail.ParseConfig(configPath)
	if err != nil {
		panic(err)
	}
	accountRef := args[1]
	username, domain, ok := strings.Cut(accountRef, "@")
	if !ok {
		println("Invalid email.")
		os.Exit(1)
	}
	d, ok := cfg.Domains[domain]
	if !ok {
		println("Domain not found.")
		os.Exit(1)
	}
	if d.Exists("", username) == nil {
		println("Account not found.")
		os.Exit(1)
	}
	switch args[0] {
	case "read":
		read(accountRef, args[2:])
	case "list", "ls":
		list(accountRef, args[2:])
	default:
		println("Unknown command.")
		usage()
		os.Exit(1)
	}
}

func usage() {
	println("atmail", Version)
	println("Commands:")
	println("- read <account> <mail>")
	println("- list <account> [mailbox]")
	println("- passwd [-cost <cost>]")
}
