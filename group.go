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
	"sync"
	"time"

	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/groups"
	"github.com/topfreegames/pitaya/logger"
)

// Group represents an agglomeration of UIDs which is used to manage
// users. Data sent to the group will be sent to all users in it.

var (
	groupServiceInstance groups.GroupService
	groupsOnce           sync.Once
)

// InitGroups should be called once at the beginning of the application to setup the service type that will manage the groups
func InitGroups(groupService groups.GroupService) {
	groupsOnce.Do(func() {
		groupServiceInstance = groupService
	})
}

// GroupCreate creates a group
func GroupCreate(ctx context.Context, groupName string) error {
	return groupServiceInstance.GroupCreate(ctx, groupName)
}

// GroupCreateWithTTL creates a group with given TTL
func GroupCreateWithTTL(ctx context.Context, groupName string, ttlTime time.Duration) error {
	return groupServiceInstance.GroupCreateWithTTL(ctx, groupName, ttlTime)
}

// GroupMembers returns all member's UIDs
func GroupMembers(ctx context.Context, groupName string) ([]string, error) {
	return groupServiceInstance.GroupMembers(ctx, groupName)
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

func sendDataToMembers(uids []string, frontendType, route string, v interface{}) error {
	errUids, err := SendPushToUsers(route, v, uids, frontendType)
	if err != nil {
		logger.Log.Errorf("Group push message error, UID=%v, Error=%s", errUids, err.Error())
		return err
	}
	return nil
}

// GroupContainsMember checks whether an UID is contained in group or not
func GroupContainsMember(ctx context.Context, groupName, uid string) (bool, error) {
	if uid == "" {
		return false, constants.ErrEmptyUID
	}
	return groupServiceInstance.GroupContainsMember(ctx, groupName, uid)
}

// GroupAddMember adds UID to group
func GroupAddMember(ctx context.Context, groupName, uid string) error {
	if uid == "" {
		return constants.ErrEmptyUID
	}
	logger.Log.Debugf("Add user to group %s, UID=%s", groupName, uid)
	return groupServiceInstance.GroupAddMember(ctx, groupName, uid)
}

// GroupRemoveMember removes specified UID from group
func GroupRemoveMember(ctx context.Context, groupName, uid string) error {
	logger.Log.Debugf("Remove user from group %s, UID=%s", groupName, uid)
	return groupServiceInstance.GroupRemoveMember(ctx, groupName, uid)
}

// GroupRemoveAll clears all UIDs
func GroupRemoveAll(ctx context.Context, groupName string) error {
	return groupServiceInstance.GroupRemoveAll(ctx, groupName)
}

// GroupCountMembers get current member amount in group
func GroupCountMembers(ctx context.Context, groupName string) (int, error) {
	return groupServiceInstance.GroupCountMembers(ctx, groupName)
}

// GroupRenewTTL renews group with the initial TTL
func GroupRenewTTL(ctx context.Context, groupName string) error {
	return groupServiceInstance.GroupRenewTTL(ctx, groupName)
}

// GroupDelete deletes whole group, including UIDs and base group
func GroupDelete(ctx context.Context, groupName string) error {
	return groupServiceInstance.GroupDelete(ctx, groupName)
}
