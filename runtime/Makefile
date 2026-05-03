.PHONY: all build test clean install run

all: build

build:
	cd /home/sorc/test/test2/runtime && go build -o anthill-runtime ./cmd/node
	cd /home/sorc/test/test2/runtime && go build -o tcp-cli ./cmd/cli

test:
	cd /home/sorc/test/test2/runtime && go test ./...

clean:
	rm -f anthill-runtime tcp-cli

run:
	./anthill-runtime -admin wss://localhost:8080/ws

install:
	cp anthill-runtime /usr/local/bin/
	cp tcp-cli /usr/local/bin/