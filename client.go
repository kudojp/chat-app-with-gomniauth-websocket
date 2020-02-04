package main

import (
	"time"

	"github.com/gorilla/websocket"
)

// クライアントモデル
type client struct {
	socket *websocket.Conn // websocketへのコネクション
	send   chan *message   // メッセージ
	room   *chatroom       // 所属するチャットルーム
}

// websocketに書き込まれたメッセージを読み込みroomのforwardチャネルへ送る
func (c *client) read() {
	// webソケットからjson形式でメッセージを読み出し、forwardチャネルに流す
	// 読み込みは無限ループで実行される
	for {
		var msg *message
		// クライアントからWSでメッセージを受け取る
		if err := c.socket.ReadJSON(&msg); err == nil {
			t := time.Now()
			layout := "2006-01-02 15:04:05"
			msg.Time = t.Format(layout)
			c.room.forward <- msg
		} else {
			break
		}
	}
	// c.socket.Close()
}

// roomから送られたメッセージを読み込みwebsocketに書き込む
func (c *client) write() {
	// チャネルが閉じるまでの実質無限ループ
	for msg := range c.send {
		// チャネル内のメッセージをクライアントへ送る
		if err := c.socket.WriteJSON(msg); err != nil {
			break
		}
	}
}
