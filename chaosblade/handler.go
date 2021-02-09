package chaosblade

import (
	"github.com/chaosblade-io/chaos-agent/transport"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	transport.InterceptorRequestHandler
	blade *ChaosBlade
}

//GetFaultInjectHandler
func GetChaosBladeHandler(blade *ChaosBlade) *Handler {
	handler := &Handler{
		blade: blade,
	}
	requestHandler := transport.NewCommonHandler(handler)
	handler.InterceptorRequestHandler = requestHandler
	return handler
}

//Handle
func (handler *Handler) Handle(request *transport.Request) *transport.Response {
	logrus.Infof("chaosblade: %+v", request)
	if handler.blade.IsStopped() {
		return transport.ReturnFail(transport.Code[transport.ServerError], "chaosblade service stopped")
	}
	cmd := request.Params["cmd"]
	if cmd == "" {
		return transport.ReturnFail(transport.Code[transport.ParameterEmpty], "cmd parameter is empty")
	}
	return handler.blade.exec(cmd)
}
