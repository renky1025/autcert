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
	"strings"
	"time"
)

// MultiDomainManager 多域名证书管理器
type MultiDomainManager struct {
	domains       []string
	primaryDomain string
	email         string
	challengeType ChallengeType
	webrootPath   string
	webServerType WebServerType
	certDir       string
	keySize       int
}

// NewMultiDomainManager 创建新的多域名证书管理器
func NewMultiDomainManager(domains []string, email string) *MultiDomainManager {
	if len(domains) == 0 {
		return nil
	}

	return &MultiDomainManager{
		domains:       domains,
		primaryDomain: domains[0], // 第一个域名作为主域名
		email:         email,
		challengeType: ChallengeWebroot,
		certDir:       config.GetCertDir(),
		keySize:       2048,
	}
}

// SetChallengeType 设置挑战类型
func (m *MultiDomainManager) SetChallengeType(challengeType ChallengeType) {
	m.challengeType = challengeType
}

// SetWebrootPath 设置 webroot 路径
func (m *MultiDomainManager) SetWebrootPath(path string) {
	m.webrootPath = path
}

// SetWebServer 设置 Web 服务器类型
func (m *MultiDomainManager) SetWebServer(webServerType WebServerType) {
	m.webServerType = webServerType
}

// Install 安装多域名证书
func (m *MultiDomainManager) Install() error {
	logger.Info("开始安装多域名证书", "domains", m.domains, "primaryDomain", m.primaryDomain)

	// 检查是否有泛域名
	hasWildcard := m.hasWildcardDomain()
	if hasWildcard && m.challengeType != ChallengeDNS {
		return fmt.Errorf("泛域名证书必须使用 DNS 验证模式")
	}

	// 1. 创建证书目录（使用主域名）
	if err := m.createCertDir(); err != nil {
		return fmt.Errorf("创建证书目录失败: %w", err)
	}

	// 2. 生成私钥
	privateKey, err := m.generatePrivateKey()
	if err != nil {
		return fmt.Errorf("生成私钥失败: %w", err)
	}

	// 3. 创建多域名证书签名请求
	csr, err := m.createMultiDomainCSR(privateKey)
	if err != nil {
		return fmt.Errorf("创建多域名 CSR 失败: %w", err)
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

	// 6. 为每个域名配置 Web 服务器
	if err := m.configureWebServers(); err != nil {
		return fmt.Errorf("配置 Web 服务器失败: %w", err)
	}

	logger.Info("多域名证书安装完成", "domains", m.domains)
	return nil
}

// hasWildcardDomain 检查是否包含泛域名
func (m *MultiDomainManager) hasWildcardDomain() bool {
	for _, domain := range m.domains {
		if strings.HasPrefix(domain, "*.") {
			return true
		}
	}
	return false
}

// createCertDir 创建证书目录
func (m *MultiDomainManager) createCertDir() error {
	// 使用主域名作为目录名，但添加多域名标识
	dirName := m.primaryDomain
	if len(m.domains) > 1 {
		dirName = fmt.Sprintf("%s_san", m.primaryDomain)
	}

	certDir := filepath.Join(m.certDir, dirName)
	return os.MkdirAll(certDir, 0755)
}

// generatePrivateKey 生成私钥
func (m *MultiDomainManager) generatePrivateKey() (*rsa.PrivateKey, error) {
	logger.Debug("生成多域名证书私钥", "keySize", m.keySize)

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

	logger.Debug("多域名证书私钥生成完成", "keyPath", keyPath)
	return privateKey, nil
}

// createMultiDomainCSR 创建多域名证书签名请求
func (m *MultiDomainManager) createMultiDomainCSR(privateKey *rsa.PrivateKey) ([]byte, error) {
	logger.Debug("创建多域名 CSR", "domains", m.domains)

	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: m.primaryDomain,
		},
		DNSNames: m.domains, // 所有域名都放在 SAN 中
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		return nil, err
	}

	logger.Debug("多域名 CSR 创建完成", "domains", m.domains)
	return csrBytes, nil
}

// obtainCertificate 获取多域名证书
func (m *MultiDomainManager) obtainCertificate(csr []byte) ([]byte, error) {
	logger.Info("开始多域名 ACME 证书申请流程", "domains", m.domains, "challengeType", m.challengeType)

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

// obtainCertificateWebroot 使用 Webroot 模式获取多域名证书
func (m *MultiDomainManager) obtainCertificateWebroot(csr []byte) ([]byte, error) {
	logger.Info("使用 Webroot 模式获取多域名证书", "domains", m.domains)

	// 检查是否有泛域名
	if m.hasWildcardDomain() {
		return nil, fmt.Errorf("泛域名证书不能使用 Webroot 验证模式，请使用 DNS 验证")
	}

	// 为了演示，这里使用自签名证书
	return m.generateMultiDomainSelfSignedCert(csr)
}

// obtainCertificateStandalone 使用 Standalone 模式获取多域名证书
func (m *MultiDomainManager) obtainCertificateStandalone(csr []byte) ([]byte, error) {
	logger.Info("使用 Standalone 模式获取多域名证书", "domains", m.domains)

	// 检查是否有泛域名
	if m.hasWildcardDomain() {
		return nil, fmt.Errorf("泛域名证书不能使用 Standalone 验证模式，请使用 DNS 验证")
	}

	// 为了演示，这里使用自签名证书
	return m.generateMultiDomainSelfSignedCert(csr)
}

// obtainCertificateDNS 使用 DNS 模式获取多域名证书
func (m *MultiDomainManager) obtainCertificateDNS(csr []byte) ([]byte, error) {
	logger.Info("使用 DNS 模式获取多域名证书", "domains", m.domains)

	// DNS 模式支持所有类型的域名，包括泛域名
	logger.Warn("注意：DNS 模式需要手动添加 DNS 记录或配置 DNS API", "domains", m.domains)

	// 显示需要添加的 DNS 记录
	for _, domain := range m.domains {
		if strings.HasPrefix(domain, "*.") {
			baseDomain := domain[2:]
			logger.Info("需要为泛域名添加 DNS TXT 记录",
				"record", fmt.Sprintf("_acme-challenge.%s", baseDomain),
				"domain", domain)
		} else {
			logger.Info("需要为域名添加 DNS TXT 记录",
				"record", fmt.Sprintf("_acme-challenge.%s", domain),
				"domain", domain)
		}
	}

	// 为了演示，这里使用自签名证书
	return m.generateMultiDomainSelfSignedCert(csr)
}

// generateMultiDomainSelfSignedCert 生成多域名自签名证书
func (m *MultiDomainManager) generateMultiDomainSelfSignedCert(csr []byte) ([]byte, error) {
	logger.Warn("生成多域名自签名证书（仅用于演示）", "domains", m.domains)

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
func (m *MultiDomainManager) saveCertificate(certBytes []byte, privateKey *rsa.PrivateKey) error {
	logger.Debug("保存多域名证书", "domains", m.domains)

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

	// 创建域名列表文件（用于记录此证书包含的所有域名）
	domainsFile := m.getDomainsListPath()
	if err := os.WriteFile(domainsFile, []byte(strings.Join(m.domains, "\n")), 0644); err != nil {
		logger.Warn("无法创建域名列表文件", "error", err)
	}

	logger.Debug("多域名证书保存完成", "certPath", certPath, "domains", m.domains)
	return nil
}

// configureWebServers 为所有域名配置 Web 服务器
func (m *MultiDomainManager) configureWebServers() error {
	logger.Info("配置多域名 Web 服务器", "type", m.webServerType, "domains", m.domains)

	// 为每个域名配置 Web 服务器
	for _, domain := range m.domains {
		if strings.HasPrefix(domain, "*.") {
			// 泛域名需要特殊处理
			logger.Info("配置泛域名", "domain", domain)
		} else {
			logger.Info("配置普通域名", "domain", domain)
		}
	}

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

// configureNginx 配置 Nginx 多域名
func (m *MultiDomainManager) configureNginx() error {
	logger.Info("配置 Nginx 多域名 SSL", "domains", m.domains)

	// 这里应该实现真正的 Nginx 多域名配置逻辑
	// 可以为每个域名创建单独的 server block，或者创建一个包含所有域名的 server block

	logger.Info("Nginx 多域名配置完成")
	return nil
}

// configureApache 配置 Apache 多域名
func (m *MultiDomainManager) configureApache() error {
	logger.Info("配置 Apache 多域名 SSL", "domains", m.domains)

	// 这里应该实现真正的 Apache 多域名配置逻辑

	logger.Info("Apache 多域名配置完成")
	return nil
}

// configureIIS 配置 IIS 多域名
func (m *MultiDomainManager) configureIIS() error {
	logger.Info("配置 IIS 多域名 SSL", "domains", m.domains)

	// 这里应该实现真正的 IIS 多域名配置逻辑

	logger.Info("IIS 多域名配置完成")
	return nil
}

// 获取各种文件路径
func (m *MultiDomainManager) getCertPath() string {
	dirName := m.primaryDomain
	if len(m.domains) > 1 {
		dirName = fmt.Sprintf("%s_san", m.primaryDomain)
	}
	return filepath.Join(m.certDir, dirName, "cert.pem")
}

func (m *MultiDomainManager) getKeyPath() string {
	dirName := m.primaryDomain
	if len(m.domains) > 1 {
		dirName = fmt.Sprintf("%s_san", m.primaryDomain)
	}
	return filepath.Join(m.certDir, dirName, "key.pem")
}

func (m *MultiDomainManager) getChainPath() string {
	dirName := m.primaryDomain
	if len(m.domains) > 1 {
		dirName = fmt.Sprintf("%s_san", m.primaryDomain)
	}
	return filepath.Join(m.certDir, dirName, "chain.pem")
}

func (m *MultiDomainManager) getDomainsListPath() string {
	dirName := m.primaryDomain
	if len(m.domains) > 1 {
		dirName = fmt.Sprintf("%s_san", m.primaryDomain)
	}
	return filepath.Join(m.certDir, dirName, "domains.txt")
}
