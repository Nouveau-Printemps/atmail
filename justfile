dev:
    go generate
    go run . -config dev.toml -dev

build:
    go generate
    go build -ldflags "-s -w" .

clean:
    rm -fr data/*
    rm debug.db
