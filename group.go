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
	"sync/atomic"

	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/groups"
	"github.com/topfreegames/pitaya/logger"
)

const (
	groupStatusWorking = 0
	groupStatusClosed  = 1
)

// Group represents an agglomeration of UIDs which is used to manage
// users. Data sent to the group will be sent to all users in it.
type Group struct {
	groupService groups.GroupService
	status       int32  // channel current status
	name         string // channel name
}

// NewGroup returns a new group instance
func NewGroup(n string, gs groups.GroupService) *Group {
	return &Group{
		status:       groupStatusWorking,
		name:         n,
		groupService: gs,
	}
}

// MemberGroups returns all groups that the member takes part
func (c *Group) MemberGroups(uid string) ([]string, error) {
	if c.isClosed() {
		return nil, constants.ErrClosedGroup
	}
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return c.groupService.MemberGroups(uid)
}

// Member returns member payload
func (c *Group) Member(uid string) (*groups.Payload, error) {
	if c.isClosed() {
		return nil, constants.ErrClosedGroup
	}
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return c.groupService.Member(c.name, uid)
}

// Members returns all member's UIDs in current group
func (c *Group) Members() (map[string]*groups.Payload, error) {
	if c.isClosed() {
		return nil, constants.ErrClosedGroup
	}
	return c.groupService.Members(c.name)
}

// Multicast  push  the message to the filtered clients
func (c *Group) Multicast(frontendType, route string, v interface{}, uids []string) error {
	if c.isClosed() {
		return constants.ErrClosedGroup
	}
	logger.Log.Debugf("Type=Multicast Route=%s, Data=%+v", route, v)

	errUids, err := SendPushToUsers(route, v, uids, frontendType)
	if err != nil {
		for _, eUID := range errUids {
			logger.Log.Errorf("Group push message error, UID=%d, Error=%s", eUID, err.Error())
		}
	}
	return err
}

// Broadcast pushes the message to all members
func (c *Group) Broadcast(frontendType, route string, v interface{}) error {
	if c.isClosed() {
		return constants.ErrClosedGroup
	}
	logger.Log.Debugf("Type=Broadcast Route=%s, Data=%+v", route, v)

	members, err := c.Members()
	if err != nil {
		return err
	}
	uids := make([]string, 0, len(members))
	for uid := range members {
		uids = append(uids, uid)
	}
	return c.Multicast(frontendType, route, v, uids)
}

// Contains check whether a UID is contained in current group or not
func (c *Group) Contains(uid string) (bool, error) {
	if uid == "" {
		return false, constants.ErrNoUIDBind
	}
	if c.isClosed() {
		return false, constants.ErrClosedGroup
	}
	return c.groupService.Contains(c.name, uid)
}

// Add adds UID to group
func (c *Group) Add(uid string, payload *groups.Payload) error {
	if uid == "" {
		return constants.ErrNoUIDBind
	}
	if c.isClosed() {
		return constants.ErrClosedGroup
	}
	logger.Log.Debugf("Add user to group %s, UID=%d", c.name, uid)

	return c.groupService.Add(c.name, uid, payload)
}

// Leave removes specified UID from group
func (c *Group) Leave(uid string) error {
	if c.isClosed() {
		return constants.ErrClosedGroup
	}
	logger.Log.Debugf("Remove user from group %s, UID=%d", c.name, uid)

	return c.groupService.Leave(c.name, uid)
}

// LeaveAll clears all UIDs in the group
func (c *Group) LeaveAll() error {
	if c.isClosed() {
		return constants.ErrClosedGroup
	}

	return c.groupService.LeaveAll(c.name)
}

// Count get current member amount in the group
func (c *Group) Count() (int, error) {
	if c.isClosed() {
		return 0, constants.ErrClosedGroup
	}
	return c.groupService.Count(c.name)
}

func (c *Group) isClosed() bool {
	return atomic.LoadInt32(&c.status) == groupStatusClosed
}

// Close destroy group, which will release all resource in the group
func (c *Group) Close() error {
	if c.isClosed() {
		return constants.ErrCloseClosedGroup
	}

	err := c.groupService.Close(c.name)
	if err != nil {
		return err
	}

	atomic.StoreInt32(&c.status, groupStatusClosed)
	return nil
}
