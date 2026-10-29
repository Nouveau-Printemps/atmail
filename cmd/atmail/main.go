package main

import (
	"context"
	"flag"
	"os"
	"strings"

	"nouveauprintemps.org/atmail"
	"nouveauprintemps.org/atmail/auth"
	"nouveauprintemps.org/atmail/storage"
)

var (
	configPath = atmail.DefaultConfigPath
	verbose    = false
	onlyDomain string
)

var Version = "dev"

func init() {
	flag.StringVar(&configPath, "config", configPath, "sets the config path")
	flag.StringVar(&onlyDomain, "domain", onlyDomain, "sets the domain to use")
	flag.BoolVar(&verbose, "v", verbose, "increase verbosity")
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		usage()
		return
	}
	switch args[0] {
	case "passwd":
		passwd(args[1:])
		return
	case "list":
		cfg, err := atmail.ParseConfig(configPath)
		if err != nil {
			panic(err)
		}
		if onlyDomain != "" {
			cfg.Domains = map[string]auth.Config{onlyDomain: cfg.Domains[onlyDomain]}
		}
		listAccounts(&cfg, args[1:])
		return
	}
	cfg, err := atmail.ParseConfig(configPath)
	if err != nil {
		panic(err)
	}
	if len(args) < 2 {
		usage()
		os.Exit(1)
	}
	accountRef := args[0]
	username, domain, ok := strings.Cut(accountRef, "@")
	if onlyDomain != "" {
		if domain != onlyDomain {
			println("Incompatible account with domain.")
			os.Exit(1)
		}
		domain = onlyDomain
	}
	var d auth.Config
	if !ok {
		d, ok = cfg.Domains[accountRef]
		if !ok || d.CatchAll == nil {
			println("Invalid email.")
			os.Exit(1)
		}
		domain = accountRef
		username = d.CatchAll.User
		accountRef = d.CatchAll.User + "@" + accountRef
	} else {
		d, ok = cfg.Domains[domain]
		if !ok {
			println("Domain not found.")
			os.Exit(1)
		}
		if d.Exists("", username) == nil {
			println("Account not found.")
			os.Exit(1)
		}
	}
	defer func() {
		err = storage.Cache.Close(context.TODO())
		if err != nil {
			println(err.Error())
			os.Exit(3)
		}
	}()
	switch args[1] {
	case "read":
		read(accountRef, args[2:])
	case "list", "ls":
		list(accountRef, args[2:])
	case "delete", "del", "remove", "rm":
		delete(accountRef, args[2:])
	case "edit":
		edit(accountRef, args[2:])
	default:
		println("Unknown command.")
		usage()
		os.Exit(1)
	}
}

func usage() {
	println("atmail", Version)
	println("Commands:")
	println("- <account> read <mail>")
	println("- <account> delete <mail>")
	println("- <account> edit [-editor <editor>] <mailbox>")
	println("- [account] list [mailbox]")
	println("- passwd [-cost <cost>]")
}
