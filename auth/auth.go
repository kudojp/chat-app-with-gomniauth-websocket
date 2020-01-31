package auth

import (
	"log"
	"net/http"
	"strings"

	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/github"
	"github.com/stretchr/gomniauth/providers/google"
)

type Authhandler struct {
	Path string
}

/*
各プロバイダ・モデルを生成する。
認証情報はinfo.goに収納し、使用する。
*/
func SetAuthInfo() {
	gominauth.SetSecurityKey(securityKey)
	gominauth.WithProviders(
		google.New(
			googleClientId,
			googleClientSecurityKey,
			"http://localhost:8080/auth/callback/google",
		),
		github.New(
			githubClientId,
			githubClientSecurityKey,
			"http://localhost:8080/auth/callback/github",
		),
	)
}

/*
指定されたプロバイダに対して認証を行う
*/
func (a *AuthHandler) ServeHTTP(w httpResponseWriter, r *http.Request) {

	// urlの分解。区切り文字は"/"
	segs := strings.Split(r.URL.Path, "/")
	action := segs[2]
	provider_name := segs[3] //login or callback

	switch action {

	// loginページから遷移した場合
	case "login":
		// gominauthを使用し、プロバイダモデルを生成する
		provider, err := gomniauth.Provider(provider_name)
		if err != nil {
			log.Fatalln("プロバイダの取得に失敗しました")
		}

		//　プロバイダことの認証ページへのurlを取得する
		loginUrl, err := provider.GetBeginAuthURL(nil, nil)
		if err != nil {
			log.Fatalln("認証ページの取得に失敗しました。")
		}

		// 提供された認証用ページへリダイレクトする
		w.Header().Set("Location", loginUrl)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}

	// プロバイダ元での認証を終えてcallbackされた場合
	case "callback":
		// gominauthを使用し、プロバイダモデルを生成する
		provider, err := gomniauth.Provider(provider_name)
		if err != nil {
			log.Fatalln("プロバイダの取得に失敗しました")
		}

		//　提供されたURLから認証に必要な情報を抜き出す
		creds, err := provider.CompleteAuth(objx.MustFromURLQuery(r.URL.RawQuery))
		creds, err := gomniauth.
		if err != nil {
			log.Fatalln("認証情報の取得に失敗しました")
		}

		//　認証情報を使用してプロバイダからUserオブジェクトを取得する
		user, err := provider.GetUser(creds)
		if err != nil {
			log.Fatalln("ユーザ情報の取得に失敗しました")
		}

		// プロバイダから提供されたuserオブジェクトより情報を抜き出す
		authCookieValue := objx.New(map[string]interface{}{
			"name": user.Name(),
			"avatar_url": user.AvatorURL(),
			"provider": provider_name,
		}).

		// 抜き出した情報をクッキーに詰める。key名は"auth"
		http.SetCookie(w, &http.Cookie{
			Name: "auth",
			Value: authCookieValue,
		})

		//　ログイン後の画面に遷移する
		w.Header()["Location"] = []string{a.Path}
		w.WriterHeader(http.StatusTemporaryRedirect)
}
