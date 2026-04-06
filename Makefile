PROJECT := authorizer
DEFAULT_VERSION=0.1.0-local
VERSION := $(or $(VERSION),$(DEFAULT_VERSION))
DOCKER_IMAGE ?= authorizerdev/authorizer:$(VERSION)

# Full module test run. Storage provider tests honour TEST_DBS (defaults to all).
# Integration tests and memory_store/db tests always use SQLite.
# Redis memory_store tests run only when TEST_ENABLE_REDIS=1.
GO_TEST_ALL := go test -p 1 -v ./...

.PHONY: all bootstrap build build-app build-dashboard build-local-image build-push-image trivy-scan

all: build build-app build-dashboard

bootstrap:
	go install github.com/mitchellh/gox@latest

build:
	CGO_ENABLED=0 gox \
		-mod=readonly \
		-osarch="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64" \
		-ldflags="-w -X github.com/authorizerdev/authorizer/internal/constants.VERSION=$(VERSION)" \
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
# Run Trivy vulnerability scan on the Docker image (default: $(DOCKER_IMAGE)). Use IMAGE=myimage:tag to scan another image.
trivy-scan:
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
		aquasec/trivy:latest image $(or $(IMAGE),$(DOCKER_IMAGE)) \
		--severity HIGH,CRITICAL --ignore-unfixed --exit-code 1
clean:
	rm -rf build
dev:
	go run main.go --database-type=sqlite --database-url=test.db --jwt-type=HS256 --jwt-secret=test --admin-secret=admin --client-id=123456 --client-secret=secret

test:
	go clean --testcache && TEST_DBS="sqlite" $(GO_TEST_ALL)

test-postgres: test-cleanup-postgres
	docker run -d --name authorizer_postgres -p 5434:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=postgres postgres
	sleep 3
	go clean --testcache && TEST_DBS="postgres" $(GO_TEST_ALL)
	docker rm -vf authorizer_postgres

test-sqlite:
	go clean --testcache && TEST_DBS="sqlite" $(GO_TEST_ALL)

test-mongodb: test-cleanup-mongodb
	docker run -d --name authorizer_mongodb_db -p 27017:27017 mongo:4.4.15
	sleep 3
	go clean --testcache && TEST_DBS="mongodb" $(GO_TEST_ALL)
	docker rm -vf authorizer_mongodb_db

test-scylladb: test-cleanup-scylladb
	docker run -d --name authorizer_scylla_db -p 9042:9042 scylladb/scylla
	sleep 15
	go clean --testcache && TEST_DBS="scylladb" $(GO_TEST_ALL)
	docker rm -vf authorizer_scylla_db

test-arangodb: test-cleanup-arangodb
	docker run -d --name authorizer_arangodb -p 8529:8529 -e ARANGO_NO_AUTH=1 arangodb/arangodb:3.10.3
	sleep 5
	go clean --testcache && TEST_DBS="arangodb" $(GO_TEST_ALL)
	docker rm -vf authorizer_arangodb

test-dynamodb: test-cleanup-dynamodb
	docker run -d --name authorizer_dynamodb -p 8000:8000 amazon/dynamodb-local:latest
	sleep 3
	go clean --testcache && TEST_DBS="dynamodb" $(GO_TEST_ALL)
	docker rm -vf authorizer_dynamodb

test-couchbase: test-cleanup-couchbase
	docker run -d --name authorizer_couchbase -p 8091-8097:8091-8097 -p 11210:11210 -p 11207:11207 -p 18091-18095:18091-18095 -p 18096:18096 -p 18097:18097 couchbase:latest
	sh scripts/couchbase-test.sh
	go clean --testcache && TEST_DBS="couchbase" $(GO_TEST_ALL)
	docker rm -vf authorizer_couchbase

test-all-db: test-cleanup test-docker-up test-cleanup
	go clean --testcache && TEST_DBS="postgres,sqlite,mongodb,arangodb,scylladb,dynamodb,couchbase" $(GO_TEST_ALL)
	$(MAKE) test-cleanup

# Start all test database containers
test-docker-up:
	docker run -d --name authorizer_redis -p 6380:6379 redis
	docker run -d --name authorizer_postgres -p 5434:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=postgres postgres
	docker run -d --name authorizer_mongodb_db -p 27017:27017 mongo:4.4.15
	docker run -d --name authorizer_scylla_db -p 9042:9042 scylladb/scylla
	docker run -d --name authorizer_arangodb -p 8529:8529 -e ARANGO_NO_AUTH=1 arangodb/arangodb:3.10.3
	docker run -d --name authorizer_dynamodb -p 8000:8000 amazon/dynamodb-local:latest
	docker run -d --name authorizer_couchbase -p 8091-8097:8091-8097 -p 11210:11210 -p 11207:11207 -p 18091-18095:18091-18095 -p 18096:18096 -p 18097:18097 couchbase:latest
	sh scripts/couchbase-test.sh
	sleep 5

# Remove all test database containers
test-cleanup:
	-docker rm -vf authorizer_postgres
	-docker rm -vf authorizer_scylla_db
	-docker rm -vf authorizer_mongodb_db
	-docker rm -vf authorizer_arangodb
	-docker rm -vf authorizer_dynamodb
	-docker rm -vf authorizer_couchbase
	-docker rm -vf authorizer_redis

test-cleanup-postgres:
	-docker rm -vf authorizer_postgres
test-cleanup-mongodb:
	-docker rm -vf authorizer_mongodb_db
test-cleanup-scylladb:
	-docker rm -vf authorizer_scylla_db
test-cleanup-arangodb:
	-docker rm -vf authorizer_arangodb
test-cleanup-dynamodb:
	-docker rm -vf authorizer_dynamodb
test-cleanup-couchbase:
	-docker rm -vf authorizer_couchbase
generate-graphql:
	go run github.com/99designs/gqlgen --verbose generate && go mod tidy
generate-db-template:
	cp -rf internal/storage/db/provider_template internal/storage/db/${dbname}
	find internal/storage/db/${dbname} -type f -exec sed -i -e 's/provider_template/${dbname}/g' {} \;
