package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/serialize/json"
	validator "gopkg.in/go-playground/validator.v9"
)

// MetagameServer ...
type MetagameServer struct {
	component.Base
	Logger logrus.FieldLogger
}

// NewMetagameMock ...
func NewMetagameMock() *MetagameServer {
	return &MetagameServer{
		Logger: logrus.New(),
	}
}

// CreatePlayerCheatArgs ...
type CreatePlayerCheatArgs struct {
	Name         string `json:"name"`
	Email        string `json:"email" validate:"email"`
	SoftCurrency int    `json:"softCurrency" validate:"gte=0,lte=1000"`
	HardCurrency int    `json:"hardCurrency" validate:"gte=0,lte=200"`
}

// CreatePlayerCheatResponse ...
type CreatePlayerCheatResponse struct {
	Msg string `json:"msg"`
}

// CreatePlayerCheat ...
func (g *MetagameServer) CreatePlayerCheat(ctx context.Context, args *CreatePlayerCheatArgs) (*CreatePlayerCheatResponse, error) {
	// Do nothing. This is just an example of how pipelines can be helpful
	return &CreatePlayerCheatResponse{
		Msg: "ok",
	}, nil
}

// This is a beforeHandler that validates the handler argument based on the struct tags.
// As for this example, the CreatePlayerCheatArgs has the 'validate' tags for email,
// softCurrency and hardCurrency. If any of the validations fail an error will be returned
func handlerParamsValidator(ctx context.Context, in interface{}) (interface{}, error) {
	var validate *validator.Validate
	validate = validator.New()

	if err := validate.Struct(in); err != nil {
		return nil, err
	}

	return in, nil
}

// Simple example of a before pipeline that actually asserts the type of the
// in parameter.
// IMPORTANT: that this kind of pipeline will be hard to exist in real code
// as a pipeline function executes for every handler and each of them
// most probably have different parameter types.
func (g *MetagameServer) simpleBefore(ctx context.Context, in interface{}) (interface{}, error) {
	g.Logger.Info("Simple Before exec")
	createPlayerArgs := in.(*CreatePlayerCheatArgs)

	g.Logger.Infof("Name: %s", createPlayerArgs.Name)
	g.Logger.Infof("Email: %s", createPlayerArgs.Email)
	g.Logger.Infof("SoftCurrency: %d", createPlayerArgs.SoftCurrency)
	g.Logger.Infof("HardCurrency: %d", createPlayerArgs.HardCurrency)

	return in, nil
}

// Simple example of an after pipeline. The 2nd argument is the handler response.
func (g *MetagameServer) simpleAfter(ctx context.Context, resp interface{}) (interface{}, error) {
	g.Logger.Info("Simple After exec - response:", resp)

	return resp, nil
}

func main() {
	svType := flag.String("type", "metagameDemo", "the server type")
	isFrontend := flag.Bool("frontend", true, "if server is frontend")
	flag.Parse()

	defer pitaya.Shutdown()

	metagameServer := NewMetagameMock()
	pitaya.SetSerializer(json.NewSerializer())
	pitaya.Register(metagameServer,
		component.WithName("metagameHandler"),
	)

	// Pipelines registration
	pitaya.BeforeHandler(handlerParamsValidator)
	pitaya.BeforeHandler(metagameServer.simpleBefore)
	pitaya.AfterHandler(metagameServer.simpleAfter)

	port := 3251
	tcp := acceptor.NewTCPAcceptor(fmt.Sprintf(":%d", port))
	pitaya.AddAcceptor(tcp)
	pitaya.Configure(*isFrontend, *svType, pitaya.Cluster, map[string]string{})
	pitaya.Start()
}
