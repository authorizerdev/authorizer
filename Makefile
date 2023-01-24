DEFAULT_VERSION=0.1.0-local
VERSION := $(or $(VERSION),$(DEFAULT_VERSION))

cmd:
	cd server && go build -ldflags "-w -X main.VERSION=$(VERSION)" -o '../build/server'
build:
	cd server && gox \
		-osarch="linux/amd64 linux/arm64 darwin/amd64 windows/amd64" \
		-ldflags "-w -X main.VERSION=$(VERSION)" \
		-output="../build/{{.OS}}/{{.Arch}}/server" \
		./...
build-app:
	cd app && npm i && npm run build
build-dashboard:
	cd dashboard && npm i && npm run build
clean:
	rm -rf build
test:
	rm -rf server/test/test.db server/test/test.db-shm server/test/test.db-wal && rm -rf test.db test.db-shm test.db-wal && cd server && go clean --testcache && TEST_DBS="sqlite" go test -p 1 -v ./test
test-mongodb:
	docker run -d --name authorizer_mongodb_db -p 27017:27017 mongo:4.4.15
	cd server && go clean --testcache && TEST_DBS="mongodb" go test -p 1 -v ./test
	docker rm -vf authorizer_mongodb_db
test-scylladb:
	docker run -d --name authorizer_scylla_db -p 9042:9042 scylladb/scylla
	cd server && go clean --testcache && TEST_DBS="scylladb" go test -p 1 -v ./test
	docker rm -vf authorizer_scylla_db
test-arangodb:
	docker run -d --name authorizer_arangodb -p 8529:8529 -e ARANGO_NO_AUTH=1 arangodb/arangodb:3.8.4
	cd server && go clean --testcache && TEST_DBS="arangodb" go test -p 1 -v ./test
	docker rm -vf authorizer_arangodb
test-dynamodb:
	docker run -d --name dynamodb-local-test  -p 8000:8000 amazon/dynamodb-local:latest 
	cd server && go clean --testcache && TEST_DBS="dynamodb" go test -p 1 -v ./test
	docker rm -vf dynamodb-local-test
test-couchbase:
	# docker run -d --name couchbase-local-test  -p 8091-8097:8091-8097 -p 11210:11210 -p 11207:11207 -p 18091-18095:18091-18095 -p 18096:18096 -p 18097:18097 couchbase:latest
	# create a docker container, set the cluster information and then run the tests
	cd server && go clean --testcache && TEST_DBS="couchbase" go test -p 1 -v ./test
	# docker rm -vf couchbase-local-test
test-all-db:
	rm -rf server/test/test.db server/test/test.db-shm server/test/test.db-wal && rm -rf test.db test.db-shm test.db-wal
	docker run -d --name authorizer_scylla_db -p 9042:9042 scylladb/scylla
	docker run -d --name authorizer_mongodb_db -p 27017:27017 mongo:4.4.15
	docker run -d --name authorizer_arangodb -p 8529:8529 -e ARANGO_NO_AUTH=1 arangodb/arangodb:3.8.4
	docker run -d --name dynamodb-local-test  -p 8000:8000 amazon/dynamodb-local:latest
	# docker run -d --name couchbase-local-test  -p 8091-8097:8091-8097 -p 11210:11210 -p 11207:11207 -p 18091-18095:18091-18095 -p 18096:18096 -p 18097:18097 couchbase:latest
	cd server && go clean --testcache && TEST_DBS="sqlite,mongodb,arangodb,scylladb,dynamodb" go test -p 1 -v ./test
	docker rm -vf authorizer_scylla_db
	docker rm -vf authorizer_mongodb_db
	docker rm -vf authorizer_arangodb
	docker rm -vf dynamodb-local-test
	# docker rm -vf couchbase-local-test
generate:
	cd server && go run github.com/99designs/gqlgen generate && go mod tidy
