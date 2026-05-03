package tunnel

import (
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/curve25519"
)

type KeyPair struct {
	PrivateKey [32]byte
	PublicKey  [32]byte
}

func GenerateKeyPair() (*KeyPair, error) {
	var privateKey [32]byte
	var publicKey [32]byte

	_, err := rand.Read(privateKey[:])
	if err != nil {
		return nil, err
	}

	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

func ComputeSharedSecret(privateKey, peerPublicKey [32]byte) ([]byte, error) {
	var sharedSecret [32]byte

	out, err := curve25519.X25519(privateKey[:], peerPublicKey[:])
	if err != nil {
		return nil, err
	}

	copy(sharedSecret[:], out)

	if sharedSecret == [32]byte{} {
		return nil, errors.New("invalid shared secret")
	}

	return sharedSecret[:], nil
}