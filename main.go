package main

import (
	"autocert/cmd"
	"autocert/internal/logger"
	"os"
)

// 版本信息变量（通过 ldflags 设置）
var (
	version    = "dev"
	buildTime  = "unknown"
	commitHash = "unknown"
)

func main() {
	// 初始化日志
	logger.Init()

	// 设置版本信息
	cmd.SetVersionInfo(version, buildTime, commitHash)

	// 执行命令
	if err := cmd.Execute(); err != nil {
		logger.Error("程序执行失败", "error", err)
		os.Exit(1)
	}
}
