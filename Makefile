DEFAULT_VERSION=0.1.0-local
VERSION := $(or $(VERSION),$(DEFAULT_VERSION))

cmd:
	cd server && CGO_ENABLED=1 GOOS=$(GOOS) CC=$(CC) go build -ldflags "-w -X main.Version=$(VERSION)" -o '../build/server'
clean:
	rm -rf build