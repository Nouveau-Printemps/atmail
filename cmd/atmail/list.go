package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"slices"
	"text/tabwriter"

	"github.com/emersion/go-message"
	"nouveauprintemps.org/atmail"
	"nouveauprintemps.org/atmail/auth"
	"nouveauprintemps.org/atmail/storage"
	"nouveauprintemps.org/atmail/storage/store"
)

func list(account string, args []string) {
	boxes, err := storage.ListMailbox(context.Background(), account)
	if err != nil {
		println(err.Error())
		os.Exit(3)
	}
	switch len(args) {
	case 0:
		for _, box := range boxes {
			print(box.Name, " ")
		}
		println()
	case 1:
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		i := slices.IndexFunc(boxes, func(a store.ListMailboxRow) bool {
			return a.Name == args[0]
		})
		if i < 0 {
			println("Mailbox not found")
			os.Exit(2)
		}
		emails, err := storage.ListMailboxEmails(context.Background(), account, boxes[i].ID, nil)
		if err != nil {
			println(err.Error())
			os.Exit(3)
		}
		for _, email := range emails {
			rc, err := storage.ReadEmail(context.Background(), account, email)
			if err != nil {
				println(err.Error())
				os.Exit(3)
			}
			defer rc.Close()
			msg, err := message.Read(rc)
			if err != nil {
				println(err.Error())
				os.Exit(3)
			}
			fmt.Fprintf(w, "%d\t%s\n", email.ID, msg.Header.Get("Subject"))
		}
		w.Flush()
	default:
		println("Invalid list usage.")
		println("list [mailbox]")
		os.Exit(1)
	}
}

func listAccounts(cfg *atmail.Config, args []string) {
	set := flag.NewFlagSet("list", flag.ExitOnError)
	domain := set.String("domain", "", "sets the domain to search")
	set.Parse(args)
	if len(set.Args()) > 0 {
		println("Invalid list usage.")
		println("list")
		os.Exit(1)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	if *domain != "" {
		d, ok := cfg.Domains[*domain]
		if !ok {
			println(*domain, "not found")
			os.Exit(2)
		}
		cfg.Domains = map[string]auth.Config{*domain: d}
	}
	for d, v := range cfg.Domains {
		fmt.Fprint(w, d)
		switch {
		case v.Static != nil:
			fmt.Fprintln(w, " is static")
			for n := range v.Static.Users {
				fmt.Fprintf(w, "- User\t%s@%s\n", n, d)
			}
			for n := range v.Static.SystemUsers {
				fmt.Fprintf(w, "- SystemUser\t%s@%s\n", n, d)
			}
			for n, g := range v.Static.Groups {
				fmt.Fprintf(w, "- Group\t%s@%s\t->\t", n, d)
				for _, p := range g.Members {
					fmt.Fprintf(w, "%s\t", p)
				}
				fmt.Fprintln(w)
			}
		case v.CatchAll != nil:
			fmt.Fprintf(w, " is catch all\t%s@%s\n", v.CatchAll.User, d)
		}
	}
	w.Flush()
}
