PROJECT := authorizer
DEFAULT_VERSION=0.1.0-local
VERSION := $(or $(VERSION),$(DEFAULT_VERSION))
DOCKER_IMAGE ?= authorizerdev/authorizer:$(VERSION)

.PHONY: all bootstrap build build-app build-dashboard build-local-image build-push-image

all: build build-app build-dashboard

bootstrap:
	go install github.com/mitchellh/gox@latest

build:
	CGO_ENABLED=0 gox \
		-mod=readonly \
		-osarch="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64" \
		-ldflags="-w -X main.VERSION=$(VERSION)" \
		-output="./build/{{.OS}}/{{.Arch}}/$(PROJECT)" \
		-tags="netgo" \
		./...
build-app:
	cd web/app && npm ci && npm run build
build-dashboard:
	cd web/dashboard && npm ci && npm run build
build-local-image:
	docker build --build-arg VERSION=$(VERSION) -t $(DOCKER_IMAGE) .
build-push-image:
	docker buildx build --platform linux/amd64,linux/arm64 --push \
		-t $(DOCKER_IMAGE) \
		--build-arg VERSION=$(VERSION) \
		.
clean:
	rm -rf build
dev:
	go run main.go --database-type=sqlite --database-url=test.db --jwt-type=HS256 --jwt-secret=test --admin-secret=admin --client-id=123456 --client-secret=secret
# test:
# 	rm -rf server/test/test.db server/test/test.db-shm server/test/test.db-wal && rm -rf test.db test.db-shm test.db-wal && cd server && go clean --testcache && TEST_DBS="sqlite" go test -p 1 -v ./test
test:
	docker rm -vf authorizer_postgres
	docker rm -vf authorizer_scylla_db
	docker rm -vf authorizer_mongodb_db
	docker rm -vf authorizer_arangodb
	docker rm -vf authorizer_dynamodb
	docker rm -vf authorizer_couchbase
	docker rm -vf authorizer_redis
	docker run -d --name authorizer_redis -p 6380:6379 redis
	docker run --name authorizer_postgres -p 5434:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=postgres -d postgres
	docker run -d --name authorizer_scylla_db -p 9042:9042 scylladb/scylla
	docker run -d --name authorizer_mongodb_db -p 27017:27017 mongo:4.4.15
	docker run -d --name authorizer_arangodb -p 8529:8529 -e ARANGO_NO_AUTH=1 arangodb/arangodb:3.10.3
	docker run -d --name authorizer_dynamodb  -p 8000:8000 amazon/dynamodb-local:latest
	docker run -d --name authorizer_couchbase  -p 8091-8097:8091-8097 -p 11210:11210 -p 11207:11207 -p 18091-18095:18091-18095 -p 18096:18096 -p 18097:18097 couchbase:latest
	sh scripts/couchbase-test.sh
	
	go test -v ./...

	docker rm -vf authorizer_postgres
	docker rm -vf authorizer_scylla_db
	docker rm -vf authorizer_mongodb_db
	docker rm -vf authorizer_arangodb
	docker rm -vf authorizer_dynamodb
	docker rm -vf authorizer_couchbase
	docker rm -vf authorizer_redis
test-mongodb:
	docker run -d --name authorizer_mongodb_db -p 27017:27017 mongo:4.4.15
	go clean --testcache && TEST_DBS="mongodb" go test -p 1 -v ./...
	docker rm -vf authorizer_mongodb_db
test-scylladb:
	docker run -d --name authorizer_scylla_db -p 9042:9042 scylladb/scylla
	go clean --testcache && TEST_DBS="scylladb" go test -p 1 -v ./...
	docker rm -vf authorizer_scylla_db
test-arangodb:
	docker run -d --name authorizer_arangodb -p 8529:8529 -e ARANGO_NO_AUTH=1 arangodb/arangodb:3.10.3
	go clean --testcache && TEST_DBS="arangodb" go test -p 1 -v ./...
	docker rm -vf authorizer_arangodb
test-dynamodb:
	docker run -d --name dynamodb-local-test  -p 8000:8000 amazon/dynamodb-local:latest
	go clean --testcache && TEST_DBS="dynamodb" go test -p 1 -v ./...
	docker rm -vf dynamodb-local-test
test-couchbase:
	docker run -d --name couchbase-local-test  -p 8091-8097:8091-8097 -p 11210:11210 -p 11207:11207 -p 18091-18095:18091-18095 -p 18096:18096 -p 18097:18097 couchbase:latest
	sh scripts/couchbase-test.sh
	go clean --testcache && TEST_DBS="couchbase" go test -p 1 -v ./...
	docker rm -vf couchbase-local-test
test-all-db:
	rm -rf test.db test.db-shm test.db-wal
	docker run -d --name authorizer_scylla_db -p 9042:9042 scylladb/scylla
	docker run -d --name authorizer_mongodb_db -p 27017:27017 mongo:4.4.15
	docker run -d --name authorizer_arangodb -p 8529:8529 -e ARANGO_NO_AUTH=1 arangodb/arangodb:3.10.3
	docker run -d --name dynamodb-local-test  -p 8000:8000 amazon/dynamodb-local:latest
	docker run -d --name couchbase-local-test  -p 8091-8097:8091-8097 -p 11210:11210 -p 11207:11207 -p 18091-18095:18091-18095 -p 18096:18096 -p 18097:18097 couchbase:latest
	sh scripts/couchbase-test.sh
	go clean --testcache && TEST_DBS="sqlite,mongodb,arangodb,scylladb,dynamodb,couchbase" go test -p 1 -v ./...
	docker rm -vf authorizer_scylla_db
	docker rm -vf authorizer_mongodb_db
	docker rm -vf authorizer_arangodb
	docker rm -vf dynamodb-local-test
	docker rm -vf couchbase-local-test
generate-graphql:
	go run github.com/99designs/gqlgen --verbose generate && go mod tidy
generate-db-template:
	cp -rf internal/storage/db/provider_template internal/storage/db/${dbname}
	find internal/storage/db/${dbname} -type f -exec sed -i -e 's/provider_template/${dbname}/g' {} \;
