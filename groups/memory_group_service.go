package groups

import (
	"context"
	"sync"
	"time"

	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
)

var (
	memoryGroupsMu sync.RWMutex
	memoryGroups   map[string]*MemoryGroup
	memoryOnce     sync.Once
)

// MemoryGroupService base in server memory solution
type MemoryGroupService struct {
}

// MemoryGroup is the struct stored in each group key(which is the name of the group)
type MemoryGroup struct {
	Uids        []string
	LastRefresh int64
	TTL         int64
}

// NewMemoryGroupService returns a new group instance
func NewMemoryGroupService(conf *config.Config) *MemoryGroupService {
	memoryOnce.Do(func() {
		memoryGroups = make(map[string]*MemoryGroup)
		go groupTTLCleanup(conf)
	})
	return &MemoryGroupService{}
}

func groupTTLCleanup(conf *config.Config) {
	for now := range time.Tick(conf.GetDuration("pitaya.groups.memory.tickduration")) {
		memoryGroupsMu.Lock()
		for groupName, mg := range memoryGroups {
			if mg.TTL != 0 && now.UnixNano()-mg.LastRefresh > mg.TTL {
				delete(memoryGroups, groupName)
			}
		}
		memoryGroupsMu.Unlock()
	}
}

// GroupCreate creates a group without TTL
func (c *MemoryGroupService) GroupCreate(ctx context.Context, groupName string) error {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	if _, ok := memoryGroups[groupName]; ok {
		return constants.ErrGroupAlreadyExists
	}

	memoryGroups[groupName] = &MemoryGroup{}
	return nil
}

// GroupCreateWithTTL creates a group with TTL, which the go routine will clean later
func (c *MemoryGroupService) GroupCreateWithTTL(ctx context.Context, groupName string, ttlTime time.Duration) error {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	if _, ok := memoryGroups[groupName]; ok {
		return constants.ErrGroupAlreadyExists
	}

	memoryGroups[groupName] = &MemoryGroup{LastRefresh: time.Now().UnixNano(), TTL: ttlTime.Nanoseconds()}
	return nil
}

// GroupMembers returns all member's UID in given group
func (c *MemoryGroupService) GroupMembers(ctx context.Context, groupName string) ([]string, error) {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	mg, ok := memoryGroups[groupName]
	if !ok {
		return nil, constants.ErrGroupNotFound
	}

	uids := make([]string, len(mg.Uids))
	copy(uids, mg.Uids)

	return uids, nil
}

// GroupContainsMember check whether an UID is contained in given group or not
func (c *MemoryGroupService) GroupContainsMember(ctx context.Context, groupName, uid string) (bool, error) {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	mg, ok := memoryGroups[groupName]
	if !ok {
		return false, constants.ErrGroupNotFound
	}

	_, contains := elementIndex(mg.Uids, uid)
	return contains, nil
}

// GroupAddMember adds UID to group
func (c *MemoryGroupService) GroupAddMember(ctx context.Context, groupName, uid string) error {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	mg, ok := memoryGroups[groupName]
	if !ok {
		return constants.ErrGroupNotFound
	}

	_, contains := elementIndex(mg.Uids, uid)
	if contains {
		return constants.ErrMemberAlreadyExists
	}

	mg.Uids = append(mg.Uids, uid)
	memoryGroups[groupName] = mg
	return nil
}

// GroupRemoveMember removes specific UID from group
func (c *MemoryGroupService) GroupRemoveMember(ctx context.Context, groupName, uid string) error {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	mg, ok := memoryGroups[groupName]
	if !ok {
		return constants.ErrGroupNotFound
	}
	index, contains := elementIndex(mg.Uids, uid)
	if contains {
		mg.Uids[index] = mg.Uids[len(mg.Uids)-1]
		mg.Uids = mg.Uids[:len(mg.Uids)-1]
		memoryGroups[groupName] = mg
		return nil
	}

	return constants.ErrMemberNotFound
}

// GroupRemoveAll clears all UIDs from group
func (c *MemoryGroupService) GroupRemoveAll(ctx context.Context, groupName string) error {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	mg, ok := memoryGroups[groupName]
	if !ok {
		return constants.ErrGroupNotFound
	}

	mg.Uids = []string{}
	return nil
}

// GroupDelete deletes the whole group, including members and base group
func (c *MemoryGroupService) GroupDelete(ctx context.Context, groupName string) error {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	_, ok := memoryGroups[groupName]
	if !ok {
		return constants.ErrGroupNotFound
	}

	delete(memoryGroups, groupName)
	return nil
}

// GroupCountMembers get current member amount in group
func (c *MemoryGroupService) GroupCountMembers(ctx context.Context, groupName string) (int, error) {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	mg, ok := memoryGroups[groupName]
	if !ok {
		return 0, constants.ErrGroupNotFound
	}

	return len(mg.Uids), nil
}

// GroupRenewTTL will renew lease TTL
func (c *MemoryGroupService) GroupRenewTTL(ctx context.Context, groupName string) error {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	mg, ok := memoryGroups[groupName]
	if !ok {
		return constants.ErrGroupNotFound
	}

	if mg.TTL != 0 {
		mg.LastRefresh = time.Now().UnixNano()
		memoryGroups[groupName] = mg
		return nil
	}
	return constants.ErrMemoryTTLNotFound
}
