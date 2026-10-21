dev:
    go generate
    go run ./cmd/atmaild/ -config dev.toml -dev

build:
    go mod tidy
    go generate
    go build -ldflags "-s -w" ./cmd/atmaild/

clean:
    rm -fr data/*
    rm debug.db

test:
    go test ./...
