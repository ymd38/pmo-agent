package infra

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// GenerateOpaqueToken は暗号論的乱数から不透明トークン（平文）を生成する。
// 平文は利用者へ一度だけ渡し、DB にはハッシュのみを保存する。
func GenerateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashToken は平文トークンを SHA-256 でハッシュ化する（保存・照合用）。
func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
