package main

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/objx"
)

// クライアントモデル
type user struct {
	socket     *websocket.Conn // websocketへのコネクション
	send       chan *message   // 送信したメッセージを一時保存するチャネル
	room       *chatroom       // 所属するチャットルーム
	name       string
	avatar_url string
}

// websocketに書き込まれたメッセージを読み込みroomのforwardチャネルへ送る
func (cl *user) read() {
	// webソケットからjson形式でメッセージを読み出し、forwardチャネルに流す
	// 読み込みは無限ループで実行される
	for {
		var rec_msg objx.Map
		msg := message{}
		// クライアントからWSでメッセージを受け取る
		if err := cl.socket.ReadJSON(&rec_msg); err == nil {
			msg.user = cl
			msg.time = time.Now().Format("2006-01-02 15:04:05")
			msg.message = rec_msg.Get("message").MustStr()
			cl.room.forward <- &msg
		} else {
			break
		}
	}
	cl.socket.Close()
}

// roomから送られたメッセージを読み込みwebsocketに書き込む
func (cl *user) write() {
	// チャネルが閉じるまでの実質無限ループ
	for msg := range cl.send {
		// チャネル内のメッセージをクライアントへ送るjson
		// user_name、avatar_url、Time, message
		msg_map := map[string]string{
			"name":       msg.user.name,
			"avatar_url": msg.user.avatar_url,
			"time":       msg.time,
			"message":    msg.message,
		}

		if err := cl.socket.WriteJSON([]interface{}{"new_message", msg_map}); err != nil {
			break
		}
	}
	cl.socket.Close()
}

// クライアントが所属しているchatroomに所属するユーザ一覧(avatarのurl)を
//　WebSocket送信する
func (cl *user) send_members() {
	member_avatars := []string{}
	for _, each_cl := range cl.room.users {
		member_avatars = append(member_avatars, each_cl.avatar_url)
	}
	if err := cl.socket.WriteJSON([]interface{}{"member_avatars", member_avatars}); err != nil {
		panic("Could not send avatar_urls")
	}
}
