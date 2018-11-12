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
	"context"

	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/groups"
	"github.com/topfreegames/pitaya/logger"
)

// Group represents an agglomeration of UIDs which is used to manage
// users. Data sent to the group will be sent to all users in it.
type Group struct {
	groupService groups.GroupService
	name         string // channel name
}

// NewGroup returns a new group instance
func NewGroup(n string, gs groups.GroupService) *Group {
	return &Group{
		name:         n,
		groupService: gs,
	}
}

// MemberGroups returns all groups that the member takes part
func (c *Group) MemberGroups(ctx context.Context, uid string) ([]string, error) {
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return c.groupService.MemberGroups(ctx, uid)
}

// MemberSubgroups returns all subgroups from given group that the member takes part
func (c *Group) MemberSubgroups(ctx context.Context, uid string) ([]string, error) {
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return c.groupService.MemberSubgroups(ctx, c.name, uid)
}

// Member returns member payload
func (c *Group) Member(ctx context.Context, uid string) (*groups.Payload, error) {
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return c.groupService.Member(ctx, c.name, uid)
}

// SubgroupMember returns member payload from subgroup
func (c *Group) SubgroupMember(ctx context.Context, subgroupName, uid string) (*groups.Payload, error) {
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return c.groupService.SubgroupMember(ctx, c.name, subgroupName, uid)
}

// Members returns all member's UIDs and payload in current group
func (c *Group) Members(ctx context.Context) (map[string]*groups.Payload, error) {
	return c.groupService.Members(ctx, c.name)
}

// SubgroupMembers returns all member's UIDs and payload in current subgroup
func (c *Group) SubgroupMembers(ctx context.Context, subgroupName string) (map[string]*groups.Payload, error) {
	return c.groupService.SubgroupMembers(ctx, c.name, subgroupName)
}

// Multicast  push  the message to the filtered clients
func (c *Group) Multicast(ctx context.Context, frontendType, route string, v interface{}, uids []string) error {
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
func (c *Group) Broadcast(ctx context.Context, frontendType, route string, v interface{}) error {
	logger.Log.Debugf("Type=Broadcast Route=%s, Data=%+v", route, v)

	members, err := c.Members(ctx)
	if err != nil {
		return err
	}
	uids := make([]string, 0, len(members))
	for uid := range members {
		uids = append(uids, uid)
	}
	return c.Multicast(ctx, frontendType, route, v, uids)
}

// SubgroupBroadcast pushes the message to all members
func (c *Group) SubgroupBroadcast(ctx context.Context, frontendType, subgroupName, route string, v interface{}) error {
	logger.Log.Debugf("Type=SubgroupBroadcast Route=%s, Data=%+v", route, v)

	members, err := c.SubgroupMembers(ctx, subgroupName)
	if err != nil {
		return err
	}
	uids := make([]string, 0, len(members))
	for uid := range members {
		uids = append(uids, uid)
	}
	return c.Multicast(ctx, frontendType, route, v, uids)
}

// Contains check whether a UID is contained in current group or not
func (c *Group) Contains(ctx context.Context, uid string) (bool, error) {
	if uid == "" {
		return false, constants.ErrNoUIDBind
	}
	return c.groupService.Contains(ctx, c.name, uid)
}

// SubgroupContains check whether a UID is contained in current subgroup or not
func (c *Group) SubgroupContains(ctx context.Context, subgroupName, uid string) (bool, error) {
	if uid == "" {
		return false, constants.ErrNoUIDBind
	}
	return c.groupService.SubgroupContains(ctx, c.name, subgroupName, uid)
}

// SubgroupAdd adds UID to subgroup
func (c *Group) SubgroupAdd(ctx context.Context, subgroupName, uid string, payload *groups.Payload) error {
	if uid == "" {
		return constants.ErrNoUIDBind
	}
	logger.Log.Debugf("Add user to subgroup %s, UID=%d", c.name, uid)

	return c.groupService.SubgroupAdd(ctx, c.name, subgroupName, uid, payload)
}

// Add adds UID to group
func (c *Group) Add(ctx context.Context, uid string, payload *groups.Payload) error {
	if uid == "" {
		return constants.ErrNoUIDBind
	}
	logger.Log.Debugf("Add user to group %s, UID=%d", c.name, uid)
	return c.groupService.Add(ctx, c.name, uid, payload)
}

// Leave removes specified UID from group
func (c *Group) Leave(ctx context.Context, uid string) error {
	logger.Log.Debugf("Remove user from group %s, UID=%d", c.name, uid)
	return c.groupService.Leave(ctx, c.name, uid)
}

// SubgroupLeave removes specified UID from subgroup
func (c *Group) SubgroupLeave(ctx context.Context, subgroupName, uid string) error {
	logger.Log.Debugf("Remove user from subgroup %s, UID=%d", c.name, uid)
	return c.groupService.SubgroupLeave(ctx, c.name, subgroupName, uid)
}

// LeaveAll clears all UIDs in the group and contained subgroups
func (c *Group) LeaveAll(ctx context.Context) error {
	return c.groupService.LeaveAll(ctx, c.name)
}

// SubgroupLeaveAll clears all UIDs in the subgroup
func (c *Group) SubgroupLeaveAll(ctx context.Context, subgroupName string) error {
	return c.groupService.SubgroupLeaveAll(ctx, c.name, subgroupName)
}

// Count get current member amount in the group
func (c *Group) Count(ctx context.Context) (int, error) {
	return c.groupService.Count(ctx, c.name)
}

// SubgroupCount get current member amount in the subgroup
func (c *Group) SubgroupCount(ctx context.Context, subgroupName string) (int, error) {
	return c.groupService.SubgroupCount(ctx, c.name, subgroupName)
}
