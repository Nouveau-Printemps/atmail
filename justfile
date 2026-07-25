dev:
    go generate
    go run .

build:
    go generate
    go build -ldflags "-s -w" .

clean:
    rm -fr data/*
    rm debug.db
