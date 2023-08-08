# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

SHELL=/usr/bin/env bash
BIN_PATH ?= bin

default: bin
.PHONY: bin
bin:
	GOOS=linux GOARCH=amd64 go build -o "notify-run-result/main" ./notify-run-result
	GOOS=linux GOARCH=amd64 go build -o "poll-run-status/main" ./poll-run-status
	GOOS=linux GOARCH=amd64 go build -o "provisioning-operations-handler/main" ./provisioning-operations-handler
	GOOS=linux GOARCH=amd64 go build -o "send-apply/main" ./send-apply
	GOOS=linux GOARCH=amd64 go build -o "send-destroy/main" ./send-destroy
	GOOS=linux GOARCH=amd64 go build -o "terraform-parameter-parser/main" ./terraform-parameter-parser
	GOOS=linux GOARCH=amd64 go build -o "token-rotation/main" ./token-rotation