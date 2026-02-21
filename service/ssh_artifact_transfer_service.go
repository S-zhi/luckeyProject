package service

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	// DefaultStaticServerIP 默认静态服务器IP地址
	DefaultStaticServerIP = "192.168.1.100"
	// DefaultSSHServerPort 默认SSH服务器端口
	DefaultSSHServerPort = 22
	// DefaultSSHServerUser 默认SSH服务器用户名
	DefaultSSHServerUser = "root"
)

var (
	// ErrSSHClientFactoryNil SSH客户端工厂为空错误
	ErrSSHClientFactoryNil = errors.New("SSH 客户端工厂未初始化")
	// ErrSSHServerNameRequired 服务器名称必填错误
	ErrSSHServerNameRequired = errors.New("服务器名称不能为空")
	// ErrSSHServerIPRequired 服务器IP必填错误
	ErrSSHServerIPRequired = errors.New("服务器 IP 不能为空")
	// ErrSSHServerPortInvalid SSH服务器端口非法错误
	ErrSSHServerPortInvalid = errors.New("SSH 服务器端口无效")
	// ErrSSHServerUserRequired SSH服务器用户必填错误
	ErrSSHServerUserRequired = errors.New("SSH 用户名不能为空")
	// ErrSSHPrivateKeyPathRequired SSH私钥路径必填错误
	ErrSSHPrivateKeyPathRequired = errors.New("SSH 私钥路径不能为空")
	// ErrSSHFilePathRequired 文件路径必填错误
	ErrSSHFilePathRequired = errors.New("文件路径不能为空")
	// ErrLocalSourceFileNotFound 本地源文件未找到错误
	ErrLocalSourceFileNotFound = errors.New("未找到本地源文件")
	// ErrLocalSourcePathNotRegularFile 本地源路径非常规文件错误
	ErrLocalSourcePathNotRegularFile = errors.New("本地源路径不是普通文件")
	// ErrRemoteArtifactNotFound 远程构件未找到错误
	ErrRemoteArtifactNotFound = errors.New("未找到远端制品文件")
	// ErrRemoteArtifactAlreadyExists 远程构件已存在错误
	ErrRemoteArtifactAlreadyExists = errors.New("远端制品文件已存在")
	// ErrArtifactNotFoundInBackendRoots 后端根目录中未找到构件错误
	ErrArtifactNotFoundInBackendRoots = errors.New("后端根目录中未找到制品文件")
	// ErrArtifactNotFoundInRemoteOtherRoot 远程other根目录中未找到构件错误
	ErrArtifactNotFoundInRemoteOtherRoot = errors.New("远端 other 根目录中未找到制品文件")
	// ErrArtifactConflictInBackendRoots 构件在后端根目录中冲突错误
	ErrArtifactConflictInBackendRoots = errors.New("两个后端根目录都存在同名制品")
	// ErrArtifactConflictInRemoteRoots 构件在远程根目录中冲突错误
	ErrArtifactConflictInRemoteRoots = errors.New("两个远端根目录都存在同名制品")
)

var (
	// defaultSSHTimeout 默认SSH连接超时时间
	defaultSSHTimeout = 15 * time.Second
)

// SSHServerConfig SSH服务器配置信息
// 包含连接SSH服务器所需的所有配置参数
type SSHServerConfig struct {
	Name           string
	IP             string
	Port           int
	User           string
	PrivateKeyPath string
	Timeout        time.Duration
}

// SSHTransferResult SSH文件传输结果
// 记录文件传输的详细信息和统计
type SSHTransferResult struct {
	ServerName string        `json:"server_name"`
	ServerIP   string        `json:"server_ip"`
	Direction  string        `json:"direction"`
	Category   string        `json:"category,omitempty"`
	FileName   string        `json:"file_name,omitempty"`
	SourcePath string        `json:"source_path"`
	TargetPath string        `json:"target_path"`
	Bytes      int64         `json:"bytes"`
	Cost       time.Duration `json:"cost"`
}

// RemoteArtifactSearchResult 远程构件文件搜索结果
// 包含文件在远程服务器上weights和datasets目录中的存在状态
type RemoteArtifactSearchResult struct {
	ServerName        string `json:"server_name"`
	ServerIP          string `json:"server_ip"`
	FileName          string `json:"file_name"`
	WeightsPath       string `json:"weights_path"`
	DatasetsPath      string `json:"datasets_path"`
	ExistsInWeights   bool   `json:"exists_in_weights"`
	ExistsInDatasets  bool   `json:"exists_in_datasets"`
	AnyExists         bool   `json:"any_exists"`
	MatchedRemotePath string `json:"matched_remote_path,omitempty"`
}

// remoteFileClient 远程文件客户端接口
// 定义文件传输操作的标准接口
type remoteFileClient interface {
	UploadFile(localPath, remotePath string) (int64, error)
	DownloadFile(remotePath, localPath string) (int64, error)
	FileExists(remotePath string) (bool, error)
	Close() error
}

// remoteFileClientFactory 远程文件客户端工厂接口
// 用于创建remoteFileClient实例
type remoteFileClientFactory interface {
	New(server SSHServerConfig) (remoteFileClient, error)
}

// SSHArtifactTransferService SSH构件传输服务
// 提供基于SSH的文件传输功能，支持构件文件的上传、下载和搜索
type SSHArtifactTransferService struct {
	PathService   *ArtifactPathService
	serverConfigs map[string]SSHServerConfig
	defaultServer SSHServerConfig
	clientFactory remoteFileClientFactory
}

// NewSSHArtifactTransferService 创建新的SSH文件传输服务实例
// 初始化默认服务器配置和静态服务器映射
// 返回SSHArtifactTransferService指针
func NewSSHArtifactTransferService() *SSHArtifactTransferService {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}
	defaultKeyPath := filepath.Join(homeDir, ".ssh", "id_rsa")

	defaultServer := SSHServerConfig{
		Name:           "default",
		IP:             DefaultStaticServerIP,
		Port:           DefaultSSHServerPort,
		User:           DefaultSSHServerUser,
		PrivateKeyPath: defaultKeyPath,
		Timeout:        defaultSSHTimeout,
	}

	// TODO: replace static mapping with Redis lookup.
	serverConfigs := map[string]SSHServerConfig{
		"other": {
			Name:           "other",
			IP:             DefaultStaticServerIP,
			Port:           DefaultSSHServerPort,
			User:           DefaultSSHServerUser,
			PrivateKeyPath: defaultKeyPath,
			Timeout:        defaultSSHTimeout,
		},
		"other_local": {
			Name:           "other_local",
			IP:             DefaultStaticServerIP,
			Port:           DefaultSSHServerPort,
			User:           DefaultSSHServerUser,
			PrivateKeyPath: defaultKeyPath,
			Timeout:        defaultSSHTimeout,
		},
		"backend": {
			Name:           "backend",
			IP:             DefaultStaticServerIP,
			Port:           DefaultSSHServerPort,
			User:           DefaultSSHServerUser,
			PrivateKeyPath: defaultKeyPath,
			Timeout:        defaultSSHTimeout,
		},
		"baidu_netdisk": {
			Name:           "baidu_netdisk",
			IP:             DefaultStaticServerIP,
			Port:           DefaultSSHServerPort,
			User:           DefaultSSHServerUser,
			PrivateKeyPath: defaultKeyPath,
			Timeout:        defaultSSHTimeout,
		},
	}

	return &SSHArtifactTransferService{
		PathService:   NewArtifactPathService(),
		serverConfigs: serverConfigs,
		defaultServer: defaultServer,
		clientFactory: &sshSFTPClientFactory{},
	}
}

// SetServerConfig 设置指定名称的SSH服务器配置
// 参数:
//   - serverName: 服务器名称
//   - cfg: SSH服务器配置信息
//
// 返回错误信息，成功时返回nil
func (s *SSHArtifactTransferService) SetServerConfig(serverName string, cfg SSHServerConfig) error {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "SetServerConfig")

	name := strings.TrimSpace(serverName)
	if name == "" {
		logger.Warn("设置服务器配置失败：服务器名称为空")
		return ErrSSHServerNameRequired
	}

	normalized, err := normalizeServerConfig(cfg)
	if err != nil {
		logger.Error("设置服务器配置失败：配置无效", "server_name", name, "error", err)
		return err
	}
	normalized.Name = name

	if s.serverConfigs == nil {
		s.serverConfigs = make(map[string]SSHServerConfig)
	}
	s.serverConfigs[name] = normalized
	logger.Info(
		"设置服务器配置成功",
		"server_name", name,
		"server_ip", normalized.IP,
		"port", normalized.Port,
		"user", normalized.User,
		"private_key_path", normalized.PrivateKeyPath,
	)
	return nil
}

// UploadFileByPath 通过指定路径上传文件到远程服务器
// 参数:
//   - localPath: 本地文件路径
//   - remotePath: 远程目标路径
//   - serverName: 目标服务器名称
//
// 返回传输结果和错误信息
func (s *SSHArtifactTransferService) UploadFileByPath(localPath, remotePath, serverName string) (SSHTransferResult, error) {
	return s.UploadFileByPathWithPort(localPath, remotePath, serverName, 0)
}

// UploadFileByPathWithPort 通过指定路径上传文件到远程服务器（支持端口覆盖）
// 参数:
//   - localPath: 本地文件路径
//   - remotePath: 远程目标路径
//   - serverName: 目标服务器名称
//   - port: SSH端口(>0时覆盖服务器默认端口)
//
// 返回传输结果和错误信息
func (s *SSHArtifactTransferService) UploadFileByPathWithPort(localPath, remotePath, serverName string, port int) (SSHTransferResult, error) {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "UploadFileByPathWithPort")
	start := time.Now()

	logger.Info(
		"上传开始",
		"server_name", strings.TrimSpace(serverName),
		"port", port,
		"local_path", strings.TrimSpace(localPath),
		"remote_path", strings.TrimSpace(remotePath),
	)

	if strings.TrimSpace(localPath) == "" || strings.TrimSpace(remotePath) == "" {
		logger.Warn("上传失败：本地路径或远程路径为空")
		return SSHTransferResult{}, ErrSSHFilePathRequired
	}
	if s.PathService == nil {
		logger.Warn("上传失败：构件路径服务为空")
		return SSHTransferResult{}, ErrArtifactPathServiceNil
	}
	if s.clientFactory == nil {
		logger.Warn("上传失败：SSH 客户端工厂为空")
		return SSHTransferResult{}, ErrSSHClientFactoryNil
	}

	normalizedLocal := filepath.Clean(strings.TrimSpace(localPath))
	normalizedRemote, err := normalizeRemoteFilePath(remotePath)
	if err != nil {
		logger.Warn("上传失败：远程路径无效", "remote_path", remotePath, "error", err)
		return SSHTransferResult{}, err
	}

	info, err := os.Stat(normalizedLocal)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn("上传失败：本地源文件不存在", "local_path", normalizedLocal)
			return SSHTransferResult{}, ErrLocalSourceFileNotFound
		}
		logger.Error("上传失败：获取本地源文件信息失败", "local_path", normalizedLocal, "error", err)
		return SSHTransferResult{}, fmt.Errorf("获取本地源文件信息失败: %w", err)
	}
	if !info.Mode().IsRegular() {
		logger.Warn("上传失败：本地源路径不是普通文件", "local_path", normalizedLocal, "mode", info.Mode().String())
		return SSHTransferResult{}, ErrLocalSourcePathNotRegularFile
	}

	server, err := s.resolveServerWithPort(serverName, port)
	if err != nil {
		logger.Error("上传失败：解析服务器配置失败", "server_name", serverName, "port", port, "error", err)
		return SSHTransferResult{}, err
	}

	client, err := s.clientFactory.New(server)
	if err != nil {
		logger.Error("上传失败：创建 SSH 客户端失败", "server_name", server.Name, "server_ip", server.IP, "error", err)
		return SSHTransferResult{}, err
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			logger.Error("上传后关闭客户端失败", "server_name", server.Name, "error", closeErr)
		}
	}()

	written, err := client.UploadFile(normalizedLocal, normalizedRemote)
	if err != nil {
		logger.Error(
			"上传失败",
			"server_name", server.Name,
			"server_ip", server.IP,
			"local_path", normalizedLocal,
			"remote_path", normalizedRemote,
			"error", err,
		)
		return SSHTransferResult{}, err
	}

	result := SSHTransferResult{
		ServerName: server.Name,
		ServerIP:   server.IP,
		Direction:  "upload",
		SourcePath: filepath.ToSlash(normalizedLocal),
		TargetPath: normalizedRemote,
		Bytes:      written,
		Cost:       time.Since(start),
	}

	logger.Info(
		"上传成功",
		"server_name", server.Name,
		"server_ip", server.IP,
		"port", server.Port,
		"bytes", written,
		"cost_ms", result.Cost.Milliseconds(),
		"source_path", result.SourcePath,
		"target_path", result.TargetPath,
	)
	return result, nil
}

// DownloadFileByPath 通过指定路径从远程服务器下载文件
// 参数:
//   - remotePath: 远程文件路径
//   - localPath: 本地目标路径
//   - serverName: 源服务器名称
//
// 返回传输结果和错误信息
func (s *SSHArtifactTransferService) DownloadFileByPath(remotePath, localPath, serverName string) (SSHTransferResult, error) {
	return s.DownloadFileByPathWithPort(remotePath, localPath, serverName, 0)
}

// DownloadFileByPathWithPort 通过指定路径从远程服务器下载文件（支持端口覆盖）
// 参数:
//   - remotePath: 远程文件路径
//   - localPath: 本地目标路径
//   - serverName: 源服务器名称
//   - port: SSH端口(>0时覆盖服务器默认端口)
//
// 返回传输结果和错误信息
func (s *SSHArtifactTransferService) DownloadFileByPathWithPort(remotePath, localPath, serverName string, port int) (SSHTransferResult, error) {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "DownloadFileByPathWithPort")
	start := time.Now()

	logger.Info(
		"下载开始",
		"server_name", strings.TrimSpace(serverName),
		"port", port,
		"remote_path", strings.TrimSpace(remotePath),
		"local_path", strings.TrimSpace(localPath),
	)

	if strings.TrimSpace(localPath) == "" || strings.TrimSpace(remotePath) == "" {
		logger.Warn("下载失败：本地路径或远程路径为空")
		return SSHTransferResult{}, ErrSSHFilePathRequired
	}
	if s.PathService == nil {
		logger.Warn("下载失败：构件路径服务为空")
		return SSHTransferResult{}, ErrArtifactPathServiceNil
	}
	if s.clientFactory == nil {
		logger.Warn("下载失败：SSH 客户端工厂为空")
		return SSHTransferResult{}, ErrSSHClientFactoryNil
	}

	normalizedRemote, err := normalizeRemoteFilePath(remotePath)
	if err != nil {
		logger.Warn("下载失败：远程路径无效", "remote_path", remotePath, "error", err)
		return SSHTransferResult{}, err
	}
	normalizedLocal := filepath.Clean(strings.TrimSpace(localPath))

	server, err := s.resolveServerWithPort(serverName, port)
	if err != nil {
		logger.Error("下载失败：解析服务器配置失败", "server_name", serverName, "port", port, "error", err)
		return SSHTransferResult{}, err
	}

	client, err := s.clientFactory.New(server)
	if err != nil {
		logger.Error("下载失败：创建 SSH 客户端失败", "server_name", server.Name, "server_ip", server.IP, "error", err)
		return SSHTransferResult{}, err
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			logger.Error("下载后关闭客户端失败", "server_name", server.Name, "error", closeErr)
		}
	}()

	exists, err := client.FileExists(normalizedRemote)
	if err != nil {
		logger.Error("下载失败：获取远程文件信息失败", "remote_path", normalizedRemote, "error", err)
		return SSHTransferResult{}, err
	}
	if !exists {
		logger.Warn("下载失败：远程文件不存在", "remote_path", normalizedRemote)
		return SSHTransferResult{}, ErrRemoteArtifactNotFound
	}

	written, err := client.DownloadFile(normalizedRemote, normalizedLocal)
	if err != nil {
		logger.Error(
			"下载失败",
			"server_name", server.Name,
			"server_ip", server.IP,
			"remote_path", normalizedRemote,
			"local_path", normalizedLocal,
			"error", err,
		)
		return SSHTransferResult{}, err
	}

	result := SSHTransferResult{
		ServerName: server.Name,
		ServerIP:   server.IP,
		Direction:  "download",
		SourcePath: normalizedRemote,
		TargetPath: filepath.ToSlash(normalizedLocal),
		Bytes:      written,
		Cost:       time.Since(start),
	}

	logger.Info(
		"下载成功",
		"server_name", server.Name,
		"server_ip", server.IP,
		"port", server.Port,
		"bytes", written,
		"cost_ms", result.Cost.Milliseconds(),
		"source_path", result.SourcePath,
		"target_path", result.TargetPath,
	)
	return result, nil
}

// SearchRemoteFileInDefaultOtherRoots 在默认的other根目录中搜索远程文件
// 在weights和datasets两个目录中查找指定文件
// 参数:
//   - fileName: 要搜索的文件名
//   - serverName: 目标服务器名称
//
// 返回搜索结果，包含文件在各目录的存在状态
func (s *SSHArtifactTransferService) SearchRemoteFileInDefaultOtherRoots(fileName, serverName string) (RemoteArtifactSearchResult, error) {
	return s.SearchRemoteFileInDefaultOtherRootsWithPort(fileName, serverName, 0)
}

// SearchRemoteFileInDefaultOtherRootsWithPort 在默认的other根目录中搜索远程文件（支持端口覆盖）
// 参数:
//   - fileName: 要搜索的文件名
//   - serverName: 目标服务器名称
//   - port: SSH端口(>0时覆盖服务器默认端口)
//
// 返回搜索结果，包含文件在各目录的存在状态
func (s *SSHArtifactTransferService) SearchRemoteFileInDefaultOtherRootsWithPort(fileName, serverName string, port int) (RemoteArtifactSearchResult, error) {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "SearchRemoteFileInDefaultOtherRootsWithPort")
	start := time.Now()

	logger.Info(
		"远程文件搜索开始",
		"server_name", strings.TrimSpace(serverName),
		"port", port,
		"file_name", strings.TrimSpace(fileName),
		"weights_root", strings.TrimSpace(safeRoot(s, ArtifactCategoryWeights)),
		"datasets_root", strings.TrimSpace(safeRoot(s, ArtifactCategoryDatasets)),
	)

	if s.PathService == nil {
		logger.Warn("远程文件搜索失败：构件路径服务为空")
		return RemoteArtifactSearchResult{}, ErrArtifactPathServiceNil
	}
	if s.clientFactory == nil {
		logger.Warn("远程文件搜索失败：SSH 客户端工厂为空")
		return RemoteArtifactSearchResult{}, ErrSSHClientFactoryNil
	}

	name, err := normalizeArtifactFileName(fileName)
	if err != nil {
		logger.Warn("远程文件搜索失败：文件名无效", "file_name", fileName, "error", err)
		return RemoteArtifactSearchResult{}, err
	}

	weightsPath, err := s.PathService.BuildPath(ArtifactCategoryWeights, StorageTargetOtherLocal, name)
	if err != nil {
		logger.Error("远程文件搜索失败：构建权重路径失败", "file_name", name, "error", err)
		return RemoteArtifactSearchResult{}, err
	}
	datasetsPath, err := s.PathService.BuildPath(ArtifactCategoryDatasets, StorageTargetOtherLocal, name)
	if err != nil {
		logger.Error("远程文件搜索失败：构建数据集路径失败", "file_name", name, "error", err)
		return RemoteArtifactSearchResult{}, err
	}

	server, err := s.resolveServerWithPort(serverName, port)
	if err != nil {
		logger.Error("远程文件搜索失败：解析服务器配置失败", "server_name", serverName, "port", port, "error", err)
		return RemoteArtifactSearchResult{}, err
	}

	client, err := s.clientFactory.New(server)
	if err != nil {
		logger.Error("远程文件搜索失败：创建 SSH 客户端失败", "server_name", server.Name, "server_ip", server.IP, "error", err)
		return RemoteArtifactSearchResult{}, err
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			logger.Error("远程文件搜索后关闭客户端失败", "server_name", server.Name, "error", closeErr)
		}
	}()

	weightsExists, err := client.FileExists(weightsPath)
	if err != nil {
		logger.Error("远程文件搜索失败：获取权重路径文件信息失败", "remote_path", weightsPath, "error", err)
		return RemoteArtifactSearchResult{}, err
	}
	datasetsExists, err := client.FileExists(datasetsPath)
	if err != nil {
		logger.Error("远程文件搜索失败：获取数据集路径文件信息失败", "remote_path", datasetsPath, "error", err)
		return RemoteArtifactSearchResult{}, err
	}

	result := RemoteArtifactSearchResult{
		ServerName:       server.Name,
		ServerIP:         server.IP,
		FileName:         name,
		WeightsPath:      filepath.ToSlash(weightsPath),
		DatasetsPath:     filepath.ToSlash(datasetsPath),
		ExistsInWeights:  weightsExists,
		ExistsInDatasets: datasetsExists,
		AnyExists:        weightsExists || datasetsExists,
	}

	if weightsExists && !datasetsExists {
		result.MatchedRemotePath = result.WeightsPath
	}
	if datasetsExists && !weightsExists {
		result.MatchedRemotePath = result.DatasetsPath
	}

	logger.Info(
		"远程文件搜索成功",
		"server_name", server.Name,
		"server_ip", server.IP,
		"port", server.Port,
		"file_name", name,
		"exists_in_weights", weightsExists,
		"exists_in_datasets", datasetsExists,
		"any_exists", result.AnyExists,
		"matched_remote_path", result.MatchedRemotePath,
		"cost_ms", time.Since(start).Milliseconds(),
	)
	return result, nil
}

// UploadArtifactByName 根据文件名上传构件文件
// 自动解析文件类别并在后端找到对应文件，然后上传到远程other目录
// 参数:
//   - fileName: 构件文件名
//   - serverName: 目标服务器名称
//
// 返回传输结果和错误信息
func (s *SSHArtifactTransferService) UploadArtifactByName(fileName, serverName string) (SSHTransferResult, error) {
	return s.UploadArtifactByNameWithPort(fileName, serverName, 0)
}

// UploadArtifactByNameWithPort 根据文件名上传构件文件（支持端口覆盖）
// 参数:
//   - fileName: 构件文件名
//   - serverName: 目标服务器名称
//   - port: SSH端口(>0时覆盖服务器默认端口)
//
// 返回传输结果和错误信息
func (s *SSHArtifactTransferService) UploadArtifactByNameWithPort(fileName, serverName string, port int) (SSHTransferResult, error) {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "UploadArtifactByNameWithPort")
	start := time.Now()

	logger.Info(
		"按名称上传构件开始",
		"server_name", strings.TrimSpace(serverName),
		"port", port,
		"file_name", strings.TrimSpace(fileName),
	)

	if s.PathService == nil {
		logger.Warn("按名称上传构件失败：构件路径服务为空")
		return SSHTransferResult{}, ErrArtifactPathServiceNil
	}

	name, err := normalizeArtifactFileName(fileName)
	if err != nil {
		logger.Warn("按名称上传构件失败：文件名无效", "file_name", fileName, "error", err)
		return SSHTransferResult{}, err
	}

	localPath, category, err := s.resolveLocalBackendFile(name)
	if err != nil {
		logger.Warn("按名称上传构件失败：解析本地后端文件失败", "file_name", name, "error", err)
		return SSHTransferResult{}, err
	}

	searchResult, err := s.SearchRemoteFileInDefaultOtherRootsWithPort(name, serverName, port)
	if err != nil {
		logger.Error("按名称上传构件失败：远程文件搜索失败", "file_name", name, "server_name", serverName, "port", port, "error", err)
		return SSHTransferResult{}, err
	}

	switch category {
	case ArtifactCategoryWeights:
		if searchResult.ExistsInWeights {
			logger.Warn("按名称上传构件失败：远程权重目录已存在同名文件", "remote_path", searchResult.WeightsPath)
			return SSHTransferResult{}, ErrRemoteArtifactAlreadyExists
		}
	case ArtifactCategoryDatasets:
		if searchResult.ExistsInDatasets {
			logger.Warn("按名称上传构件失败：远程数据集目录已存在同名文件", "remote_path", searchResult.DatasetsPath)
			return SSHTransferResult{}, ErrRemoteArtifactAlreadyExists
		}
	}

	remotePath, err := s.PathService.BuildPath(category, StorageTargetOtherLocal, name)
	if err != nil {
		logger.Error("按名称上传构件失败：构建远程目标路径失败", "category", category, "file_name", name, "error", err)
		return SSHTransferResult{}, err
	}

	result, err := s.UploadFileByPathWithPort(localPath, remotePath, serverName, port)
	if err != nil {
		logger.Error("按名称上传构件失败：按路径上传失败", "error", err)
		return SSHTransferResult{}, err
	}

	result.FileName = name
	result.Category = category
	result.Cost = time.Since(start)

	logger.Info(
		"按名称上传构件成功",
		"server_name", result.ServerName,
		"server_ip", result.ServerIP,
		"port", port,
		"file_name", result.FileName,
		"category", result.Category,
		"bytes", result.Bytes,
		"source_path", result.SourcePath,
		"target_path", result.TargetPath,
		"cost_ms", result.Cost.Milliseconds(),
	)
	return result, nil
}

// DownloadArtifactByName 根据文件名下载构件文件
// 在远程other目录中搜索文件并下载到本地后端目录
// 参数:
//   - fileName: 构件文件名
//   - serverName: 源服务器名称
//
// 返回传输结果和错误信息
func (s *SSHArtifactTransferService) DownloadArtifactByName(fileName, serverName string) (SSHTransferResult, error) {
	return s.DownloadArtifactByNameWithPort(fileName, serverName, 0)
}

// DownloadArtifactByNameWithPort 根据文件名下载构件文件（支持端口覆盖）
// 参数:
//   - fileName: 构件文件名
//   - serverName: 源服务器名称
//   - port: SSH端口(>0时覆盖服务器默认端口)
//
// 返回传输结果和错误信息
func (s *SSHArtifactTransferService) DownloadArtifactByNameWithPort(fileName, serverName string, port int) (SSHTransferResult, error) {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "DownloadArtifactByNameWithPort")
	start := time.Now()

	logger.Info(
		"按名称下载构件开始",
		"server_name", strings.TrimSpace(serverName),
		"port", port,
		"file_name", strings.TrimSpace(fileName),
	)

	if s.PathService == nil {
		logger.Warn("按名称下载构件失败：构件路径服务为空")
		return SSHTransferResult{}, ErrArtifactPathServiceNil
	}

	name, err := normalizeArtifactFileName(fileName)
	if err != nil {
		logger.Warn("按名称下载构件失败：文件名无效", "file_name", fileName, "error", err)
		return SSHTransferResult{}, err
	}

	searchResult, err := s.SearchRemoteFileInDefaultOtherRootsWithPort(name, serverName, port)
	if err != nil {
		logger.Error("按名称下载构件失败：远程文件搜索失败", "file_name", name, "server_name", serverName, "port", port, "error", err)
		return SSHTransferResult{}, err
	}
	if !searchResult.AnyExists {
		logger.Warn(
			"按名称下载构件失败：远程根目录未找到文件",
			"file_name", name,
			"weights_path", searchResult.WeightsPath,
			"datasets_path", searchResult.DatasetsPath,
		)
		return SSHTransferResult{}, ErrArtifactNotFoundInRemoteOtherRoot
	}
	if searchResult.ExistsInWeights && searchResult.ExistsInDatasets {
		logger.Warn(
			"按名称下载构件失败：远程权重和数据集目录均存在同名文件",
			"file_name", name,
			"weights_path", searchResult.WeightsPath,
			"datasets_path", searchResult.DatasetsPath,
		)
		return SSHTransferResult{}, ErrArtifactConflictInRemoteRoots
	}

	category := ArtifactCategoryWeights
	remotePath := searchResult.WeightsPath
	if searchResult.ExistsInDatasets {
		category = ArtifactCategoryDatasets
		remotePath = searchResult.DatasetsPath
	}

	localPath, err := s.PathService.BuildPath(category, StorageTargetBackend, name)
	if err != nil {
		logger.Error("按名称下载构件失败：构建本地目标路径失败", "category", category, "file_name", name, "error", err)
		return SSHTransferResult{}, err
	}

	result, err := s.DownloadFileByPathWithPort(remotePath, localPath, serverName, port)
	if err != nil {
		logger.Error("按名称下载构件失败：按路径下载失败", "error", err)
		return SSHTransferResult{}, err
	}

	result.FileName = name
	result.Category = category
	result.Cost = time.Since(start)

	logger.Info(
		"按名称下载构件成功",
		"server_name", result.ServerName,
		"server_ip", result.ServerIP,
		"port", port,
		"file_name", result.FileName,
		"category", result.Category,
		"bytes", result.Bytes,
		"source_path", result.SourcePath,
		"target_path", result.TargetPath,
		"cost_ms", result.Cost.Milliseconds(),
	)
	return result, nil
}

// resolveServer 解析服务器配置
// 根据服务器名称查找对应的SSH配置信息
// 如果找不到则回退到默认配置
// 参数:
//   - serverName: 服务器名称
//
// 返回服务器配置和错误信息
func (s *SSHArtifactTransferService) resolveServer(serverName string) (SSHServerConfig, error) {
	return s.resolveServerWithPort(serverName, 0)
}

// resolveServerWithPort 解析服务器配置并支持端口覆盖
// 参数:
//   - serverName: 服务器名称
//   - port: SSH端口(>0时覆盖服务器默认端口)
//
// 返回服务器配置和错误信息
func (s *SSHArtifactTransferService) resolveServerWithPort(serverName string, port int) (SSHServerConfig, error) {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "resolveServerWithPort")

	name := strings.TrimSpace(serverName)
	if name == "" {
		logger.Warn("解析服务器配置失败：服务器名称为空")
		return SSHServerConfig{}, ErrSSHServerNameRequired
	}

	if s.serverConfigs != nil {
		if cfg, ok := s.serverConfigs[name]; ok {
			normalized, err := normalizeServerConfig(cfg)
			if err != nil {
				logger.Error("解析服务器配置失败：映射配置无效", "server_name", name, "error", err)
				return SSHServerConfig{}, err
			}
			normalized.Name = name
			if port > 0 {
				normalized.Port = port
			}
			if normalized.Port <= 0 || normalized.Port > 65535 {
				logger.Error("解析服务器配置失败：端口无效", "server_name", name, "port", normalized.Port)
				return SSHServerConfig{}, ErrSSHServerPortInvalid
			}
			logger.Info(
				"从静态映射解析服务器配置",
				"server_name", name,
				"server_ip", normalized.IP,
				"port", normalized.Port,
				"user", normalized.User,
				"private_key_path", normalized.PrivateKeyPath,
			)
			return normalized, nil
		}
	}

	fallback, err := normalizeServerConfig(s.defaultServer)
	if err != nil {
		logger.Error("解析服务器配置失败：默认配置无效", "error", err)
		return SSHServerConfig{}, err
	}
	fallback.Name = name
	if port > 0 {
		fallback.Port = port
	}
	if fallback.Port <= 0 || fallback.Port > 65535 {
		logger.Error("解析服务器配置失败：端口无效", "server_name", name, "port", fallback.Port)
		return SSHServerConfig{}, ErrSSHServerPortInvalid
	}
	logger.Warn(
		"静态映射中未找到服务器，使用默认静态IP",
		"server_name", name,
		"server_ip", fallback.IP,
		"port", fallback.Port,
		"user", fallback.User,
		"private_key_path", fallback.PrivateKeyPath,
	)
	return fallback, nil
}

// normalizeServerConfig 标准化SSH服务器配置
// 清理字符串字段，设置默认值，验证必需字段
// 参数:
//   - cfg: 原始服务器配置
//
// 返回标准化后的配置和错误信息
func normalizeServerConfig(cfg SSHServerConfig) (SSHServerConfig, error) {
	normalized := cfg
	normalized.IP = strings.TrimSpace(normalized.IP)
	normalized.User = strings.TrimSpace(normalized.User)
	normalized.PrivateKeyPath = strings.TrimSpace(normalized.PrivateKeyPath)
	if normalized.Port == 0 {
		normalized.Port = DefaultSSHServerPort
	}
	if normalized.Port <= 0 || normalized.Port > 65535 {
		return SSHServerConfig{}, ErrSSHServerPortInvalid
	}
	if normalized.Timeout <= 0 {
		normalized.Timeout = defaultSSHTimeout
	}
	if normalized.IP == "" {
		return SSHServerConfig{}, ErrSSHServerIPRequired
	}
	if normalized.User == "" {
		return SSHServerConfig{}, ErrSSHServerUserRequired
	}
	if normalized.PrivateKeyPath == "" {
		return SSHServerConfig{}, ErrSSHPrivateKeyPathRequired
	}
	return normalized, nil
}

// resolveLocalBackendFile 解析本地后端文件路径
// 在weights和datasets目录中查找指定文件，确定其完整路径和类别
// 参数:
//   - fileName: 文件名
//
// 返回文件完整路径、文件类别和错误信息
func (s *SSHArtifactTransferService) resolveLocalBackendFile(fileName string) (string, string, error) {
	weightsPath, err := s.PathService.BuildPath(ArtifactCategoryWeights, StorageTargetBackend, fileName)
	if err != nil {
		return "", "", err
	}
	datasetsPath, err := s.PathService.BuildPath(ArtifactCategoryDatasets, StorageTargetBackend, fileName)
	if err != nil {
		return "", "", err
	}

	weightsExists, err := localRegularFileExists(weightsPath)
	if err != nil {
		return "", "", err
	}
	datasetsExists, err := localRegularFileExists(datasetsPath)
	if err != nil {
		return "", "", err
	}

	switch {
	case weightsExists && datasetsExists:
		return "", "", ErrArtifactConflictInBackendRoots
	case weightsExists:
		return filepath.Clean(weightsPath), ArtifactCategoryWeights, nil
	case datasetsExists:
		return filepath.Clean(datasetsPath), ArtifactCategoryDatasets, nil
	default:
		return "", "", ErrArtifactNotFoundInBackendRoots
	}
}

// localRegularFileExists 检查本地常规文件是否存在
// 验证指定路径是否为存在的常规文件
// 参数:
//   - filePath: 文件路径
//
// 返回是否存在以及错误信息
func localRegularFileExists(filePath string) (bool, error) {
	info, err := os.Stat(filepath.Clean(filePath))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, ErrLocalSourcePathNotRegularFile
	}
	return true, nil
}

// safeRoot 安全获取存储根目录路径
// 防止空指针异常的安全包装函数
// 参数:
//   - s: SSHArtifactTransferService实例
//   - category: 存储类别
//
// 返回根目录路径，失败时返回空字符串
func safeRoot(s *SSHArtifactTransferService, category string) string {
	if s == nil || s.PathService == nil {
		return ""
	}
	root, err := s.PathService.ResolveRoot(category, StorageTargetOtherLocal)
	if err != nil {
		return ""
	}
	return root
}

// normalizeRemoteFilePath 标准化远程文件路径
// 清理路径字符串，转换反斜杠，确保绝对路径格式
// 参数:
//   - rawPath: 原始路径字符串
//
// 返回标准化后的路径和错误信息
func normalizeRemoteFilePath(rawPath string) (string, error) {
	value := strings.TrimSpace(strings.ReplaceAll(rawPath, "\\", "/"))
	if value == "" {
		return "", ErrSSHFilePathRequired
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = path.Clean(value)
	if value == "/" || value == "." {
		return "", ErrSSHFilePathRequired
	}
	return value, nil
}

// sshSFTPClientFactory SFTP客户端工厂实现
// 负责创建SSH SFTP客户端连接
type sshSFTPClientFactory struct{}

// New 创建新的远程文件客户端
// 实现remoteFileClientFactory接口
// 参数:
//   - server: SSH服务器配置
//
// 返回远程文件客户端和错误信息
func (f *sshSFTPClientFactory) New(server SSHServerConfig) (remoteFileClient, error) {
	return newSSHSFTPClient(server)
}

// sshSFTPClient SSH SFTP客户端实现
// 封装SSH连接和SFTP客户端功能
type sshSFTPClient struct {
	server     SSHServerConfig // 服务器配置
	sshClient  *ssh.Client     // SSH客户端连接
	sftpClient *sftp.Client    // SFTP客户端
}

// newSSHSFTPClient 创建新的SSH SFTP客户端
// 建立SSH连接并初始化SFTP客户端
// 参数:
//   - server: SSH服务器配置
//
// 返回SSH SFTP客户端和错误信息
func newSSHSFTPClient(server SSHServerConfig) (*sshSFTPClient, error) {
	normalized, err := normalizeServerConfig(server)
	if err != nil {
		return nil, err
	}

	keyBytes, err := os.ReadFile(normalized.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("读取私钥失败: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %w", err)
	}

	clientConfig := &ssh.ClientConfig{
		User: normalized.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         normalized.Timeout,
	}

	address := net.JoinHostPort(normalized.IP, strconv.Itoa(normalized.Port))
	sshClient, err := ssh.Dial("tcp", address, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("建立 SSH 连接失败: %w", err)
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, fmt.Errorf("创建 SFTP 客户端失败: %w", err)
	}

	return &sshSFTPClient{
		server:     normalized,
		sshClient:  sshClient,
		sftpClient: sftpClient,
	}, nil
}

// UploadFile 上传本地文件到远程服务器
// 实现remoteFileClient接口的文件上传功能
// 参数:
//   - localPath: 本地源文件路径
//   - remotePath: 远程目标文件路径
//
// 返回传输的字节数和错误信息
func (c *sshSFTPClient) UploadFile(localPath, remotePath string) (int64, error) {
	normalizedRemote, err := normalizeRemoteFilePath(remotePath)
	if err != nil {
		return 0, err
	}

	src, err := os.Open(filepath.Clean(localPath))
	if err != nil {
		return 0, fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return 0, fmt.Errorf("获取本地文件信息失败: %w", err)
	}
	if !info.Mode().IsRegular() {
		return 0, ErrLocalSourcePathNotRegularFile
	}

	remoteDir := path.Dir(normalizedRemote)
	if err := c.sftpClient.MkdirAll(remoteDir); err != nil {
		return 0, fmt.Errorf("创建远端目录失败: %w", err)
	}

	dst, err := c.sftpClient.Create(normalizedRemote)
	if err != nil {
		return 0, fmt.Errorf("创建远端文件失败: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, src)
	if err != nil {
		return 0, fmt.Errorf("写入远端文件失败: %w", err)
	}

	return written, nil
}

// DownloadFile 从远程服务器下载文件到本地
// 实现remoteFileClient接口的文件下载功能
// 参数:
//   - remotePath: 远程源文件路径
//   - localPath: 本地目标文件路径
//
// 返回传输的字节数和错误信息
func (c *sshSFTPClient) DownloadFile(remotePath, localPath string) (int64, error) {
	normalizedRemote, err := normalizeRemoteFilePath(remotePath)
	if err != nil {
		return 0, err
	}
	normalizedLocal := filepath.Clean(strings.TrimSpace(localPath))
	if normalizedLocal == "" {
		return 0, ErrSSHFilePathRequired
	}

	src, err := c.sftpClient.Open(normalizedRemote)
	if err != nil {
		if isNotExistError(err) {
			return 0, ErrRemoteArtifactNotFound
		}
		return 0, fmt.Errorf("打开远端文件失败: %w", err)
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(normalizedLocal), 0o755); err != nil {
		return 0, fmt.Errorf("创建本地目录失败: %w", err)
	}

	dst, err := os.Create(normalizedLocal)
	if err != nil {
		return 0, fmt.Errorf("创建本地文件失败: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, src)
	if err != nil {
		return 0, fmt.Errorf("写入本地文件失败: %w", err)
	}

	return written, nil
}

// FileExists 检查远程文件是否存在
// 实现remoteFileClient接口的文件存在性检查功能
// 参数:
//   - remotePath: 远程文件路径
//
// 返回文件是否存在和错误信息
func (c *sshSFTPClient) FileExists(remotePath string) (bool, error) {
	normalizedRemote, err := normalizeRemoteFilePath(remotePath)
	if err != nil {
		return false, err
	}

	_, err = c.sftpClient.Stat(normalizedRemote)
	if err != nil {
		if isNotExistError(err) {
			return false, nil
		}
		return false, fmt.Errorf("获取远端文件信息失败: %w", err)
	}
	return true, nil
}

// Close 关闭SSH和SFTP客户端连接
// 实现remoteFileClient接口的资源清理功能
// 返回关闭过程中可能发生的第一个错误
func (c *sshSFTPClient) Close() error {
	var firstErr error
	if c.sftpClient != nil {
		if err := c.sftpClient.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// isNotExistError 判断错误是否表示文件不存在
// 统一处理不同形式的"文件不存在"错误
// 参数:
//   - err: 要检查的错误
//
// 返回是否为文件不存在错误
func isNotExistError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) || os.IsNotExist(err) {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "not exist") || strings.Contains(message, "no such file")
}
