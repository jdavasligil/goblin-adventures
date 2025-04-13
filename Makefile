build_tags = "linux sqlite_foreign_keys"

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
	@go build -tags $(build_tags) -o ./build/bot/bot ./cmd/bot

.PHONY: test
test: cleantest
	@go test -tags $(build_tags) -v ./...

.PHONY: run
run:
	@./build/bot/bot

.PHONY: clean
clean:
	@go clean && rm -rf ./build/*

.PHONY: cleantest
cleantest:
	@rm -f ./db/test.db

.PHONY: cleandb
cleandb:
	@rm -rf ./db/*
