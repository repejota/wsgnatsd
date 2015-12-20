build:
	go build .

deps:
	go get -u github.com/nats-io/gnatsd

clean:
	rm -rf wsgnatsd

