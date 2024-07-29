#!/usr/bin/env bash

# Local Version
# Use when running from your local machine
# Optional linters included but commented out
go fmt ./...;
#gofmt -e -s -w ./; # Diff: Add -d, remove -w
go vet ./...;
#gosec -quiet -color -tests ./...;
#golint ./...; # -v
go mod tidy;
go run -race ./cmd/load-test/main.go;

# # Static Version
# # Use this to deploy to multiple machines in a cluster
# # -race (requires dynamic linking)
# # -ldflags="-extldflags=-static"
# #
# mkdir -p ./bin/;
# CGO_ENABLED=0 go build -o ./bin/load-test.bin -ldflags='-s -w' ./cmd/load-test/main.go;
