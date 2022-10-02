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
	rm -rf server/test/test.db && rm -rf test.db && cd server && go clean --testcache && TEST_DBS="sqlite" go test -p 1 -v ./test
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
test-all-db:
	rm -rf server/test/test.db && rm -rf test.db
	docker run -d --name authorizer_scylla_db -p 9042:9042 scylladb/scylla
	docker run -d --name authorizer_mongodb_db -p 27017:27017 mongo:4.4.15
	docker run -d --name authorizer_arangodb -p 8529:8529 -e ARANGO_NO_AUTH=1 arangodb/arangodb:3.8.4
	docker run -d --name dynamodb-local-test  -p 8000:8000 amazon/dynamodb-local:latest 
	cd server && go clean --testcache && TEST_DBS="sqlite,mongodb,arangodb,scylladb,dynamodb" go test -p 1 -v ./test
	docker rm -vf authorizer_scylla_db
	docker rm -vf authorizer_mongodb_db
	docker rm -vf authorizer_arangodb
	docker rm -vf dynamodb-local-test
generate:
	cd server && go get github.com/99designs/gqlgen/cmd@v0.14.0 && go run github.com/99designs/gqlgen generate
	