/*
 * Copyright 2025 The ChaosBlade Authors
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

package options

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/chaosblade-io/chaos-agent/pkg/tools"
)

const (
	LogFileOutput = "file"
	LogStdOutput  = "stdout"
	// Version       = "1.13.2"

	ProgramName      = "CHAOS_AGENT"
	BladeProgramName = "CHAOS_BLADE"

	InstallOperatorLinux   = "linux"
	InstallOperatorWindows = "windows"

	AgentHostMode      = "host"
	AgentK8sMode       = "k8s"
	AgentCSK8sMode     = "cs_k8s"
	AgentCSSwarmMode   = "cs_swarm"
	AgentK8sHelmMode   = "k8s_helm"
	AgentCSK8sHelmMode = "cs_k8s_helm"
)

const (
	Host = iota
	Container
)

// application
const (
	AppInstanceKeyName = "appInstance"
	AppGroupKeyName    = "appGroup"

	DefaultApplicationInstance = "chaos-default-app"
	DefaultApplicationGroup    = "chaos-default-app-group"

	StartManualMode  = "manual"
	StartConsoleMode = "console"
	StartCrontabMode = "crontab"
	StartUpgradeMode = "upgrade"
)

var Opts *Options

type Options struct {
	LogConfig LogConfig
	Help      bool

	Environment     string
	IsVpc           bool
	VpcId           string
	Pid             string
	Cid             string
	Uid             string
	Ip              string
	HostName        string
	InstanceId      string
	Namespace       string
	License         string
	AgentMode       string
	InstallOperator string

	// server port
	Port string

	// heartbeat config
	HeartbeatConfig HeartbeatConfig

	//// metric report config
	//MetricReportConfig MetricReportConfig

	// transport config
	TransportConfig TransportConfig

	// application
	ApplicationInstance string
	ApplicationGroup    string
	StartupMode         string
	RestrictedVpc       bool

	// version
	Version            string
	ChaosbladeVersion  string
	LitmusChaosVerison string

	// k8s cluster
	ClusterId   string
	ClusterName string
	// k8s metric report flag
	PodMetricFlag    bool
	ExternalIpEnable bool

	// download url
	ChaosAgentBinUrl string
	ChaosAgentSHUrl  string
	ChaosBladeTarUrl string
	LitmusChartUrl   string
	CertUrl          string

	Flags *pflag.FlagSet
}

type HeartbeatConfig struct {
	Period time.Duration
}

type LogConfig struct {
	// debug|info|warn|error|fatal|panic
	Level string
	// maximum log file size
	MaxFileSize int
	// maximum log file count
	MaxFileCount int
	// file/stdout
	LogOutput string
}

type TransportConfig struct {
	Environment string
	// Endpoint is server address with port
	Endpoint string
	// Timeout is the maximum amount of time a client will wait for a connect to complete
	Timeout time.Duration
	// Secure is setting the socket encrypted or not
	Secure bool
}

func NewOptions() {
	Opts = &Options{}

	Opts.AddFlags()

	err := Opts.Parse()
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}

	if Opts.Help {
		Opts.Usage()
		os.Exit(0)
	}
}

func (o *Options) AddFlags() {
	o.Flags = pflag.NewFlagSet("", pflag.ExitOnError)
	o.Flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		o.Flags.PrintDefaults()
	}

	o.Flags.StringVar(&o.LogConfig.Level, "log.level", "info", "logging level: debug|info|warn|error|fatal|panic")
	o.Flags.IntVar(&o.LogConfig.MaxFileSize, "log.size", 10, "log file size, unit: m")
	o.Flags.IntVar(&o.LogConfig.MaxFileCount, "log.count", 1, "log file count, default value is 1")
	o.Flags.StringVar(&o.LogConfig.LogOutput, "log.output", LogFileOutput, "log output, file|stdout")

	o.Flags.StringVar(&o.Environment, "environment", "prod", "environment: prod|pre|test|dev")
	o.Flags.StringVar(&o.Namespace, "namespace", "default", "namespace where the host is located")
	o.Flags.StringVar(&o.License, "license", "", "license")
	o.Flags.StringVar(&o.AgentMode, "agent.mode", AgentHostMode, "mode of the agent")
	o.Flags.StringVar(&o.InstallOperator, "install.operator", InstallOperatorLinux, "operator of the agent")

	o.Flags.DurationVar(&o.HeartbeatConfig.Period, "heartbeat.period", 5*time.Second, "the period of heartbeat")

	o.Flags.StringVar(&o.TransportConfig.Endpoint, "transport.endpoint", "", "the server endpoint, ip:port")
	o.Flags.DurationVar(&o.TransportConfig.Timeout, "transport.timeout", 3*time.Second, "connect timeout with server")
	o.Flags.BoolVar(&o.TransportConfig.Secure, "transport.secure", true, "transport in secure or not, default value is true")

	o.Flags.StringVar(&o.ApplicationInstance, AppInstanceKeyName, DefaultApplicationInstance, "application instance name")
	o.Flags.StringVar(&o.ApplicationGroup, AppGroupKeyName, DefaultApplicationGroup, "application group name")
	o.Flags.StringVar(&o.StartupMode, "startup.mode", StartConsoleMode, "startup mode")
	o.Flags.BoolVar(&o.RestrictedVpc, "restrict", false, "use license as user, default value is false")

	o.Flags.StringVar(&o.ClusterId, "kubernetes.cluster.id", "", "the cluster id")
	o.Flags.StringVar(&o.ClusterName, "kubernetes.cluster.name", "", "the cluster name")
	o.Flags.BoolVar(&o.PodMetricFlag, "kubernetes.pod.report", false, "the flag of pod metric")
	o.Flags.BoolVar(&o.ExternalIpEnable, "kubernetes.externalIp.enable", false, "the flag of use external ip or not")

	o.Flags.StringVar(&o.ChaosAgentBinUrl, "agent.bin.url", "", "the download url of chaos-agent binary")
	o.Flags.StringVar(&o.ChaosAgentSHUrl, "agent.sh.url", "", "the download url of chaos-agent start shell")
	o.Flags.StringVar(&o.ChaosBladeTarUrl, "blade.tar.url", "", "the download url of chaosblade tar package")
	o.Flags.StringVar(&o.LitmusChartUrl, "litmus.chart.url", "", "the chart repositories of litmusChaos")
	o.Flags.StringVar(&o.CertUrl, "cert.url", "", "the download url of cert")

	o.Flags.StringVar(&o.Port, "port", "19527", "the agent server port")

	o.Flags.BoolVarP(&o.Help, "help", "h", false, "Print Help text")
}

func (o *Options) SetOthersByFlags() {
	o.TransportConfig.Environment = o.Environment

	o.Pid = o.GetPid()
	o.IsVpc = false
	o.VpcId = o.License
	o.Uid = ""
	o.Ip = o.GetPrivateIp()
	o.HostName = o.GetHostName()
	o.InstanceId = o.GetHostName()
	o.Version = "1.1.0"
	o.InitApplicationInfo(o.ApplicationInstance, o.ApplicationGroup)

	var err error
	if o.ChaosbladeVersion, err = GetChaosBladeVersion(); err != nil {
		logrus.Errorf("Get chaosblade version failed, err: %s", err.Error())
		os.Exit(1)
	}
}

func (o *Options) InitApplicationInfo(appInstance string, appGroup string) {
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

		return
	}

	if err := tools.RecordApplicationToFile(appInstance, appGroup, true); err != nil {
		logrus.WithError(err).Warningln("record application info to local file failed")
	}
}

func (o *Options) SetUid(uid string) {
	o.Uid = uid
}

func (o *Options) SetCid(cid string) {
	o.Cid = cid
}

func (o *Options) Parse() error {
	return o.Flags.Parse(os.Args)
}

func (o *Options) Usage() {
	o.Flags.Usage()
}

func (o *Options) SetClusterIdIfNotPresent(clusterId string) {
	if o.ClusterId != "" {
		logrus.Infof("[options] cluster id is %s, not empty, so skip to set new value", o.ClusterId)
		return
	}

	o.ClusterId = clusterId
}

// SetChaosBladeVersion
func (o *Options) SetChaosBladeVersion(chaosBladeVersion string) {
	logrus.Infof("[options] Set chaosblade version from %s to %s", o.ChaosbladeVersion, chaosBladeVersion)
	o.ChaosbladeVersion = chaosBladeVersion
}

func (o *Options) IsK8sMode() bool {
	return o.AgentMode == AgentK8sMode || o.AgentMode == AgentCSK8sMode ||
		o.AgentMode == AgentK8sHelmMode || o.AgentMode == AgentCSK8sHelmMode
}

func (o *Options) IsHostMode() bool {
	return o.AgentMode == AgentHostMode
}

func (o *Options) GetPid() string {
	return strconv.Itoa(os.Getpid())
}

func (o *Options) GetPrivateIp() string {
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

func (o *Options) GetHostName() string {
	name, err := os.Hostname()
	if err != nil {
		logrus.Warningln("Cannot get hostname", err)
		return ""
	}
	return name
}
