package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// tokenBytes 是会话 token 的随机字节数（256 bit）。
const tokenBytes = 32

// NewToken 生成一个新的会话 token（43 字符的 base64url）。
// 明文 token 只下发到 Cookie，不落库。
func NewToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken 返回 token 的 sha256 hex（64 字符），用于入库与查询。
//
// 存哈希而非明文：数据库被拖走也换不出可用会话。
// 这里用裸 sha256 而非 bcrypt——token 是 256 bit 高熵随机串，
// 不存在字典攻击面，不需要慢哈希。
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
