package main

import (
	"context"
	"io"
	"os"
	"strconv"

	"nouveauprintemps.org/atmail/storage"
)

func read(account string, args []string) {
	if len(args) != 1 {
		println("Invalid read usage.")
		println("read <mail>")
		os.Exit(1)
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		println("Invalid read usage.")
		println(err.Error())
		println("read <mail>")
		os.Exit(2)
	}
	rc, err := storage.Read(context.Background(), account, int64(id))
	if err != nil {
		println(err.Error())
		os.Exit(3)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		println(err.Error())
		os.Exit(3)
	}
	os.Stderr.Write(b)
}
