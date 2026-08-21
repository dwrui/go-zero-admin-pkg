package authcenter

import (
	"context"
	"sync"

	auth2 "github.com/dwrui/go-zero-admin-pkg/rpcclient/authcenter/auth"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/zrpc"
)

var (
	defaultClient *AuthClient
	once          sync.Once
)

type AuthClient struct {
	client auth2.AuthServiceClient
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
			timeout = 5000
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
		defaultClient = &AuthClient{
			client: auth2.NewAuthServiceClient(conn.Conn()),
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

func GetClient() *AuthClient {
	if defaultClient == nil {
		panic("authcenter client not initialized, please call Init first")
	}
	return defaultClient
}

// TryGetClient 返回已初始化的客户端；未初始化时返回 nil。
func TryGetClient() *AuthClient {
	return defaultClient
}

func (c *AuthClient) CheckToken(ctx context.Context, token string, permission string) (string, error) {
	resp, err := c.client.CheckToken(ctx, &auth2.CheckTokenRequest{
		Token:      token,
		Permission: permission,
	})
	if err != nil {
		return "", err
	}
	return resp.GetNewToken(), nil
}

func (c *AuthClient) GetFieldAccess(ctx context.Context, userId uint64, tableName string) (*auth2.GetFieldAccessResponse, error) {
	resp, err := c.client.GetFieldAccess(ctx, &auth2.GetFieldAccessRequest{
		UserId:    userId,
		TableName: tableName,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthClient) GetUserPermission(ctx context.Context, userId uint64, tableName string, businessId uint64) (*auth2.GetUserPermissionResponse, error) {
	resp, err := c.client.GetUserPermission(ctx, &auth2.GetUserPermissionRequest{
		UserId:     userId,
		TableName:  tableName,
		BusinessId: businessId,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *AuthClient) RawClient() auth2.AuthServiceClient {
	return c.client
}

func CheckToken(ctx context.Context, token string, permission string) (string, error) {
	return GetClient().CheckToken(ctx, token, permission)
}

func GetFieldAccess(ctx context.Context, userId uint64, tableName string) (*auth2.GetFieldAccessResponse, error) {
	return GetClient().GetFieldAccess(ctx, userId, tableName)
}

func GetUserPermission(ctx context.Context, userId uint64, tableName string, businessId uint64) (*auth2.GetUserPermissionResponse, error) {
	return GetClient().GetUserPermission(ctx, userId, tableName, businessId)
}
