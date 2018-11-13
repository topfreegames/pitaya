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
		MemberGroups(ctx context.Context, uid string) ([]string, error)
		MemberSubgroups(ctx context.Context, groupName, uid string) ([]string, error)
		Member(ctx context.Context, groupName, uid string) (*Payload, error)
		SubgroupMember(ctx context.Context, groupName, subgroupName, uid string) (*Payload, error)
		Members(ctx context.Context, groupName string) (map[string]*Payload, error)
		SubgroupMembers(ctx context.Context, groupName, subgroupName string) (map[string]*Payload, error)
		Contains(ctx context.Context, groupName, uid string) (bool, error)
		SubgroupContains(ctx context.Context, groupName, subgroupName, uid string) (bool, error)
		Add(ctx context.Context, groupName, uid string, payload *Payload) error
		SubgroupAdd(ctx context.Context, groupName, subgroupName, uid string, payload *Payload) error
		Leave(ctx context.Context, groupName, uid string) error
		SubgroupLeave(ctx context.Context, groupName, subgroupName, uid string) error
		LeaveAll(ctx context.Context, groupName string) error
		SubgroupLeaveAll(ctx context.Context, groupName, subgroupName string) error
		Count(ctx context.Context, groupName string) (int, error)
		SubgroupCount(ctx context.Context, groupName, subgroupName string) (int, error)
	}
)
