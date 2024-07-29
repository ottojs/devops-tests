
go fmt ./...;
go vet ./...;
go mod tidy;
$env:CGO_ENABLED = 0;
go run ./cmd/load-test/main.go;
