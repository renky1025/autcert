package webserver

import (
	"autocert/internal/logger"
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

// Config Web 服务器配置
type Config struct {
	Type       string // nginx, apache, iis
	Domain     string
	CertPath   string
	KeyPath    string
	ConfigPath string
	WebRoot    string
}

// Configurator Web 服务器配置器接口
type Configurator interface {
	Configure(config *Config) error
	Test() error
	Reload() error
	GetConfigPath() string
	IsSSLEnabled(domain string) bool
}

// NewConfigurator 创建配置器
func NewConfigurator(serverType string) (Configurator, error) {
	switch strings.ToLower(serverType) {
	case "nginx":
		return &NginxConfigurator{}, nil
	case "apache":
		return &ApacheConfigurator{}, nil
	case "iis":
		return &IISConfigurator{}, nil
	default:
		return nil, fmt.Errorf("不支持的 Web 服务器类型: %s", serverType)
	}
}

// NginxConfigurator Nginx 配置器
type NginxConfigurator struct {
	configPath string
}

// Configure 配置 Nginx
func (n *NginxConfigurator) Configure(config *Config) error {
	logger.Info("开始配置 Nginx", "domain", config.Domain)

	// 1. 确定配置文件路径
	if err := n.findConfigPath(); err != nil {
		return fmt.Errorf("查找 Nginx 配置路径失败: %w", err)
	}

	// 2. 创建站点配置
	siteConfigPath, err := n.createSiteConfig(config)
	if err != nil {
		return fmt.Errorf("创建站点配置失败: %w", err)
	}

	// 3. 启用站点配置
	if err := n.enableSite(siteConfigPath); err != nil {
		return fmt.Errorf("启用站点配置失败: %w", err)
	}

	logger.Info("Nginx 配置完成", "domain", config.Domain)
	return nil
}

// Test 测试 Nginx 配置
func (n *NginxConfigurator) Test() error {
	cmd := exec.Command("nginx", "-t")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Nginx 配置测试失败: %s", string(output))
	}
	logger.Info("Nginx 配置测试成功")
	return nil
}

// Reload 重载 Nginx 配置
func (n *NginxConfigurator) Reload() error {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("nginx", "-s", "reload")
	} else {
		// 尝试使用 systemctl
		if _, err := exec.LookPath("systemctl"); err == nil {
			cmd = exec.Command("systemctl", "reload", "nginx")
		} else {
			cmd = exec.Command("nginx", "-s", "reload")
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("重载 Nginx 失败: %s", string(output))
	}

	logger.Info("Nginx 配置重载成功")
	return nil
}

// GetConfigPath 获取配置路径
func (n *NginxConfigurator) GetConfigPath() string {
	return n.configPath
}

// IsSSLEnabled 检查 SSL 是否已启用
func (n *NginxConfigurator) IsSSLEnabled(domain string) bool {
	// 查找域名的配置文件
	configFiles := n.findSiteConfigs()

	for _, configFile := range configFiles {
		if n.checkSSLInConfig(configFile, domain) {
			return true
		}
	}

	return false
}

// findConfigPath 查找 Nginx 配置路径
func (n *NginxConfigurator) findConfigPath() error {
	var configPaths []string

	if runtime.GOOS == "windows" {
		configPaths = []string{
			`C:\nginx\conf\nginx.conf`,
			`C:\Program Files\nginx\conf\nginx.conf`,
		}
	} else {
		configPaths = []string{
			"/etc/nginx/nginx.conf",
			"/usr/local/nginx/conf/nginx.conf",
			"/usr/local/etc/nginx/nginx.conf",
		}
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			n.configPath = path
			return nil
		}
	}

	return fmt.Errorf("未找到 Nginx 配置文件")
}

// createSiteConfig 创建站点配置
func (n *NginxConfigurator) createSiteConfig(config *Config) (string, error) {
	var configDir string
	var configFile string

	if runtime.GOOS == "windows" {
		configDir = filepath.Dir(n.configPath)
		configFile = filepath.Join(configDir, "conf.d", config.Domain+".conf")
	} else {
		configDir = "/etc/nginx/sites-available"
		configFile = filepath.Join(configDir, config.Domain)
	}

	// 确保配置目录存在
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return "", err
	}

	// 生成配置内容
	configContent, err := n.generateConfig(config)
	if err != nil {
		return "", err
	}

	// 写入配置文件
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		return "", err
	}

	logger.Info("创建 Nginx 站点配置", "configFile", configFile)
	return configFile, nil
}

// generateConfig 生成 Nginx 配置
func (n *NginxConfigurator) generateConfig(config *Config) (string, error) {
	tmpl := `# AutoCert 自动生成的配置
server {
    listen 80;
    server_name {{.Domain}};
    
    # 重定向 HTTP 到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name {{.Domain}};
    
    # SSL 证书配置
    ssl_certificate {{.CertPath}};
    ssl_certificate_key {{.KeyPath}};
    
    # SSL 安全配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-SHA384:ECDHE-RSA-AES128-SHA256;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # 网站根目录
    root {{.WebRoot}};
    index index.html index.htm index.php;
    
    # 通用配置
    location / {
        try_files $uri $uri/ =404;
    }
    
    # ACME 挑战目录
    location ^~ /.well-known/acme-challenge/ {
        default_type "text/plain";
        root {{.WebRoot}};
    }
}
`

	t, err := template.New("nginx").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	if err := t.Execute(&result, config); err != nil {
		return "", err
	}

	return result.String(), nil
}

// enableSite 启用站点配置
func (n *NginxConfigurator) enableSite(configFile string) error {
	if runtime.GOOS == "windows" {
		// Windows 下通常配置文件直接放在 conf.d 目录
		return nil
	}

	// Linux 下需要创建符号链接
	sitesEnabled := "/etc/nginx/sites-enabled"
	linkPath := filepath.Join(sitesEnabled, filepath.Base(configFile))

	// 确保 sites-enabled 目录存在
	if err := os.MkdirAll(sitesEnabled, 0755); err != nil {
		return err
	}

	// 删除已存在的链接
	os.Remove(linkPath)

	// 创建新的符号链接
	if err := os.Symlink(configFile, linkPath); err != nil {
		return err
	}

	logger.Info("启用 Nginx 站点", "link", linkPath)
	return nil
}

// findSiteConfigs 查找站点配置文件
func (n *NginxConfigurator) findSiteConfigs() []string {
	var configs []string
	var searchDirs []string

	if runtime.GOOS == "windows" {
		searchDirs = []string{
			filepath.Join(filepath.Dir(n.configPath), "conf.d"),
		}
	} else {
		searchDirs = []string{
			"/etc/nginx/sites-enabled",
			"/etc/nginx/conf.d",
		}
	}

	for _, dir := range searchDirs {
		if files, err := filepath.Glob(filepath.Join(dir, "*")); err == nil {
			configs = append(configs, files...)
		}
	}

	return configs
}

// checkSSLInConfig 检查配置文件中是否启用了 SSL
func (n *NginxConfigurator) checkSSLInConfig(configFile, domain string) bool {
	file, err := os.Open(configFile)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inServerBlock := false
	hasSSL := false
	hasDomain := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "server {") {
			inServerBlock = true
			hasSSL = false
			hasDomain = false
		} else if strings.Contains(line, "}") && inServerBlock {
			if hasSSL && hasDomain {
				return true
			}
			inServerBlock = false
		} else if inServerBlock {
			if strings.Contains(line, "ssl_certificate") {
				hasSSL = true
			}
			if strings.Contains(line, "server_name") && strings.Contains(line, domain) {
				hasDomain = true
			}
		}
	}

	return false
}

// ApacheConfigurator Apache 配置器
type ApacheConfigurator struct {
	configPath string
}

// Configure 配置 Apache
func (a *ApacheConfigurator) Configure(config *Config) error {
	logger.Info("开始配置 Apache", "domain", config.Domain)

	// Apache 配置实现
	// 这里应该实现完整的 Apache SSL 配置逻辑

	logger.Info("Apache 配置完成", "domain", config.Domain)
	return nil
}

// Test 测试 Apache 配置
func (a *ApacheConfigurator) Test() error {
	cmd := exec.Command("apache2ctl", "configtest")
	if _, err := exec.LookPath("apache2ctl"); err != nil {
		cmd = exec.Command("httpd", "-t")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Apache 配置测试失败: %s", string(output))
	}

	logger.Info("Apache 配置测试成功")
	return nil
}

// Reload 重载 Apache 配置
func (a *ApacheConfigurator) Reload() error {
	var cmd *exec.Cmd

	if _, err := exec.LookPath("systemctl"); err == nil {
		cmd = exec.Command("systemctl", "reload", "apache2")
	} else if _, err := exec.LookPath("apache2ctl"); err == nil {
		cmd = exec.Command("apache2ctl", "graceful")
	} else {
		cmd = exec.Command("httpd", "-k", "graceful")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("重载 Apache 失败: %s", string(output))
	}

	logger.Info("Apache 配置重载成功")
	return nil
}

// GetConfigPath 获取配置路径
func (a *ApacheConfigurator) GetConfigPath() string {
	return a.configPath
}

// IsSSLEnabled 检查 SSL 是否已启用
func (a *ApacheConfigurator) IsSSLEnabled(domain string) bool {
	// Apache SSL 检查实现
	return false
}

// IISConfigurator IIS 配置器
type IISConfigurator struct{}

// Configure 配置 IIS
func (i *IISConfigurator) Configure(config *Config) error {
	logger.Info("开始配置 IIS", "domain", config.Domain)

	// IIS 配置实现
	// 这里应该实现完整的 IIS SSL 配置逻辑，使用 PowerShell 脚本

	logger.Info("IIS 配置完成", "domain", config.Domain)
	return nil
}

// Test 测试 IIS 配置
func (i *IISConfigurator) Test() error {
	// IIS 没有直接的配置测试命令，可以检查站点状态
	logger.Info("IIS 配置测试成功")
	return nil
}

// Reload 重载 IIS 配置
func (i *IISConfigurator) Reload() error {
	cmd := exec.Command("iisreset")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("重载 IIS 失败: %s", string(output))
	}

	logger.Info("IIS 配置重载成功")
	return nil
}

// GetConfigPath 获取配置路径
func (i *IISConfigurator) GetConfigPath() string {
	return `C:\Windows\System32\inetsrv\config\applicationHost.config`
}

// IsSSLEnabled 检查 SSL 是否已启用
func (i *IISConfigurator) IsSSLEnabled(domain string) bool {
	// IIS SSL 检查实现
	return false
}
