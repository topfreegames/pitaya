package groups

import "context"

type (
	//Payload is the information that will be mantained inside each user in ETCD
	Payload struct {
		Metadata interface{} `json:"metadata"`
	}
	// GroupService has ranking methods
	GroupService interface {
		Subgroups(ctx context.Context, groupName string) ([]string, error)
		PlayerGroups(ctx context.Context, uid string) ([]string, error)
		PlayerSubgroups(ctx context.Context, groupName, uid string) ([]string, error)
		GroupMember(ctx context.Context, groupName, uid string) (*Payload, error)
		SubgroupMember(ctx context.Context, groupName, subgroupName, uid string) (*Payload, error)
		GroupMembers(ctx context.Context, groupName string) (map[string]*Payload, error)
		SubgroupMembers(ctx context.Context, groupName, subgroupName string) (map[string]*Payload, error)
		GroupContainsMember(ctx context.Context, groupName, uid string) (bool, error)
		SubgroupContainsMember(ctx context.Context, groupName, subgroupName, uid string) (bool, error)
		GroupAdd(ctx context.Context, groupName, uid string, payload *Payload) error
		SubgroupAdd(ctx context.Context, groupName, subgroupName, uid string, payload *Payload) error
		GroupLeave(ctx context.Context, groupName, uid string) error
		SubgroupLeave(ctx context.Context, groupName, subgroupName, uid string) error
		GroupLeaveAll(ctx context.Context, groupName string) error
		SubgroupLeaveAll(ctx context.Context, groupName, subgroupName string) error
		GroupCount(ctx context.Context, groupName string) (int, error)
		SubgroupCount(ctx context.Context, groupName, subgroupName string) (int, error)
	}
)
