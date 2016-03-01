build:
	go build .

deps:
	go get -u github.com/nats-io/gnatsd
	go get -u github.com/gorilla/websocket

clean:
	rm -rf wsgnatsd
