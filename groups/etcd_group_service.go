package groups

import (
	"context"
	"encoding/json"
	"strings"
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

func groupPrefix(groupName string) string {
	return "groups/" + groupName
}

func subGroupPrefix(groupName, subgroupName string) string {
	return groupPrefix(groupName) + "/subgroups/" + subgroupName
}

func memberGroupKey(groupName, uid string) string {
	return "uids/" + uid + "/groups/" + groupName
}

func memberSubgroupKey(groupName, subgroupName, uid string) string {
	return groupPrefix(groupName) + "/uids/" + uid + "/subgroups/" + subgroupName
}

func memberKey(groupName, uid string) string {
	return groupPrefix(groupName) + "/uids/" + uid
}

func memberSubKey(groupName, subgroupName, uid string) string {
	return subGroupPrefix(groupName, subgroupName) + "/uids/" + uid
}

// Subgroups returns all subgroups that are contained inside given group
func (c *EtcdGroupService) Subgroups(ctx context.Context, groupName string) ([]string, error) {
	subgroupPrefix := subGroupPrefix(groupName, "")
	etcdRes, err := clientInstance.Get(ctx, subgroupPrefix, clientv3.WithKeysOnly(), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	subMap := make(map[string]bool)
	subgroups := make([]string, 0, len(etcdRes.Kvs))
	for _, kv := range etcdRes.Kvs {
		subgroupName := takeSubgroupName(string(kv.Key), subgroupPrefix)
		if _, ok := subMap[subgroupName]; !ok {
			subMap[subgroupName] = true
			subgroups = append(subgroups, subgroupName)
		}
	}
	return subgroups, nil
}

func takeSubgroupName(memberKey, subgroupPrefix string) string {
	memberKey = memberKey[len(subgroupPrefix):]
	return strings.Split(memberKey, "/uids/")[0]
}

// MemberGroups returns all groups which member takes part
func (c *EtcdGroupService) MemberGroups(ctx context.Context, uid string) ([]string, error) {
	return getGroupNames(ctx, memberGroupKey("", uid))
}

// MemberSubgroups returns all subgroups which member takes part
func (c *EtcdGroupService) MemberSubgroups(ctx context.Context, groupName, uid string) ([]string, error) {
	return getGroupNames(ctx, memberSubgroupKey(groupName, "", uid))
}

func getGroupNames(ctx context.Context, prefix string) ([]string, error) {
	etcdRes, err := clientInstance.Get(ctx, prefix, clientv3.WithKeysOnly(), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	groups := make([]string, etcdRes.Count)
	for i, kv := range etcdRes.Kvs {
		groups[i] = string(kv.Key)[len(prefix):]
	}
	return groups, nil
}

// Members returns all member's UID and payload in current group
func (c *EtcdGroupService) Members(ctx context.Context, groupName string) (map[string]*Payload, error) {
	return getMembers(ctx, memberKey(groupName, ""))
}

// SubgroupMembers returns all member's UID and payload in current subgroup
func (c *EtcdGroupService) SubgroupMembers(ctx context.Context, groupName, subgroupName string) (map[string]*Payload, error) {
	return getMembers(ctx, memberSubKey(groupName, subgroupName, ""))
}

func getMembers(ctx context.Context, prefix string) (map[string]*Payload, error) {
	etcdRes, err := clientInstance.Get(ctx, prefix, clientv3.WithPrefix())
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
func (c *EtcdGroupService) Member(ctx context.Context, groupName, uid string) (*Payload, error) {
	return getMember(ctx, memberKey(groupName, uid))
}

// SubgroupMember returns the Payload from User
func (c *EtcdGroupService) SubgroupMember(ctx context.Context, groupName, subgroupName, uid string) (*Payload, error) {
	return getMember(ctx, memberSubKey(groupName, subgroupName, uid))
}

func getMember(ctx context.Context, memberKey string) (*Payload, error) {
	etcdRes, err := clientInstance.Get(ctx, memberKey)
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
func (c *EtcdGroupService) Contains(ctx context.Context, groupName, uid string) (bool, error) {
	return contain(ctx, memberKey(groupName, uid))
}

// SubgroupContains check whether a UID is contained in current group or not
func (c *EtcdGroupService) SubgroupContains(ctx context.Context, groupName, subgroupName, uid string) (bool, error) {
	return contain(ctx, memberSubKey(groupName, subgroupName, uid))
}

func contain(ctx context.Context, memberKey string) (bool, error) {
	etcdRes, err := clientInstance.Get(ctx, memberKey, clientv3.WithCountOnly())
	if err != nil {
		return false, err
	}
	return etcdRes.Count > 0, nil
}

// Add adds UID and payload to group. If the group doesn't exist, it is created
func (c *EtcdGroupService) Add(ctx context.Context, groupName, uid string, payload *Payload) error {
	return add(ctx, map[string]*Payload{memberKey(groupName, uid): payload, memberGroupKey(groupName, uid): nil})
}

// SubgroupAdd adds UID and payload to subgroup. If the subgroup doesn't exist, it is created
func (c *EtcdGroupService) SubgroupAdd(ctx context.Context, groupName, subgroupName, uid string, payload *Payload) error {
	return add(ctx, map[string]*Payload{memberSubKey(groupName, subgroupName, uid): payload, memberSubgroupKey(groupName, subgroupName, uid): nil})
}

func add(ctx context.Context, kvs map[string]*Payload) error {
	for k, v := range kvs {
		jsonPayload, err := json.Marshal(v)
		if err != nil {
			return err
		}
		_, err = clientInstance.Put(ctx, k, string(jsonPayload), clientv3.WithLease(leaseID))
		if err != nil {
			return err
		}
	}
	return nil
}

// Leave removes specified UID from group
func (c *EtcdGroupService) Leave(ctx context.Context, groupName, uid string) error {
	return leave(ctx, memberKey(groupName, uid), memberGroupKey(groupName, uid))
}

// SubgroupLeave removes specified UID from group
func (c *EtcdGroupService) SubgroupLeave(ctx context.Context, groupName, subgroupName, uid string) error {
	return leave(ctx, memberSubKey(groupName, subgroupName, uid), memberSubgroupKey(groupName, subgroupName, uid))
}

func leave(ctx context.Context, memberKeys ...string) error {
	for _, memberKey := range memberKeys {
		_, err := clientInstance.Delete(ctx, memberKey)
		if err != nil {
			return err
		}
	}
	return nil
}

// LeaveAll clears all UIDs in the group
func (c *EtcdGroupService) LeaveAll(ctx context.Context, groupName string) error {
	dResp, err := clientInstance.Delete(ctx, groupPrefix(groupName), clientv3.WithPrefix())
	if err != nil {
		return err
	}
	prefix := memberKey(groupName, "")
	for _, kv := range dResp.PrevKvs {
		if strings.Contains(string(kv.Key), prefix) {
			uid := string(kv.Key)[len(prefix):]
			if err = leave(ctx, memberGroupKey(groupName, uid)); err != nil {
				logger.Log.Warnf("[groups] sd: error deleting key from etcd: %s", err.Error())
			}
		}
	}
	return err
}

// SubgroupLeaveAll clears all UIDs in the subgroup
func (c *EtcdGroupService) SubgroupLeaveAll(ctx context.Context, groupName, subgroupName string) error {
	prefix := memberSubKey(groupName, subgroupName, "")
	dResp, err := clientInstance.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, kv := range dResp.PrevKvs {
		uid := string(kv.Key)[len(prefix):]
		if err = leave(ctx, memberSubgroupKey(groupName, subgroupName, uid)); err != nil {
			logger.Log.Warn("[subgroups] sd: error deleting key from etcd")
		}
	}
	return err
}

// Count get current member amount in the group
func (c *EtcdGroupService) Count(ctx context.Context, groupName string) (int, error) {
	return count(ctx, memberKey(groupName, ""))
}

// SubgroupCount get current member amount in the group
func (c *EtcdGroupService) SubgroupCount(ctx context.Context, groupName, subgroupName string) (int, error) {
	return count(ctx, memberSubKey(groupName, subgroupName, ""))
}

func count(ctx context.Context, memberKeyPrefix string) (int, error) {
	etcdRes, err := clientInstance.Get(ctx, memberKeyPrefix, clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		return 0, err
	}
	return int(etcdRes.Count), nil
}
