package pitaya

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/sirupsen/logrus"
	pitayaclient "github.com/topfreegames/pitaya/v3/pkg/client"
	"github.com/topfreegames/pitaya/v3/pkg/session"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

type (
	// RootModule is the global module instance that will create Client
	// instances for each VU.
	RootModule struct{}

	// ModuleInstance represents an instance of the JS module.
	ModuleInstance struct {
		vu modules.VU
		*Client
		metrics *pitayaMetrics
	}
)

// Ensure the interfaces are implemented correctly
var (
	_ modules.Instance = &ModuleInstance{}
	_ modules.Module   = &RootModule{}
)

// New returns a pointer to a new RootModule instance
func New() *RootModule {
	return &RootModule{}
}

// NewModuleInstance implements the modules.Module interface and returns
// a new instance for each VU.
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	m, err := registerMetrics(vu)
	if err != nil {
		common.Throw(vu.Runtime(), err)
	}
	return &ModuleInstance{vu: vu, Client: &Client{vu: vu}, metrics: &m}
}

// Exports implements the modules.Instance interface and returns
// the exports of the JS module.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{Named: map[string]interface{}{
		"Client": mi.NewClient,
	}}
}

// NewClient is the JS constructor function for the Client type.
// It returns a new Client instance for each VU.
// The first argument is an options object with the following fields:
// - handshakeData: the handshake data to send to the server
// - requestTimeoutMs: the timeout for requests in milliseconds
// - logLevel: the log level to use
func (mi *ModuleInstance) NewClient(call goja.ConstructorCall) *goja.Object {
	rt := mi.vu.Runtime()

	var optionsArg map[string]interface{}
	err := rt.ExportTo(call.Arguments[0], &optionsArg)
	if err != nil {
		common.Throw(rt, errors.New("unable to parse options object"))
	}

	opts, err := newOptionsFrom(optionsArg)
	if err != nil {
		common.Throw(rt, fmt.Errorf("invalid options; reason: %w", err))
	}

	client := &Client{
		vu:        mi.vu,
		client:    nil,
		handshake: opts.HandshakeData,
		responses: make(map[uint]chan []byte, 100),
		pushes:    make(map[string]chan []byte, 100),
		timeout:   time.Duration(opts.RequestTimeoutMs) * time.Millisecond,
		metrics:   mi.metrics,
	}

	client.client = pitayaclient.New(logrus.InfoLevel)
	client.client.SetClientHandshakeData(opts.HandshakeData)

	return rt.ToValue(client).ToObject(rt)
}

type options struct {
	HandshakeData    *session.HandshakeData `json:"handshakeData"`
	RequestTimeoutMs int                    `json:"requestTimeoutMs"`
}

// newOptionsFrom validates and instantiates an options struct from its map representation
// as obtained by calling a Goja's Runtime.ExportTo.
func newOptionsFrom(argument map[string]interface{}) (*options, error) {
	jsonStr, err := json.Marshal(argument)
	if err != nil {
		return nil, fmt.Errorf("unable to serialize options to JSON %w", err)
	}

	// Instantiate a JSON decoder which will error on unknown
	// fields. As a result, if the input map contains an unknown
	// option, this function will produce an error.
	decoder := json.NewDecoder(bytes.NewReader(jsonStr))
	decoder.DisallowUnknownFields()

	var opts options
	err = decoder.Decode(&opts)
	if err != nil {
		return nil, fmt.Errorf("unable to decode options %w", err)
	}

	return &opts, nil
}
