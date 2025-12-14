package provider

import "sync"

// Registry Provider 注册表接口
type Registry interface {
	Register(p Provider)
	Get(name string) (Provider, bool)
	GetByModel(model string) (Provider, bool)
	List() []Provider
}

// registry Provider 注册表实现
type registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	models    map[string]string // model -> provider name
}

// NewRegistry 创建注册表
func NewRegistry() Registry {
	return &registry{
		providers: make(map[string]Provider),
		models:    make(map[string]string),
	}
}

// Register 注册 Provider
func (r *registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[p.Name()] = p
	for _, model := range p.SupportedModels() {
		r.models[model] = p.Name()
	}
}

// Get 根据名称获取 Provider
func (r *registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	return p, ok
}

// GetByModel 根据模型获取 Provider
func (r *registry) GetByModel(model string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	name, ok := r.models[model]
	if !ok {
		return nil, false
	}
	return r.providers[name], true
}

// List 列出所有 Provider
func (r *registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		list = append(list, p)
	}
	return list
}
