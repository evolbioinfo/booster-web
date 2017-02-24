package config

type Provider interface {
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetStringMap(key string) map[string]interface{}
	GetStringMapString(key string) map[string]string
	Get(key string) interface{}
	Set(key string, value interface{})
	IsSet(key string) bool
}
