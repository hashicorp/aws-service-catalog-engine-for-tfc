SHELL=/usr/bin/env bash
ORG = hashicorp
STABLE = $(shell [[ "$(VERSION)" =~ ^[0-9]+(\.[0-9]+){2}$$ ]] && echo 1)
BRANCH = $(shell git branch --show-current)
BIN_PATH ?= bin

default: bin
.PHONY: bin
bin:
	GOOS=linux GOARCH=amd64 go build -o "notify-run-result/main" -trimpath ./notify-run-result
	GOOS=linux GOARCH=amd64 go build -o "poll-run-status/main" -trimpath ./poll-run-status
	GOOS=linux GOARCH=amd64 go build -o "send-apply/main" -trimpath ./send-apply
	GOOS=linux GOARCH=amd64 go build -o "send-destroy/main" -trimpath ./send-destroy