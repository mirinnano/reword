package mail

import "fmt"

// Links carry tokens in the URL fragment (#token=...). Fragments are never
// sent to servers or in Referer headers, so tokens stay out of access logs.

const signature = "\n\n--\nReWord（仮称）\nこのメールに心当たりがない場合は、破棄してください。"

func VerifySignup(to, appOrigin, token string) Message {
	return Message{
		To:      to,
		Subject: "【ReWord】メールアドレスの確認",
		Text: fmt.Sprintf("ReWord へのご登録ありがとうございます。\n\n"+
			"次のリンクを開くと登録が完了します（24時間有効）。\n%s/verify-email#token=%s%s",
			appOrigin, token, signature),
	}
}

func AlreadyRegistered(to, appOrigin string) Message {
	return Message{
		To:      to,
		Subject: "【ReWord】このメールアドレスは登録済みです",
		Text: fmt.Sprintf("このメールアドレスで新規登録の申し込みがありましたが、すでにアカウントがあります。\n\n"+
			"ログイン: %s/login\nパスワードを忘れた場合: %s/forgot-password%s",
			appOrigin, appOrigin, signature),
	}
}

func ResetPassword(to, appOrigin, token string) Message {
	return Message{
		To:      to,
		Subject: "【ReWord】パスワードの再設定",
		Text: fmt.Sprintf("パスワード再設定の申し込みを受け付けました。\n\n"+
			"次のリンクから新しいパスワードを設定してください（30分有効）。\n%s/reset-password#token=%s%s",
			appOrigin, token, signature),
	}
}

func PasswordChanged(to, appOrigin string) Message {
	return Message{
		To:      to,
		Subject: "【ReWord】パスワードが変更されました",
		Text: fmt.Sprintf("アカウントのパスワードが変更され、すべての端末からログアウトしました。\n\n"+
			"心当たりがない場合は、すぐにパスワードを再設定してください。\n%s/forgot-password%s",
			appOrigin, signature),
	}
}
