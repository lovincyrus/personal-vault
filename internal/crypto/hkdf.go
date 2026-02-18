package crypto

import (
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// DeriveSubkey derives a 256-bit subkey from the vault key for a category.
// Uses HKDF-SHA256 with the vault salt and category name as info.
func DeriveSubkey(vaultKey, salt []byte, category string) ([]byte, error) {
	r := hkdf.New(sha256.New, vaultKey, salt, []byte(category))
	subkey := make([]byte, keyLen)
	if _, err := io.ReadFull(r, subkey); err != nil {
		return nil, fmt.Errorf("deriving subkey for %s: %w", category, err)
	}
	return subkey, nil
}
