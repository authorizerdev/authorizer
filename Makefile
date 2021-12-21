DEFAULT_VERSION=0.1.0-local
VERSION := $(or $(VERSION),$(DEFAULT_VERSION))

cmd:
	cd server && go build -ldflags "-w -X main.Version=$(VERSION)" -o '../build/server'
clean:
	rm -rf build
test:
	cd server && go clean --testcache && go test -v ./...