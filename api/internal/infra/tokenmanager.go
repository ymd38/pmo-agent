package infra

// TokenManager は不透明トークンの生成とハッシュ化をまとめる（usecase.TokenManager 実装）。
type TokenManager struct{}

func NewTokenManager() *TokenManager { return &TokenManager{} }

func (TokenManager) Generate() (string, error) { return GenerateOpaqueToken() }

func (TokenManager) Hash(plain string) string { return HashToken(plain) }
