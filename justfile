dev:
    go generate
    zig build run -- -config dev.toml -v

build:
    go generate
    zig build --summary all

clean:
    rm -fr data/*
    rm debug.db

test:
    zig run test --summary all
