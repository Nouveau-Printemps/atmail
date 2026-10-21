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

func init() {
	flag.StringVar(&configPath, "config", configPath, "sets the config path")
	flag.BoolVar(&verbose, "v", verbose, "increase verbosity")
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		println("Commands:")
		println("read <account> <mail>")
		println("list <account> [mailbox]")
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
	case "list", "ls":
		list(accountRef, args[2:])
	default:
		println("Unknown command.")
		os.Exit(1)
	}
}
