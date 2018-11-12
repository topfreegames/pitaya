package groups

import "context"

type (
	//Payload is the information that will be mantained inside each user in ETCD
	Payload struct {
		Metadata interface{}
	}
	// GroupService has ranking methods
	GroupService interface {
		MemberGroups(ctx context.Context, uid string) ([]string, error)
		Member(ctx context.Context, groupName, uid string) (*Payload, error)
		Members(ctx context.Context, groupName string) (map[string]*Payload, error)
		Contains(ctx context.Context, groupName, uid string) (bool, error)
		Add(ctx context.Context, groupName, uid string, payload *Payload) error
		Leave(ctx context.Context, groupName, uid string) error
		LeaveAll(ctx context.Context, groupName string) error
		Count(ctx context.Context, groupName string) (int, error)
		Close(ctx context.Context, groupName string) error
	}
)
