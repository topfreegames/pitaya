package groups

import (
	"context"
	"time"
)

type (
	// GroupService has ranking methods
	GroupService interface {
		GroupCreate(ctx context.Context, groupName string) error
		GroupCreateWithTTL(ctx context.Context, groupName string, ttlTime time.Duration) error
		GroupDelete(ctx context.Context, groupName string) error
		GroupMembers(ctx context.Context, groupName string) ([]string, error)
		GroupContainsMember(ctx context.Context, groupName, uid string) (bool, error)
		GroupAddMember(ctx context.Context, groupName, uid string) error
		GroupRemoveMember(ctx context.Context, groupName, uid string) error
		GroupRemoveAll(ctx context.Context, groupName string) error
		GroupCountMembers(ctx context.Context, groupName string) (int, error)
		GroupRenewTTL(ctx context.Context, groupName string) error
	}
)

func elementIndex(slice []string, element string) (int, bool) {
	for i, sliceElement := range slice {
		if element == sliceElement {
			return i, true
		}
	}
	return 0, false
}
