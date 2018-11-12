package groups

type (
	//Payload is the information that will be mantained inside each user in ETCD
	Payload struct {
		Metadata interface{}
	}
	// GroupService has ranking methods
	GroupService interface {
		MemberGroups(uid string) ([]string, error)
		Member(groupName, uid string) (*Payload, error)
		Members(groupName string) (map[string]*Payload, error)
		Contains(groupName, uid string) (bool, error)
		Add(groupName, uid string, payload *Payload) error
		Leave(groupName, uid string) error
		LeaveAll(groupName string) error
		Count(groupName string) (int, error)
		Close(groupName string) error
	}
)
