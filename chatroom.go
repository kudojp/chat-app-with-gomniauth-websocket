package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/objx"
)

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

// websocketの変数
var upgrader = &websocket.Upgrader{
	ReadBufferSize:  socketBufferSize,
	WriteBufferSize: socketBufferSize,
}

// チャットルームモデル
type chatroom struct {
	forward chan *message // 新着メッセージを一時保存するチャネル
	join    chan *client // 新入ユーザを一時的に保存するチャネル
	leave   chan *client // 退出ユーザを一時的に保存するチャネル
	clients []*client // ルーム内のuser名をキーとする配列
}

// chatroomをhttp.handlerに適合させる
// websocketの開設かつclientの生成
func (cr *chatroom) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	fmt.Print(r.Proto)

	// 初回時のみでいい
	// websocketの開設
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatalln("websocketの開設に失敗しました", err)
	}

	// 初回時のみでいい
	// まずheaderからuser情報の抽出
	authCookie, err := r.Cookie("auth")
	var user objx.Map
	if err == nil && authCookie != nil {
		user = objx.MustFromBase64(authCookie.Value)
	}

	// クライアントの初期化
	client := &client{
		socket: socket,
		send:   make(chan *message, messageBufferSize),
		room:   cr,
		name: 	user.Get("name").MustStr(),
		avatar_url: user.Get("avatar_url").MustStr(),
	}

	// 初回時のみでいい
	// チャットルームのjoinチャネルにアクセスし、クライアントを入室させる
	cr.join <- client

	defer func() {cr.leave <- client}()

	// 初回時のみでいい
	// チャットルームのメンバー一覧(avatar url)を送信する
	// client.send_members()

	// ずっと
	// 無限ループでチャネルの変更をWSでクライアントサイドへ送信する
	go client.write()
	// 無限ループでWSで受信したメッセージをforwardチャネルに書き込む
	client.read()
}

// チャットルームを生成する
func newRoom() *chatroom {
	t := time.Now()
	layout := "2006-01-02 15:05:04"
	fmt.Println("chatroomが生成されました:", t.Format(layout))
	return &chatroom{
		forward: make(chan *message),
		join:    make(chan *client),
		leave:   make(chan *client),
	}
}

// チャットルームを起動する
func (cr *chatroom) run(){

	// チャットルームは無限ループで起動する
	for {
		// チャネルの動きを監視し、処理を決定する
		select {

		// joinチャネルに動きがあった(クライアントが入室した)場合
		case client := <-cr.join:
			// 入室したクライアントを属性に追加
			cr.clients = append(cr.clients, client)
			for _, each_member := range(cr.clients){
				each_member.send_members()
			}
			fmt.Printf("クライアントが入室しました。現在　%x 人のクライアントが存在しています\n", len(cr.clients))

		// leaveチャネルに動きがあった(クライアントが退室した)場合
		case client := <-cr.leave:
			//　クライアントmapから対象クライアントを削除する
			cr.remove(client)
			for _, each_member := range(cr.clients){
				each_member.send_members()
			}
			fmt.Printf("クライアントが退出しました。現在 %x 人のクライアントが存在しています\n", len(cr.clients))

		// forwardチャネルに動きがあった(メッセージを受信した)場合
		case msg := <-cr.forward:
			fmt.Println("メッセージを受信しました")
			// 存在するクライアント全てに対してメッセージを送信する
			for _, target := range cr.clients {
				select {
				case target.send <- msg:
					fmt.Println("メッセージの送信に成功しました")
				default:
					// このユーザは取り除く
					cr.remove(target)
					fmt.Println("メッセージの送信に失敗しました")
				}
			}
		}
	}
}

// チャットルームモデルからユーザを取り除くプライベート関数
func (cr *chatroom) remove(cl *client) {
	// スライスの中身削除
	clients_remaining := []*client{}
	for _, c := range cr.clients {
		if c != cl {
			clients_remaining = append(clients_remaining, c)
		}
	}
	cr.clients = clients_remaining
}