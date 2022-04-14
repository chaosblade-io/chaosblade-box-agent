package closer

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/transport"
)

type ClientCloserHandler struct {
	transportClient *transport.TransportClient
}

func NewClientCloseHandler(transportClient *transport.TransportClient) *ClientCloserHandler {
	return &ClientCloserHandler{
		transportClient: transportClient,
	}
}

func (close *ClientCloserHandler) Shutdown() {
	logrus.Infoln("Agent closing")
	go func() {
		logrus.Infof("Invoking chaos-chaos service to close")
		// invoke monkeyking
		request := transport.NewRequest()
		uri := transport.TransportUriMap[transport.API_CLOSE]
		response, err := close.transportClient.Invoke(uri, request, true)
		if err != nil {
			logrus.Warningf("Invoking %s service err: %v", uri.ServerName, err)
			return
		}
		if !response.Success {
			logrus.Warningf("Invoking chaos-chaos service failed, %s", response.Error)
		}
	}()
	time.Sleep(2 * time.Second)
	logrus.Infoln("Agent closed")
}
