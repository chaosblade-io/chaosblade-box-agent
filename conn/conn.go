/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
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

package conn

import (
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

type ClientHandle interface {
	Start() error
	Stop(stopCh chan bool) error
}
type Conn struct {
	clientHandlers map[string]ClientHandle
	locker         sync.Mutex
}

func NewConn() *Conn {
	return &Conn{
		clientHandlers: make(map[string]ClientHandle),
	}
}

func (c *Conn) Register(clientHandlerName string, clientHandler ClientHandle) {
	c.locker.Lock()
	defer c.locker.Unlock()
	c.clientHandlers[clientHandlerName] = clientHandler
}

func (c *Conn) Start() {
	if len(c.clientHandlers) <= 0 {
		return
	}

	var errCh chan error
	for clientHandlerName, clientHandler := range c.clientHandlers {
		go func(clientHandlerName string, clientHandler ClientHandle) {
			logrus.WithField("clientHandlerName", clientHandlerName).Infof("conn start")
			if err := clientHandler.Start(); err != nil {
				logrus.WithField("clientHandlerName", clientHandlerName).Warnf("conn start failed, err: %s", err.Error())
				errCh <- err
			}
		}(clientHandlerName, clientHandler)
	}

	go func() {
		for {
			select {
			case err := <-errCh:
				if err != nil {
					logrus.Errorf("register conn failed, err: %s", err.Error())
					os.Exit(1)
				}
			}
		}
	}()

}
