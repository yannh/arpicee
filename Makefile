#!/usr/bin/make -f

.PHONY: all build test

all: test build

build:
	go build -o bin/ ./...

test:
	go test -count=1 -race ./...

goreleaser-build:
	docker run -t -e GOOS=linux -e GOARCH=amd64 -v $$PWD:/go/src/github.com/yannh/arpicee -w /go/src/github.com/yannh/arpicee goreleaser/goreleaser:v1.18.2 build --single-target --skip-post-hooks --rm-dist --snapshot
	cp dist/arpicee_linux_amd64_v1/arpicee-slackbot bin/

release:
	docker run -e GITHUB_TOKEN -e GIT_OWNER -t -v /var/run/docker.sock:/var/run/docker.sock -v $$PWD:/go/src/github.com/yannh/arpicee -w /go/src/github.com/yannh/arpicee goreleaser/goreleaser:v1.18.2 release --rm-dist
