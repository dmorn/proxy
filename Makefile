all:
	go build -o bin/socks5 cmd/socks5/main.go
clean:
	rm -r bin
test:
	go test ./...
format:
	gofmt -w ./..
