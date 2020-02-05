# chat-app-with-gomniauth-websocket

このプロジェクトは[GO 言語で認証機能つきのチャットアプリを作ってみた。](http://wild-data-chase.com/index.php/2019/03/28/post-686/#outline__4_7)を参考にしている。最初はこのサイトの写経からはじめ、のちに不具合修正、機能追加と行った形で発展させていく。

## 実行方法

```
$ go run *.go
```

## どのような設計か

ユーザが入室した時、ブラウザからの ws 通信は chatroom ハンドラで対処され、受信したユーザは join チャネルに入れる。これを go routine で走っている chatroom.run において対処する。

## 写経終了後に発展させられそうな点(バックログ)

- 日本語のメッセージを送れない
- Enter でメッセージを送れるようにする
- 認証未完了の状態でも"/chat"ページへアクセスできてしまい、それがユーザとしてカウントされ、またメッセージを送れてしまう
- チャットルーム内の各ユーザ一覧(と合計ユーザ数)を表示したい
- ルームへのメンバーの入退室をリアルタイムでクライアントのユーザ一覧を更新する
- サーバにデプロイする
- ある一つメッセージを送ったらそのユーザはルームを自動的に退出することになる(リロードで再入室する)
- データの永続化
- chatroom を複数用意する

## 発展のログ

> ~~ある一つメッセージを送ったらそのユーザはルームを自動的に退出することになる(リロードで再入室する)~~

~~200205: 今回のチャットルームへのアクセスは chatroom 構造体の ServeHTTP メソッドによってハンドリングされる。この中で以下の三点で websocket の通信が切断されていたので、これらを取り除いた。~~

- ~~chatroom.ServeHTTP()における`defer func() {c.leave <- client}()`~~
- ~~client.read()の最後の`c.socket.Close()`~~
- ~~client.write()の最後の`c.socket.Close()`~~

[のちに追記]以上は全くもって不必要でトンチンカンな処理であった。メッセージ送信時に自動退出してしまうことに関しては他のどこかが問題になっていたようだ。上記３つはのちに復活させた。

> チャットルーム内の各ユーザのプロフィール(と合計ユーザ数)を表示したい

200205:

- ~~ユーザは入室時に現在入室中のユーザ情報が(サーバーサイドで既に)レンダリングされた html を取得する~~ (ユーザが chatroom に入室するのは chat.html 取得時ではなく、html をブラウザが受信して WS 通信をはじめた時。したがってユーザの描写はこの WS 通信開設後しか不可能)
- ユーザは WS 通信を開始時に WS 通信でユーザ一覧情報を受信し、DOM 操作でページを書き換える
- ユーザが入室 or 退出した際には~~その旨を~~_毎回 avator 一覧を_ WS でクライアントサイドに送信し、DOM 操作でページを書き換える

これを直す過程でまず以下を行った。

- avatar の URL は message ではなく、client が持つべき
- message 構造はどの client によるものなのかを持つべき
- message 送信の際には user 情報は json に含めない。user 情報は WS 接続確率時にクッキーからサーバーサイドで取り出す。接続後のメッセージはサーバーサイドに置いて user 情報と紐づけた message 構造体を作る。

終了後、以下を実装した。

- ユーザは WS 通信を開始時に WS 通信でユーザ一覧情報を受信し、DOM 操作でページを書き換える

ただし、これを実装したところ、以下の問題が生じた

- WS 通信で受信するデータが現状で「入室時のルーム内のユーザ一覧」と「新規メッセージ」の二種類になる

これに対処するために、websocket で送信する json を以下のフォーマットに統一する

- chatroom のメンバー一覧は`{'member_avatars': ['url1', 'url2'] }`
- 新規メッセージは`{'new_message': '新規メッセージ'}`

なお、chatroom members に関しては、自分自身の avator は表示されない。これは、user が chatroom.serveHTTP()における処理の順番が以下だからである。

1. cookie からユーザ自身に相当するクライアント構造体の初期化
2. client.send_members()で websocket 通信でメンバー一覧をブラウザに送信
3. クライアント構造体を join チャネルに追加
4. join チャネルに追加されたクライアント構造体を chatroom.clients に追加

