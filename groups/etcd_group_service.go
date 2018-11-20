package groups

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/namespace"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
)

var (
	clientInstance *clientv3.Client
	etcdOnce       sync.Once
)

// EtcdGroupService base ETCD struct solution
type EtcdGroupService struct {
}

// EtcdGroupPayload is the payload stored in each Etcd group key
type EtcdGroupPayload struct {
	Uids    []string         `json:"uids"`
	LeaseID clientv3.LeaseID `json:"leaseID"`
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
	etcdOnce.Do(func() {
		if clientOrNil != nil {
			clientInstance = clientOrNil
		} else {
			clientInstance, err = createBaseClient(config)
		}
		if err != nil {
			logger.Log.Fatalf("error initializing singleton etcd client in groups: %s", err.Error())
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

func groupKey(groupName string) string {
	return fmt.Sprintf("groups/%s", groupName)
}

func getGroupPayload(ctx context.Context, groupName string) (*EtcdGroupPayload, error) {
	etcdRes, err := clientInstance.Get(ctx, groupKey(groupName))
	if err != nil {
		return nil, err
	}
	if etcdRes.Count == 0 {
		return nil, constants.ErrGroupNotFound
	}

	payload := &EtcdGroupPayload{}
	if err = json.Unmarshal(etcdRes.Kvs[0].Value, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func putGroupPayload(ctx context.Context, groupName string, payload *EtcdGroupPayload) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if payload.LeaseID != 0 {
		_, err = clientInstance.Put(ctx, groupKey(groupName), string(jsonPayload), clientv3.WithLease(payload.LeaseID))
	} else {
		_, err = clientInstance.Put(ctx, groupKey(groupName), string(jsonPayload))
	}
	if err != nil {
		return err
	}

	return nil
}

// GroupCreate returns all member's UID and payload in current group
func (c *EtcdGroupService) GroupCreate(ctx context.Context, groupName string) error {
	etcdRes, err := clientInstance.Get(ctx, groupKey(groupName), clientv3.WithCountOnly())
	if err != nil {
		return err
	}
	if etcdRes.Count != 0 {
		return constants.ErrGroupAlreadyExists
	}
	return putGroupPayload(ctx, groupName, &EtcdGroupPayload{})
}

// GroupCreateWithTTL returns all member's UID and payload in current group
func (c *EtcdGroupService) GroupCreateWithTTL(ctx context.Context, groupName string, ttlTime time.Duration) error {
	etcdRes, err := clientInstance.Get(ctx, groupKey(groupName), clientv3.WithCountOnly())
	if err != nil {
		return err
	}
	if etcdRes.Count != 0 {
		return constants.ErrGroupAlreadyExists
	}

	lease, err := clientInstance.Grant(ctx, int64(ttlTime.Seconds()))
	if err != nil {
		return err
	}
	return putGroupPayload(ctx, groupName, &EtcdGroupPayload{LeaseID: lease.ID})
}

// GroupMembers returns all member's UID and payload in current group
func (c *EtcdGroupService) GroupMembers(ctx context.Context, groupName string) ([]string, error) {
	payload, err := getGroupPayload(ctx, groupName)
	if err != nil {
		return nil, err
	}
	return payload.Uids, nil
}

// GroupContainsMember check whether a UID is contained in current group or not
func (c *EtcdGroupService) GroupContainsMember(ctx context.Context, groupName, uid string) (bool, error) {
	payload, err := getGroupPayload(ctx, groupName)
	if err != nil {
		return false, err
	}

	_, contains := elementIndex(payload.Uids, uid)

	return contains, nil
}

// GroupAddMember adds UID and payload to group. If the group doesn't exist, it is created
func (c *EtcdGroupService) GroupAddMember(ctx context.Context, groupName, uid string) error {
	payload, err := getGroupPayload(ctx, groupName)
	if err != nil {
		return err
	}

	_, contains := elementIndex(payload.Uids, uid)
	if contains {
		return constants.ErrMemberAlreadyExists
	}

	payload.Uids = append(payload.Uids, uid)
	return putGroupPayload(ctx, groupName, payload)
}

// GroupRemoveMember removes specified UID from group
func (c *EtcdGroupService) GroupRemoveMember(ctx context.Context, groupName, uid string) error {
	payload, err := getGroupPayload(ctx, groupName)
	if err != nil {
		return err
	}
	index, contains := elementIndex(payload.Uids, uid)
	if contains {
		payload.Uids[index] = payload.Uids[len(payload.Uids)-1]
		payload.Uids = payload.Uids[:len(payload.Uids)-1]
		return putGroupPayload(ctx, groupName, payload)
	}

	return constants.ErrMemberNotFound
}

// GroupRemoveAll clears all UIDs in the group and also subgroups contained
func (c *EtcdGroupService) GroupRemoveAll(ctx context.Context, groupName string) error {
	_, err := clientInstance.Delete(ctx, groupKey(groupName))
	return err
}

// GroupCountMembers get current member amount in the group
func (c *EtcdGroupService) GroupCountMembers(ctx context.Context, groupName string) (int, error) {
	payload, err := getGroupPayload(ctx, groupName)
	if err != nil {
		return 0, err
	}
	return len(payload.Uids), nil
}

// GroupRenewTTL will create/renew lease TTL
func (c *EtcdGroupService) GroupRenewTTL(ctx context.Context, groupName string) error {
	payload, err := getGroupPayload(ctx, groupName)
	if err != nil {
		return err
	}
	if payload.LeaseID != 0 {
		_, err = clientInstance.KeepAliveOnce(ctx, payload.LeaseID)
		return err
	}
	return constants.ErrEtcdLeaseNotFound
}
