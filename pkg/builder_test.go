// Copyright (c) nano Author and TFG Co. All Rights Reserved.
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

package pitaya

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v3/pkg/acceptor"
	"github.com/topfreegames/pitaya/v3/pkg/config"
)

func TestPostBuildHooks(t *testing.T) {
	acc := acceptor.NewTCPAcceptor("0.0.0.0:0")
	for _, table := range tables {
		builderConfig := config.NewDefaultPitayaConfig()

		t.Run("with_post_build_hooks", func(t *testing.T) {
			called := false
			builder := NewDefaultBuilder(table.isFrontend, table.serverType, table.serverMode, table.serverMetadata, *builderConfig)
			builder.AddAcceptor(acc)
			builder.AddPostBuildHook(func(app Pitaya) {
				called = true
			})
			app := builder.Build()

			assert.True(t, called)
			assert.NotNil(t, app)
		})

		t.Run("without_post_build_hooks", func(t *testing.T) {
			called := false
			builder := NewDefaultBuilder(table.isFrontend, table.serverType, table.serverMode, table.serverMetadata, *builderConfig)
			builder.AddAcceptor(acc)
			app := builder.Build()

			assert.False(t, called)
			assert.NotNil(t, app)
		})
	}
}
