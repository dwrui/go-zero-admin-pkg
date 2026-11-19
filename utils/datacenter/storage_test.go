package datacenter_test

import (
	"testing"

	"datacenter/pkg/datacenter"
)

func TestBuildFilename(t *testing.T) {
	name := datacenter.BuildFilename("{biz_type}_{date}_{task_no}", datacenter.FilenameMeta{
		BizType: "warehouse",
		TaskNo:  "EX202607080001",
	})
	if name == "" {
		t.Fatal("empty filename")
	}
}

func TestFileRefRoundTrip(t *testing.T) {
	ref := datacenter.FileRef{
		Backend:  datacenter.BackendRedis,
		Key:      "datacenter:export:result:1",
		Filename: "a.xlsx",
	}
	raw := datacenter.EncodeFileRef(ref)
	parsed, ok := datacenter.ParseFileRef(raw)
	if !ok || parsed.Key != ref.Key {
		t.Fatalf("parse failed: %v %v", parsed, ok)
	}
}
