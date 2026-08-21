package configcenter

import (
	"context"
	"fmt"
	"sync"
	"time"

	configcenter2 "github.com/dwrui/go-zero-admin-pkg/rpcclient/configcenter/configcenter"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/zrpc"
)

const defaultCategoryCacheTTL = 2 * time.Minute

var (
	defaultClient *ConfigClient
	once          sync.Once
)

type ConfigClient struct {
	client      configcenter2.ConfigApiServiceClient
	itemClient  configcenter2.ConfigItemServiceClient
	filesClient configcenter2.FilesServiceClient
	conn        zrpc.Client
	valueCache  *collection.Cache
}

type Config struct {
	Etcd         discov.EtcdConf
	Timeout      int64
	CacheTTL     time.Duration // 分类配置本地缓存 TTL；0 使用默认 2 分钟；<0 关闭缓存
	DisableCache bool
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
		client := &ConfigClient{
			client:      configcenter2.NewConfigApiServiceClient(conn.Conn()),
			itemClient:  configcenter2.NewConfigItemServiceClient(conn.Conn()),
			filesClient: configcenter2.NewFilesServiceClient(conn.Conn()),
			conn:        conn,
		}
		if !c.DisableCache && c.CacheTTL != -1 {
			ttl := c.CacheTTL
			if ttl <= 0 {
				ttl = defaultCategoryCacheTTL
			}
			valueCache, cacheErr := collection.NewCache(ttl)
			if cacheErr == nil {
				client.valueCache = valueCache
			}
		}
		defaultClient = client
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

// TryGetClient 返回已初始化的客户端；未初始化时返回 nil。
func TryGetClient() *ConfigClient {
	return defaultClient
}

func (c *ConfigClient) GetConfig(ctx context.Context, categoryKey string) (map[string]string, error) {
	return c.GetCategoryValues(ctx, 0, categoryKey)
}

func (c *ConfigClient) GetCategoryValues(ctx context.Context, businessId int64, categoryKey string) (map[string]string, error) {
	cacheKey := categoryCacheKey(categoryKey, businessId)
	if c.valueCache != nil {
		if v, ok := c.valueCache.Get(cacheKey); ok {
			if m, ok := v.(map[string]string); ok {
				return cloneStringMap(m), nil
			}
		}
	}
	resp, err := c.itemClient.GetValue(ctx, &configcenter2.GetConfigValueRequest{
		CategoryKey: categoryKey,
		BusinessId:  businessId,
	})
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		if item == nil {
			continue
		}
		out[item.GetConfigKey()] = item.GetConfigValue()
	}
	if c.valueCache != nil {
		c.valueCache.Set(cacheKey, cloneStringMap(out))
	}
	return out, nil
}

func (c *ConfigClient) GetConfigInfo(ctx context.Context, configType string) (string, error) {
	resp, err := c.client.GetConfigInfo(ctx, &configcenter2.GetConfigInfoRequest{
		ConfigType: configType,
	})
	if err != nil {
		return "", err
	}
	return resp.GetConfigValue(), nil
}

func (c *ConfigClient) RawClient() configcenter2.ConfigApiServiceClient {
	return c.client
}

func (c *ConfigClient) RawItemClient() configcenter2.ConfigItemServiceClient {
	return c.itemClient
}

func (c *ConfigClient) DelFiles(ctx context.Context, ids []uint64) error {
	if c == nil || c.filesClient == nil || len(ids) == 0 {
		return nil
	}
	_, err := c.filesClient.DelFile(ctx, &configcenter2.DelFileRequest{Ids: ids})
	return err
}

// UploadFileParams 上传文件到 configcenter FilesService（OSS 在 configcenter 侧配置）。
type UploadFileParams struct {
	BusinessID  uint64
	Filename    string
	ContentType string
	Data        []byte
	StorageName string
	IsCommon    bool
}

// UploadFileResult 上传结果。
type UploadFileResult struct {
	ID  uint64
	URL string
}

// UploadFile 上传二进制内容并返回文件 ID 与访问 URL。
func (c *ConfigClient) UploadFile(ctx context.Context, p UploadFileParams) (*UploadFileResult, error) {
	if c == nil || c.filesClient == nil {
		return nil, fmt.Errorf("configcenter files client not ready")
	}
	if len(p.Data) == 0 {
		return nil, fmt.Errorf("upload content is empty")
	}
	resp, err := c.filesClient.UploadFile(ctx, &configcenter2.UploadFileRequest{
		Content:     p.Data,
		Filename:    p.Filename,
		ContentType: p.ContentType,
		Size:        int64(len(p.Data)),
		BusinessId:  p.BusinessID,
		StorageName: p.StorageName,
		IsCommon:    p.IsCommon,
	})
	if err != nil {
		return nil, err
	}
	return &UploadFileResult{
		ID:  resp.GetId(),
		URL: resp.GetUrl(),
	}, nil
}

func GetConfig(ctx context.Context, categoryKey string) (map[string]string, error) {
	return GetClient().GetConfig(ctx, categoryKey)
}

func GetCategoryValues(ctx context.Context, businessId int64, categoryKey string) (map[string]string, error) {
	return GetClient().GetCategoryValues(ctx, businessId, categoryKey)
}

func GetConfigInfo(ctx context.Context, configType string) (string, error) {
	return GetClient().GetConfigInfo(ctx, configType)
}

func categoryCacheKey(categoryKey string, businessId int64) string {
	return fmt.Sprintf("%s:%d", categoryKey, businessId)
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
