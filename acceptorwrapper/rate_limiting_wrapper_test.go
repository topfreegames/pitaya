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

package acceptorwrapper

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/metrics"
	"github.com/topfreegames/pitaya/mocks"
)

func TestNewRateLimitingWrapper(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockedApp := mocks.NewMockPitaya(ctrl)
	mockedApp.EXPECT().GetMetricsReporters().Return([]metrics.Reporter{}).AnyTimes()

	rateLimitingWrapper := NewRateLimitingWrapper(mockedApp, NewDefaultRateLimitingConfig())
	expected := NewRateLimiter(mockedApp, nil, 20, time.Second, false)
	assert.Equal(t, expected, rateLimitingWrapper.wrapConn(nil))
}
