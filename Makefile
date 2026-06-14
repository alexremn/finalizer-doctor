.PHONY: test cover lint build integration e2e tidy

test:
	go test -race ./...

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

lint:
	golangci-lint run

build:
	go build -o bin/kubectl-finalizer_doctor ./cmd/finalizer-doctor
	go build -o bin/kubectl-fid ./cmd/finalizer-doctor

integration:
	go test -tags integration ./internal/cluster/...

e2e:
	go test -tags e2e ./test/e2e/...

tidy:
	go mod tidy
