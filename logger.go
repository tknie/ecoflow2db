package ecoflow2db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	tlog "github.com/tknie/log"
	"github.com/tknie/services"
)

var logRus = logrus.StandardLogger()

var logger = &logBridge{}

type logBridge struct {
}

func (l *logBridge) Println(v ...interface{}) {
	s := ""
	for range v {
		s += "%v "
	}
	s += "\n"
	tlog.Log.Debugf(s, v...)
}

func (l *logBridge) Printf(format string, v ...interface{}) {
	tlog.Log.Debugf(format, v...)
}

func StartLog(fileName string) {
	services.ServerMessage("Init logging")
	level := os.Getenv("ENABLE_ECO2DB_DEBUG")
	logLevel := logrus.WarnLevel
	switch level {
	case "debug", "1":
		tlog.SetDebugLevel(true)
		logLevel = logrus.DebugLevel
	case "info", "2":
		tlog.SetDebugLevel(false)
		logLevel = logrus.InfoLevel
	default:
		tlog.SetDebugLevel(false)
		logLevel = logrus.ErrorLevel
	}
	logRus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05",
	})
	logRus.SetLevel(logLevel)
	p := os.Getenv("LOGPATH")
	if p == "" {
		p = "."
	}

	path := filepath.FromSlash(p + string(os.PathSeparator) + fileName)
	f, err := os.OpenFile(path,
		os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	logRus.SetOutput(f)
	logRus.Infof("Init logrus")
	tlog.Log = logRus
	services.ServerMessage("Logging initiated with %v...", logRus.Level)
}
