package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	// 基础配置
	LogLevel  string `mapstructure:"log_level"`
	ConfigDir string `mapstructure:"config_dir"`
	CertDir   string `mapstructure:"cert_dir"`
	LogDir    string `mapstructure:"log_dir"`

	// ACME 配置
	ACME ACMEConfig `mapstructure:"acme"`

	// 通知配置
	Notification NotificationConfig `mapstructure:"notification"`

	// Web 服务器配置
	WebServer WebServerConfig `mapstructure:"webserver"`
}

// ACMEConfig ACME 相关配置
type ACMEConfig struct {
	Server  string `mapstructure:"server"`   // Let's Encrypt 服务器
	Email   string `mapstructure:"email"`    // 邮箱地址
	KeyType string `mapstructure:"key_type"` // 密钥类型
	KeySize int    `mapstructure:"key_size"` // 密钥大小
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	Email   EmailConfig `mapstructure:"email"`
	Webhook string      `mapstructure:"webhook"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	SMTP     string `mapstructure:"smtp"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	To       string `mapstructure:"to"`
}

// WebServerConfig Web 服务器配置
type WebServerConfig struct {
	Type       string `mapstructure:"type"`        // nginx, apache, iis
	ConfigPath string `mapstructure:"config_path"` // 配置文件路径
	ReloadCmd  string `mapstructure:"reload_cmd"`  // 重载命令
}

var (
	// AppConfig 全局配置实例
	AppConfig *Config
)

// Load 加载配置
func Load() {
	AppConfig = &Config{}

	// 设置默认值
	setDefaults()

	// 从配置文件和环境变量加载
	if err := viper.Unmarshal(AppConfig); err != nil {
		// 如果解析失败，使用默认配置
		AppConfig = getDefaultConfig()
	}
}

// setDefaults 设置默认配置值
func setDefaults() {
	// 根据操作系统设置默认路径
	if runtime.GOOS == "windows" {
		viper.SetDefault("config_dir", filepath.Join(os.Getenv("PROGRAMDATA"), "AutoCert"))
		viper.SetDefault("cert_dir", filepath.Join(os.Getenv("PROGRAMDATA"), "AutoCert", "certs"))
		viper.SetDefault("log_dir", filepath.Join(os.Getenv("PROGRAMDATA"), "AutoCert", "logs"))
		viper.SetDefault("webserver.type", "iis")
	} else {
		viper.SetDefault("config_dir", "/etc/autocert")
		viper.SetDefault("cert_dir", "/etc/autocert/certs")
		viper.SetDefault("log_dir", "/var/log")
		viper.SetDefault("webserver.type", "nginx")
	}

	// 其他默认值
	viper.SetDefault("log_level", "info")
	viper.SetDefault("acme.server", "https://acme-v02.api.letsencrypt.org/directory")
	viper.SetDefault("acme.key_type", "rsa")
	viper.SetDefault("acme.key_size", 2048)
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() *Config {
	config := &Config{
		LogLevel: "info",
		ACME: ACMEConfig{
			Server:  "https://acme-v02.api.letsencrypt.org/directory",
			KeyType: "rsa",
			KeySize: 2048,
		},
	}

	if runtime.GOOS == "windows" {
		config.ConfigDir = filepath.Join(os.Getenv("PROGRAMDATA"), "AutoCert")
		config.CertDir = filepath.Join(os.Getenv("PROGRAMDATA"), "AutoCert", "certs")
		config.LogDir = filepath.Join(os.Getenv("PROGRAMDATA"), "AutoCert", "logs")
		config.WebServer.Type = "iis"
	} else {
		config.ConfigDir = "/etc/autocert"
		config.CertDir = "/etc/autocert/certs"
		config.LogDir = "/var/log"
		config.WebServer.Type = "nginx"
	}

	return config
}

// GetConfigDir 获取配置目录
func GetConfigDir() string {
	if AppConfig != nil {
		return AppConfig.ConfigDir
	}
	return getDefaultConfig().ConfigDir
}

// GetCertDir 获取证书目录
func GetCertDir() string {
	if AppConfig != nil {
		return AppConfig.CertDir
	}
	return getDefaultConfig().CertDir
}
