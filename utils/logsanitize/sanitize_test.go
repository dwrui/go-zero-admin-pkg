package logsanitize

import (
	"strings"
	"testing"
)

func TestSanitizeJSONString_masksPassword(t *testing.T) {
	raw := `{"username":"u1","password":"secret123"}`
	out := SanitizeJSONString(raw, 0)
	if out == raw {
		t.Fatal("password should be masked")
	}
	if strings.Contains(out, "secret123") {
		t.Fatalf("leaked password: %s", out)
	}
}

func TestSanitizeJSONString_truncates(t *testing.T) {
	raw := `{"data":"` + string(make([]byte, 100)) + `"}`
	out := SanitizeJSONString(raw, 20)
	if len(out) <= 20 {
		return
	}
	if !strings.Contains(out, "truncated") {
		t.Fatalf("expected truncated marker: %s", out)
	}
}

func TestShouldSkipBody(t *testing.T) {
	if !ShouldSkipBody("/datacenter/files/upload", []string{"/datacenter/files/"}) {
		t.Fatal("should skip upload path")
	}
	if ShouldSkipBody("/api/admin/user/info", []string{"/user/login"}) {
		t.Fatal("should not skip unrelated path")
	}
}
