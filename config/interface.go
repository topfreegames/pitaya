package config

import "time"

type Config interface {
	// Get returns the configuration path as an interface. Default: nil
	Get(string) interface{}

	// GetString returns the configuration path as a string. Default: ""
	GetString(string) string

	// GetStringSlice returns the value associated with the key as a slice of strings.
	GetStringSlice(string) []string

	// GetStringMapString return the configuration path as a map of strings.
	// Default: map[string]string
	GetStringMapString(string) map[string]string

	// GetStringMap return the configuration path as a map of string and interfaces.
	// Default: map[string]interface{}
	GetStringMap(key string) map[string]interface{}

	// GetInt returns the configuration path as an int. Default: 0
	GetInt(string) int

	// GetFloat64 returns the configuration path as a float64. Default: 0.0
	GetFloat64(string) float64

	// GetBool returns the configuration path as a boolean. Default: false
	GetBool(string) bool

	// GetDuration returns a time.Duration of the config. Default: 0
	GetDuration(string) time.Duration

	// Set sets the value at the given key
	Set(string, interface{})

	// UnmarshalKey can be used to scan values in the config instance and unmarshal
	// into a struct.
	UnmarshalKey(string, interface{}) error
}
