package bootstrap

import (
	"testing"

	"github.com/zeromicro/go-zero/core/discov"
)

func TestEtcdConfigured(t *testing.T) {
	if EtcdConfigured(discovEmpty()) {
		t.Fatal("empty etcd should not be configured")
	}
	c := discovSample()
	if !EtcdConfigured(c) {
		t.Fatal("hosts+key should be configured")
	}
	c.Key = ""
	if EtcdConfigured(c) {
		t.Fatal("empty key should fail")
	}
}

func discovEmpty() discov.EtcdConf {
	return discov.EtcdConf{}
}

func discovSample() discov.EtcdConf {
	return discov.EtcdConf{Hosts: []string{"127.0.0.1:2379"}, Key: "demo.rpc"}
}
