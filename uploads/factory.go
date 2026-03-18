package uploads

import (
	"errors"
	"fmt"
	"sync"

	"github.com/dwrui/go-zero-admin/pkg/uploads/aliyun"
	"github.com/dwrui/go-zero-admin/pkg/uploads/local"
	"github.com/dwrui/go-zero-admin/pkg/uploads/qiniu"
	"github.com/dwrui/go-zero-admin/pkg/uploads/tencent"
	"github.com/dwrui/go-zero-admin/pkg/uploads/types"
)

var (
	instances     = make(map[string]types.Storage)
	instancesLock sync.RWMutex
	defaultName   = "default"
)

func NewStorage(cfg *types.Config) (types.Storage, error) {
	if cfg == nil {
		return nil, errors.New("storage config is required")
	}

	switch cfg.Type {
	case types.StorageLocal:
		if cfg.Local == nil {
			return nil, errors.New("local storage config is required")
		}
		return local.NewLocalStorage(cfg.Local)

	case types.StorageAliyun:
		if cfg.Aliyun == nil {
			return nil, errors.New("aliyun storage config is required")
		}
		return aliyun.NewAliyunStorage(cfg.Aliyun)

	case types.StorageTencent:
		if cfg.Tencent == nil {
			return nil, errors.New("tencent storage config is required")
		}
		return tencent.NewTencentStorage(cfg.Tencent)

	case types.StorageQiniu:
		if cfg.Qiniu == nil {
			return nil, errors.New("qiniu storage config is required")
		}
		return qiniu.NewQiniuStorage(cfg.Qiniu)

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}

func Init(name string, cfg *types.Config) error {
	storage, err := NewStorage(cfg)
	if err != nil {
		return err
	}

	instancesLock.Lock()
	defer instancesLock.Unlock()

	instances[name] = storage
	return nil
}

func InitDefault(cfg *types.Config) error {
	return Init(defaultName, cfg)
}

func Get(name ...string) (types.Storage, error) {
	instanceName := defaultName
	if len(name) > 0 && name[0] != "" {
		instanceName = name[0]
	}

	instancesLock.RLock()
	defer instancesLock.RUnlock()

	storage, ok := instances[instanceName]
	if !ok {
		return nil, fmt.Errorf("storage instance '%s' not found", instanceName)
	}

	return storage, nil
}

func GetDefault() (types.Storage, error) {
	return Get()
}

func MustGet(name ...string) types.Storage {
	storage, err := Get(name...)
	if err != nil {
		panic(err)
	}
	return storage
}

func Close(name ...string) error {
	instanceName := defaultName
	if len(name) > 0 && name[0] != "" {
		instanceName = name[0]
	}

	instancesLock.Lock()
	defer instancesLock.Unlock()

	delete(instances, instanceName)
	return nil
}

func CloseAll() {
	instancesLock.Lock()
	defer instancesLock.Unlock()

	instances = make(map[string]types.Storage)
}

type Manager struct {
	storages map[string]types.Storage
	default_ string
	mu       sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		storages: make(map[string]types.Storage),
		default_: defaultName,
	}
}

func (m *Manager) Add(name string, storage types.Storage, isDefault ...bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.storages[name] = storage

	if len(isDefault) > 0 && isDefault[0] {
		m.default_ = name
	}
}

func (m *Manager) AddFromConfig(name string, cfg *types.Config, isDefault ...bool) error {
	storage, err := NewStorage(cfg)
	if err != nil {
		return err
	}

	m.Add(name, storage, isDefault...)
	return nil
}

func (m *Manager) Get(name ...string) (types.Storage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instanceName := m.default_
	if len(name) > 0 && name[0] != "" {
		instanceName = name[0]
	}

	storage, ok := m.storages[instanceName]
	if !ok {
		return nil, fmt.Errorf("storage instance '%s' not found", instanceName)
	}

	return storage, nil
}

func (m *Manager) GetDefault() (types.Storage, error) {
	return m.Get()
}

func (m *Manager) SetDefault(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.storages[name]; !ok {
		return fmt.Errorf("storage instance '%s' not found", name)
	}

	m.default_ = name
	return nil
}

func (m *Manager) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.storages, name)

	if m.default_ == name {
		m.default_ = defaultName
	}
}

func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.storages))
	for name := range m.storages {
		names = append(names, name)
	}
	return names
}

func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.storages = make(map[string]types.Storage)
	m.default_ = defaultName
}
