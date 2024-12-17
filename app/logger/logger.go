package logger

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger = *log.Logger

var (
	loggerInstances sync.Map
)

func GetLogger(filename string) *log.Logger {
	instance, ok := loggerInstances.Load(filename)
	if ok {
		return instance.(*log.Logger)
	}
	newInstance := log.New()
	rotateLogger := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    200,
		MaxBackups: 1,
		MaxAge:     28,
		Compress:   true,
	}
	newInstance.SetOutput(rotateLogger)
	loggerInstances.Store(filename, newInstance)
	newInstance.Infof("Logger created for file: %s", filename)
	return newInstance
}
