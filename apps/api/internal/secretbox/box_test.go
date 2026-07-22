package secretbox

import (
	"bytes"
	"testing"
)

func TestBoxEncryptsDecryptsAndFingerprintsCredential(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	box, err := New(key)
	if err != nil {
		t.Fatalf("创建密钥保险箱失败：%v", err)
	}
	plaintext := []byte("sk-upstream-secret")
	ciphertext, err := box.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("加密失败：%v", err)
	}
	if bytes.Contains(ciphertext, plaintext) {
		t.Fatal("密文包含密钥正文")
	}
	decrypted, err := box.Decrypt(ciphertext)
	if err != nil || !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("解密结果=%q，错误=%v", decrypted, err)
	}
	fingerprint := box.Fingerprint(plaintext)
	if len(fingerprint) != 64 || fingerprint != box.Fingerprint(plaintext) {
		t.Fatalf("指纹不稳定：%q", fingerprint)
	}
	if fingerprint == box.Fingerprint([]byte("sk-other-secret")) {
		t.Fatal("不同密钥产生相同指纹")
	}
}

func TestBoxRejectsInvalidKeyAndTamperedCiphertext(t *testing.T) {
	if _, err := New([]byte("short")); err == nil {
		t.Fatal("错误长度的主密钥应被拒绝")
	}
	box, err := New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := box.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext[len(ciphertext)-1] ^= 0xff
	if _, err := box.Decrypt(ciphertext); err == nil {
		t.Fatal("被篡改的密文应被拒绝")
	}
}
