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

package errors

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	t.Parallel()

	const code = "code"

	type (
		input struct {
			err      error
			code     string
			metadata map[string]string
		}

		expected struct {
			code     string
			metadata map[string]string
		}
	)

	tables := []struct {
		name     string
		input    input
		expected expected
	}{
		{"nil_metadata",
			input{err: errors.New(uuid.New().String()), code: code, metadata: nil},
			expected{code: code, metadata: nil},
		},
		{"empty_metadata",
			input{err: errors.New(uuid.New().String()), code: code, metadata: map[string]string{}},
			expected{code: code, metadata: map[string]string{}},
		},
		{"non_empty_metadata",
			input{err: errors.New(uuid.New().String()), code: code, metadata: map[string]string{"key": "value"}},
			expected{code: code, metadata: map[string]string{"key": "value"}},
		},
		{"pitaya_error",
			input{
				err:      NewError(errors.New(uuid.New().String()), code, map[string]string{"key1": "value1", "key2": "value2"}),
				code:     "another-code",
				metadata: map[string]string{"key1": "new-value1", "key3": "value3"},
			},
			expected{code: code, metadata: map[string]string{"key1": "new-value1", "key2": "value2", "key3": "value3"}},
		},
		{"pitaya_error_nil_metadata",
			input{
				err:      NewError(errors.New(uuid.New().String()), code),
				code:     "another-code",
				metadata: map[string]string{"key1": "value1", "key2": "value2"},
			},
			expected{code: code, metadata: map[string]string{"key1": "value1", "key2": "value2"}},
		},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			var err *Error
			if table.input.metadata != nil {
				err = NewError(table.input.err, table.input.code, table.input.metadata)
			} else {
				err = NewError(table.input.err, table.input.code)
			}
			assert.NotNil(t, err)
			assert.Equal(t, table.input.err.Error(), err.Message)
			assert.Equal(t, table.expected.code, err.Code)
			assert.Equal(t, table.expected.metadata, err.Metadata)
		})
	}
}

func TestErrorError(t *testing.T) {
	t.Parallel()

	sourceErr := errors.New(uuid.New().String())
	err := NewError(sourceErr, uuid.New().String())

	errStr := err.Error()
	assert.Equal(t, sourceErr.Error(), errStr)
}

func TestCodeFromError(t *testing.T) {
	t.Parallel()

	errTest := errors.New("error")
	codeNotFound := "GAME-404"

	tables := map[string]struct {
		err  error
		code string
	}{
		"test_not_error": {
			err:  nil,
			code: "",
		},

		"test_not_pitaya_error": {
			err:  errTest,
			code: ErrUnknownCode,
		},

		"test_nil_pitaya_error": {
			err:  func() *Error { var err *Error; return err }(),
			code: "",
		},

		"test_pitaya_error": {
			err:  NewError(errTest, codeNotFound),
			code: codeNotFound,
		},
	}

	for name, table := range tables {
		t.Run(name, func(t *testing.T) {
			code := CodeFromError(table.err)
			assert.Equal(t, table.code, code)
		})
	}
}
