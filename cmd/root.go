package cmd

import (
	"autocert/internal/config"
	"autocert/internal/logger"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "autocert",
		Short: "Let's Encrypt HTTPS 证书一键安装部署工具",
		Long: `AutoCert 是一个跨平台的 Let's Encrypt HTTPS 证书管理工具，
支持一键安装、自动更新、跨机器迁移等功能。

支持的 Web 服务器：
- Linux: Nginx, Apache
- Windows: IIS, Nginx for Windows`,
	}
)

// Execute 执行根命令
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// 全局标志
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径 (默认搜索路径: $HOME/.autocert.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "详细输出")

	// 绑定标志到 viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig 初始化配置
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// 查找配置文件
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".autocert")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		logger.Info("使用配置文件", "config", viper.ConfigFileUsed())
	}

	// 应用配置
	config.Load()
}
