package main

import (
	"github.com/sirupsen/logrus"
	"os"
)

var logger *logrus.Logger

func InitLogger() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn", "warning":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logger.SetLevel(logrus.FatalLevel)
	case "panic":
		logger.SetLevel(logrus.PanicLevel)
	default:
		// 默认使用 InfoLevel
		logger.SetLevel(logrus.InfoLevel)
	}
	logOutput := os.Getenv("LOG_OUTPUT")
	switch logOutput {
	case "file":
		// 设置日志输出到文件
		logFile, err := os.OpenFile("/var/log/minimcd.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			// 如果文件打开失败，回退到标准输出
			logger.SetOutput(os.Stdout)
			logger.Warn("Failed to log to file, using default stdout")
		} else {
			logger.SetOutput(logFile)
		}
	case "stdout", "":
		// 默认输出到控制台
		logger.SetOutput(os.Stdout)
	default:
		// 默认输出到控制台
		logger.SetOutput(os.Stdout)
	}
}
func GetLogger() *logrus.Logger {
	return logger
}
