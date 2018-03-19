package pipeline

import "github.com/topfreegames/pitaya/session"

// Pipeline struct
var Pipeline = struct {
	Outbound, Inbound *pipelineChannel
}{
	&pipelineChannel{}, &pipelineChannel{},
}

type (
	pipelineHandler func(s *session.Session, in []byte) (out []byte, err error)

	pipelineChannel struct {
		Handlers []pipelineHandler
	}
)

// PushFront should not be used after pitaya running
func (p *pipelineChannel) PushFront(h pipelineHandler) {
	Handlers := make([]pipelineHandler, len(p.Handlers)+1)
	Handlers[0] = h
	copy(Handlers[1:], p.Handlers)
	p.Handlers = Handlers
}

// PushBack should not be used after pitaya running
func (p *pipelineChannel) PushBack(h pipelineHandler) {
	p.Handlers = append(p.Handlers, h)
}
