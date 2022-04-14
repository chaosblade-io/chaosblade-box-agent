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
package asyncreport

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaos-agent/transport"
)

type AsyncReportHandler struct {
	transportClient *transport.TransportClient
}

func NewClientCloseHandler(transportClient *transport.TransportClient) *AsyncReportHandler {
	return &AsyncReportHandler{
		transportClient: transportClient,
	}
}

// for chaos tools, async report chaos exec result
func (arh *AsyncReportHandler) ReportStatus(uid, status, errorMsg, toolType string, uri transport.Uri) {
	recordMsg := fmt.Sprintf("uid: %s, status: %s", uid, status)
	request := transport.NewRequest()
	request.AddParam("uid", uid).AddParam("status", status)
	if errorMsg != "" {
		request.AddParam("error", errorMsg)
	}
	if toolType != "" {
		request.AddParam("ToolType", toolType)
	}

	logrus.Infof("report install status: %v", request)
	response, err := arh.transportClient.Invoke(uri, request, true)
	if err != nil {
		logrus.Warningf("Report status err, %v, %s", err, recordMsg)
		return
	}
	if !response.Success {
		logrus.Warningf("Report status failed, %s, %s", response.Error, recordMsg)
		return
	}
	logrus.Infof("Report status success, %s", recordMsg)
}
