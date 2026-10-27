package main

import (
	"context"
	"os"
	"strconv"

	"nouveauprintemps.org/atmail/storage"
)

func delete(accountRef string, args []string) {
	if len(args) != 1 {
		println("Invalid delete usage.")
		println("delete <mail>")
		os.Exit(1)
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		println("Invalid delete usage.")
		println(err.Error())
		println("delete <mail>")
		os.Exit(2)
	}
	err = storage.DeleteEmail(context.Background(), accountRef, int64(id))
	if err != nil {
		panic(err)
	}
}
