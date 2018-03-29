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

package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// TODO allow receiving a config from the user

func fillDefaultValues() {
	viper.SetDefault("pitaya.buffer.agent.messages", 16)
	viper.SetDefault("pitaya.buffer.cluster.rpc.server.messages", 1000)
	viper.SetDefault("pitaya.buffer.cluster.rpc.server.push", 100)
	viper.SetDefault("pitaya.buffer.handler.localprocess", 10)
	viper.SetDefault("pitaya.buffer.handler.remoteprocess", 10)
	viper.SetDefault("pitaya.buffer.timer", 1<<8)

	viper.SetDefault("pitaya.concurrency.handler.dispatch", 10)
	viper.SetDefault("pitaya.concurrency.remote.service", 10)

	viper.SetDefault("pitaya.cluster.rpc.client.nats.connect", "nats://localhost:4222")
	viper.SetDefault("pitaya.cluster.rpc.client.nats.requesttimeout", "5s")
	viper.SetDefault("pitaya.cluster.rpc.server.nats.connect", "nats://localhost:4222")
	viper.SetDefault("pitaya.cluster.sd.etcd.dialtimeout", "5s")
	viper.SetDefault("pitaya.cluster.sd.etcd.endpoints", "localhost:2379")
	viper.SetDefault("pitaya.cluster.sd.etcd.heartbeat.interval", "20s")
	viper.SetDefault("pitaya.cluster.sd.etcd.heartbeat.ttl", "60s")
	viper.SetDefault("pitaya.cluster.sd.etcd.syncservers.interval", "120s")

	viper.SetDefault("pitaya.heartbeat.interval", "30s")
}

// GetConcurrency retrieves concurrency config for a given suffix
func GetConcurrency(s string) int {
	return viper.GetInt(fmt.Sprintf("pitaya.concurrency.%s", s))
}

// GetBuffer retrieves buffer config for a given suffix
func GetBuffer(s string) int {
	return viper.GetInt(fmt.Sprintf("pitaya.buffer.%s", s))
}

func init() {
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	fillDefaultValues()
}
