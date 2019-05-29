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
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/mocks"
)

func TestRateLimiterRead(t *testing.T) {
	t.Parallel()

	var (
		limit    = 3
		interval = time.Second
		buf      = []byte{}
		errTest  = errors.New("error")

		mockConn *mocks.MockConn
		r        *RateLimiter
	)

	tables := map[string]struct {
		forceDisable bool
		mock         func()
		expected     int
		err          error
	}{
		"test_can_read_on_first_time": {
			forceDisable: false,
			mock: func() {
				mockConn.EXPECT().Read(buf).Return(10, nil)
			},
			expected: 10,
			err:      nil,
		},

		"test_read_return_error": {
			forceDisable: false,
			mock: func() {
				mockConn.EXPECT().Read(buf).Return(0, errTest)
			},
			expected: 0,
			err:      errTest,
		},

		"test_exceed_limit": {
			forceDisable: false,
			mock: func() {
				for i := 0; i < limit; i++ {
					mockConn.EXPECT().Read(buf).Return(10, nil)
					_, err := r.Read(buf)
					assert.NoError(t, err)
				}

				// exceed after this call
				mockConn.EXPECT().Read(buf).Return(10, nil)
				// back to for begin, return error to leave for loop
				mockConn.EXPECT().Read(buf).Return(0, errTest)
			},
			expected: 0,
			err:      errTest,
		},

		"test_force_disable": {
			forceDisable: true,
			mock: func() {
				for i := 0; i < limit; i++ {
					mockConn.EXPECT().Read(buf).Return(10, nil)
					_, err := r.Read(buf)
					assert.NoError(t, err)
				}

				mockConn.EXPECT().Read(buf).Return(10, nil)
			},
			expected: 10, // exceed but ignored, so return the value of read
			err:      nil,
		},
	}

	for name, table := range tables {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockConn = mocks.NewMockConn(ctrl)

			r = NewRateLimiter(mockConn, limit, interval, table.forceDisable)

			table.mock()
			n, err := r.Read(buf)
			assert.Equal(t, table.err, err)
			assert.Equal(t, table.expected, n)
		})
	}
}

func TestRateLimiterShouldRateLimit(t *testing.T) {
	t.Parallel()

	var (
		limit    = 3
		interval = time.Second
		now      = time.Now()
		r        *RateLimiter
	)

	tables := map[string]struct {
		before func()
		should bool
	}{
		"test_should_not_on_first_time": {
			before: func() {},
			should: false,
		},
		"test_should_not_missing_one_to_limit": {
			before: func() {
				r.shouldRateLimit(now)
				r.shouldRateLimit(now)
			},
			should: false,
		},
		"test_should_not_when_oldest_request_expired": {
			before: func() {
				r.shouldRateLimit(now.Add(-2 * interval))
				r.shouldRateLimit(now)
				r.shouldRateLimit(now)
			},
			should: false,
		},
		"test_should_when_exceeded_limit": {
			before: func() {
				r.shouldRateLimit(now)
				r.shouldRateLimit(now)
				r.shouldRateLimit(now)
			},
			should: true,
		},
	}

	for name, table := range tables {
		t.Run(name, func(t *testing.T) {
			r = NewRateLimiter(nil, limit, interval, false)

			table.before()
			should := r.shouldRateLimit(now)
			assert.Equal(t, table.should, should)
		})
	}
}
