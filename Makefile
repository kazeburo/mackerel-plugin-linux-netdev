VERSION=0.0.2
LDFLAGS=-ldflags "-w -s -X main.version=${VERSION}"
all: mackerel-plugin-linux-netdev

.PHONY: mackerel-plugin-linux-process-status

mackerel-plugin-linux-netdev: main.go
	go build $(LDFLAGS) -o mackerel-plugin-linux-netdev

linux: main.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o mackerel-plugin-linux-netdev

fmt:
	go fmt ./...

check:
	go test ./...

clean:
	rm -rf mackerel-plugin-linux-netdev

tag:
	git tag v${VERSION}
	git push origin v${VERSION}
	git push origin main
