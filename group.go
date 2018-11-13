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

	"github.com/coreos/etcd/clientv3"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/groups"
	"github.com/topfreegames/pitaya/logger"
)

// Group represents an agglomeration of UIDs which is used to manage
// users. Data sent to the group will be sent to all users in it.

var groupServiceInstance groups.GroupService

// InitGroups should be called ONCE in the beginning of the application, if it is intended to use groups functionality
func InitGroups(config *config.Config, clientOrNil *clientv3.Client) {
	gsi, err := groups.NewEtcdGroupService(config, clientOrNil)
	if err != nil {
		logger.Log.Fatalf("error initializing groupServiceInstance in groups: %s", err.Error())
	}
	groupServiceInstance = gsi
}

// Subgroups returns all subgroups that the group has
func Subgroups(ctx context.Context, groupName string) ([]string, error) {
	return groupServiceInstance.Subgroups(ctx, groupName)
}

// MemberGroups returns all groups that the member takes part
func MemberGroups(ctx context.Context, uid string) ([]string, error) {
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return groupServiceInstance.MemberGroups(ctx, uid)
}

// MemberSubgroups returns all subgroups from given group that the member takes part
func MemberSubgroups(ctx context.Context, groupName, uid string) ([]string, error) {
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return groupServiceInstance.MemberSubgroups(ctx, groupName, uid)
}

// Member returns member payload
func Member(ctx context.Context, groupName, uid string) (*groups.Payload, error) {
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return groupServiceInstance.Member(ctx, groupName, uid)
}

// SubgroupMember returns member payload from subgroup
func SubgroupMember(ctx context.Context, groupName, subgroupName, uid string) (*groups.Payload, error) {
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return groupServiceInstance.SubgroupMember(ctx, groupName, subgroupName, uid)
}

// Members returns all member's UIDs and payload in current group
func Members(ctx context.Context, groupName string) (map[string]*groups.Payload, error) {
	return groupServiceInstance.Members(ctx, groupName)
}

// SubgroupMembers returns all member's UIDs and payload in current subgroup
func SubgroupMembers(ctx context.Context, groupName, subgroupName string) (map[string]*groups.Payload, error) {
	return groupServiceInstance.SubgroupMembers(ctx, groupName, subgroupName)
}

// Multicast  pushes  the message to the filtered clients
func Multicast(ctx context.Context, frontendType, route string, v interface{}, uids []string) error {
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
func Broadcast(ctx context.Context, frontendType, groupName, route string, v interface{}) error {
	logger.Log.Debugf("Type=Broadcast Route=%s, Data=%+v", route, v)

	members, err := Members(ctx, groupName)
	if err != nil {
		return err
	}
	uids := make([]string, 0, len(members))
	for uid := range members {
		uids = append(uids, uid)
	}
	return Multicast(ctx, frontendType, route, v, uids)
}

// SubgroupBroadcast pushes the message to all members
func SubgroupBroadcast(ctx context.Context, frontendType, groupName, subgroupName, route string, v interface{}) error {
	logger.Log.Debugf("Type=SubgroupBroadcast Route=%s, Data=%+v", route, v)

	members, err := SubgroupMembers(ctx, groupName, subgroupName)
	if err != nil {
		return err
	}
	uids := make([]string, 0, len(members))
	for uid := range members {
		uids = append(uids, uid)
	}
	return Multicast(ctx, frontendType, route, v, uids)
}

// Contains checks whether a UID is contained in current group or not
func Contains(ctx context.Context, groupName, uid string) (bool, error) {
	if uid == "" {
		return false, constants.ErrNoUIDBind
	}
	return groupServiceInstance.Contains(ctx, groupName, uid)
}

// SubgroupContains check whether a UID is contained in current subgroup or not
func SubgroupContains(ctx context.Context, groupName, subgroupName, uid string) (bool, error) {
	if uid == "" {
		return false, constants.ErrNoUIDBind
	}
	return groupServiceInstance.SubgroupContains(ctx, groupName, subgroupName, uid)
}

// SubgroupAdd adds UID to subgroup
func SubgroupAdd(ctx context.Context, groupName, subgroupName, uid string, payload *groups.Payload) error {
	if uid == "" {
		return constants.ErrNoUIDBind
	}
	logger.Log.Debugf("Add user to subgroup %s, UID=%d", groupName, uid)

	return groupServiceInstance.SubgroupAdd(ctx, groupName, subgroupName, uid, payload)
}

// Add adds UID to group
func Add(ctx context.Context, groupName, uid string, payload *groups.Payload) error {
	if uid == "" {
		return constants.ErrNoUIDBind
	}
	logger.Log.Debugf("Add user to group %s, UID=%d", groupName, uid)
	return groupServiceInstance.Add(ctx, groupName, uid, payload)
}

// Leave removes specified UID from group
func Leave(ctx context.Context, groupName, uid string) error {
	logger.Log.Debugf("Remove user from group %s, UID=%d", groupName, uid)
	return groupServiceInstance.Leave(ctx, groupName, uid)
}

// SubgroupLeave removes specified UID from subgroup
func SubgroupLeave(ctx context.Context, groupName, subgroupName, uid string) error {
	logger.Log.Debugf("Remove user from subgroup %s, UID=%d", groupName, uid)
	return groupServiceInstance.SubgroupLeave(ctx, groupName, subgroupName, uid)
}

// LeaveAll clears all UIDs in the group and contained subgroups
func LeaveAll(ctx context.Context, groupName string) error {
	return groupServiceInstance.LeaveAll(ctx, groupName)
}

// SubgroupLeaveAll clears all UIDs in the subgroup
func SubgroupLeaveAll(ctx context.Context, groupName, subgroupName string) error {
	return groupServiceInstance.SubgroupLeaveAll(ctx, groupName, subgroupName)
}

// Count get current member amount in the group
func Count(ctx context.Context, groupName string) (int, error) {
	return groupServiceInstance.Count(ctx, groupName)
}

// SubgroupCount get current member amount in the subgroup
func SubgroupCount(ctx context.Context, groupName, subgroupName string) (int, error) {
	return groupServiceInstance.SubgroupCount(ctx, groupName, subgroupName)
}
