.PHONY: run fmt test

run:
	go run ./cmd/api

fmt:
	gofmt -w ./cmd ./internal ./pkg

test:
	go test ./...

