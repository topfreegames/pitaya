package groups

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/namespace"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/logger"
)

var (
	clientInstance *clientv3.Client
	leaseTTL       time.Duration
	once           sync.Once
	leaseID        clientv3.LeaseID
)

// EtcdGroupService base ETCD struct solution
type EtcdGroupService struct {
}

// NewEtcdGroupService returns a new group instance
func NewEtcdGroupService(conf *config.Config, clientOrNil *clientv3.Client) (*EtcdGroupService, error) {
	err := initClientInstance(conf, clientOrNil)
	if err != nil {
		return nil, err
	}
	return &EtcdGroupService{}, err
}

func initClientInstance(config *config.Config, clientOrNil *clientv3.Client) error {
	var err error
	once.Do(func() {
		leaseTTL = config.GetDuration("pitaya.groups.etcd.leasettl")
		if clientOrNil != nil {
			clientInstance = clientOrNil
		} else {
			clientInstance, err = createBaseClient(config)
		}
		if err != nil {
			logger.Log.Fatalf("error initializing singleton etcd client in groups: %s", err.Error())
			return
		}
		err = bootstrapLease()
		if err != nil {
			logger.Log.Fatalf("error initializing bootstrap lease in groups: %s", err.Error())
			return
		}
	})
	return err
}

func createBaseClient(config *config.Config) (*clientv3.Client, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   config.GetStringSlice("pitaya.groups.etcd.endpoints"),
		DialTimeout: config.GetDuration("pitaya.groups.etcd.dialtimeout"),
	})
	if err != nil {
		return nil, err
	}
	cli.KV = namespace.NewKV(cli.KV, config.GetString("pitaya.groups.etcd.prefix"))
	return cli, nil
}

func bootstrapLease() error {
	// grab lease
	l, err := clientInstance.Grant(context.TODO(), int64(leaseTTL.Seconds()))
	if err != nil {
		return err
	}
	leaseID = l.ID
	logger.Log.Debugf("[groups] sd: got leaseID: %x", l.ID)
	// this will keep alive forever, when channel c is closed
	// it means we probably have to rebootstrap the lease
	c, err := clientInstance.KeepAlive(context.TODO(), leaseID)
	if err != nil {
		return err
	}
	// need to receive here as per etcd docs
	<-c
	go watchLeaseChan(c)
	return nil
}

func watchLeaseChan(c <-chan *clientv3.LeaseKeepAliveResponse) {
	for {
		kaRes := <-c
		if kaRes == nil {
			logger.Log.Warn("[groups] sd: error renewing etcd lease, rebootstrapping")
			for {
				err := bootstrapLease()
				if err != nil {
					logger.Log.Warn("[groups] sd: error rebootstrapping lease, will retry in 5 seconds")
					time.Sleep(5 * time.Second)
					continue
				} else {
					return
				}
			}
		}
	}
}

func userGroupKey(groupName, uid string) string {
	return "uids/" + uid + "/groups/" + groupName
}

func memberKey(groupName, uid string) string {
	return "groups/" + groupName + "/uids/" + uid
}

// MemberGroups returns all groups which member takes part
func (c *EtcdGroupService) MemberGroups(uid string) ([]string, error) {
	prefix := userGroupKey("", uid)
	etcdRes, err := clientInstance.Get(context.Background(), prefix, clientv3.WithKeysOnly(), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	groups := make([]string, etcdRes.Count)
	for i, kv := range etcdRes.Kvs {
		groups[i] = string(kv.Key)[len(prefix):]
	}
	return groups, nil
}

// Members returns all member's UID in current group
func (c *EtcdGroupService) Members(groupName string) (map[string]*Payload, error) {
	prefix := memberKey(groupName, "")
	etcdRes, err := clientInstance.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	members := make(map[string]*Payload, etcdRes.Count)
	for _, kv := range etcdRes.Kvs {
		payload := &Payload{}
		if err = json.Unmarshal(kv.Value, payload); err != nil {
			return nil, err
		}
		members[string(kv.Key)[len(prefix):]] = payload
	}
	return members, nil
}

// Member returns the Payload from User
func (c *EtcdGroupService) Member(groupName, uid string) (*Payload, error) {
	etcdRes, err := clientInstance.Get(context.Background(), memberKey(groupName, uid))
	if err != nil {
		return nil, err
	}
	payload := &Payload{}
	if err = json.Unmarshal(etcdRes.Kvs[0].Value, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// Contains check whether a UID is contained in current group or not
func (c *EtcdGroupService) Contains(groupName, uid string) (bool, error) {
	etcdRes, err := clientInstance.Get(context.Background(), memberKey(groupName, uid), clientv3.WithCountOnly())
	if err != nil {
		return false, err
	}
	return etcdRes.Count > 0, nil
}

// Add adds UID and payload to group. If the group doesn't exist, it is created
func (c *EtcdGroupService) Add(groupName, uid string, payload *Payload) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = clientInstance.Put(context.Background(), memberKey(groupName, uid), string(jsonPayload), clientv3.WithLease(leaseID))
	if err != nil {
		return err
	}
	_, err = clientInstance.Put(context.Background(), userGroupKey(groupName, uid), "", clientv3.WithLease(leaseID))
	return err
}

// Leave removes specified UID from group
func (c *EtcdGroupService) Leave(groupName, uid string) error {
	_, err := clientInstance.Delete(context.Background(), memberKey(groupName, uid))
	if err != nil {
		return err
	}
	_, err = clientInstance.Delete(context.Background(), userGroupKey(groupName, uid))
	return err
}

// LeaveAll clears all UIDs in the group
func (c *EtcdGroupService) LeaveAll(groupName string) error {
	dResp, err := clientInstance.Delete(context.Background(), memberKey(groupName, ""), clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, kv := range dResp.PrevKvs {
		_, err = clientInstance.Delete(context.Background(), string(kv.Key))
		if err != nil {
			logger.Log.Warn("[groups] sd: error deleting key from etcd")
		}
	}
	return err
}

// Count get current member amount in the group
func (c *EtcdGroupService) Count(groupName string) (int, error) {
	etcdRes, err := clientInstance.Get(context.Background(), memberKey(groupName, ""), clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		return 0, err
	}
	return int(etcdRes.Count), nil
}

// Close destroy group, which will release all resource in the group
func (c *EtcdGroupService) Close(groupName string) error {
	return c.LeaveAll(groupName)
}
