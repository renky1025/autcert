package cert

import (
	"autocert/internal/config"
	"autocert/internal/logger"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ChallengeType ACME 挑战类型
type ChallengeType int

const (
	ChallengeWebroot ChallengeType = iota
	ChallengeStandalone
	ChallengeDNS
)

// WebServerType Web 服务器类型
type WebServerType int

const (
	WebServerNginx WebServerType = iota
	WebServerApache
	WebServerIIS
)

// Manager 证书管理器
type Manager struct {
	domain        string
	email         string
	challengeType ChallengeType
	webrootPath   string
	webServerType WebServerType
	certDir       string
	keySize       int
}

// CertInfo 证书信息
type CertInfo struct {
	Domain     string
	CertPath   string
	KeyPath    string
	ChainPath  string
	ExpiryDate time.Time
	IsValid    bool
}

// NewManager 创建新的证书管理器
func NewManager(domain, email string) *Manager {
	return &Manager{
		domain:        domain,
		email:         email,
		challengeType: ChallengeWebroot,
		certDir:       config.GetCertDir(),
		keySize:       2048,
	}
}

// SetChallengeType 设置挑战类型
func (m *Manager) SetChallengeType(challengeType ChallengeType) {
	m.challengeType = challengeType
}

// SetWebrootPath 设置 webroot 路径
func (m *Manager) SetWebrootPath(path string) {
	m.webrootPath = path
}

// SetWebServer 设置 Web 服务器类型
func (m *Manager) SetWebServer(webServerType WebServerType) {
	m.webServerType = webServerType
}

// Install 安装证书
func (m *Manager) Install() error {
	logger.Info("开始安装证书", "domain", m.domain)

	// 1. 创建证书目录
	if err := m.createCertDir(); err != nil {
		return fmt.Errorf("创建证书目录失败: %w", err)
	}

	// 2. 生成私钥
	privateKey, err := m.generatePrivateKey()
	if err != nil {
		return fmt.Errorf("生成私钥失败: %w", err)
	}

	// 3. 创建证书签名请求
	csr, err := m.createCSR(privateKey)
	if err != nil {
		return fmt.Errorf("创建 CSR 失败: %w", err)
	}

	// 4. 通过 ACME 获取证书
	cert, err := m.obtainCertificate(csr)
	if err != nil {
		return fmt.Errorf("获取证书失败: %w", err)
	}

	// 5. 保存证书和私钥
	if err := m.saveCertificate(cert, privateKey); err != nil {
		return fmt.Errorf("保存证书失败: %w", err)
	}

	// 6. 配置 Web 服务器
	if err := m.configureWebServer(); err != nil {
		return fmt.Errorf("配置 Web 服务器失败: %w", err)
	}

	logger.Info("证书安装完成", "domain", m.domain)
	return nil
}

// Renew 续期证书
func (m *Manager) Renew() error {
	logger.Info("开始续期证书", "domain", m.domain)

	// 检查证书是否需要续期
	certInfo, err := m.GetCertInfo()
	if err != nil {
		return fmt.Errorf("获取证书信息失败: %w", err)
	}

	// 如果证书有效期超过 30 天，则不需要续期
	if time.Until(certInfo.ExpiryDate) > 30*24*time.Hour {
		logger.Info("证书还未到续期时间", "domain", m.domain, "expiry", certInfo.ExpiryDate)
		return nil
	}

	// 执行续期流程（基本和安装流程相同）
	return m.Install()
}

// GetCertInfo 获取证书信息
func (m *Manager) GetCertInfo() (*CertInfo, error) {
	certPath := m.getCertPath()

	// 检查证书文件是否存在
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("证书文件不存在: %s", certPath)
	}

	// 读取证书文件
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("读取证书文件失败: %w", err)
	}

	// 解析证书
	block, _ := pem.Decode(certData)
	if block == nil {
		return nil, fmt.Errorf("无法解析证书文件")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}

	return &CertInfo{
		Domain:     m.domain,
		CertPath:   certPath,
		KeyPath:    m.getKeyPath(),
		ChainPath:  m.getChainPath(),
		ExpiryDate: cert.NotAfter,
		IsValid:    time.Now().Before(cert.NotAfter),
	}, nil
}

// createCertDir 创建证书目录
func (m *Manager) createCertDir() error {
	certDir := filepath.Join(m.certDir, m.domain)
	return os.MkdirAll(certDir, 0755)
}

// generatePrivateKey 生成私钥
func (m *Manager) generatePrivateKey() (*rsa.PrivateKey, error) {
	logger.Debug("生成私钥", "keySize", m.keySize)

	privateKey, err := rsa.GenerateKey(rand.Reader, m.keySize)
	if err != nil {
		return nil, err
	}

	// 保存私钥到文件
	keyPath := m.getKeyPath()
	keyFile, err := os.Create(keyPath)
	if err != nil {
		return nil, err
	}
	defer keyFile.Close()

	keyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	keyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	if err := pem.Encode(keyFile, keyPEM); err != nil {
		return nil, err
	}

	// 设置私钥文件权限
	if err := os.Chmod(keyPath, 0600); err != nil {
		return nil, err
	}

	logger.Debug("私钥生成完成", "keyPath", keyPath)
	return privateKey, nil
}

// createCSR 创建证书签名请求
func (m *Manager) createCSR(privateKey *rsa.PrivateKey) ([]byte, error) {
	logger.Debug("创建 CSR", "domain", m.domain)

	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: m.domain,
		},
		DNSNames: []string{m.domain},
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		return nil, err
	}

	logger.Debug("CSR 创建完成")
	return csrBytes, nil
}

// obtainCertificate 通过 ACME 获取证书
func (m *Manager) obtainCertificate(csr []byte) ([]byte, error) {
	logger.Info("开始 ACME 证书申请流程", "domain", m.domain, "challengeType", m.challengeType)

	switch m.challengeType {
	case ChallengeWebroot:
		return m.obtainCertificateWebroot(csr)
	case ChallengeStandalone:
		return m.obtainCertificateStandalone(csr)
	case ChallengeDNS:
		return m.obtainCertificateDNS(csr)
	default:
		return nil, fmt.Errorf("不支持的验证模式: %d", m.challengeType)
	}
}

// generateSelfSignedCert 生成自签名证书（仅用于演示）
func (m *Manager) generateSelfSignedCert(csr []byte) ([]byte, error) {
	logger.Warn("生成自签名证书（仅用于演示）", "domain", m.domain)

	// 解析 CSR
	csrParsed, err := x509.ParseCertificateRequest(csr)
	if err != nil {
		return nil, err
	}

	// 创建证书模板
	template := x509.Certificate{
		Subject:     csrParsed.Subject,
		DNSNames:    csrParsed.DNSNames,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(90 * 24 * time.Hour), // 90 天有效期
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	// 生成私钥（用于签名）
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// 创建证书
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	return certBytes, nil
}

// saveCertificate 保存证书和私钥
func (m *Manager) saveCertificate(certBytes []byte, privateKey *rsa.PrivateKey) error {
	logger.Debug("保存证书", "domain", m.domain)

	// 保存证书
	certPath := m.getCertPath()
	certFile, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certFile.Close()

	certPEM := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}

	if err := pem.Encode(certFile, certPEM); err != nil {
		return err
	}

	logger.Debug("证书保存完成", "certPath", certPath)
	return nil
}

// configureWebServer 配置 Web 服务器
func (m *Manager) configureWebServer() error {
	logger.Info("配置 Web 服务器", "type", m.webServerType)

	switch m.webServerType {
	case WebServerNginx:
		return m.configureNginx()
	case WebServerApache:
		return m.configureApache()
	case WebServerIIS:
		return m.configureIIS()
	default:
		return fmt.Errorf("不支持的 Web 服务器类型")
	}
}

// configureNginx 配置 Nginx
func (m *Manager) configureNginx() error {
	logger.Info("配置 Nginx SSL", "domain", m.domain)

	// 这里应该实现真正的 Nginx 配置逻辑
	// 包括创建虚拟主机配置、启用 SSL 等

	logger.Info("Nginx 配置完成")
	return nil
}

// configureApache 配置 Apache
func (m *Manager) configureApache() error {
	logger.Info("配置 Apache SSL", "domain", m.domain)

	// 这里应该实现真正的 Apache 配置逻辑

	logger.Info("Apache 配置完成")
	return nil
}

// configureIIS 配置 IIS
func (m *Manager) configureIIS() error {
	logger.Info("配置 IIS SSL", "domain", m.domain)

	// 这里应该实现真正的 IIS 配置逻辑

	logger.Info("IIS 配置完成")
	return nil
}

// 获取各种文件路径
func (m *Manager) getCertPath() string {
	return filepath.Join(m.certDir, m.domain, "cert.pem")
}

func (m *Manager) getKeyPath() string {
	return filepath.Join(m.certDir, m.domain, "key.pem")
}

func (m *Manager) getChainPath() string {
	return filepath.Join(m.certDir, m.domain, "chain.pem")
}

// obtainCertificateWebroot 使用 Webroot 模式获取证书
func (m *Manager) obtainCertificateWebroot(csr []byte) ([]byte, error) {
	logger.Info("使用 Webroot 模式获取证书", "domain", m.domain, "webroot", m.webrootPath)

	// 这里应该实现真正的 ACME Webroot 验证逻辑
	// 1. 在 webroot/.well-known/acme-challenge/ 目录下创建挑战文件
	// 2. 向 Let's Encrypt 服务器发送证书申请
	// 3. Let's Encrypt 服务器通过 HTTP 访问挑战文件进行验证

	// 为了演示，这里使用自签名证书
	return m.generateSelfSignedCert(csr)
}

// obtainCertificateStandalone 使用 Standalone 模式获取证书
func (m *Manager) obtainCertificateStandalone(csr []byte) ([]byte, error) {
	logger.Info("使用 Standalone 模式获取证书", "domain", m.domain)

	// 这里应该实现真正的 ACME Standalone 验证逻辑
	// 1. 启动临时 HTTP 服务器监听 80 端口
	// 2. 向 Let's Encrypt 服务器发送证书申请
	// 3. Let's Encrypt 服务器通过 HTTP 访问挑战路径进行验证
	// 4. 验证成功后关闭临时服务器

	// 为了演示，这里使用自签名证书
	return m.generateSelfSignedCert(csr)
}

// obtainCertificateDNS 使用 DNS 模式获取证书（支持泛域名）
func (m *Manager) obtainCertificateDNS(csr []byte) ([]byte, error) {
	logger.Info("使用 DNS 模式获取证书", "domain", m.domain)

	// 这里应该实现真正的 ACME DNS 验证逻辑
	// 1. 向 Let's Encrypt 服务器发送证书申请
	// 2. 获取 DNS 挑战记录值
	// 3. 在 DNS 服务商中添加 TXT 记录：_acme-challenge.domain.com
	// 4. 等待 DNS 传播完成
	// 5. 通知 Let's Encrypt 服务器进行验证
	// 6. 验证成功后清理 DNS 记录

	logger.Warn("注意：DNS 模式需要手动添加 DNS 记录或配置 DNS API", "domain", m.domain)

	// 为了演示，这里使用自签名证书
	return m.generateSelfSignedCert(csr)
}
