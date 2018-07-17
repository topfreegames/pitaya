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
	"sync"
	"sync/atomic"

	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/util"
)

const (
	groupStatusWorking = 0
	groupStatusClosed  = 1
)

// SessionFilter represents a filter which is used to filter sessions when Multicast,
// the session will receive the message when the filter returns true.
type SessionFilter func(*session.Session) bool

// Group represents a session group which is used to manage a number of
// sessions, data sent to the group will be sent to all sessions in it.
type Group struct {
	mu       sync.RWMutex
	status   int32                       // channel current status
	name     string                      // channel name
	sessions map[string]*session.Session // session id map to session instance
}

// NewGroup returns a new group instance
func NewGroup(n string) *Group {
	return &Group{
		status:   groupStatusWorking,
		name:     n,
		sessions: make(map[string]*session.Session),
	}
}

// Member returns specified UID's session
func (c *Group) Member(uid string) (*session.Session, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	s, ok := c.sessions[uid]
	if ok {
		return s, nil
	}
	return nil, constants.ErrMemberNotFound
}

// Members returns all member's UID in current group
func (c *Group) Members() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	members := []string{}
	for _, s := range c.sessions {
		members = append(members, s.UID())
	}

	return members
}

// Multicast  push  the message to the filtered clients
func (c *Group) Multicast(route string, v interface{}, filter SessionFilter) error {
	if c.isClosed() {
		return constants.ErrClosedGroup
	}

	data, err := util.SerializeOrRaw(app.serializer, v)
	if err != nil {
		return err
	}

	logger.Log.Debugf("Type=Multicast Route=%s, Data=%+v", route, v)

	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, s := range c.sessions {
		if !filter(s) {
			continue
		}
		if err = s.Push(route, data); err != nil {
			logger.Log.Error(err.Error())
		}
	}

	return nil
}

// Broadcast pushes the message to all members
func (c *Group) Broadcast(route string, v interface{}) error {
	if c.isClosed() {
		return constants.ErrClosedGroup
	}

	data, err := util.SerializeOrRaw(app.serializer, v)
	if err != nil {
		return err
	}

	logger.Log.Debugf("Type=Broadcast Route=%s, Data=%+v", route, v)

	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, s := range c.sessions {
		if err = s.Push(route, data); err != nil {
			logger.Log.Errorf("Session push message error, ID=%d, UID=%d, Error=%s", s.ID(), s.UID(), err.Error())
		}
	}

	return err
}

// Contains check whether a UID is contained in current group or not
func (c *Group) Contains(uid string) bool {
	_, err := c.Member(uid)
	return err == nil
}

// Add adds session to group
func (c *Group) Add(session *session.Session) error {
	if session.UID() == "" {
		return constants.ErrNoUIDBind
	}
	if c.isClosed() {
		return constants.ErrClosedGroup
	}

	logger.Log.Debugf("Add session to group %s, ID=%d, UID=%d", c.name, session.ID(), session.UID())

	c.mu.Lock()
	defer c.mu.Unlock()

	id := session.UID()
	_, ok := c.sessions[id]
	if ok {
		return constants.ErrSessionDuplication
	}

	c.sessions[id] = session
	return nil
}

// Leave removes specified UID related session from group
func (c *Group) Leave(s *session.Session) error {
	if c.isClosed() {
		return constants.ErrClosedGroup
	}

	logger.Log.Debugf("Remove session from group %s, UID=%d", c.name, s.UID())

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.sessions, s.UID())
	return nil
}

// LeaveAll clears all sessions in the group
func (c *Group) LeaveAll() error {
	if c.isClosed() {
		return constants.ErrClosedGroup
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.sessions = make(map[string]*session.Session)
	return nil
}

// Count get current member amount in the group
func (c *Group) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.sessions)
}

func (c *Group) isClosed() bool {
	if atomic.LoadInt32(&c.status) == groupStatusClosed {
		return true
	}
	return false
}

// Close destroy group, which will release all resource in the group
func (c *Group) Close() error {
	if c.isClosed() {
		return constants.ErrCloseClosedGroup
	}

	atomic.StoreInt32(&c.status, groupStatusClosed)

	// release all references
	c.sessions = make(map[string]*session.Session)
	return nil
}
