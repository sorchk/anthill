package tunnel

import (
	"crypto/hkdf"
	"crypto/sha256"
	"fmt"
)

const (
	KeySize     = 32
	DerivedLen  = 48
)

var defaultSalt = sha256.Sum256([]byte("anthill-tunnel-e2e-v1"))

type DerivedKeys struct {
	EncryptKey []byte
}

func DeriveKeys(sharedSecret []byte, tunnelID string) (*DerivedKeys, error) {
	if len(sharedSecret) != 32 {
		return nil, fmt.Errorf("invalid shared secret length: %d", len(sharedSecret))
	}

	info := "tunnel:" + tunnelID

	keyMaterial, err := hkdf.Key(sha256.New, sharedSecret, defaultSalt[:], info, DerivedLen)
	if err != nil {
		return nil, fmt.Errorf("hkdf key derivation failed: %w", err)
	}

	return &DerivedKeys{
		EncryptKey: keyMaterial[:KeySize],
	}, nil
}