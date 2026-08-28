package openapiauth

import "testing"

func TestSignAndVerify(t *testing.T) {
	secret := "test-secret-key-40chars-abcdefghijklmn"
	ts := "1710000000"
	method := "POST"
	path := "/api/open/v1/demo/list"
	query := "page=1"
	body := []byte(`{"name":"a"}`)
	sig := BuildSignature(secret, ts, method, path, query, body)
	if !VerifySignature(secret, ts, method, path, query, body, sig) {
		t.Fatal("verify failed")
	}
	if VerifySignature(secret, ts, method, path, query, body, "bad") {
		t.Fatal("bad signature should fail")
	}
}

func TestEncryptDecryptSecret(t *testing.T) {
	master := "goza-open-api-master-key-change-me"
	plain := "SKABCDEFGHijklmnop1234567890xyz"
	enc, err := EncryptSecret(master, plain)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecryptSecret(master, enc)
	if err != nil {
		t.Fatal(err)
	}
	if got != plain {
		t.Fatalf("got %s", got)
	}
}
