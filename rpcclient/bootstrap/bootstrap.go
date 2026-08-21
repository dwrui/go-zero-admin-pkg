package bootstrap

import (
	"strings"

	"github.com/dwrui/go-zero-admin-pkg/rpcclient/authcenter"
	"github.com/dwrui/go-zero-admin-pkg/rpcclient/configcenter"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
)

// EtcdConfigured 判断 Etcd 服务发现是否已配置（Hosts + Key 均非空）。
func EtcdConfigured(c discov.EtcdConf) bool {
	return len(c.Hosts) > 0 && strings.TrimSpace(c.Key) != ""
}

// RpcTimeouts 各 RPC 客户端超时（毫秒）；0 表示使用各包默认值。
type RpcTimeouts struct {
	Auth         int64
	ConfigCenter int64
}

// OptionalClients 可选 RPC 客户端；未配置 Etcd 或连接失败时为 nil。
type OptionalClients struct {
	Auth   *authcenter.AuthClient
	Config *configcenter.ConfigClient
}

// InitOptionalClients 按 Etcd 配置安全初始化 auth / configcenter 客户端，失败仅打日志不 panic。
func InitOptionalClients(authEtcd, configEtcd discov.EtcdConf, timeouts RpcTimeouts) OptionalClients {
	var out OptionalClients
	if EtcdConfigured(authEtcd) {
		if err := authcenter.Init(authcenter.Config{
			Etcd:    authEtcd,
			Timeout: timeouts.Auth,
		}); err != nil {
			logx.Errorf("authcenter rpc init failed: %v", err)
		} else {
			out.Auth = authcenter.TryGetClient()
		}
	} else {
		logx.Infof("authcenter etcd not configured, skip rpc client")
	}
	if EtcdConfigured(configEtcd) {
		if err := configcenter.Init(configcenter.Config{
			Etcd:    configEtcd,
			Timeout: timeouts.ConfigCenter,
		}); err != nil {
			logx.Errorf("configcenter rpc init failed: %v", err)
		} else {
			out.Config = configcenter.TryGetClient()
		}
	} else {
		logx.Infof("configcenter etcd not configured, skip rpc client")
	}
	return out
}
