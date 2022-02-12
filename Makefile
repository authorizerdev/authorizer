DEFAULT_VERSION=0.1.0-local
VERSION := $(or $(VERSION),$(DEFAULT_VERSION))

cmd:
	cd server && go build -ldflags "-w -X main.VERSION=$(VERSION)" -o '../build/server'
build-app:
	cd app && npm i && npm run build
build-dashboard:
	cd dashboard && npm i && npm run build
clean:
	rm -rf build
test:
	cd server && go clean --testcache && go test -v ./test
generate:
	cd server && go get github.com/99designs/gqlgen/cmd@v0.14.0 && go run github.com/99designs/gqlgen generate
	