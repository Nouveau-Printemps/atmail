package main

import (
	"flag"
	"io"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func passwd(args []string) {
	set := flag.NewFlagSet("passwd", flag.ExitOnError)
	cost := set.Uint("cost", uint(bcrypt.DefaultCost+2), "sets bcrypt cost")
	_ = set.Parse(args)
	if len(set.Args()) > 0 {
		println("Invalid passwd usage.")
		println("passwd [-cost <cost>]")
		os.Exit(1)
	}
	if *cost < uint(bcrypt.MinCost) || *cost > uint(bcrypt.MaxCost) {
		println("Invalid cost. Value given:", *cost)
		println("Min cost:", bcrypt.MinCost)
		println("Max cost:", bcrypt.MaxCost)
		os.Exit(1)
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	b, err = bcrypt.GenerateFromPassword(b, int(*cost))
	if err != nil {
		panic(err)
	}
	println(string(b))
}
