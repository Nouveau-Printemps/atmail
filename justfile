build:
    go generate
    go build -ldflags "-s -w" .

dev:
    go generate
    go run .
