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

package timer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/helpers"
)

type alwaysRunCondition struct{}

func (a *alwaysRunCondition) Check(now time.Time) bool {
	return true
}

type neverRunCondition struct{}

func (a *neverRunCondition) Check(now time.Time) bool {
	return false
}

var tables = []struct {
	name      string
	f         func()
	interval  time.Duration
	counter   int
	condition Condition
}{
	{"1sec1rep", func() {}, 1 * time.Second, 1, &neverRunCondition{}},
	{"10sec10rep", func() {}, 10 * time.Second, 10, &neverRunCondition{}},
	{"50milli1rep", func() {}, 50 * time.Millisecond, 1, &neverRunCondition{}},
	{"50milli2rep", func() {}, 50 * time.Millisecond, 2, &alwaysRunCondition{}},
}

func TestInit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, Manager.ChClosingTimer)
	assert.NotNil(t, Manager.ChCreatedTimer)
}

func TestNewTimer(t *testing.T) {
	t.Parallel()
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			tm := NewTimer(table.f, table.interval, table.counter)
			assert.NotNil(t, tm)
			assert.Equal(t, table.counter, tm.counter)
			assert.Equal(t, table.interval, tm.interval)
		})
	}
}

func TestAddTimer(t *testing.T) {
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			tm := NewTimer(table.f, table.interval, table.counter)
			AddTimer(tm)
			tt, ok := Manager.timers.Load(tm.ID)
			assert.True(t, ok)
			assert.Equal(t, tm, tt)
		})
	}
}

func TestRemoveTimer(t *testing.T) {
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			tm := NewTimer(table.f, table.interval, table.counter)
			AddTimer(tm)
			tt, ok := Manager.timers.Load(tm.ID)
			assert.True(t, ok)
			assert.Equal(t, tm, tt)
			RemoveTimer(tm.ID)
			_, ok = Manager.timers.Load(tm.ID)
			assert.False(t, ok)
		})
	}
}

func TestSetCondition(t *testing.T) {
	t.Parallel()
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			tm := NewTimer(table.f, table.interval, table.counter)
			tm.SetCondition(table.condition)
			assert.Equal(t, table.condition, tm.condition)
		})
	}
}

func TestStop(t *testing.T) {
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			tm := NewTimer(table.f, table.interval, table.counter)
			tm.Stop()
			closedID := helpers.ShouldEventuallyReceive(t, Manager.ChClosingTimer).(int64)
			assert.Equal(t, tm.ID, closedID)
			assert.Equal(t, int32(1), tm.closed)
		})
	}
}

func TestSetTimerBacklog(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 1<<8, timerBacklog)
	SetTimerBacklog(10)
	// if c is < 16, we force 16
	assert.Equal(t, 1<<4, timerBacklog)
	SetTimerBacklog(32)
	assert.Equal(t, 1<<5, timerBacklog)
}

func TestPexec(t *testing.T) {
	t.Parallel()
	i := 0
	dFunc := func() {
		i++
	}
	pexec(10, dFunc)
	assert.Equal(t, 1, i)

	panicFunc := func() {
		panic("bla")
	}

	// should not panic because the execution is protected
	assert.NotPanics(t, func() {
		pexec(20, panicFunc)
	})
}

func TestCron(t *testing.T) {
	i := 0
	j := 0
	k := 0

	fi := func() {
		i++
	}

	fj := func() {
		j++
	}

	fk := func() {
		k++
	}

	tmi := NewTimer(fi, 20*time.Millisecond, 1)
	tmi.SetCondition(&alwaysRunCondition{})

	tmj := NewTimer(fj, 20*time.Millisecond, 2)

	tmk := NewTimer(fk, 20*time.Millisecond, 2)
	tmk.SetCondition(&neverRunCondition{})

	AddTimer(tmi)
	AddTimer(tmj)
	AddTimer(tmk)

	time.Sleep(10 * time.Millisecond)
	Cron()
	// after 10ms only tmi should run because it has always run condition
	assert.Equal(t, i, i)
	assert.Equal(t, 0, j)
	assert.Equal(t, 0, k)

	time.Sleep(15 * time.Millisecond)
	Cron()
	// after 25ms cron run, i should be 2, j should be 1, and k should be 0 because it has neverRunCondition
	assert.Equal(t, 2, i)
	assert.Equal(t, 1, j)
	assert.Equal(t, 0, k)

	time.Sleep(25 * time.Millisecond)
	Cron()

	// after more 25ms j should be 2
	assert.Equal(t, 2, j)

	time.Sleep(25 * time.Millisecond)
	Cron()
	// after more 25ms j should be still 2 because of the counter
	assert.Equal(t, 2, j)
}
