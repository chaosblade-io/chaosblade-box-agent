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
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/service"
	"github.com/chaosblade-io/chaos-agent/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

// serviceName
const (
	All        = "_all"
	StartCmd   = "start"
	StopCmd    = "stop"
	ChaosBlade = "chaosblade"
)

//Controller controls service start or stop except heartbeat or transport service
type Controller struct {
	services   map[string]service.LifeCycle
	serviceKey []string
	transport  *transport.Transport
	handler    *transport.InterceptorRequestHandler
	mutex      sync.Mutex
	*service.Controller
	shutdownFuncList []func() error
}

func (controller *Controller) Shutdown() error {
	for _, shutdownFunc := range controller.shutdownFuncList {
		if err := shutdownFunc(); err != nil {
			logrus.Warnln(err.Error())
		}
	}
	return controller.Stop()
}

//NewController
func NewController(transport0 *transport.Transport) *Controller {
	control := &Controller{
		services:         make(map[string]service.LifeCycle, 0),
		serviceKey:       make([]string, 0),
		transport:        transport0,
		shutdownFuncList: make([]func() error, 0),
	}
	control.Controller = service.NewController(control)
	control.handler = &GetControllerHandler(control).InterceptorRequestHandler
	return control
}

//Register service for control
func (controller *Controller) Register(serviceName string, service service.LifeCycle) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	if controller.services[serviceName] == nil {
		controller.serviceKey = append(controller.serviceKey, serviceName)
		controller.services[serviceName] = service
		logrus.Infof("[Controller] register %s service to controller", serviceName)
	}
}

func (controller *Controller) RegisterWithShutdownFunc(serviceName string, service service.LifeCycle, shutDownFunc func() error) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	if controller.services[serviceName] == nil {
		controller.serviceKey = append(controller.serviceKey, serviceName)
		controller.services[serviceName] = service
		controller.shutdownFuncList = append(controller.shutdownFuncList, shutDownFunc)
		logrus.Infof("[Controller] register %s service to controller with shutdown function", serviceName)
	}
}

func (controller *Controller) StopWithReason(reason string) error {
	logrus.Warningf("[Controller] send stop event to server, reason: %s", reason)
	return controller.Stop()
}

func (controller *Controller) StartWithReason(reason string) error {
	logrus.Infof("[Controller] send start event to server, reason: %s", reason)
	return controller.Start()
}

func (controller *Controller) DoStart() error {
	// start all register service
	go func() {
		defer tools.PrintPanicStack()
		controller.mutex.Lock()
		defer controller.mutex.Unlock()
		for _, key := range controller.serviceKey {
			lifeCycle := controller.services[key]
			if lifeCycle != nil {
				err := lifeCycle.Start()
				if err != nil {
					logrus.Warningf("[Controller] start %s service failed, err: %s", key, err.Error())
					continue
				}
				logrus.Infof("[Controller] start %s service successfully.", key)
			}
		}
	}()
	return nil
}

func (controller *Controller) DoStop() error {
	// stop all register service
	go func() {
		defer tools.PrintPanicStack()
		controller.mutex.Lock()
		defer controller.mutex.Unlock()
		length := len(controller.serviceKey)
		for i := length - 1; i >= 0; i-- {
			key := controller.serviceKey[i]
			lifeCycle := controller.services[key]
			if lifeCycle != nil {
				err := lifeCycle.Stop()
				if err != nil {
					logrus.Warningf("[Controller] stop %s service failed, err: %s", key, err.Error())
					continue
				}
				logrus.Infof("[Controller] stop %s service successfully.", key)
			}
		}
	}()
	return nil
}
