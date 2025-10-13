package ws

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		c.conn.WriteMessage(websocket.TextMessage, msg)
	}
}

func ServeWs(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		client := &Client{conn: conn, send: make(chan []byte, 256)}
		hub.register <- client

		go client.writePump()
	}
}
