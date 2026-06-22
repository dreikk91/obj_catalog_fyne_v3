package config

// Preferences is the subset of preference storage used by config loaders.
type Preferences interface {
	BoolWithFallback(key string, fallback bool) bool
	FloatWithFallback(key string, fallback float64) float64
	IntWithFallback(key string, fallback int) int
	String(key string) string
	StringWithFallback(key string, fallback string) string
	SetBool(key string, value bool)
	SetFloat(key string, value float64)
	SetInt(key string, value int)
	SetString(key string, value string)
}
