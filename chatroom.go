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
	join    chan *user // 新入ユーザを一時的に保存するチャネル
	leave   chan *user // 退出ユーザを一時的に保存するチャネル
	users []*user // ルーム内のuser名をキーとする配列
}

// chatroomをhttp.handlerに適合させる
// websocketの開設かつuserの生成
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
	var user_info objx.Map
	if err == nil && authCookie != nil {
		user_info = objx.MustFromBase64(authCookie.Value)
	}

	// クライアントの初期化
	user := &user{
		socket: socket,
		send:   make(chan *message, messageBufferSize),
		room:   cr,
		name: 	user_info.Get("name").MustStr(),
		avatar_url: user_info.Get("avatar_url").MustStr(),
	}

	// 初回時のみでいい
	// チャットルームのjoinチャネルにアクセスし、クライアントを入室させる
	cr.join <- user

	defer func() {cr.leave <- user}()

	// 初回時のみでいい
	// チャットルームのメンバー一覧(avatar url)を送信する
	// user.send_members()

	// ずっと
	// 無限ループでチャネルの変更をWSでクライアントサイドへ送信する
	go user.write()
	// 無限ループでWSで受信したメッセージをforwardチャネルに書き込む
	user.read()
}

// チャットルームを生成する
func newRoom() *chatroom {
	t := time.Now()
	layout := "2006-01-02 15:05:04"
	fmt.Println("chatroomが生成されました:", t.Format(layout))
	return &chatroom{
		forward: make(chan *message),
		join:    make(chan *user),
		leave:   make(chan *user),
	}
}

// チャットルームを起動する
func (cr *chatroom) run(){

	// チャットルームは無限ループで起動する
	for {
		// チャネルの動きを監視し、処理を決定する
		select {

		// joinチャネルに動きがあった(クライアントが入室した)場合
		case user := <-cr.join:
			// 入室したクライアントを属性に追加
			cr.users = append(cr.users, user)
			for _, each_member := range(cr.users){
				each_member.send_members()
			}
			fmt.Printf("クライアントが入室しました。現在　%x 人のクライアントが存在しています\n", len(cr.users))

		// leaveチャネルに動きがあった(クライアントが退室した)場合
		case user := <-cr.leave:
			//　クライアントmapから対象クライアントを削除する
			cr.remove(user)
			for _, each_member := range(cr.users){
				each_member.send_members()
			}
			fmt.Printf("クライアントが退出しました。現在 %x 人のクライアントが存在しています\n", len(cr.users))

		// forwardチャネルに動きがあった(メッセージを受信した)場合
		case msg := <-cr.forward:
			fmt.Println("メッセージを受信しました")
			// 存在するクライアント全てに対してメッセージを送信する
			for _, target := range cr.users {
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
func (cr *chatroom) remove(cl *user) {
	// スライスの中身削除
	users_remaining := []*user{}
	for _, c := range cr.users {
		if c != cl {
			users_remaining = append(users_remaining, c)
		}
	}
	cr.users = users_remaining
}