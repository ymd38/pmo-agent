package domain

import "errors"

// ドメイン共通エラー。usecase/handler は errors.Is で分類して HTTP ステータスへ写像する。
var (
	ErrNotFound           = errors.New("リソースが見つかりません")
	ErrInvalidCredentials = errors.New("メールアドレスまたはパスワードが正しくありません")
	ErrInactiveUser       = errors.New("このアカウントは無効化されています")
	ErrNotActivated       = errors.New("このアカウントはまだ有効化されていません")
	ErrTokenInvalid       = errors.New("トークンが無効か、有効期限が切れています")
	ErrTokenReuse         = errors.New("トークンが再利用されました")
	ErrConflict           = errors.New("リソースが競合しています")
	ErrValidation         = errors.New("入力内容が正しくありません")
	ErrForbidden          = errors.New("この操作を行う権限がありません")
)
