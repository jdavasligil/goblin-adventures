.PHONY: all
all: fmt tidy test build run

.PHONY: fmt
fmt:
	@go fmt ./...

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: build
build:
	@go build -tags "linux" -o ./build/bot ./cmd/bot

.PHONY: test
test: cleantest
	@go test -v ./...

.PHONY: run
run:
	@./build/bot

.PHONY: clean
clean:
	@go clean && rm -rf ./build/*

.PHONY: cleantest
cleantest:
	@rm -f ./db/test.db

.PHONY: cleandb
cleandb:
	@rm -rf ./db/*
