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

package log

import (
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/chaosblade-io/chaos-agent/pkg/options"
	"github.com/chaosblade-io/chaos-agent/pkg/tools"
)

const DebugLevel = "debug"

func InitLog(cfg *options.LogConfig) {
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(level)
	}

	switch cfg.LogOutput {
	case options.LogFileOutput:
		fileName := tools.GetAgentLogFilePath()
		logrus.SetOutput(&lumberjack.Logger{
			Filename:   fileName,
			MaxSize:    cfg.MaxFileSize, // m
			MaxBackups: cfg.MaxFileCount,
			MaxAge:     2, // days
			Compress:   false,
		})
	case options.LogStdOutput:
		logrus.SetOutput(os.Stdout)
	}
}
