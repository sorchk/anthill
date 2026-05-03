.PHONY: all build test clean install run

all: build

build:
	cd /home/sorc/test/test2/runtime && go build -o anthill-runtime ./cmd/node
	cd /home/sorc/test/test2/runtime && go build -o anthill-cli ./cmd/cli

test:
	cd /home/sorc/test/test2/runtime && go test ./...

clean:
	rm -f anthill-runtime anthill-cli

run:
	./anthill-runtime -admin wss://localhost:8080/ws

install:
	cp anthill-runtime /usr/local/bin/
	cp anthill-cli /usr/local/bin/