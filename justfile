dev:
    go generate
    go run ./cmd/atmaild/ -config dev.toml -dev

build:
    go mod tidy
    go generate
    go build -ldflags "-s -w" ./cmd/atmail/
    go build -ldflags "-s -w" ./cmd/atmaild/

install:
    install -m 755 atmail /usr/local/bin
    install -m 755 atmaild /usr/local/bin

clean:
    rm -fr data/*
    rm debug.db

test:
    go test ./...
