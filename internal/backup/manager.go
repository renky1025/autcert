package backup

import (
	"archive/tar"
	"archive/zip"
	"autocert/internal/config"
	"autocert/internal/logger"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manager 备份管理器
type Manager struct {
	certDir   string
	configDir string
}

// ExportOptions 导出选项
type ExportOptions struct {
	OutputFile string
	Format     string // tar.gz, zip
	Domain     string // 可选，只导出指定域名
}

// ImportOptions 导入选项
type ImportOptions struct {
	InputFile       string
	RestoreSchedule bool
}

// BackupMetadata 备份元数据
type BackupMetadata struct {
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	Platform    string    `json:"platform"`
	Domains     []string  `json:"domains"`
	HasSchedule bool      `json:"has_schedule"`
}

// NewManager 创建备份管理器
func NewManager() *Manager {
	return &Manager{
		certDir:   config.GetCertDir(),
		configDir: config.GetConfigDir(),
	}
}

// Export 导出证书和配置
func (m *Manager) Export(options *ExportOptions) error {
	logger.Info("开始导出", "format", options.Format, "output", options.OutputFile)

	// 收集要导出的文件
	files, err := m.collectFiles(options.Domain)
	if err != nil {
		return fmt.Errorf("收集文件失败: %w", err)
	}

	// 创建元数据
	metadata, err := m.createMetadata(files, options.Domain)
	if err != nil {
		return fmt.Errorf("创建元数据失败: %w", err)
	}

	// 根据格式选择导出方法
	switch strings.ToLower(options.Format) {
	case "tar.gz", "tgz":
		return m.exportTarGz(options.OutputFile, files, metadata)
	case "zip":
		return m.exportZip(options.OutputFile, files, metadata)
	default:
		return fmt.Errorf("不支持的导出格式: %s", options.Format)
	}
}

// Import 导入证书和配置
func (m *Manager) Import(options *ImportOptions) error {
	logger.Info("开始导入", "input", options.InputFile)

	// 检查文件是否存在
	if _, err := os.Stat(options.InputFile); os.IsNotExist(err) {
		return fmt.Errorf("导入文件不存在: %s", options.InputFile)
	}

	// 根据文件扩展名选择导入方法
	ext := strings.ToLower(filepath.Ext(options.InputFile))
	switch ext {
	case ".gz":
		if strings.HasSuffix(options.InputFile, ".tar.gz") {
			return m.importTarGz(options.InputFile, options.RestoreSchedule)
		}
		return fmt.Errorf("不支持的文件格式: %s", options.InputFile)
	case ".zip":
		return m.importZip(options.InputFile, options.RestoreSchedule)
	default:
		return fmt.Errorf("不支持的文件格式: %s", options.InputFile)
	}
}

// collectFiles 收集要导出的文件
func (m *Manager) collectFiles(domain string) (map[string]string, error) {
	files := make(map[string]string) // key: 归档路径, value: 本地路径

	// 收集证书文件
	if domain != "" {
		// 只导出指定域名
		domainDir := filepath.Join(m.certDir, domain)
		if _, err := os.Stat(domainDir); err == nil {
			if err := m.addDomainFiles(files, domain, domainDir); err != nil {
				return nil, err
			}
		}
	} else {
		// 导出所有域名
		entries, err := os.ReadDir(m.certDir)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("读取证书目录失败: %w", err)
			}
		} else {
			for _, entry := range entries {
				if entry.IsDir() {
					domainDir := filepath.Join(m.certDir, entry.Name())
					if err := m.addDomainFiles(files, entry.Name(), domainDir); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	// 收集配置文件
	if err := m.addConfigFiles(files); err != nil {
		return nil, err
	}

	return files, nil
}

// addDomainFiles 添加域名相关文件
func (m *Manager) addDomainFiles(files map[string]string, domain, domainDir string) error {
	entries, err := os.ReadDir(domainDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			localPath := filepath.Join(domainDir, entry.Name())
			archivePath := filepath.Join("certs", domain, entry.Name())
			files[archivePath] = localPath
		}
	}

	return nil
}

// addConfigFiles 添加配置文件
func (m *Manager) addConfigFiles(files map[string]string) error {
	configFiles := []string{
		".autocert.yaml",
		"autocert.yaml",
		"config.yaml",
	}

	// 检查配置目录
	if _, err := os.Stat(m.configDir); err == nil {
		entries, err := os.ReadDir(m.configDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
					localPath := filepath.Join(m.configDir, entry.Name())
					archivePath := filepath.Join("config", entry.Name())
					files[archivePath] = localPath
				}
			}
		}
	}

	// 检查用户目录下的配置文件
	homeDir, err := os.UserHomeDir()
	if err == nil {
		for _, configFile := range configFiles {
			configPath := filepath.Join(homeDir, configFile)
			if _, err := os.Stat(configPath); err == nil {
				archivePath := filepath.Join("config", configFile)
				files[archivePath] = configPath
			}
		}
	}

	return nil
}

// createMetadata 创建备份元数据
func (m *Manager) createMetadata(files map[string]string, domain string) (*BackupMetadata, error) {
	metadata := &BackupMetadata{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		Platform:    getOSInfo(),
		HasSchedule: false,
	}

	// 提取域名列表
	domainsMap := make(map[string]bool)
	for archivePath := range files {
		if strings.HasPrefix(archivePath, "certs/") {
			parts := strings.Split(archivePath, "/")
			if len(parts) >= 2 {
				domainsMap[parts[1]] = true
			}
		}
	}

	for d := range domainsMap {
		metadata.Domains = append(metadata.Domains, d)
	}

	// 检查是否有定时任务（这里简化处理）
	metadata.HasSchedule = true

	return metadata, nil
}

// exportTarGz 导出为 tar.gz 格式
func (m *Manager) exportTarGz(outputFile string, files map[string]string, metadata *BackupMetadata) error {
	logger.Debug("导出为 tar.gz 格式", "output", outputFile)

	// 创建输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 创建 gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// 创建 tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// 添加元数据文件
	if err := m.addMetadataToTar(tarWriter, metadata); err != nil {
		return err
	}

	// 添加文件
	for archivePath, localPath := range files {
		if err := m.addFileToTar(tarWriter, archivePath, localPath); err != nil {
			logger.Warn("跳过文件", "file", localPath, "error", err)
			continue
		}
	}

	logger.Debug("tar.gz 导出完成")
	return nil
}

// exportZip 导出为 zip 格式
func (m *Manager) exportZip(outputFile string, files map[string]string, metadata *BackupMetadata) error {
	logger.Debug("导出为 zip 格式", "output", outputFile)

	// 创建输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 创建 zip writer
	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// 添加元数据文件
	if err := m.addMetadataToZip(zipWriter, metadata); err != nil {
		return err
	}

	// 添加文件
	for archivePath, localPath := range files {
		if err := m.addFileToZip(zipWriter, archivePath, localPath); err != nil {
			logger.Warn("跳过文件", "file", localPath, "error", err)
			continue
		}
	}

	logger.Debug("zip 导出完成")
	return nil
}

// importTarGz 导入 tar.gz 格式
func (m *Manager) importTarGz(inputFile string, restoreSchedule bool) error {
	logger.Debug("导入 tar.gz 格式", "input", inputFile)

	// 打开文件
	file, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建 gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	// 创建 tar reader
	tarReader := tar.NewReader(gzReader)

	// 读取文件
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := m.extractFileFromTar(tarReader, header, restoreSchedule); err != nil {
			logger.Warn("提取文件失败", "file", header.Name, "error", err)
			continue
		}
	}

	logger.Debug("tar.gz 导入完成")
	return nil
}

// importZip 导入 zip 格式
func (m *Manager) importZip(inputFile string, restoreSchedule bool) error {
	logger.Debug("导入 zip 格式", "input", inputFile)

	// 打开 zip 文件
	zipReader, err := zip.OpenReader(inputFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	// 提取文件
	for _, file := range zipReader.File {
		if err := m.extractFileFromZip(file, restoreSchedule); err != nil {
			logger.Warn("提取文件失败", "file", file.Name, "error", err)
			continue
		}
	}

	logger.Debug("zip 导入完成")
	return nil
}

// 辅助方法

func (m *Manager) addMetadataToTar(tarWriter *tar.Writer, metadata *BackupMetadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name: "metadata.json",
		Size: int64(len(data)),
		Mode: 0644,
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	_, err = tarWriter.Write(data)
	return err
}

func (m *Manager) addFileToTar(tarWriter *tar.Writer, archivePath, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name: archivePath,
		Size: info.Size(),
		Mode: int64(info.Mode()),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)
	return err
}

func (m *Manager) addMetadataToZip(zipWriter *zip.Writer, metadata *BackupMetadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	writer, err := zipWriter.Create("metadata.json")
	if err != nil {
		return err
	}

	_, err = writer.Write(data)
	return err
}

func (m *Manager) addFileToZip(zipWriter *zip.Writer, archivePath, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := zipWriter.Create(archivePath)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

func (m *Manager) extractFileFromTar(tarReader *tar.Reader, header *tar.Header, restoreSchedule bool) error {
	// 跳过元数据文件（已经处理）
	if header.Name == "metadata.json" {
		return nil
	}

	// 确定目标路径
	targetPath, err := m.getTargetPath(header.Name)
	if err != nil {
		return err
	}

	// 创建目录
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	// 创建文件
	outFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 复制内容
	_, err = io.Copy(outFile, tarReader)
	if err != nil {
		return err
	}

	// 设置权限
	if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
		logger.Warn("设置文件权限失败", "file", targetPath, "error", err)
	}

	logger.Debug("文件提取完成", "source", header.Name, "target", targetPath)
	return nil
}

func (m *Manager) extractFileFromZip(file *zip.File, restoreSchedule bool) error {
	// 跳过元数据文件
	if file.Name == "metadata.json" {
		return nil
	}

	// 确定目标路径
	targetPath, err := m.getTargetPath(file.Name)
	if err != nil {
		return err
	}

	// 打开源文件
	srcFile, err := file.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 创建目录
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	// 创建目标文件
	dstFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// 复制内容
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	logger.Debug("文件提取完成", "source", file.Name, "target", targetPath)
	return nil
}

func (m *Manager) getTargetPath(archivePath string) (string, error) {
	// 根据归档路径确定本地目标路径
	if strings.HasPrefix(archivePath, "certs/") {
		// 证书文件
		relativePath := strings.TrimPrefix(archivePath, "certs/")
		return filepath.Join(m.certDir, relativePath), nil
	} else if strings.HasPrefix(archivePath, "config/") {
		// 配置文件
		fileName := strings.TrimPrefix(archivePath, "config/")
		if strings.HasPrefix(fileName, ".") {
			// 用户配置文件
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(homeDir, fileName), nil
		} else {
			// 系统配置文件
			return filepath.Join(m.configDir, fileName), nil
		}
	}

	return "", fmt.Errorf("未知的归档路径: %s", archivePath)
}

func getOSInfo() string {
	// 简化的操作系统信息
	return fmt.Sprintf("%s/%s", "go", "1.21")
}
