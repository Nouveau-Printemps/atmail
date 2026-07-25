dev:
    go generate
    go run .

build:
    go generate
    go build -ldflags "-s -w" .
