#!/usr/bin/make -f

.PHONY: all build test

all: test build

build:
	go build -o bin/ ./...

test:
	go test -count=1 -race ./...
