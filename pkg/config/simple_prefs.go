package config

import (
	"encoding/json"
	"os"
	"sync"
)

type SimplePreferences struct {
	data map[string]interface{}
	path string
	mu   sync.RWMutex
}

func NewSimplePreferences(path string) *SimplePreferences {
	p := &SimplePreferences{
		data: make(map[string]interface{}),
		path: path,
	}
	p.load()
	return p
}

func (p *SimplePreferences) load() {
	p.mu.Lock()
	defer p.mu.Unlock()
	file, err := os.ReadFile(p.path)
	if err == nil {
		json.Unmarshal(file, &p.data)
	}
}

func (p *SimplePreferences) save() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	file, _ := json.MarshalIndent(p.data, "", "  ")
	os.WriteFile(p.path, file, 0644)
}

func (p *SimplePreferences) String(key string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if v, ok := p.data[key].(string); ok {
		return v
	}
	return ""
}

func (p *SimplePreferences) StringWithFallback(key, fallback string) string {
	v := p.String(key)
	if v == "" {
		return fallback
	}
	return v
}

func (p *SimplePreferences) SetString(key, value string) {
	p.mu.Lock()
	p.data[key] = value
	p.mu.Unlock()
	p.save()
}

func (p *SimplePreferences) Float(key string) float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if v, ok := p.data[key].(float64); ok {
		return v
	}
	return 0
}

func (p *SimplePreferences) FloatWithFallback(key string, fallback float64) float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if v, ok := p.data[key].(float64); ok {
		return v
	}
	return fallback
}

func (p *SimplePreferences) SetFloat(key string, value float64) {
	p.mu.Lock()
	p.data[key] = value
	p.mu.Unlock()
	p.save()
}
