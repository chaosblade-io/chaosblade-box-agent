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
