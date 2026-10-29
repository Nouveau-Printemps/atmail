package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"strconv"

	"nouveauprintemps.org/atmail/storage"
	"nouveauprintemps.org/atmail/storage/store"
)

func edit(accountRef string, args []string) {
	set := flag.NewFlagSet("edit", flag.ExitOnError)
	editor := set.String("editor", os.Getenv("EDITOR"), "sets the editor")
	set.Parse(args)
	args = set.Args()
	if len(args) != 1 {
		println("Invalid edit usage.")
		println("edit [-editor <editor>] <mail>")
		os.Exit(1)
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		println("Invalid edit usage.")
		println(err.Error())
		println("edit [-editor <editor>] <mail>")
		os.Exit(1)
	}
	email, err := storage.GetEmail(context.Background(), accountRef, id)
	if errors.Is(err, sql.ErrNoRows) {
		println("Mail not found.")
		os.Exit(2)
	}
	if err != nil {
		println(err.Error())
		os.Exit(3)
	}
	f, err := os.CreateTemp("", "atmail_edit")
	if err != nil {
		println(err.Error())
		os.Exit(3)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()
	rc, flags, err := storage.ReadEmail(context.Background(), accountRef, email)
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
	_, err = f.Write(b)
	if err != nil {
		println(err.Error())
		os.Exit(3)
	}
	cmd := exec.Command(*editor, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		println(err.Error())
		os.Exit(4)
	}
	f.Seek(0, io.SeekStart)
	b, err = io.ReadAll(f)
	if err != nil {
		println(err.Error())
		os.Exit(3)
	}
	err = storage.EditEmail(
		context.Background(),
		accountRef,
		email,
		flags&store.EmailEncrypted != 0,
		b,
	)
	if err != nil {
		println(err.Error())
		os.Exit(3)
	}
}
