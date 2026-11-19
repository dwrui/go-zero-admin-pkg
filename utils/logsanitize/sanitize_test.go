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

func TestSanitizeJSONString_oversizeReturnsEmptyObject(t *testing.T) {
	raw := `{"data":"` + strings.Repeat("a", 100) + `"}`
	out := SanitizeJSONString(raw, 20)
	if out != "{}" {
		t.Fatalf("expected empty object for oversize json, got: %s", out)
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
