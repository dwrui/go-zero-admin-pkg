package gvalid

import "testing"

func TestParseAndBuildValidateStructTag(t *testing.T) {
	raw := `{"tags":["chinaMobile","max=11"],"messages":{"chinaMobile":"手机号不对"}}`
	tag := BuildValidateStructTag(1, raw)
	if tag == "" {
		t.Fatal("expected validate tag")
	}
	if !containsAll(tag, "required", "chinaMobile", "max=11") {
		t.Fatalf("unexpected tag: %s", tag)
	}
}

func TestBuildValidateStructTagOmitempty(t *testing.T) {
	raw := `{"tags":["email"]}`
	tag := BuildValidateStructTag(0, raw)
	if !containsAll(tag, "omitempty", "email") {
		t.Fatalf("expected omitempty, got %s", tag)
	}
}

func TestRegexpTagBase64(t *testing.T) {
	raw := `{"regex":{"pattern":"^a,b$","message":"err"}}`
	tags := MergeValidateTags(0, raw)
	found := false
	for _, tg := range tags {
		if len(tg) > 7 && tg[:7] == "regexp=" {
			found = true
		}
	}
	if !found {
		t.Fatalf("regexp tag not found: %v", tags)
	}
}

func TestSuggestDefaultValidateRules(t *testing.T) {
	r := SuggestDefaultValidateRules("mobile", "手机号", "text")
	if len(r.Tags) != 1 || r.Tags[0] != "chinaMobile" {
		t.Fatalf("unexpected suggest: %v", r.Tags)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
