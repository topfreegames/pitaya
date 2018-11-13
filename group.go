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

// Subgroups returns all subgroups that the group contains
func Subgroups(ctx context.Context, groupName string) ([]string, error) {
	return groupServiceInstance.Subgroups(ctx, groupName)
}

// PlayerGroups returns all groups that the player takes part
func PlayerGroups(ctx context.Context, uid string) ([]string, error) {
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return groupServiceInstance.PlayerGroups(ctx, uid)
}

// PlayerSubgroups returns all subgroups from given group that the player takes part
func PlayerSubgroups(ctx context.Context, groupName, uid string) ([]string, error) {
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return groupServiceInstance.PlayerSubgroups(ctx, groupName, uid)
}

// GroupMember returns member payload
func GroupMember(ctx context.Context, groupName, uid string) (*groups.Payload, error) {
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return groupServiceInstance.GroupMember(ctx, groupName, uid)
}

// SubgroupMember returns member payload from subgroup
func SubgroupMember(ctx context.Context, groupName, subgroupName, uid string) (*groups.Payload, error) {
	if uid == "" {
		return nil, constants.ErrNoUIDBind
	}
	return groupServiceInstance.SubgroupMember(ctx, groupName, subgroupName, uid)
}

// GroupMembers returns all member's UIDs and payload in current group
func GroupMembers(ctx context.Context, groupName string) (map[string]*groups.Payload, error) {
	return groupServiceInstance.GroupMembers(ctx, groupName)
}

// SubgroupMembers returns all member's UIDs and payload in current subgroup
func SubgroupMembers(ctx context.Context, groupName, subgroupName string) (map[string]*groups.Payload, error) {
	return groupServiceInstance.SubgroupMembers(ctx, groupName, subgroupName)
}

// GroupBroadcast pushes the message to all members inside group
func GroupBroadcast(ctx context.Context, frontendType, groupName, route string, v interface{}) error {
	logger.Log.Debugf("Type=Broadcast Route=%s, Data=%+v", route, v)

	members, err := GroupMembers(ctx, groupName)
	if err != nil {
		return err
	}
	return sendDataToMembers(members, frontendType, route, v)
}

// SubgroupBroadcast pushes the message to all member inside subgroup
func SubgroupBroadcast(ctx context.Context, frontendType, groupName, subgroupName, route string, v interface{}) error {
	logger.Log.Debugf("Type=SubgroupBroadcast Route=%s, Data=%+v", route, v)

	members, err := SubgroupMembers(ctx, groupName, subgroupName)
	if err != nil {
		return err
	}
	return sendDataToMembers(members, frontendType, route, v)
}

func sendDataToMembers(members map[string]*groups.Payload, frontendType, route string, v interface{}) error {
	uids := make([]string, 0, len(members))
	for uid := range members {
		uids = append(uids, uid)
	}
	errUids, err := SendPushToUsers(route, v, uids, frontendType)
	if err != nil {
		logger.Log.Errorf("Group push message error, UID=%d, Error=%s", errUids, err.Error())
		return err
	}
	return nil
}

// GroupContainsMember checks whether an UID is contained in group or not
func GroupContainsMember(ctx context.Context, groupName, uid string) (bool, error) {
	if uid == "" {
		return false, constants.ErrNoUIDBind
	}
	return groupServiceInstance.GroupContainsMember(ctx, groupName, uid)
}

// SubgroupContainsMember check whether a UID is contained in subgroup or not
func SubgroupContainsMember(ctx context.Context, groupName, subgroupName, uid string) (bool, error) {
	if uid == "" {
		return false, constants.ErrNoUIDBind
	}
	return groupServiceInstance.SubgroupContainsMember(ctx, groupName, subgroupName, uid)
}

// SubgroupAdd adds UID to subgroup
func SubgroupAdd(ctx context.Context, groupName, subgroupName, uid string, payload *groups.Payload) error {
	if uid == "" {
		return constants.ErrNoUIDBind
	}
	logger.Log.Debugf("Add user to subgroup %s, UID=%d", groupName, uid)

	return groupServiceInstance.SubgroupAdd(ctx, groupName, subgroupName, uid, payload)
}

// GroupAdd adds UID to group
func GroupAdd(ctx context.Context, groupName, uid string, payload *groups.Payload) error {
	if uid == "" {
		return constants.ErrNoUIDBind
	}
	logger.Log.Debugf("Add user to group %s, UID=%d", groupName, uid)
	return groupServiceInstance.GroupAdd(ctx, groupName, uid, payload)
}

// GroupRemove removes specified UID from group
func GroupRemove(ctx context.Context, groupName, uid string) error {
	logger.Log.Debugf("Remove user from group %s, UID=%d", groupName, uid)
	return groupServiceInstance.GroupRemove(ctx, groupName, uid)
}

// SubgroupRemove removes specified UID from subgroup
func SubgroupRemove(ctx context.Context, groupName, subgroupName, uid string) error {
	logger.Log.Debugf("Remove user from subgroup %s, UID=%d", groupName, uid)
	return groupServiceInstance.SubgroupRemove(ctx, groupName, subgroupName, uid)
}

// GroupRemoveAll clears all UIDs in the group and contained subgroups
func GroupRemoveAll(ctx context.Context, groupName string) error {
	return groupServiceInstance.GroupRemoveAll(ctx, groupName)
}

// SubgroupRemoveAll clears all UIDs in the subgroup
func SubgroupRemoveAll(ctx context.Context, groupName, subgroupName string) error {
	return groupServiceInstance.SubgroupRemoveAll(ctx, groupName, subgroupName)
}

// GroupCount get current member amount in the group
func GroupCount(ctx context.Context, groupName string) (int, error) {
	return groupServiceInstance.GroupCount(ctx, groupName)
}

// SubgroupCount get current member amount in the subgroup
func SubgroupCount(ctx context.Context, groupName, subgroupName string) (int, error) {
	return groupServiceInstance.SubgroupCount(ctx, groupName, subgroupName)
}
