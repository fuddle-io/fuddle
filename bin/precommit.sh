#!/bin/bash

go fmt ./...
go generate ./...
go build ./...
go test ./...
golangci-lint run
