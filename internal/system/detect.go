package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// OSInfo 操作系统信息
type OSInfo struct {
	Type         string // windows, linux
	Distribution string // ubuntu, centos, debian, etc.
	Version      string
	Architecture string
}

// WebServerInfo Web 服务器信息
type WebServerInfo struct {
	Type       string // nginx, apache, iis
	Version    string
	ConfigPath string
	IsRunning  bool
}

// SystemInfo 系统信息
type SystemInfo struct {
	OS         OSInfo
	WebServers []WebServerInfo
	HasRoot    bool // 是否有管理员权限
}

// DetectSystem 检测系统环境
func DetectSystem() (*SystemInfo, error) {
	info := &SystemInfo{}

	// 检测操作系统
	if err := detectOS(&info.OS); err != nil {
		return nil, fmt.Errorf("检测操作系统失败: %w", err)
	}

	// 检测权限
	info.HasRoot = hasAdminPrivileges()

	// 检测 Web 服务器
	webServers, err := detectWebServers()
	if err != nil {
		return nil, fmt.Errorf("检测 Web 服务器失败: %w", err)
	}
	info.WebServers = webServers

	return info, nil
}

// detectOS 检测操作系统信息
func detectOS(osInfo *OSInfo) error {
	osInfo.Type = runtime.GOOS
	osInfo.Architecture = runtime.GOARCH

	switch runtime.GOOS {
	case "windows":
		return detectWindowsVersion(osInfo)
	case "linux":
		return detectLinuxDistribution(osInfo)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// detectWindowsVersion 检测 Windows 版本
func detectWindowsVersion(osInfo *OSInfo) error {
	cmd := exec.Command("powershell", "-Command", "(Get-WmiObject -Class Win32_OperatingSystem).Caption")
	output, err := cmd.Output()
	if err != nil {
		osInfo.Version = "Unknown"
		return nil
	}

	osInfo.Version = strings.TrimSpace(string(output))
	osInfo.Distribution = "windows"
	return nil
}

// detectLinuxDistribution 检测 Linux 发行版
func detectLinuxDistribution(osInfo *OSInfo) error {
	// 尝试读取 /etc/os-release
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "ID=") {
				osInfo.Distribution = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			}
			if strings.HasPrefix(line, "VERSION_ID=") {
				osInfo.Version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
			}
		}
		return nil
	}

	// 备用方案：检查其他文件
	distFiles := []struct {
		file string
		dist string
	}{
		{"/etc/ubuntu-release", "ubuntu"},
		{"/etc/centos-release", "centos"},
		{"/etc/redhat-release", "rhel"},
		{"/etc/debian_version", "debian"},
	}

	for _, df := range distFiles {
		if _, err := os.Stat(df.file); err == nil {
			osInfo.Distribution = df.dist
			if data, err := os.ReadFile(df.file); err == nil {
				osInfo.Version = strings.TrimSpace(string(data))
			}
			return nil
		}
	}

	osInfo.Distribution = "unknown"
	osInfo.Version = "unknown"
	return nil
}

// hasAdminPrivileges 检查是否有管理员权限
func hasAdminPrivileges() bool {
	if runtime.GOOS == "windows" {
		// Windows: 检查是否以管理员身份运行
		cmd := exec.Command("net", "session")
		err := cmd.Run()
		return err == nil
	} else {
		// Linux: 检查是否为 root 用户
		return os.Geteuid() == 0
	}
}

// detectWebServers 检测已安装的 Web 服务器
func detectWebServers() ([]WebServerInfo, error) {
	var servers []WebServerInfo

	if runtime.GOOS == "windows" {
		// 检测 IIS
		if iisInfo := detectIIS(); iisInfo != nil {
			servers = append(servers, *iisInfo)
		}

		// 检测 Windows 上的 Nginx
		if nginxInfo := detectNginxWindows(); nginxInfo != nil {
			servers = append(servers, *nginxInfo)
		}
	} else {
		// 检测 Linux 上的 Nginx
		if nginxInfo := detectNginxLinux(); nginxInfo != nil {
			servers = append(servers, *nginxInfo)
		}

		// 检测 Apache
		if apacheInfo := detectApache(); apacheInfo != nil {
			servers = append(servers, *apacheInfo)
		}
	}

	return servers, nil
}

// detectIIS 检测 IIS
func detectIIS() *WebServerInfo {
	// 检查 IIS 是否安装
	cmd := exec.Command("powershell", "-Command", "Get-WindowsFeature -Name IIS-WebServer | Select-Object InstallState")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	if strings.Contains(string(output), "Installed") {
		return &WebServerInfo{
			Type:       "iis",
			Version:    "Unknown",
			ConfigPath: `C:\inetpub\wwwroot`,
			IsRunning:  isServiceRunning("W3SVC"),
		}
	}

	return nil
}

// detectNginxWindows 检测 Windows 上的 Nginx
func detectNginxWindows() *WebServerInfo {
	// 常见的 Nginx 安装路径
	paths := []string{
		`C:\nginx\nginx.exe`,
		`C:\Program Files\nginx\nginx.exe`,
		`C:\nginx-*\nginx.exe`,
	}

	for _, path := range paths {
		if matches, _ := filepath.Glob(path); len(matches) > 0 {
			nginxPath := matches[0]
			version := getNginxVersion(nginxPath)
			configPath := filepath.Dir(nginxPath) + `\conf\nginx.conf`

			return &WebServerInfo{
				Type:       "nginx",
				Version:    version,
				ConfigPath: configPath,
				IsRunning:  isProcessRunning("nginx.exe"),
			}
		}
	}

	return nil
}

// detectNginxLinux 检测 Linux 上的 Nginx
func detectNginxLinux() *WebServerInfo {
	// 检查 nginx 命令是否存在
	_, err := exec.LookPath("nginx")
	if err != nil {
		return nil
	}

	// 获取版本
	cmd := exec.Command("nginx", "-v")
	output, err := cmd.CombinedOutput()
	version := "Unknown"
	if err == nil {
		version = strings.TrimSpace(string(output))
	}

	// 查找配置文件
	configPaths := []string{
		"/etc/nginx/nginx.conf",
		"/usr/local/nginx/conf/nginx.conf",
	}

	configPath := ""
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	return &WebServerInfo{
		Type:       "nginx",
		Version:    version,
		ConfigPath: configPath,
		IsRunning:  isServiceRunning("nginx"),
	}
}

// detectApache 检测 Apache
func detectApache() *WebServerInfo {
	// 检查常见的 Apache 命令
	commands := []string{"apache2", "httpd"}
	var apacheCmd string

	for _, cmd := range commands {
		if _, err := exec.LookPath(cmd); err == nil {
			apacheCmd = cmd
			break
		}
	}

	if apacheCmd == "" {
		return nil
	}

	// 获取版本
	cmd := exec.Command(apacheCmd, "-v")
	output, err := cmd.Output()
	version := "Unknown"
	if err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			version = strings.TrimSpace(lines[0])
		}
	}

	// 查找配置文件
	configPaths := []string{
		"/etc/apache2/apache2.conf",
		"/etc/httpd/conf/httpd.conf",
		"/usr/local/apache2/conf/httpd.conf",
	}

	configPath := ""
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	return &WebServerInfo{
		Type:       "apache",
		Version:    version,
		ConfigPath: configPath,
		IsRunning:  isServiceRunning(apacheCmd),
	}
}

// getNginxVersion 获取 Nginx 版本
func getNginxVersion(nginxPath string) string {
	cmd := exec.Command(nginxPath, "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(output))
}

// isServiceRunning 检查服务是否运行
func isServiceRunning(serviceName string) bool {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("sc", "query", serviceName)
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), "RUNNING")
	} else {
		cmd := exec.Command("systemctl", "is-active", serviceName)
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.TrimSpace(string(output)) == "active"
	}
}

// isProcessRunning 检查进程是否运行
func isProcessRunning(processName string) bool {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName))
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), processName)
	} else {
		cmd := exec.Command("pgrep", processName)
		err := cmd.Run()
		return err == nil
	}
}
