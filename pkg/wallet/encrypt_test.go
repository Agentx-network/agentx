package wallet

import (
	"bytes"
	"testing"
)

// H2 (audit): verify the passphrase-aware encryption is correct AND backward
// compatible — a wallet encrypted before a passphrase was set must still
// decrypt afterward via the machine-key fallback.

func TestEncryptDecrypt_NoPassphrase(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AGENTX_WALLET_PASSPHRASE", "")

	secret := []byte("0xdeadbeefprivkey")
	ct, err := EncryptKey(secret)
	if err != nil {
		t.Fatalf("EncryptKey: %v", err)
	}
	pt, err := DecryptKey(ct)
	if err != nil {
		t.Fatalf("DecryptKey: %v", err)
	}
	if !bytes.Equal(pt, secret) {
		t.Errorf("roundtrip mismatch: got %q", pt)
	}
}

func TestEncryptDecrypt_WithPassphrase(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AGENTX_WALLET_PASSPHRASE", "correct horse battery staple")

	secret := []byte("0xprivkeywithpassphrase")
	ct, err := EncryptKey(secret)
	if err != nil {
		t.Fatalf("EncryptKey: %v", err)
	}
	pt, err := DecryptKey(ct)
	if err != nil {
		t.Fatalf("DecryptKey: %v", err)
	}
	if !bytes.Equal(pt, secret) {
		t.Errorf("roundtrip mismatch: got %q", pt)
	}
}

func TestDecrypt_LegacyWalletStillReadableAfterPassphraseSet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Encrypt with NO passphrase (legacy machine-key wallet).
	t.Setenv("AGENTX_WALLET_PASSPHRASE", "")
	secret := []byte("0xlegacykey")
	ct, err := EncryptKey(secret)
	if err != nil {
		t.Fatalf("EncryptKey: %v", err)
	}

	// User later sets a passphrase. The legacy wallet must still decrypt via the
	// machine-key fallback (no data loss / no lockout).
	t.Setenv("AGENTX_WALLET_PASSPHRASE", "added-later")
	pt, err := DecryptKey(ct)
	if err != nil {
		t.Fatalf("legacy wallet failed to decrypt after passphrase set: %v", err)
	}
	if !bytes.Equal(pt, secret) {
		t.Errorf("legacy roundtrip mismatch: got %q", pt)
	}
}

func TestDecrypt_WrongPassphraseFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	t.Setenv("AGENTX_WALLET_PASSPHRASE", "right-pass")
	ct, err := EncryptKey([]byte("0xsecret"))
	if err != nil {
		t.Fatalf("EncryptKey: %v", err)
	}

	// A different passphrase must NOT decrypt it (machine-key fallback also fails
	// because it was sealed with the scrypt key).
	t.Setenv("AGENTX_WALLET_PASSPHRASE", "wrong-pass")
	if _, err := DecryptKey(ct); err == nil {
		t.Error("expected decryption to fail with the wrong passphrase")
	}
}
