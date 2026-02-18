package crypto

import (
	"bytes"
	"testing"
)

func TestDeriveVaultKey_Deterministic(t *testing.T) {
	password := []byte("hunter2")
	secret := []byte("0123456789abcdef")
	salt := []byte("saltsaltsaltsaltsaltsaltsaltsalt")

	k1 := DeriveVaultKey(password, secret, salt)
	k2 := DeriveVaultKey(password, secret, salt)

	if !bytes.Equal(k1, k2) {
		t.Fatal("same inputs should produce same key")
	}
	if len(k1) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(k1))
	}
}

func TestDeriveVaultKey_DifferentPassword(t *testing.T) {
	secret := []byte("0123456789abcdef")
	salt := []byte("saltsaltsaltsaltsaltsaltsaltsalt")

	k1 := DeriveVaultKey([]byte("password1"), secret, salt)
	k2 := DeriveVaultKey([]byte("password2"), secret, salt)

	if bytes.Equal(k1, k2) {
		t.Fatal("different passwords should produce different keys")
	}
}

func TestDeriveVaultKey_DifferentSecretKey(t *testing.T) {
	password := []byte("hunter2")
	salt := []byte("saltsaltsaltsaltsaltsaltsaltsalt")

	k1 := DeriveVaultKey(password, []byte("0123456789abcdef"), salt)
	k2 := DeriveVaultKey(password, []byte("fedcba9876543210"), salt)

	if bytes.Equal(k1, k2) {
		t.Fatal("different secret keys should produce different keys")
	}
}

func TestGenerateSalt_Length(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	if len(salt) != 32 {
		t.Fatalf("expected 32-byte salt, got %d", len(salt))
	}
}

func TestGenerateSecretKey_Length(t *testing.T) {
	key, err := GenerateSecretKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 16 {
		t.Fatalf("expected 16-byte key, got %d", len(key))
	}
}

func TestGenerateSecretKey_Unique(t *testing.T) {
	k1, _ := GenerateSecretKey()
	k2, _ := GenerateSecretKey()
	if bytes.Equal(k1, k2) {
		t.Fatal("two generated keys should differ")
	}
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := make([]byte, 32)
	copy(key, "test-key-32-bytes-long-padding!!")
	plaintext := []byte("hello, vault")

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncrypt_DifferentNonces(t *testing.T) {
	key := make([]byte, 32)
	copy(key, "test-key-32-bytes-long-padding!!")
	plaintext := []byte("same content")

	e1, _ := Encrypt(key, plaintext)
	e2, _ := Encrypt(key, plaintext)

	if bytes.Equal(e1, e2) {
		t.Fatal("two encryptions of same plaintext should differ (random nonce)")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	copy(key, "test-key-32-bytes-long-padding!!")

	encrypted, _ := Encrypt(key, []byte("secret"))
	encrypted[len(encrypted)-1] ^= 0xff // flip last byte

	_, err := Decrypt(key, encrypted)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	copy(key1, "key-one-32-bytes-long-padding!!!")
	copy(key2, "key-two-32-bytes-long-padding!!!")

	encrypted, _ := Encrypt(key1, []byte("secret"))

	_, err := Decrypt(key2, encrypted)
	if err == nil {
		t.Fatal("expected error for wrong key")
	}
}

func TestEncryptDecryptBase64_Roundtrip(t *testing.T) {
	key := make([]byte, 32)
	copy(key, "test-key-32-bytes-long-padding!!")
	plaintext := []byte("base64 test value")

	encoded, err := EncryptToBase64(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := DecryptFromBase64(key, encoded)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestDeriveSubkey_DifferentCategories(t *testing.T) {
	vaultKey := make([]byte, 32)
	copy(vaultKey, "vault-key-32-bytes-long-padding!")
	salt := []byte("test-salt-16bytes")

	k1, err := DeriveSubkey(vaultKey, salt, "identity")
	if err != nil {
		t.Fatal(err)
	}
	k2, err := DeriveSubkey(vaultKey, salt, "financial")
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(k1, k2) {
		t.Fatal("different categories should produce different subkeys")
	}
}

func TestDeriveSubkey_Deterministic(t *testing.T) {
	vaultKey := make([]byte, 32)
	copy(vaultKey, "vault-key-32-bytes-long-padding!")
	salt := []byte("test-salt-16bytes")

	k1, _ := DeriveSubkey(vaultKey, salt, "identity")
	k2, _ := DeriveSubkey(vaultKey, salt, "identity")

	if !bytes.Equal(k1, k2) {
		t.Fatal("same inputs should produce same subkey")
	}
	if len(k1) != 32 {
		t.Fatalf("expected 32-byte subkey, got %d", len(k1))
	}
}

func TestHashSecretKey_Deterministic(t *testing.T) {
	key := []byte("0123456789abcdef")

	h1 := HashSecretKey(key)
	h2 := HashSecretKey(key)

	if !bytes.Equal(h1, h2) {
		t.Fatal("same input should produce same hash")
	}
	if len(h1) != 32 {
		t.Fatalf("expected 32-byte hash, got %d", len(h1))
	}
}
