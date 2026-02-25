package security

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
)

const (
	Algorithm = "aes-256-gcm"
	KeyLength = 32
	IVLength  = 16
	TagLength = 16
)

var fixedIV = []byte("ThisIsAFixedIV16")

func stringKeyToBuffer(keyString string) []byte {
	keyBytes := []byte(keyString)
	keyBuffer := make([]byte, KeyLength)

	if len(keyBytes) < KeyLength {
		copy(keyBuffer, keyBytes)
	} else {
		copy(keyBuffer, keyBytes[:KeyLength])
	}

	return keyBuffer
}

func Encrypt(plaintext string, keyString string) (string, error) {
	key := stringKeyToBuffer(keyString)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCMWithNonceSize(block, IVLength)
	if err != nil {
		return "", err
	}

	ciphertext := aesgcm.Seal(nil, fixedIV, []byte(plaintext), nil)

	output := append(fixedIV, ciphertext...)
	return base64.StdEncoding.EncodeToString(output), nil
}

func Decrypt(encryptedText string, keyString string) (string, error) {
	key := stringKeyToBuffer(keyString)

	data, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	if len(data) < IVLength+TagLength {
		return "", errors.New("invalid encrypted data format: too short")
	}

	iv := data[:IVLength]
	ciphertextWithTag := data[IVLength:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCMWithNonceSize(block, IVLength)
	if err != nil {
		return "", err
	}

	plaintext, err := aesgcm.Open(nil, iv, ciphertextWithTag, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
