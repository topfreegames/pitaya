package logic

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/examples/demo/tadpole/logic/protocol"
	"github.com/topfreegames/pitaya/session"
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
func (w *World) Enter(s *session.Session, msg []byte) (*protocol.EnterWorldResponse, error) {
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
func (w *World) Update(s *session.Session, msg []byte) {
	err := w.Broadcast("update", msg)
	if err != nil {
		fmt.Println("error broadcasting message", err)
	}
}

// Message handler was used to communicate with each other
func (w *World) Message(s *session.Session, msg *protocol.WorldMessage) ([]byte, error) {
	msg.ID = s.ID()
	err := w.Broadcast("message", msg)
	if err != nil {
		return nil, err
	}
	return []byte("ok"), nil
}
