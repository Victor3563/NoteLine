BINARY=noteline
CMD=./cmd/noteline

.PHONY: build test static static-all

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -ldflags "-s -w -X 'main.version=dev'" \
		-o dist/noteline-linux-amd64 $(CMD)

static-all:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -ldflags "-s -w -X 'main.version=dev'" \
		-o dist/noteline-linux-amd64 $(CMD)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		go build -ldflags "-s -w -X 'main.version=dev'" \
		-o dist/noteline-linux-arm64 $(CMD)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		go build -ldflags "-s -w -X 'main.version=dev'" \
		-o dist/noteline-windows-amd64.exe $(CMD)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
		go build -ldflags "-s -w -X 'main.version=dev'" \
		-o dist/noteline-darwin-arm64 $(CMD)
