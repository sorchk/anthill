.PHONY: all build up down logs ps test clean dev admin web

all: build

build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

logs-admin:
	docker compose logs -f admin

logs-web:
	docker compose logs -f web

ps:
	docker compose ps

test:
	./test-integration.sh

clean:
	docker compose down -v --rmi local
	rm -rf data/

dev:
	@$(MAKE) admin &
	@$(MAKE) web

admin:
	cd admin && air

web:
	cd web && npm run dev