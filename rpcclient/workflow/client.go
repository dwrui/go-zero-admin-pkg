package workflow

import (
	"context"
	"sync"
	"time"

	wfpb "github.com/dwrui/go-zero-admin-pkg/rpcclient/workflow/workflow"

	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

var (
	defaultClient *Client
	once          sync.Once
)

// Client 工作流 RPC 客户端，封装 InstanceService。
type Client struct {
	instance wfpb.InstanceServiceClient
	conn     zrpc.Client
}

// Config 工作流 etcd 配置。
type Config struct {
	Etcd    discov.EtcdConf
	Timeout int64
}

// Init 初始化全局工作流客户端。
func Init(c Config) error {
	var initErr error
	once.Do(func() {
		timeout := c.Timeout
		if timeout <= 0 {
			timeout = 50000
		}
		conn, err := zrpc.NewClient(zrpc.RpcClientConf{
			Etcd:          c.Etcd,
			NonBlock:      true,
			Timeout:       timeout,
			KeepaliveTime: 10 * time.Second,
		})
		if err != nil {
			initErr = err
			return
		}
		defaultClient = &Client{
			instance: wfpb.NewInstanceServiceClient(conn.Conn()),
			conn:     conn,
		}
	})
	return initErr
}

// MustInit 初始化工作流客户端，失败时 panic。
func MustInit(c Config) {
	if err := Init(c); err != nil {
		panic(err)
	}
}

// GetClient 获取全局工作流客户端。
func GetClient() *Client {
	if defaultClient == nil {
		panic("workflow client not initialized, please call Init first")
	}
	return defaultClient
}

// NewClient 创建独立的工作流客户端（不写入全局单例）。
func NewClient(c Config) (*Client, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 50000
	}
	conn, err := zrpc.NewClient(zrpc.RpcClientConf{
		Etcd:          c.Etcd,
		NonBlock:      true,
		Timeout:       timeout,
		KeepaliveTime: 10 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		instance: wfpb.NewInstanceServiceClient(conn.Conn()),
		conn:     conn,
	}, nil
}

func (c *Client) StartInstance(ctx context.Context, in *wfpb.StartInstanceRequest, opts ...grpc.CallOption) (*wfpb.StartInstanceResponse, error) {
	return c.instance.StartInstance(ctx, in, opts...)
}

func (c *Client) RawInstanceClient() wfpb.InstanceServiceClient {
	return c.instance
}
