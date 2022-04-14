package heartbeat

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
)

type ClientHeartbeatHandler struct {
	heartbeatConfig options.HeartbeatConfig
	transportClient *transport.TransportClient
}

type HBSnapshot struct {
	Success bool
}

var HBSnapshotList, _ = tools.NewLimitedSortList(26)

func NewClientHeartbeatHandler(heartbeatConfig options.HeartbeatConfig, transportClient *transport.TransportClient) *ClientHeartbeatHandler {
	return &ClientHeartbeatHandler{
		heartbeatConfig: heartbeatConfig,
		transportClient: transportClient,
	}
}

func (chh *ClientHeartbeatHandler) Start() error {
	ticker := time.NewTicker(chh.heartbeatConfig.Period)
	go func() {
		defer tools.PanicPrintStack()
		for range ticker.C {
			request := transport.NewRequest()

			uri := transport.TransportUriMap[transport.API_HEARTBEAT]

			request.AddParam("ip", options.Opts.Ip)
			request.AddParam(options.AppInstanceKeyName, options.Opts.ApplicationInstance)
			request.AddParam(options.AppGroupKeyName, options.Opts.ApplicationGroup)
			chh.sendHeartbeat(uri, request)
		}
	}()
	log().Infoln("[heartbeat] start successfully")
	return nil
}

// todo 这里后面完善
func (cc *ClientHeartbeatHandler) Stop(stopCh chan bool) error {
	return nil
}

// sendHeartbeat
func (chh *ClientHeartbeatHandler) sendHeartbeat(uri transport.Uri, request *transport.Request) {
	response, err := chh.transportClient.Invoke(uri, request, true)
	if err != nil {
		log().Errorln("[heartbeat] send failed.", err)
		chh.record(false)
		return
	}
	if !response.Success {
		log().Errorf("[heartbeat] send failed. %+v", response)
		chh.record(false)
		return
	}
	log().Infoln("[heartbeat] success")
	chh.record(true)
}

// recode heartbeat result, for monitor heartbeat status
func (chh *ClientHeartbeatHandler) record(success bool) {
	HBSnapshotList.Put(HBSnapshot{
		Success: success,
	})
}

func log() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"cid":         options.Opts.Cid,
		"ver":         options.Opts.Version,
		"vpcId":       options.Opts.VpcId,
		"cbv":         options.Opts.ChaosbladeVersion,
		"appInstance": options.Opts.ApplicationInstance,
		"appGroup":    options.Opts.ApplicationGroup,
	})
}
