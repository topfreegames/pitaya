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

package worker

import (
	"context"
	"encoding/json"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"

	workers "github.com/topfreegames/go-workers"
)

// Worker executes RPCs with retry and backoff time
type Worker struct {
	concurrency int
	registered  bool
	opts        *EnqueueOpts
	config      *config.Config
	started     bool
}

// NewWorker configures and returns a *Worker
func NewWorker(config *config.Config) (*Worker, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	workers.Configure(map[string]string{
		"server":    config.GetString("pitaya.worker.redis.url"),
		"pool":      config.GetString("pitaya.worker.redis.pool"),
		"password":  config.GetString("pitaya.worker.redis.password"),
		"namespace": config.GetString("pitaya.worker.namespace"),
		"process":   hostname,
	})

	return &Worker{
		concurrency: config.GetInt("pitaya.worker.concurrency"),
		opts:        NewEnqueueOpts(config),
		config:      config,
	}, nil
}

// SetLogger overwrites worker logger
func (w *Worker) SetLogger(logger logger.Logger) {
	workers.Logger = logger
}

// Start starts worker in another gorotine
func (w *Worker) Start() {
	go workers.Start()
	w.started = true
}

// Started returns true if worker was started
func (w *Worker) Started() bool {
	return w != nil && w.started
}

// EnqueueRPC enqueues rpc job to worker
func (w *Worker) EnqueueRPC(
	routeStr string,
	metadata map[string]interface{},
	reply, arg proto.Message,
) (jid string, err error) {
	opts := w.enqueueOptions(w.opts)
	return workers.EnqueueWithOptions(rpcQueue, class, &rpcInfo{
		Route:    routeStr,
		Metadata: metadata,
		Arg:      arg,
		Reply:    reply,
	}, opts)
}

// EnqueueRPCWithOptions enqueues rpc job to worker
func (w *Worker) EnqueueRPCWithOptions(
	routeStr string,
	metadata map[string]interface{},
	reply, arg proto.Message,
	opts *EnqueueOpts,
) (jid string, err error) {
	return workers.EnqueueWithOptions(rpcQueue, class, &rpcInfo{
		Route:    routeStr,
		Metadata: metadata,
		Arg:      arg,
		Reply:    reply,
	}, w.enqueueOptions(opts))
}

// RegisterRPCJob registers a RPC job
func (w *Worker) RegisterRPCJob(rpcJob RPCJob) error {
	if w.registered {
		return constants.ErrRPCJobAlreadyRegistered
	}

	job := w.parsedRPCJob(rpcJob)
	workers.Process(rpcQueue, job, w.concurrency)
	w.registered = true
	return nil
}

func (w *Worker) parsedRPCJob(rpcJob RPCJob) func(*workers.Msg) {
	return func(jobArg *workers.Msg) {
		logger.Log.Debug("executing rpc job")
		bts, rpcRoute, err := w.unmarshalRouteMetadata(jobArg)
		if err != nil {
			logger.Log.Errorf("failed to get job arg: %q", err)
			panic(err)
		}

		logger.Log.Debug("getting route arg and reply")
		arg, reply, err := rpcJob.GetArgReply(rpcRoute.Route)
		if err != nil {
			logger.Log.Errorf("failed to get methods arg and reply: %q", err)
			panic(err)
		}
		rpcInfo := &rpcInfo{
			Arg:   arg,
			Reply: reply,
		}

		logger.Log.Debug("unmarshalling rpc info")
		err = json.Unmarshal(bts, rpcInfo)
		if err != nil {
			logger.Log.Errorf("failed to unmarshal rpc info: %q", err)
			panic(err)
		}

		logger.Log.Debug("choosing server to make rpc")
		serverID, err := rpcJob.ServerDiscovery(rpcInfo.Route, rpcInfo.Metadata)
		if err != nil {
			logger.Log.Errorf("failed get server: %q", err)
			panic(err)
		}

		ctx := context.Background()

		logger.Log.Debugf("executing rpc func to %s", rpcInfo.Route)
		err = rpcJob.RPC(ctx, serverID, rpcInfo.Route, reply, arg)
		if err != nil {
			logger.Log.Errorf("failed make rpc: %q", err)
			panic(err)
		}

		logger.Log.Debug("finished executing rpc job")
	}
}

func (w *Worker) enqueueOptions(
	opts *EnqueueOpts,
) workers.EnqueueOptions {
	return workers.EnqueueOptions{
		Retry:    opts.RetryEnabled,
		RetryMax: opts.MaxRetries,
		RetryOptions: workers.RetryOptions{
			Exp:      opts.ExponentialFactor,
			MinDelay: opts.MinDelayToRetry,
			MaxDelay: opts.MaxDelayToRetry,
			MaxRand:  opts.MaxRandom,
		},
	}
}

func (w *Worker) unmarshalRouteMetadata(
	jobArg *workers.Msg,
) ([]byte, *rpcRoute, error) {
	bts, err := jobArg.Args().MarshalJSON()
	if err != nil {
		return nil, nil, err
	}

	rpcRoute := new(rpcRoute)
	err = json.Unmarshal(bts, rpcRoute)
	if err != nil {
		return nil, nil, err
	}

	return bts, rpcRoute, nil
}
