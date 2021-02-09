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

package meta

import (
	"github.com/sirupsen/logrus"
	"net"
	"os"
	"strconv"

	"github.com/chaosblade-io/chaos-agent/tools"
)

const (
	ProgramName = "CHAOS_AGENT"
)

const (
	StartConsoleMode = "console"

	AnsibleModel = "ansible"
	SSHMode      = "ssh"
	K8sMode      = "k8s"
	K8sHelmMode  = "k8s_helm"
)

// application
const (
	DefaultApplicationInstance = "chaos-default-app"
	DefaultApplicationGroup    = "chaos-default-app-group"
)

type Meta struct {
	AgentId             string
	Ip                  string
	Port                string
	HostName            string
	Pid                 string
	Namespace           string
	InstanceId          string
	Uid                 string
	Debugging           bool
	Version             string
	Env                 string
	StartupMode         string
	AgentInstallMode    string
	ApplicationInstance string
	ApplicationGroup    string
}

var Info *Meta

func Init(namespace string, debugging bool,
	agentId, startupMode, agentMode, appInstance, appGroup, port string) error {
	Info = &Meta{
		AgentId:    agentId,
		Ip:         getPrivateIp(),
		HostName:   getHostName(),
		Pid:        getPid(),
		InstanceId: getHostName(),
		Uid:        "", // get from server
		Debugging:  debugging,
		Port:       port,
	}
	Info.Namespace = namespace
	Info.StartupMode = startupMode
	Info.AgentInstallMode = agentMode
	Info.Version = "0.0.1"

	Info.initApplicationInfo(appInstance, appGroup)
	return nil
}

func (info *Meta) initApplicationInfo(appInstance string, appGroup string) {
	if !IsHostMode() {
		return
	}
	if tools.IsExist(tools.AppFile) && appInstance == DefaultApplicationInstance && appGroup == DefaultApplicationGroup {
		// read from local file
		instance, group, err := tools.ReadAppInfoFromFile()
		if err != nil {
			logrus.WithError(err).Warningln("failed read application info from local file")
		}
		if instance != "" {
			appInstance = instance
		}
		if group != "" {
			appGroup = group
		}
	} else {
		// record to local file
		if err := tools.RecordApplicationToFile(appInstance, appGroup, true); err != nil {
			logrus.WithError(err).Warningln("record application info to local file failed")
		}
	}
	Info.ApplicationInstance = appInstance
	Info.ApplicationGroup = appGroup
}

func SetUid(uid string) {
	Info.Uid = uid
}

func getPid() string {
	return strconv.Itoa(os.Getpid())
}

func getHostName() string {
	name, err := os.Hostname()
	if err != nil {
		logrus.Warningln("Cannot get hostname", err)
		return ""
	}
	return name
}

func getPrivateIp() string {
	ifs, err := net.Interfaces()
	if err != nil {
		logrus.Fatalln("Cannot get host ip address! Please use --localIp flag to specify the host ip!!", err)
	}
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 {
			continue
		}
		if i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			logrus.Warningln(i, "get it's address error.", err)
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			return ip.String()
		}
	}
	logrus.Fatalln("Cannot get host ip address! Please use --localIp flag to specify the host ip")
	return ""
}

func IsK8sMode() bool {
	return Info.AgentInstallMode == K8sMode || Info.AgentInstallMode == K8sHelmMode
}

func IsHostMode() bool {
	return Info.AgentInstallMode == AnsibleModel || Info.AgentInstallMode == SSHMode
}
