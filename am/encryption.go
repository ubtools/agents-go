package am

import (
	"bytes"
	"crypto/rand"
	"errors"
	"log"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

type Encryption interface {
	GetKey() []byte
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}

func GetEncryption(key []byte) Encryption {
	if len(key) == 0 {
		return &NoEncryption{}
	}
	return &Chacha20poly1305Encryption{Key: key}
}

type NoEncryption struct {
}

func (enc *NoEncryption) GetKey() []byte {
	return nil
}

func (enc *NoEncryption) Encrypt(data []byte) ([]byte, error) {
	return data, nil
}

func (enc *NoEncryption) Decrypt(data []byte) ([]byte, error) {
	return data, nil
}

type Chacha20poly1305Encryption struct {
	Key []byte
}

func (enc *Chacha20poly1305Encryption) GetKey() []byte {
	return enc.Key
}

func random(size uint) []byte {
	data := make([]byte, size)
	_, err := rand.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

func deriveKey(password []byte, salt []byte) []byte {
	// Use Argon2 as the KDF
	key := argon2.Key(password, salt, 3, 32*1024, 4, 32)
	return key
}

const SALT_SIZE = 16
const NONCE_SIZE = 24

func (enc *Chacha20poly1305Encryption) Encrypt(data []byte) ([]byte, error) {
	salt := random(SALT_SIZE)
	actualKey := deriveKey(enc.Key, salt)
	aead, err := chacha20poly1305.NewX(actualKey)
	if err != nil {
		return nil, err
	}

	nonce := random(NONCE_SIZE)

	encrypted := aead.Seal(nil, nonce, data, nil)
	return bytes.Join([][]byte{salt, nonce, encrypted}, []byte{}), nil
}

func (enc *Chacha20poly1305Encryption) Decrypt(data []byte) ([]byte, error) {
	if len(data) < SALT_SIZE+NONCE_SIZE+1 {
		return nil, errors.New("invalid data")
	}
	salt := data[:SALT_SIZE]
	nonce := data[SALT_SIZE : NONCE_SIZE+SALT_SIZE]
	data = data[NONCE_SIZE+SALT_SIZE:]
	actualKey := deriveKey(enc.Key, salt)
	aead, err := chacha20poly1305.NewX(actualKey)
	if err != nil {
		return nil, err
	}
	decrypted, err := aead.Open(nil, nonce, data, nil)
	return decrypted, err
}
