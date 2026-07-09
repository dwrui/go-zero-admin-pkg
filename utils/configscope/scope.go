package configscope

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 与 config/config_scope.yaml 结构一致
type Config struct {
	PlatformCategories []string            `yaml:"PlatformCategories"`
	HybridCategories   []string            `yaml:"HybridCategories"`
	PlatformKeys       map[string][]string `yaml:"PlatformKeys"`
}

var global *Config

// Load 加载 YAML（可多次调用，后加载覆盖）
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	c.normalize()
	global = &c
	return &c, nil
}

// MustLoad 启动时加载，失败 panic
func MustLoad(path string) *Config {
	c, err := Load(path)
	if err != nil {
		panic("configscope: load " + path + ": " + err.Error())
	}
	return c
}

// Get 返回已加载配置（未加载则 nil）
func Get() *Config {
	return global
}

func (c *Config) normalize() {
	if c.PlatformKeys == nil {
		c.PlatformKeys = map[string][]string{}
	}
	set := map[string]struct{}{}
	for _, cat := range c.PlatformCategories {
		set[cat] = struct{}{}
	}
	for _, cat := range c.HybridCategories {
		set[cat] = struct{}{}
	}
	_ = set
}

// IsPlatformCategory 整类系统维护
func (c *Config) IsPlatformCategory(categoryKey string) bool {
	if c == nil {
		return false
	}
	for _, k := range c.PlatformCategories {
		if k == categoryKey {
			return true
		}
	}
	return false
}

// IsHybridCategory 租户可覆盖（含系统子项）
func (c *Config) IsHybridCategory(categoryKey string) bool {
	if c == nil {
		return false
	}
	for _, k := range c.HybridCategories {
		if k == categoryKey {
			return true
		}
	}
	return false
}

// IsPlatformKey 类内系统维护 key（仅 item 表）
func (c *Config) IsPlatformKey(categoryKey, configKey string) bool {
	if c == nil {
		return false
	}
	if c.IsPlatformCategory(categoryKey) {
		return true
	}
	keys, ok := c.PlatformKeys[categoryKey]
	if !ok {
		return false
	}
	for _, k := range keys {
		if k == configKey {
			return true
		}
	}
	return false
}

// IsTenantKey 租户可写 key
func (c *Config) IsTenantKey(categoryKey, configKey string) bool {
	if c == nil {
		return false
	}
	if !c.IsHybridCategory(categoryKey) {
		return false
	}
	return !c.IsPlatformKey(categoryKey, configKey)
}

// SystemCategoryKeys 系统配置页展示的分类
func (c *Config) SystemCategoryKeys() []string {
	if c == nil {
		return nil
	}
	out := append([]string{}, c.PlatformCategories...)
	out = append(out, c.HybridCategories...)
	return out
}

// TenantCategoryKeys 租户配置页展示的分类
func (c *Config) TenantCategoryKeys() []string {
	if c == nil {
		return nil
	}
	return append([]string{}, c.HybridCategories...)
}

// ContainsCategory 是否在 scope 定义内
func (c *Config) ContainsCategory(categoryKey string) bool {
	return c.IsPlatformCategory(categoryKey) || c.IsHybridCategory(categoryKey)
}

// PlatformDefault 取 item 平台默认
func PlatformDefault(itemConfigValue, itemDefaultValue string) string {
	if v := strings.TrimSpace(itemConfigValue); v != "" {
		return v
	}
	return strings.TrimSpace(itemDefaultValue)
}
