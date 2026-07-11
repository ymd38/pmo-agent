package domain

import "errors"

// ドメイン共通エラー。usecase/handler は errors.Is で分類して HTTP ステータスへ写像する。
var (
	ErrNotFound           = errors.New("リソースが見つかりません")
	ErrInvalidCredentials = errors.New("メールアドレスまたはパスワードが正しくありません")
	ErrTokenInvalid       = errors.New("トークンが無効か、有効期限が切れています")
	ErrTokenReuse         = errors.New("トークンが再利用されました")
	ErrConflict           = errors.New("リソースが競合しています")
	ErrValidation         = errors.New("入力内容が正しくありません")
)
