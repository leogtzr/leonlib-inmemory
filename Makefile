.DEFAULT_GOAL := build

BIN_FILE=leonlib

build:
	@go build -o leonlib ./cmd/webapp

clean:
	go clean
	rm -f "cp.out"
	rm -f nohup.out
	rm -f "${BIN_FILE}"

test:
	go test

check:
	go test

cover:
	go test -coverprofile cp.out
	go tool cover -html=cp.out

run:
	./"${BIN_FILE}"

build_search:
	@go build -o cmd/search/search cmd/search/search.go

lint:
	golangci-lint run --enable-all
