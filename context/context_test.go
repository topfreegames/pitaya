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

package context

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/helpers"
)

var update = flag.Bool("update", false, "update .golden files")

type unregisteredStruct struct{}
type registeredStruct struct{}

func TestAddToPropagateCtx(t *testing.T) {
	tables := []struct {
		name  string
		items map[string]interface{}
	}{
		{"one_element", map[string]interface{}{"key1": "val1"}},
		{"two_elements", map[string]interface{}{"key1": "val1", "key2": "val2"}},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctx := context.Background()
			for k, v := range table.items {
				ctx = AddToPropagateCtx(ctx, k, v)
			}
			val := ctx.Value(constants.PropagateCtxKey)
			assert.IsType(t, map[string]interface{}{}, val)
			assert.Equal(t, table.items, val.(map[string]interface{}))
		})
	}
}

func TestGetFromPropagateCtx(t *testing.T) {
	tables := []struct {
		name  string
		items map[string]interface{}
	}{
		{"one_element", map[string]interface{}{"key1": "val1"}},
		{"two_elements", map[string]interface{}{"key1": "val1", "key2": "val2"}},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctx := context.Background()
			for k, v := range table.items {
				ctx = AddToPropagateCtx(ctx, k, v)
			}

			for k, v := range table.items {
				val := GetFromPropagateCtx(ctx, k)
				assert.Equal(t, v, val)
			}
		})
	}
}

func TestGetFromPropagateCtxReturnsNilIfNotFound(t *testing.T) {
	ctx := context.Background()
	val := GetFromPropagateCtx(ctx, "key")
	assert.Nil(t, val)
}

func TestToMap(t *testing.T) {
	tables := []struct {
		name  string
		items map[string]interface{}
	}{
		{"no_elements", map[string]interface{}{}},
		{"one_element", map[string]interface{}{"key1": "val1"}},
		{"two_elements", map[string]interface{}{"key1": "val1", "key2": "val2"}},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctx := context.Background()
			for k, v := range table.items {
				ctx = AddToPropagateCtx(ctx, k, v)
			}

			val := ToMap(ctx)
			assert.Equal(t, table.items, val)
		})
	}
}

func TestToMapReturnsEmptyIfNothingInPropagateKey(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "key", "val")
	val := ToMap(ctx)
	assert.Equal(t, map[string]interface{}{}, val)
}

func TestFromMap(t *testing.T) {
	tables := []struct {
		name  string
		items map[string]interface{}
	}{
		{"one_element", map[string]interface{}{"key1": "val1"}},
		{"two_elements", map[string]interface{}{"key1": "val1", "key2": "val2"}},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctx := FromMap(table.items)
			for k, v := range table.items {
				val := GetFromPropagateCtx(ctx, k)
				assert.Equal(t, v, val)
			}

		})
	}
}

func TestEncode(t *testing.T) {
	tables := []struct {
		name  string
		items map[string]interface{}
		err   error
	}{
		{"no_elements", map[string]interface{}{}, nil},
		{"one_element", map[string]interface{}{"key1": "val1"}, nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctx := FromMap(table.items)

			if len(table.items) > 0 && table.err == nil {
				gp := filepath.Join("fixtures", table.name+".golden")
				if *update {
					b, err := json.Marshal(table.items)
					require.NoError(t, err)
					t.Log("updating golden file")
					helpers.WriteFile(t, gp, b)
				}
				expectedEncoded := helpers.ReadFile(t, gp)

				encoded, err := Encode(ctx)
				assert.Equal(t, table.err, err)
				assert.Equal(t, expectedEncoded, encoded)
			} else {
				encoded, err := Encode(ctx)
				assert.Nil(t, encoded)
				assert.Equal(t, table.err, err)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tables := []struct {
		name  string
		items map[string]interface{}
	}{
		{"one_element", map[string]interface{}{"key1": "val1"}},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctx := FromMap(table.items)

			gp := filepath.Join("fixtures", table.name+".golden")
			encoded := helpers.ReadFile(t, gp)

			decoded, err := Decode(encoded)
			assert.NoError(t, err)
			assert.Equal(t, decoded, ctx)
		})
	}
}

func TestDecodeFailsIfBadEncodedData(t *testing.T) {
	decoded, err := Decode([]byte("oh noes"))
	assert.Equal(t, errors.New("invalid character 'o' looking for beginning of value").Error(), err.Error())
	assert.Nil(t, decoded)
}

func TestDecodeWithEmptyData(t *testing.T) {
	decoded, err := Decode([]byte(""))
	assert.Nil(t, err)
	assert.Nil(t, decoded)
}
