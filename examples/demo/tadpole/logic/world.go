package logic

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/tadpole/logic/protocol"
)

// World contains all tadpoles
type World struct {
	component.Base
	*pitaya.Group
}

// NewWorld returns a world instance
func NewWorld() *World {
	return &World{
		Group: pitaya.NewGroup(uuid.New().String()),
	}
}

// Init initialize world component
func (w *World) Init() {

}

// Enter was called when new guest enter
func (w *World) Enter(ctx context.Context, msg []byte) (*protocol.EnterWorldResponse, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	s.OnClose(func() {
		w.Leave(s)
		w.Broadcast("leave", &protocol.LeaveWorldResponse{ID: s.ID()})
		log.Println(fmt.Sprintf("session count: %d", w.Count()))
	})
	w.Add(s)
	log.Println(fmt.Sprintf("session count: %d", w.Count()))
	resp := &protocol.EnterWorldResponse{ID: s.ID()}
	return resp, nil
}

// Update refresh tadpole's position
func (w *World) Update(ctx context.Context, msg []byte) {
	err := w.Broadcast("update", msg)
	if err != nil {
		fmt.Println("error broadcasting message", err)
	}
}

// Message handler was used to communicate with each other
func (w *World) Message(ctx context.Context, msg *protocol.WorldMessage) ([]byte, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	msg.ID = s.ID()
	err := w.Broadcast("message", msg)
	if err != nil {
		return nil, err
	}
	return []byte("ok"), nil
}
