package manager

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func pemEncode(blockType string, der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
}
func pemEncodeECKey(key *ecdsa.PrivateKey) []byte {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		panic(err)
	}
	return pemEncode("EC PRIVATE KEY", der)
}

func TestPEMHelper(t *testing.T) {
	if len(pemEncode("TEST", []byte("value"))) == 0 {
		t.Fatal("expected PEM output")
	}
}
