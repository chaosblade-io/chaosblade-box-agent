package api

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/transport"
	"github.com/chaosblade-io/chaos-agent/web"
)

type ServerRequestHandler struct {
	Interceptor transport.RequestInterceptor
	Handler     web.ApiHandler
	Ctx         context.Context
}

func NewServerRequestHandler(handler web.ApiHandler) *ServerRequestHandler {
	if handler == nil {
		return nil
	}

	return &ServerRequestHandler{
		Interceptor: transport.BuildInterceptor(),
		Handler:     handler,
		Ctx:         context.Background(),
	}
}

// handle(request string) (string, error)
func (handler *ServerRequestHandler) Handle(request string) (string, error) {
	logrus.Debugf("Handle: %+v", request)
	var response *transport.Response
	select {
	case <-handler.Ctx.Done():
		response = transport.ReturnFail(transport.HandlerClosed)
	default:
		// decode
		req := &transport.Request{}
		err := json.Unmarshal([]byte(request), req)
		if err != nil {
			return "", err
		}

		response = handler.Handler.Handle(req)
	}
	// encode
	bytes, err := json.Marshal(response)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
