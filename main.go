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

package main

import (
	"flag"
	"github.com/chaosblade-io/chaos-agent/chaosblade"
	"github.com/chaosblade-io/chaos-agent/config"
	"github.com/chaosblade-io/chaos-agent/controller"
	"github.com/chaosblade-io/chaos-agent/heartbeat"
	"github.com/chaosblade-io/chaos-agent/meta"
	"github.com/chaosblade-io/chaos-agent/service"
	"github.com/chaosblade-io/chaos-agent/tools"
	"github.com/chaosblade-io/chaos-agent/transport"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"net/http"
	"os"
	"strconv"
	"time"
)

var pidFile = "/var/run/chaosagent.pid"

func initConfig() (*config.Config, error) {
	cfg := &config.Config{}
	cfg.Controller = service.NewController(cfg)

	// common
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.StringVar(&cfg.AgentId, "agentId", uuid.New().String(), "agent id")
	flag.BoolVar(&cfg.Debugging, "debug", false, "debug mode")
	flag.StringVar(&cfg.Namespace, "namespace", "default", "namespace where the host is located")
	flag.StringVar(&cfg.StartupMode, "startup.mode", meta.StartConsoleMode, "startup mode")
	flag.StringVar(&cfg.AgentMode, "agent.mode", meta.AnsibleModel, "mode of the agent")
	flag.StringVar(&cfg.Port, "port", "19527", "the agent server port")

	// log
	flag.StringVar(&cfg.Level, "log.level", "info", "logging level: debug|info|warn|error|fatal|panic")
	flag.IntVar(&cfg.MaxFileSize, "log.size", 10, "log file size, unit: m")
	flag.IntVar(&cfg.MaxFileCount, "log.count", 1, "log file count, default value is 1")
	flag.StringVar(&cfg.LogOutput, "log.output", config.FileOutput, "log output, file|stdout")

	flag.StringVar(&cfg.TransportConfig.Endpoint, "transport.endpoint", "127.0.0.1:8080", "the server endpoint")
	flag.DurationVar(&cfg.TransportConfig.Timeout, "transport.timeout", 3*time.Second, "connect timeout with server")

	flag.DurationVar(&cfg.HeartbeatConfig.Period, "heartbeat.period", 5*time.Second, "the period of heartbeat")

	// application
	flag.StringVar(&cfg.ApplicationInstance, "appInstance", meta.DefaultApplicationInstance, "application instance name")
	flag.StringVar(&cfg.ApplicationGroup, "appGroup", meta.DefaultApplicationGroup, "application group name")

	flag.Parse()
	init0, err := cfg.Init0()
	if err != nil {
		return cfg, err
	}
	logrus.Infoln("Init config completed")
	return init0, nil
}

func main() {
	// init config service and start it
	mainConfig, err := initConfig()
	if err != nil {
		logrus.Errorf(err.Error())
		handlerErr(mainConfig, err)
	}

	bashChannel := tools.GetInstance()
	bashChannel.Start()

	// start transport service
	httpTransport, err := transport.New(&mainConfig.TransportConfig)
	if err != nil {
		handlerErr(mainConfig, err)
	}
	_, err = httpTransport.Start()
	handlerErr(mainConfig, err)

	defer tools.PrintPanicStack()

	// init LOG，must place the init after httpTransport.Start()，
	// because of logrus default output is stderr, if connect err, user can view the error message
	initLog(mainConfig)

	// start heartbeat service
	heartbeat.New(mainConfig.HeartbeatConfig, httpTransport).Start()

	// create chaosblade service
	chaosBlade := chaosblade.New(httpTransport)

	// register all service exclude transport and heartbeat
	ctl := controller.NewController(httpTransport)
	ctl.Register(controller.ChaosBlade, chaosBlade)

	ctl.Start()

	go func() {
		defer tools.PrintPanicStack()
		err := http.ListenAndServe(":"+mainConfig.Port, nil)
		if err != nil {
			logrus.Warningln("Start http server failed")
		}
	}()

	handlerSuccess(mainConfig)
	tools.Hold(ctl, httpTransport)
}

func handlerSuccess(cfg *config.Config) {
	pid := os.Getpid()
	err := writePid(pid)
	if err == nil || cfg.Debugging {
		return
	}
	if err != nil {
		logrus.Panic("write pid: ", pidFile, " failed. ", err)
	}
}

func handlerErr(cfg *config.Config, err error) {
	if err == nil {
		return
	}
	logrus.Warningf("start chaos failed because of %v", err)
	writePid(-1)
	logrus.Errorf("chaos agent will exit")
	os.Exit(1)
}

func writePid(pid int) error {
	file, err := os.OpenFile(pidFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(strconv.Itoa(pid))
	return err
}

func initLog(cfg *config.Config) {
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(level)
	}
	if cfg.Debugging {
		logrus.SetLevel(logrus.DebugLevel)
	}
	switch cfg.LogOutput {
	case config.FileOutput:
		fileName := tools.GetAgentLogFilePath()
		logrus.SetOutput(&lumberjack.Logger{
			Filename:   fileName,
			MaxSize:    cfg.MaxFileSize, // m
			MaxBackups: cfg.MaxFileCount,
			MaxAge:     2, // days
			Compress:   false,
		})
	case config.StdOutput:
		logrus.SetOutput(os.Stdout)
	}
}
