.PHONY: fmt
fmt:
	golangci-lint run --enable-only goimports --fix ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: test
test:
	go test -race -cover ./... -coverprofile=coverage.out

.PHONY: coverage
coverage: test
	go tool cover -html=coverage.out


.PHONY: run
run:
	go run ./cmd

