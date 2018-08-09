all:
	go build -o bin/proxy cmd/proxy/main.go
clean:
	rm -r bin
test:
	go test ./...
format:
	gofmt -w ./..
