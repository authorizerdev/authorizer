package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

// These pin that both PEM encodings openssl emits are accepted.
//
// openssl 3.x — the default on macOS and current Linux — writes PKCS#8
// ("BEGIN PRIVATE KEY") from `genrsa`/`genpkey`, and PKIX ("BEGIN PUBLIC KEY")
// from `rsa -pubout`. Only the older PKCS#1 forms were accepted, so keys
// generated the standard way failed — and failed LATE: the server started,
// signup worked, and only token issuance and /.well-known/jwks.json broke, on
// an instance that otherwise looked healthy.

func pemEncode(t *testing.T, typ string, der []byte) string {
	t.Helper()
	return string(pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}))
}

func TestParseRsaPrivateKeyAcceptsBothEncodings(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}

	for _, tc := range []struct {
		name, typ string
		der       []byte
	}{
		{"PKCS#1 (openssl 1.x genrsa)", "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key)},
		{"PKCS#8 (openssl 3.x genrsa)", "PRIVATE KEY", pkcs8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseRsaPrivateKeyFromPemStr(pemEncode(t, tc.typ, tc.der))
			if err != nil {
				t.Fatalf("parse: %v — token issuance would fail on this key", err)
			}
			if !got.Equal(key) {
				t.Fatal("parsed a different key")
			}
		})
	}
}

func TestParseRsaPublicKeyAcceptsBothEncodings(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	pkix, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal pkix: %v", err)
	}

	for _, tc := range []struct {
		name, typ string
		der       []byte
	}{
		{"PKCS#1", "RSA PUBLIC KEY", x509.MarshalPKCS1PublicKey(&key.PublicKey)},
		{"PKIX (openssl rsa -pubout)", "PUBLIC KEY", pkix},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseRsaPublicKeyFromPemStr(pemEncode(t, tc.typ, tc.der))
			if err != nil {
				t.Fatalf("parse: %v — jwks.json would fail on this key", err)
			}
			if !got.Equal(&key.PublicKey) {
				t.Fatal("parsed a different key")
			}
		})
	}
}

func TestParseEcdsaPrivateKeyAcceptsBothEncodings(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	sec1, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal sec1: %v", err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}

	for _, tc := range []struct {
		name, typ string
		der       []byte
	}{
		{"SEC1", "EC PRIVATE KEY", sec1},
		{"PKCS#8 (openssl genpkey)", "PRIVATE KEY", pkcs8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseEcdsaPrivateKeyFromPemStr(pemEncode(t, tc.typ, tc.der))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !got.Equal(key) {
				t.Fatal("parsed a different key")
			}
		})
	}
}

// A non-matching key type must be a clear error, not a panic from a bad cast.
func TestParseRsaPrivateKeyRejectsAnEcdsaKey(t *testing.T) {
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(ec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := ParseRsaPrivateKeyFromPemStr(pemEncode(t, "PRIVATE KEY", pkcs8)); err == nil {
		t.Fatal("an ECDSA key must not parse as RSA")
	}
}
