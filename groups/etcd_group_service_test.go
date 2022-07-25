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

package groups

import (
	"testing"

	"github.com/topfreegames/pitaya/v2/config"
	"go.etcd.io/etcd/tests/v3/framework/integration"
)

func setup(t *testing.T) (*integration.Cluster, GroupService) {
	integration.BeforeTest(t)
	cluster := integration.NewCluster(t, &integration.ClusterConfig{Size: 1})
	cli := cluster.RandClient()
	etcdGroupService, err := NewEtcdGroupService(*config.NewDefaultEtcdGroupServiceConfig(), cli)
	if err != nil {
		panic(err)
	}

	return cluster, etcdGroupService
}

func TestEtcdCreateDuplicatedGroup(t *testing.T) {
	cluster, etcdGroupService := setup(t)
	defer cluster.Terminate(t)
	testCreateDuplicatedGroup(etcdGroupService, t)
}

func TestEtcdCreateGroup(t *testing.T) {
	cluster, etcdGroupService := setup(t)
	defer cluster.Terminate(t)
	testCreateGroup(etcdGroupService, t)
}

func TestEtcdCreateGroupWithTTL(t *testing.T) {
	cluster, etcdGroupService := setup(t)
	defer cluster.Terminate(t)
	testCreateGroupWithTTL(etcdGroupService, t)
}

func TestEtcdGroupAddMember(t *testing.T) {
	cluster, etcdGroupService := setup(t)
	defer cluster.Terminate(t)
	testGroupAddMember(etcdGroupService, t)
}

func TestEtcdGroupAddDuplicatedMember(t *testing.T) {
	cluster, etcdGroupService := setup(t)
	defer cluster.Terminate(t)
	testGroupAddDuplicatedMember(etcdGroupService, t)
}

func TestEtcdGroupContainsMember(t *testing.T) {
	cluster, etcdGroupService := setup(t)
	defer cluster.Terminate(t)
	testGroupContainsMember(etcdGroupService, t)
}

func TestEtcdRemove(t *testing.T) {
	cluster, etcdGroupService := setup(t)
	defer cluster.Terminate(t)
	testRemove(etcdGroupService, t)
}

func TestEtcdDelete(t *testing.T) {
	cluster, etcdGroupService := setup(t)
	defer cluster.Terminate(t)
	testDelete(etcdGroupService, t)
}

func TestEtcdRemoveAll(t *testing.T) {
	cluster, etcdGroupService := setup(t)
	defer cluster.Terminate(t)
	testRemoveAll(etcdGroupService, t)
}

func TestEtcdCount(t *testing.T) {
	cluster, etcdGroupService := setup(t)
	defer cluster.Terminate(t)
	testCount(etcdGroupService, t)
}

func TestEtcdMembers(t *testing.T) {
	cluster, etcdGroupService := setup(t)
	defer cluster.Terminate(t)
	testMembers(etcdGroupService, t)
}
