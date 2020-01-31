package main

import (
	"html/template"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/stretchr/objx"
)

/*
HTMLテンプレートをサーブするためのハンドラ
*/
type templateHandler struct {
	once     sync.Once          // HTMLを一度だけコンパイルするという指定
	filename string             // テンプレートとして使うHTMLファイル名
	tmpl     *template.Template //　テンプレート
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// テンプレートディレクトリを指定する
	path, err := filepath.Abs("./templates/")
	if err != nil {
		panic(err)
	}

	// 指定された名称のテンプレートファイルを一回だけコンパイルする
	t.once.Do(
		func() {
			t.tmpl = template.Must(template.ParseFiles(path + t.filename))
		})

	// HTMLに渡すデータ
	data := map[string]interface{}{
		"Host": r.Host,
	}

	//　認証が済んでいればクッキーの値も渡す
	authCookie, err := r.Cookie("auth")
	if err != nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}

	// テンプレートにデータを埋め込みresponse
	t.tmpl.Execute(w, data)
}

/*
ユーザのステータスに応じて遷移先を変更するハンドラ
ルートディレクトリのアクセスの際に呼び出される
*/
func moveHandler(w http.ResponseWriter, r *http.Request) {

	// 認証情報有無を確認する
	authCookie, _ := r.Cookie("auth")
	if authCookie != nil {
		//認証済みならchatページへアクセス
		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		// 認証が済んでいなければloginページへリダイレクト
		w.Header()["Location"] = []string{"/login"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}

/*
クッキー情報を削除し、login画面へ遷移させるハンドラ。
"/logout"へのアクセスの際に呼び刺される
*/
func logoutHandler(w http.ResponseWriter, r *http.Request) {

	//すでに設定されている"auth"のクッキー情報を削除する
	http.SetCookie(w, &http.Cookie{
		Name:   "auth",
		Value:  "",
		Path:   "",
		MaxAge: -1,
	})

	w.Header()["Location"] = []string{"/login"}
	w.WriteHeader(http.StatusTemporaryRedirect)
}
