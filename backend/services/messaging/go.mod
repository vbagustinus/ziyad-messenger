module lan-chat/messaging

go 1.22

require (
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/mattn/go-sqlite3 v1.14.22
	lan-chat/protocol v0.0.0
)

replace lan-chat/protocol => ../../pkg/protocol
