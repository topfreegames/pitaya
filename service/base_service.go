package service

import "github.com/topfreegames/pitaya/pipeline"

type baseService struct {
	handlerHooks *pipeline.HandlerHooks
}

func (h *baseService) SetHandlerHooks(handlerHooks *pipeline.HandlerHooks) {
	h.handlerHooks = handlerHooks
}
