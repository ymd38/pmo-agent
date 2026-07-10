package infra

import "golang.org/x/crypto/bcrypt"

// PasswordHasher は bcrypt によるパスワードのハッシュ化・照合。
type PasswordHasher struct{}

func NewPasswordHasher() *PasswordHasher { return &PasswordHasher{} }

// Hash は平文パスワードを bcrypt ハッシュへ変換する。
func (PasswordHasher) Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare はハッシュと平文を照合する。一致すれば nil。
func (PasswordHasher) Compare(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
