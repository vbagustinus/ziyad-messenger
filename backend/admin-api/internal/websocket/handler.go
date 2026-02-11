package websocket

import (
	"net/http"
	"time"

	"admin-service/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var claims *auth.Claims
		if val, ok := c.Get(auth.ClaimsKey); ok {
			claims, _ = val.(*auth.Claims)
		}
		if claims == nil {
			tokenStr := c.Query("token")
			if tokenStr != "" {
				claims, _ = auth.ParseToken(tokenStr)
			}
		}
		if claims == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		client := &Client{
			ID:     uuid.New().String(),
			Send:   make(chan []byte, 256),
			UserID: claims.UserID,
		}
		hub.Register(client)

		go func() {
			defer func() {
				hub.Unregister(client)
				conn.Close()
			}()
			for msg := range client.Send {
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			}
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}
}
