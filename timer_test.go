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

package pitaya

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/timer"
)

type MyCond struct{}

func (m *MyCond) Check(now time.Time) bool {
	return false
}

func TestNewTimer(t *testing.T) {
	t.Parallel()
	tt := NewTimer(100*time.Millisecond, func() {})
	assert.NotNil(t, tt)
}

func TestNewCountTimer(t *testing.T) {
	t.Parallel()
	tt := NewCountTimer(100*time.Millisecond, 10, func() {})
	assert.NotNil(t, tt)
}

func TestNewAfterTimer(t *testing.T) {
	t.Parallel()
	tt := NewAfterTimer(100*time.Millisecond, func() {})
	assert.NotNil(t, tt)
}

func TestNewCondTimer(t *testing.T) {
	t.Parallel()
	_, err := NewCondTimer(nil, func() {})
	assert.EqualError(t, constants.ErrNilCondition, err.Error())

	tt, err := NewCondTimer(&MyCond{}, func() {})
	assert.NoError(t, err)
	assert.NotNil(t, tt)
}

func TestSetTimerPrecision(t *testing.T) {
	t.Parallel()
	dur := 33 * time.Millisecond
	SetTimerPrecision(dur)
	assert.Equal(t, dur, timer.Precision)
}

func TestSetTimerBacklog(t *testing.T) {
	backlog := 1 << 4
	SetTimerBacklog(backlog)
}
