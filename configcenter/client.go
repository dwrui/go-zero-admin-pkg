package configcenter

import (
	"configcenter/configcenter"
	"context"
	"sync"

	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/zrpc"
)

var (
	defaultClient *ConfigClient
	once          sync.Once
)

type ConfigClient struct {
	client configcenter.ConfigApiServiceClient
	conn   zrpc.Client
}

type Config struct {
	Etcd    discov.EtcdConf
	Timeout int64
}

func Init(c Config) error {
	var initErr error
	once.Do(func() {
		timeout := c.Timeout
		if timeout <= 0 {
			timeout = 3000
		}
		conn, err := zrpc.NewClient(zrpc.RpcClientConf{
			Etcd:     c.Etcd,
			NonBlock: true,
			Timeout:  timeout,
		})
		if err != nil {
			initErr = err
			return
		}
		defaultClient = &ConfigClient{
			client: configcenter.NewConfigApiServiceClient(conn.Conn()),
			conn:   conn,
		}
	})
	return initErr
}

func MustInit(c Config) {
	if err := Init(c); err != nil {
		panic(err)
	}
}

func GetClient() *ConfigClient {
	if defaultClient == nil {
		panic("configcenter client not initialized, please call Init first")
	}
	return defaultClient
}

func (c *ConfigClient) GetConfig(ctx context.Context, categoryKey string) (map[string]string, error) {
	resp, err := c.client.GetConfig(ctx, &configcenter.GetConfigRequest{
		CategoryKey: categoryKey,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetConfigs(), nil
}

func (c *ConfigClient) GetConfigInfo(ctx context.Context, configType string) (string, error) {
	resp, err := c.client.GetConfigInfo(ctx, &configcenter.GetConfigInfoRequest{
		ConfigType: configType,
	})
	if err != nil {
		return "", err
	}
	return resp.GetConfigValue(), nil
}

func (c *ConfigClient) RawClient() configcenter.ConfigApiServiceClient {
	return c.client
}

func GetConfig(ctx context.Context, categoryKey string) (map[string]string, error) {
	return GetClient().GetConfig(ctx, categoryKey)
}

func GetConfigInfo(ctx context.Context, configType string) (string, error) {
	return GetClient().GetConfigInfo(ctx, configType)
}
