/*
 * Copyright 1999-2021 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controller

import (
	"errors"
	"fmt"
	"github.com/chaosblade-io/chaos-agent/transport"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	transport.InterceptorRequestHandler
	controller *Controller
}

//GetControllerHandler
func GetControllerHandler(controller *Controller) *Handler {
	handler := &Handler{controller: controller}
	requestHandler := transport.NewCommonHandler(handler)
	handler.InterceptorRequestHandler = requestHandler
	return handler
}

//Handle service switch
func (handler *Handler) Handle(request *transport.Request) *transport.Response {
	command := request.Params["cmd"]
	var err error
	if command == StartCmd {
		err = handler.Start(request.Params["name"])
	} else if command == StopCmd {
		err = handler.Stop(request.Params["name"])
	} else {
		err = errors.New(fmt.Sprintf("not find switch command: %s", command))
	}
	if err != nil {
		return transport.ReturnFail(transport.Code[transport.ServiceSwitchError], err.Error())
	}
	return transport.ReturnSuccess("success")
}

//Start
func (handler *Handler) Start(serviceName string) error {
	if serviceName == "" {
		logrus.Warningln("Start service err: service name is empty")
		return errors.New("service name is empty")
	}
	if serviceName == All {
		// 直接启动整个 controller，会默认启动下面管理的所有服务
		handler.controller.StartWithReason("start command that comes from server")
		return nil
	}
	service := handler.controller.services[serviceName]
	if service == nil {
		logrus.Warningln("Start service err: cannot find the service:", serviceName)
		return errors.New("can not find the service")
	}
	err := service.Start()
	if err != nil {
		handler.controller.StartWithReason(fmt.Sprintf("start %s service by server command executed failed. %s.", serviceName, err.Error()))
		return err
	}
	handler.controller.StartWithReason(fmt.Sprintf("start %s service by server command executed successfully.", serviceName))
	return nil
}

//Stop
func (handler *Handler) Stop(serviceName string) error {
	if serviceName == "" {
		logrus.Warningln("Stop service err: service name is empty")
		return errors.New("service name is empty")
	}
	if serviceName == All {
		handler.controller.StopWithReason("stop command that comes from server")
		return nil
	}
	service := handler.controller.services[serviceName]
	if service == nil {
		logrus.Warningln("Stop service err: cannot find the service:", serviceName)
		return errors.New("can not find the service")
	}
	err := service.Stop()
	if err != nil {
		handler.controller.StopWithReason(
			fmt.Sprintf("stop %s service by server command executed failed. %s.", serviceName, err.Error()))
		return err
	}
	handler.controller.StopWithReason(fmt.Sprintf("stop %s service by server command executed successfully.", serviceName))
	return nil
}
