package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"slices"
	"text/tabwriter"

	"github.com/emersion/go-message"
	"nouveauprintemps.org/atmail/storage"
	"nouveauprintemps.org/atmail/storage/store"
)

func list(account string, args []string) {
	boxes, err := storage.ListMailbox(context.Background(), account)
	if err != nil {
		println(err.Error())
		os.Exit(2)
	}
	switch len(args) {
	case 0:
		for _, box := range boxes {
			print(box.Name, " ")
		}
		println()
	case 1:
		w := tabwriter.NewWriter(os.Stderr, 0, 2, 2, ' ', 0)
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
			b, err := storage.ReadEmail(context.Background(), account, email)
			if err != nil {
				println(err.Error())
				os.Exit(3)
			}
			msg, err := message.Read(bytes.NewBuffer(b))
			if err != nil {
				println(err.Error())
				os.Exit(3)
			}
			fmt.Fprintf(w, "%d\t%s\n", email.ID, msg.Header.Get("Subject"))
		}
		w.Flush()
	default:
		println("Invalid list usage.")
		println("list <account> [mailbox]")
		os.Exit(1)
	}
}
