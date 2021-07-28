DEFAULT_VERSION=0.1.0-local
VERSION := $(or $(VERSION),$(DEFAULT_VERSION))

cmd:
	go build -ldflags "-w -X main.Version=$(VERSION)"
clean:
	rm -rf server