package pkg

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type Node struct {
	Data chan []byte     `json:"data"`
	Conn *websocket.Conn `json:"conn"`
}

var UserClient = make(map[int]Node)

// 消息方向常量
const (
	DirectionSend    = 1 // 发送
	DirectionReceive = 2 // 接收
)

type Message struct {
	UserId      int    `json:"user_id"`
	DisId       int    `json:"dis_id"`
	Cmd         int    `json:"cmd"`
	MessageType int    `json:"message_type"`
	Content     string `json:"content"`
	Direction   int    `json:"direction"` // 添加消息方向字段
}

func Chat(c *gin.Context) {
	userId := c.GetUint("user_id")
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	node := Node{
		Data: make(chan []byte, 50),
		Conn: conn,
	}

	conn.WriteMessage(websocket.TextMessage, []byte("欢迎使用websocket"))

	UserClient[int(userId)] = node

	go write(node)
	go read(node, userId)
}

func read(node Node, userId uint) {
	for {
		_, message, err := node.Conn.ReadMessage()
		if err != nil {
			log.Println("读取消息失败:", err)
			return
		}
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Println("解析消息失败:", err)
			continue
		}

		// 使用服务器验证过的用户ID替代客户端提供的ID
		msg.UserId = int(userId)

		if msg.Cmd == 1 {
			if _, ok := UserClient[msg.DisId]; ok {
				// 重新序列化消息，使用正确的用户ID
				_, err := json.Marshal(msg)
				if err != nil {
					log.Println("重新序列化消息失败:", err)
					continue
				}

				// 发送消息给目标用户，设置为接收方向
				receiveMsg := msg
				receiveMsg.Direction = DirectionReceive
				receiveData, _ := json.Marshal(receiveMsg)
				UserClient[msg.DisId].Data <- receiveData

				// 发送确认消息给发送者，设置为发送方向
				sendMsg := msg
				sendMsg.Direction = DirectionSend
				sendData, _ := json.Marshal(sendMsg)
				UserClient[int(userId)].Data <- sendData
			} else {
				content := Message{
					Cmd:         3,
					MessageType: 1,
					Content:     "用户不存在",
				}
				marshal, _ := json.Marshal(content)
				UserClient[int(userId)].Data <- marshal
			}
		} else {
			content := Message{
				Cmd:         3,
				MessageType: 1,
				Content:     "消息类型错误",
			}
			marshal, _ := json.Marshal(content)
			UserClient[int(userId)].Data <- marshal
		}
	}
}

func write(node Node) {
	for {
		data, ok := <-node.Data
		if !ok {
			return
		}
		if err := node.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Println("传输失败:", err)
			return
		}
	}
}
