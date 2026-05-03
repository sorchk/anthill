package tunnel

import (
    "crypto/rand"
    "fmt"
    "io"

    "golang.org/x/crypto/chacha20poly1305"
)

type E2EEncryptor struct {
    key        []byte
    nonceCounter uint64
}

func NewE2EEncryptor(key []byte) (*E2EEncryptor, error) {
    if len(key) != chacha20poly1305.KeySize {
        return nil, fmt.Errorf("invalid key size: need %d, got %d", chacha20poly1305.KeySize, len(key))
    }
    return &E2EEncryptor{key: key}, nil
}

func (e *E2EEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
    nonce := make([]byte, 12)
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    aead, err := chacha20poly1305.NewX(e.key)
    if err != nil {
        return nil, err
    }

    ciphertext := aead.Seal(nil, nonce, plaintext, nil)

    result := make([]byte, 12+len(ciphertext))
    copy(result[:12], nonce)
    copy(result[12:], ciphertext)

    return result, nil
}

func (e *E2EEncryptor) Decrypt(data []byte) ([]byte, error) {
    if len(data) < 12+chacha20poly1305.Overhead {
        return nil, fmt.Errorf("data too short")
    }

    nonce := data[:12]
    ciphertext := data[12:]

    aead, err := chacha20poly1305.NewX(e.key)
    if err != nil {
        return nil, err
    }

    plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, err
    }

    return plaintext, nil
}

func GenerateKey() ([]byte, error) {
    key := make([]byte, chacha20poly1305.KeySize)
    if _, err := rand.Read(key); err != nil {
        return nil, err
    }
    return key, nil
}