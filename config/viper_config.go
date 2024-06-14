// Copyright (c) TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package config

import (
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/spf13/viper"
)

// Config is a wrapper around a viper config
type viperImpl struct {
	*viper.Viper
}

func NewViperConfig(cfg *viper.Viper) Config {
	return &viperImpl{cfg}
}

// NewConfig creates a new config with a given viper config if given
func NewConfig(cfgs ...viper.Viper) Config {
	var cfg *viper.Viper
	if len(cfgs) > 0 {
		cfg = &cfgs[0]
	} else {
		cfg = viper.New()
	}

	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	cfg.AutomaticEnv()
	c := &viperImpl{cfg}
	fillDefaultValues(c)
	return c
}

// UnmarshalKey unmarshals key into v
func (c *viperImpl) UnmarshalKey(key string, rawVal interface{}) error {
	key = strings.ToLower(key)
	delimiter := "."
	prefix := key + delimiter

	i := c.Get(key)
	if i == nil {
		return nil
	}
	if isStringMapInterface(i) {
		val := i.(map[string]interface{})
		keys := c.AllKeys()
		for _, k := range keys {
			if !strings.HasPrefix(k, prefix) {
				continue
			}
			mk := strings.TrimPrefix(k, prefix)
			mk = strings.Split(mk, delimiter)[0]
			if _, exists := val[mk]; exists {
				continue
			}
			mv := c.Get(key + delimiter + mk)
			if mv == nil {
				continue
			}
			val[mk] = mv
		}
		i = val
	}
	return decode(i, defaultDecoderConfig(rawVal))
}

func isStringMapInterface(val interface{}) bool {
	vt := reflect.TypeOf(val)
	return vt.Kind() == reflect.Map &&
		vt.Key().Kind() == reflect.String &&
		vt.Elem().Kind() == reflect.Interface
}

// A wrapper around mapstructure.Decode that mimics the WeakDecode functionality
func decode(input interface{}, config *mapstructure.DecoderConfig) error {
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

// defaultDecoderConfig returns default mapstructure.DecoderConfig with support
// of time.Duration values & string slices
func defaultDecoderConfig(output interface{}, opts ...viper.DecoderConfigOption) *mapstructure.DecoderConfig {
	c := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c

}
