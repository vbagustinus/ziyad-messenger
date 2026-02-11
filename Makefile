.PHONY: build run-all stop-all test-auth test-messaging admin-run admin-build

build:
	docker-compose build

run-all:
	docker-compose up -d

stop-all:
	docker-compose down

logs:
	docker-compose logs -f

# Manual Test Commands (Requires curl). Run after: make run-all
test-auth-register:
	curl -s -X POST http://localhost:8086/register -H "Content-Type: application/json" -d '{"username":"admin","password":"password","role":"admin"}'

test-auth-login:
	curl -s -X POST http://localhost:8086/login -H "Content-Type: application/json" -d '{"username":"admin","password":"password"}'

test-send-message:
	curl -s -X POST http://localhost:8081/send -H "Content-Type: application/json" -H "X-User-ID: test-user" -d '{"channel_id":"general","content":"SGVsbG8gV29ybGQ=","type":1}'

test-heartbeat:
	curl -s -X POST http://localhost:8083/heartbeat -H "Content-Type: application/json" -d '{"user_id":"test-user","status":1}'

# Admin service (from project root; admin-api is in go.work)
admin-run:
	go run ./backend/admin-api/cmd/server

admin-build:
	go build -o bin/admin-server ./backend/admin-api/cmd/server

dashboard-run:
	cd frontend/admin-dashboard && npm run dev

dashboard-build:
	cd frontend/admin-dashboard && npm run build

# Quick health checks
health:
	@echo "Auth:" && curl -s http://localhost:8086/health && echo ""
	@echo "Presence status (test-user):" && curl -s "http://localhost:8083/status?user_id=test-user" && echo ""
