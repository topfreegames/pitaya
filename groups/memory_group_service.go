package groups

import (
	"context"
	"sync"
	"time"

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
func NewMemoryGroupService() *MemoryGroupService {
	memoryOnce.Do(func() {
		memoryGroups = make(map[string]*MemoryGroup)
		go func() {
			for now := range time.Tick(time.Second) {
				memoryGroupsMu.Lock()
				for groupName, mg := range memoryGroups {
					if mg.TTL != 0 && now.UnixNano()-mg.LastRefresh > mg.TTL {
						delete(memoryGroups, groupName)
					}
				}
				memoryGroupsMu.Unlock()
			}
		}()
	})
	return &MemoryGroupService{}
}

func getMemoryGroup(groupName string) (*MemoryGroup, error) {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	mg, ok := memoryGroups[groupName]
	if !ok {
		return nil, constants.ErrGroupNotFound
	}

	return mg, nil
}

// GroupCreate returns all member's UID and payload in current group
func (c *MemoryGroupService) GroupCreate(ctx context.Context, groupName string) error {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	if _, ok := memoryGroups[groupName]; ok {
		return constants.ErrGroupDuplication
	}

	memoryGroups[groupName] = &MemoryGroup{}
	return nil
}

// GroupCreateWithTTL returns all member's UID and payload in current group
func (c *MemoryGroupService) GroupCreateWithTTL(ctx context.Context, groupName string, ttlTime time.Duration) error {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	if _, ok := memoryGroups[groupName]; ok {
		return constants.ErrGroupDuplication
	}

	memoryGroups[groupName] = &MemoryGroup{LastRefresh: time.Now().UnixNano(), TTL: ttlTime.Nanoseconds()}
	return nil
}

// GroupMembers returns all member's UID and payload in current group
func (c *MemoryGroupService) GroupMembers(ctx context.Context, groupName string) ([]string, error) {
	mg, err := getMemoryGroup(groupName)
	if err != nil {
		return nil, err
	}

	return mg.Uids, nil
}

// GroupContainsMember check whether a UID is contained in current group or not
func (c *MemoryGroupService) GroupContainsMember(ctx context.Context, groupName, uid string) (bool, error) {
	mg, err := getMemoryGroup(groupName)
	if err != nil {
		return false, err
	}

	_, contains := elementIndex(mg.Uids, uid)
	return contains, nil
}

// GroupAddMember adds UID and payload to group. If the group doesn't exist, it is created
func (c *MemoryGroupService) GroupAddMember(ctx context.Context, groupName, uid string) error {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	mg, ok := memoryGroups[groupName]
	if !ok {
		return constants.ErrGroupNotFound
	}

	_, contains := elementIndex(mg.Uids, uid)
	if contains {
		return constants.ErrMemberDuplication
	}

	mg.Uids = append(mg.Uids, uid)
	memoryGroups[groupName] = mg
	return nil
}

// GroupRemoveMember removes specified UID from group
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

// GroupRemoveAll clears all UIDs in the group and also subgroups contained
func (c *MemoryGroupService) GroupRemoveAll(ctx context.Context, groupName string) error {
	memoryGroupsMu.Lock()
	defer memoryGroupsMu.Unlock()

	_, ok := memoryGroups[groupName]
	if !ok {
		return constants.ErrGroupNotFound
	}

	delete(memoryGroups, groupName)
	return nil
}

// GroupCountMembers get current member amount in the group
func (c *MemoryGroupService) GroupCountMembers(ctx context.Context, groupName string) (int, error) {
	mg, err := getMemoryGroup(groupName)
	if err != nil {
		return 0, err
	}
	return len(mg.Uids), nil
}

// GroupRenewTTL will create/renew lease TTL
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
