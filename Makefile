VERSION=`cat ./VERSION`

LDFLAGS=-ldflags "-X main.Version=${VERSION}"

build:
	go build -v ${LDFLAGS}

test:
	go test -v ./...

cover:
	go test -v ./... --cover

deps: dev-deps
	go get -u github.com/nats-io/gnatsd
	go get -u github.com/gorilla/websocket

dev-deps:
	go get -u github.com/golang/lint/golint

lint:
	$(GOPATH)/bin/golint ./...
	go vet ./...

clean:
	go clean
	rm -rf wsgnatsd
