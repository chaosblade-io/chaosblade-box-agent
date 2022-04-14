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
