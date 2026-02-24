.PHONY: build run-all stop-all test-auth test-messaging admin-run admin-build

MAKE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
BACKEND_ROOT := $(abspath $(MAKE_DIR)/..)

build:
	docker-compose -f $(MAKE_DIR)/docker-compose.yml build

run-all:
	docker-compose -f $(MAKE_DIR)/docker-compose.yml up -d

stop-all:
	docker-compose -f $(MAKE_DIR)/docker-compose.yml down

logs:
	docker-compose -f $(MAKE_DIR)/docker-compose.yml logs -f

# Manual Test Commands (Requires curl). Run after: make run-all
test-auth-register:
	curl -s -X POST http://localhost:8086/register -H "Content-Type: application/json" -d '{"username":"admin","password":"password","role":"admin"}'

test-auth-login:
	curl -s -X POST http://localhost:8086/login -H "Content-Type: application/json" -d '{"username":"admin","password":"password"}'

test-send-message:
	curl -s -X POST http://localhost:8081/send -H "Content-Type: application/json" -H "X-User-ID: test-user" -d '{"channel_id":"general","content":"SGVsbG8gV29ybGQ=","type":1}'

test-heartbeat:
	curl -s -X POST http://localhost:8083/heartbeat -H "Content-Type: application/json" -d '{"user_id":"test-user","status":1}'

# Admin service (run via: cd backend/deploy && make admin-run)
admin-run:
	cd $(BACKEND_ROOT) && go run ./admin-api/cmd/server

admin-build:
	mkdir -p $(MAKE_DIR)/bin && cd $(BACKEND_ROOT) && go build -o $(MAKE_DIR)/bin/admin-server ./admin-api/cmd/server

# Quick health checks
health:
	@echo "Auth:" && curl -s http://localhost:8086/health && echo ""
	@echo "Presence status (test-user):" && curl -s "http://localhost:8083/status?user_id=test-user" && echo ""
