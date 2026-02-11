module lan-chat/messaging

go 1.22

require (
	github.com/google/uuid v1.6.0
	github.com/mattn/go-sqlite3 v1.14.22
	lan-chat/protocol v0.0.0
)

require github.com/gorilla/websocket v1.5.3 // indirect

replace lan-chat/protocol => ../../pkg/protocol
