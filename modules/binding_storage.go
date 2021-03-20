// Copyright (c) TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package modules

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/namespace"
	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/session"
)

// ETCDBindingStorage module that uses etcd to keep in which frontend server each user is bound
type ETCDBindingStorage struct {
	Base
	config          *config.Config
	cli             *clientv3.Client
	etcdEndpoints   []string
	etcdPrefix      string
	etcdDialTimeout time.Duration
	leaseTTL        time.Duration
	leaseID         clientv3.LeaseID
	thisServer      *cluster.Server
	stopChan        chan struct{}
}

// NewETCDBindingStorage returns a new instance of BindingStorage
func NewETCDBindingStorage(server *cluster.Server, conf *config.Config) *ETCDBindingStorage {
	b := &ETCDBindingStorage{
		config:     conf,
		thisServer: server,
		stopChan:   make(chan struct{}),
	}
	b.configure()
	return b
}

func (b *ETCDBindingStorage) configure() {
	b.etcdDialTimeout = b.config.GetDuration("pitaya.modules.bindingstorage.etcd.dialtimeout")
	b.etcdEndpoints = b.config.GetStringSlice("pitaya.modules.bindingstorage.etcd.endpoints")
	b.etcdPrefix = b.config.GetString("pitaya.modules.bindingstorage.etcd.prefix")
	b.leaseTTL = b.config.GetDuration("pitaya.modules.bindingstorage.etcd.leasettl")
}

func getUserBindingKey(uid, frontendType string) string {
	return fmt.Sprintf("bindings/%s/%s", frontendType, uid)
}

// PutBinding puts the binding info into etcd
func (b *ETCDBindingStorage) PutBinding(uid string) error {
	_, err := b.cli.Put(context.Background(), getUserBindingKey(uid, b.thisServer.Type), b.thisServer.ID, clientv3.WithLease(b.leaseID))
	return err
}

func (b *ETCDBindingStorage) removeBinding(uid string) error {
	_, err := b.cli.Delete(context.Background(), getUserBindingKey(uid, b.thisServer.Type))
	return err
}

// GetUserFrontendID gets the id of the frontend server a user is connected to
// TODO: should we set context here?
// TODO: this could be way more optimized, using watcher and local caching
func (b *ETCDBindingStorage) GetUserFrontendID(uid, frontendType string) (string, error) {
	etcdRes, err := b.cli.Get(context.Background(), getUserBindingKey(uid, frontendType))
	if err != nil {
		return "", err
	}
	if len(etcdRes.Kvs) == 0 {
		return "", constants.ErrBindingNotFound
	}
	return string(etcdRes.Kvs[0].Value), nil
}

func (b *ETCDBindingStorage) setupOnSessionCloseCB() {
	session.OnSessionClose(func(s *session.Session) {
		if s.UID() != "" {
			err := b.removeBinding(s.UID())
			if err != nil {
				logger.Log.Errorf("error removing binding info from storage: %v", err)
			}
		}
	})
}

func (b *ETCDBindingStorage) setupOnAfterSessionBindCB() {
	session.OnAfterSessionBind(func(ctx context.Context, s *session.Session) error {
		return b.PutBinding(s.UID())
	})
}

func (b *ETCDBindingStorage) watchLeaseChan(c <-chan *clientv3.LeaseKeepAliveResponse) {
	for {
		select {
		case <-b.stopChan:
			return
		case kaRes := <-c:
			if kaRes == nil {
				logger.Log.Warn("[binding storage] sd: error renewing etcd lease, rebootstrapping")
				for {
					err := b.bootstrapLease()
					if err != nil {
						logger.Log.Warn("[binding storage] sd: error rebootstrapping lease, will retry in 5 seconds")
						time.Sleep(5 * time.Second)
						continue
					} else {
						return
					}
				}
			}
		}
	}
}

func (b *ETCDBindingStorage) bootstrapLease() error {
	// grab lease
	l, err := b.cli.Grant(context.TODO(), int64(b.leaseTTL.Seconds()))
	if err != nil {
		return err
	}
	b.leaseID = l.ID
	logger.Log.Debugf("[binding storage] sd: got leaseID: %x", l.ID)
	// this will keep alive forever, when channel c is closed
	// it means we probably have to rebootstrap the lease
	c, err := b.cli.KeepAlive(context.TODO(), b.leaseID)
	if err != nil {
		return err
	}
	// need to receive here as per etcd docs
	<-c
	go b.watchLeaseChan(c)
	return nil
}

// Init starts the binding storage module
func (b *ETCDBindingStorage) Init() error {
	var cli *clientv3.Client
	var err error
	if b.cli == nil {
		cli, err = clientv3.New(clientv3.Config{
			Endpoints:   b.etcdEndpoints,
			DialTimeout: b.etcdDialTimeout,
		})
		if err != nil {
			return err
		}
		b.cli = cli
	}
	// namespaced etcd :)
	b.cli.KV = namespace.NewKV(b.cli.KV, b.etcdPrefix)
	err = b.bootstrapLease()
	if err != nil {
		return err
	}

	if b.thisServer.Frontend {
		b.setupOnSessionCloseCB()
		b.setupOnAfterSessionBindCB()
	}

	return nil
}

// Shutdown executes on shutdown and will clean etcd
func (b *ETCDBindingStorage) Shutdown() error {
	close(b.stopChan)
	return b.cli.Close()
}
