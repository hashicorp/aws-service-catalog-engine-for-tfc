# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

SHELL=/usr/bin/env bash
BIN_PATH ?= bin

default: bin
.PHONY: bin
bin:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o "poll-run-status/bootstrap" ./poll-run-status
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o "notify-run-result/bootstrap" ./notify-run-result
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o "provisioning-operations-handler/bootstrap" ./provisioning-operations-handler
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o "send-apply/bootstrap" ./send-apply
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o "send-destroy/bootstrap" ./send-destroy
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o "terraform-parameter-parser/bootstrap" ./terraform-parameter-parser
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o "token-rotation/bootstrap" ./token-rotation
