package logger

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log *logrus.Logger

// Init 初始化日志系统
func Init() {
	log = logrus.New()

	// 设置日志格式
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})

	// 设置日志级别
	if viper.GetBool("verbose") {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	// 设置日志输出
	log.SetOutput(os.Stdout)

	// 创建日志文件
	setupLogFile()
}

// setupLogFile 设置日志文件
func setupLogFile() {
	var logPath string

	if runtime.GOOS == "windows" {
		logPath = filepath.Join(os.Getenv("PROGRAMDATA"), "AutoCert", "logs", "autocert.log")
	} else {
		logPath = "/var/log/autocert.log"
	}

	// 创建日志目录
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		log.Warnf("无法创建日志目录: %v", err)
		return
	}

	// 创建或打开日志文件
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Warnf("无法打开日志文件: %v", err)
		return
	}

	// 设置多输出
	log.SetOutput(logFile)
}

// 封装常用的日志方法
func Debug(msg string, args ...interface{}) {
	if len(args) > 0 {
		log.WithFields(convertToFields(args...)).Debug(msg)
	} else {
		log.Debug(msg)
	}
}

func Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		log.WithFields(convertToFields(args...)).Info(msg)
	} else {
		log.Info(msg)
	}
}

func Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		log.WithFields(convertToFields(args...)).Warn(msg)
	} else {
		log.Warn(msg)
	}
}

func Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		log.WithFields(convertToFields(args...)).Error(msg)
	} else {
		log.Error(msg)
	}
}

func Fatal(msg string, args ...interface{}) {
	if len(args) > 0 {
		log.WithFields(convertToFields(args...)).Fatal(msg)
	} else {
		log.Fatal(msg)
	}
}

// convertToFields 将键值对转换为 logrus.Fields
func convertToFields(args ...interface{}) logrus.Fields {
	fields := make(logrus.Fields)
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			fields[key] = args[i+1]
		}
	}
	return fields
}
