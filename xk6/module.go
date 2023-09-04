package pitaya

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/sirupsen/logrus"
	pitayaclient "github.com/topfreegames/pitaya/v2/client"
	"github.com/topfreegames/pitaya/v2/session"
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
	return &ModuleInstance{vu: vu, Client: &Client{vu: vu}}
}

// Exports implements the modules.Instance interface and returns
// the exports of the JS module.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{Named: map[string]interface{}{
		"Client": mi.NewClient,
	}}
}

// NewClient is the JS constructor for the redis Client.
//
// Under the hood, the redis.UniversalClient will be used. The universal client
// supports failover/sentinel, cluster and single-node modes. Depending on the options,
// the internal universal client instance will be one of those.
//
// The type of the underlying client depends on the following conditions:
// 1. If the MasterName option is specified, a sentinel-backed FailoverClient is used.
// 2. if the number of Addrs is two or more, a ClusterClient is used.
// 3. Otherwise, a single-node Client is used.
//
// To support being instantiated in the init context, while not
// producing any IO, as it is the convention in k6, the produced
// Client is initially configured, but in a disconnected state. In
// order to connect to the configured target instance(s), the `.Connect`
// should be called.
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
	}

	switch opts.Serializer {
	case "json":
		client.client = pitayaclient.New(logrus.InfoLevel)
		break
	case "protobuf":
		if opts.DocsRoute == "" {
			common.Throw(rt, errors.New("docsRoute is required when using protobuf serializer"))
		}
		client.client = pitayaclient.NewProto(opts.DocsRoute, logrus.InfoLevel)
		break
	default:
		fmt.Printf("Serializer %s not supported, using json serializer", opts.Serializer)
		client.client = pitayaclient.New(logrus.InfoLevel)
		break
	}
	client.client.SetClientHandshakeData(opts.HandshakeData)

	return rt.ToValue(client).ToObject(rt)
}

type options struct {
	Addr             string                 `json:"addr"`
	HandshakeData    *session.HandshakeData `json:"handshakeData"`
	RequestTimeoutMs int                    `json:"requestTimeoutMs"`
	LogLevel         string                 `json:"logLevel"`
	Serializer       string                 `json:"serializer"`
	DocsRoute        string                 `json:"docsRoute"`
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
