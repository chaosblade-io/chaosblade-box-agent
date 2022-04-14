package handler

import (
	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type UpdateApplicationHandler struct {
}

func NewUpdateApplicationHandler() *UpdateApplicationHandler {
	return &UpdateApplicationHandler{}
}

func (sh *UpdateApplicationHandler) Handle(request *transport.Request) *transport.Response {
	logrus.Info("Receive server update applocation request")

	appInstance := request.Params[tools.AppInstanceKeyName]
	appGroup := request.Params[tools.AppGroupKeyName]
	if appInstance != "" {
		options.Opts.ApplicationInstance = appInstance
	}
	if appGroup != "" {
		options.Opts.ApplicationGroup = appGroup
	}
	err := tools.RecordApplicationToFile(options.Opts.ApplicationInstance, options.Opts.ApplicationGroup, true)
	if err != nil {
		errMsg := "record application info to local file failed"
		logrus.WithField(tools.AppInstanceKeyName, options.Opts.ApplicationInstance).
			WithField(tools.AppGroupKeyName, options.Opts.ApplicationGroup).Warnln(errMsg)
		return transport.ReturnFail(transport.ServerError, errMsg)
	}

	return transport.ReturnSuccess()
}
