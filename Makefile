.PHONY: all build test setup clean dev backend frontend runtime

ENV_FILE := .env
ifneq ($(wildcard $(ENV_FILE)),)
include $(ENV_FILE)
endif
export

define REQUIRE_ENV
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "Missing env file: $(ENV_FILE)"; \
		echo "Create .env from .env.example"; \
		exit 1; \
	fi
endef
all: build

build-runtime: 
	cd scripts && ls -la && ./build-runtime.sh

build: 
	cd scripts && ./build-backend.sh
	cd scripts && ./build-frontend.sh
	cd scripts && ./build-runtime.sh

test:
	./test-integration.sh

setup:
	cd admin && go mod tidy
	cd web && npm install
	cd runtime && go mod tidy
dev:
	${REQUIRE_ENV}
	@echo "Using env file: $(ENV_FILE)"
	@$(MAKE) setup
	@echo "Starting backend and frontend..."
	@trap 'kill 0' EXIT; \
		 (cd admin && air) & \
		 (cd web && npm run dev) & \
			wait
stop: ## Stop backend and frontend processes for the current checkout
	$(REQUIRE_ENV)
	@echo "Stopping services..."
	@-lsof -ti:$(PORT) | xargs kill -9 2>/dev/null
	@-lsof -ti:$(FRONTEND_PORT) | xargs kill -9 2>/dev/null
	@case "$(DATABASE_URL)" in \
		""|*@localhost:*|*@localhost/*|*@127.0.0.1:*|*@127.0.0.1/*|*@\[::1\]:*|*@\[::1\]/*) \
			echo "✓ App processes stopped. Shared PostgreSQL is still running on localhost:$(POSTGRES_PORT)." ;; \
		*) \
			echo "✓ App processes stopped. Remote PostgreSQL was not affected." ;; \
	esac

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