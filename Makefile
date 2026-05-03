.PHONY: all build test setup clean dev backend frontend runtime

all: build

build: 
	@$(MAKE) setup
	cd script && build-backend.sh
	cd script && build-frontend.sh
	cd script && build-runtime.sh

test:
	./test-integration.sh

setup:
	cd admin && go mod tidy
	cd web && npm install
	cd runtime && go mod tidy
dev:
	@$(MAKE) frontend &
	@$(MAKE) backend 

backend:
	cd admin && air

frontend:
	cd web && npm run dev

runtime:
	cd runtime && air

clean:
	cd web && npm run clean && rm -rf node_modules && rm -rf dist/
	cd admin && go clean && rm -f dist/
	cd runtime && go clean && rm -f dist/