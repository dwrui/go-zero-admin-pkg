package openapiauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

// DefaultMasterKey 默认主密钥（生产务必在 yaml OpenApiAuth.MasterKey 覆盖）
const DefaultMasterKey = "goza-open-api-master-key-change-me"

// HashSecret SK 的 SHA256 hex（审计/比对用）
func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func deriveKey(masterKey string) []byte {
	sum := sha256.Sum256([]byte(masterKey))
	return sum[:]
}

// EncryptSecret 使用主密钥 AES-GCM 加密 SK，返回 base64
func EncryptSecret(masterKey, secret string) (string, error) {
	if masterKey == "" {
		return "", errors.New("OpenApiAuth.MasterKey 未配置")
	}
	block, err := aes.NewCipher(deriveKey(masterKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nonce, nonce, []byte(secret), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

// DecryptSecret 解密 SK 密文
func DecryptSecret(masterKey, secretEnc string) (string, error) {
	if masterKey == "" {
		return "", errors.New("OpenApiAuth.MasterKey 未配置")
	}
	raw, err := base64.StdEncoding.DecodeString(secretEnc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(deriveKey(masterKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("密文无效")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
