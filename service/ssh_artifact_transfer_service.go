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
	DefaultStaticServerIP = "192.168.1.100"
	DefaultSSHServerPort  = 22
	DefaultSSHServerUser  = "root"
)

var (
	ErrSSHClientFactoryNil               = errors.New("ssh client factory is nil")
	ErrSSHServerNameRequired             = errors.New("server name is required")
	ErrSSHServerIPRequired               = errors.New("server ip is required")
	ErrSSHServerUserRequired             = errors.New("ssh server user is required")
	ErrSSHPrivateKeyPathRequired         = errors.New("ssh private key path is required")
	ErrSSHFilePathRequired               = errors.New("file path is required")
	ErrLocalSourceFileNotFound           = errors.New("local source file not found")
	ErrLocalSourcePathNotRegularFile     = errors.New("local source path is not a regular file")
	ErrRemoteArtifactNotFound            = errors.New("remote artifact not found")
	ErrRemoteArtifactAlreadyExists       = errors.New("remote artifact already exists")
	ErrArtifactNotFoundInBackendRoots    = errors.New("artifact not found in backend roots")
	ErrArtifactNotFoundInRemoteOtherRoot = errors.New("artifact not found in remote other roots")
	ErrArtifactConflictInBackendRoots    = errors.New("artifact exists in both backend roots")
	ErrArtifactConflictInRemoteRoots     = errors.New("artifact exists in both remote roots")
)

var (
	defaultSSHTimeout = 15 * time.Second
)

type SSHServerConfig struct {
	Name           string
	IP             string
	Port           int
	User           string
	PrivateKeyPath string
	Timeout        time.Duration
}

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

type remoteFileClient interface {
	UploadFile(localPath, remotePath string) (int64, error)
	DownloadFile(remotePath, localPath string) (int64, error)
	FileExists(remotePath string) (bool, error)
	Close() error
}

type remoteFileClientFactory interface {
	New(server SSHServerConfig) (remoteFileClient, error)
}

type SSHArtifactTransferService struct {
	PathService   *ArtifactPathService
	serverConfigs map[string]SSHServerConfig
	defaultServer SSHServerConfig
	clientFactory remoteFileClientFactory
}

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

func (s *SSHArtifactTransferService) SetServerConfig(serverName string, cfg SSHServerConfig) error {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "SetServerConfig")

	name := strings.TrimSpace(serverName)
	if name == "" {
		logger.Warn("set server config failed: server name is empty")
		return ErrSSHServerNameRequired
	}

	normalized, err := normalizeServerConfig(cfg)
	if err != nil {
		logger.Error("set server config failed: invalid config", "server_name", name, "error", err)
		return err
	}
	normalized.Name = name

	if s.serverConfigs == nil {
		s.serverConfigs = make(map[string]SSHServerConfig)
	}
	s.serverConfigs[name] = normalized
	logger.Info(
		"set server config success",
		"server_name", name,
		"server_ip", normalized.IP,
		"port", normalized.Port,
		"user", normalized.User,
		"private_key_path", normalized.PrivateKeyPath,
	)
	return nil
}

func (s *SSHArtifactTransferService) UploadFileByPath(localPath, remotePath, serverName string) (SSHTransferResult, error) {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "UploadFileByPath")
	start := time.Now()

	logger.Info(
		"upload begin",
		"server_name", strings.TrimSpace(serverName),
		"local_path", strings.TrimSpace(localPath),
		"remote_path", strings.TrimSpace(remotePath),
	)

	if strings.TrimSpace(localPath) == "" || strings.TrimSpace(remotePath) == "" {
		logger.Warn("upload failed: local path or remote path is empty")
		return SSHTransferResult{}, ErrSSHFilePathRequired
	}
	if s.PathService == nil {
		logger.Warn("upload failed: artifact path service is nil")
		return SSHTransferResult{}, ErrArtifactPathServiceNil
	}
	if s.clientFactory == nil {
		logger.Warn("upload failed: ssh client factory is nil")
		return SSHTransferResult{}, ErrSSHClientFactoryNil
	}

	normalizedLocal := filepath.Clean(strings.TrimSpace(localPath))
	normalizedRemote, err := normalizeRemoteFilePath(remotePath)
	if err != nil {
		logger.Warn("upload failed: invalid remote path", "remote_path", remotePath, "error", err)
		return SSHTransferResult{}, err
	}

	info, err := os.Stat(normalizedLocal)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn("upload failed: local source file not found", "local_path", normalizedLocal)
			return SSHTransferResult{}, ErrLocalSourceFileNotFound
		}
		logger.Error("upload failed: stat local source file failed", "local_path", normalizedLocal, "error", err)
		return SSHTransferResult{}, fmt.Errorf("stat local source file failed: %w", err)
	}
	if !info.Mode().IsRegular() {
		logger.Warn("upload failed: local source path is not regular file", "local_path", normalizedLocal, "mode", info.Mode().String())
		return SSHTransferResult{}, ErrLocalSourcePathNotRegularFile
	}

	server, err := s.resolveServer(serverName)
	if err != nil {
		logger.Error("upload failed: resolve server failed", "server_name", serverName, "error", err)
		return SSHTransferResult{}, err
	}

	client, err := s.clientFactory.New(server)
	if err != nil {
		logger.Error("upload failed: create ssh client failed", "server_name", server.Name, "server_ip", server.IP, "error", err)
		return SSHTransferResult{}, err
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			logger.Error("upload close client failed", "server_name", server.Name, "error", closeErr)
		}
	}()

	written, err := client.UploadFile(normalizedLocal, normalizedRemote)
	if err != nil {
		logger.Error(
			"upload failed",
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
		"upload success",
		"server_name", server.Name,
		"server_ip", server.IP,
		"bytes", written,
		"cost_ms", result.Cost.Milliseconds(),
		"source_path", result.SourcePath,
		"target_path", result.TargetPath,
	)
	return result, nil
}

func (s *SSHArtifactTransferService) DownloadFileByPath(remotePath, localPath, serverName string) (SSHTransferResult, error) {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "DownloadFileByPath")
	start := time.Now()

	logger.Info(
		"download begin",
		"server_name", strings.TrimSpace(serverName),
		"remote_path", strings.TrimSpace(remotePath),
		"local_path", strings.TrimSpace(localPath),
	)

	if strings.TrimSpace(localPath) == "" || strings.TrimSpace(remotePath) == "" {
		logger.Warn("download failed: local path or remote path is empty")
		return SSHTransferResult{}, ErrSSHFilePathRequired
	}
	if s.PathService == nil {
		logger.Warn("download failed: artifact path service is nil")
		return SSHTransferResult{}, ErrArtifactPathServiceNil
	}
	if s.clientFactory == nil {
		logger.Warn("download failed: ssh client factory is nil")
		return SSHTransferResult{}, ErrSSHClientFactoryNil
	}

	normalizedRemote, err := normalizeRemoteFilePath(remotePath)
	if err != nil {
		logger.Warn("download failed: invalid remote path", "remote_path", remotePath, "error", err)
		return SSHTransferResult{}, err
	}
	normalizedLocal := filepath.Clean(strings.TrimSpace(localPath))

	server, err := s.resolveServer(serverName)
	if err != nil {
		logger.Error("download failed: resolve server failed", "server_name", serverName, "error", err)
		return SSHTransferResult{}, err
	}

	client, err := s.clientFactory.New(server)
	if err != nil {
		logger.Error("download failed: create ssh client failed", "server_name", server.Name, "server_ip", server.IP, "error", err)
		return SSHTransferResult{}, err
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			logger.Error("download close client failed", "server_name", server.Name, "error", closeErr)
		}
	}()

	exists, err := client.FileExists(normalizedRemote)
	if err != nil {
		logger.Error("download failed: stat remote file failed", "remote_path", normalizedRemote, "error", err)
		return SSHTransferResult{}, err
	}
	if !exists {
		logger.Warn("download failed: remote file not found", "remote_path", normalizedRemote)
		return SSHTransferResult{}, ErrRemoteArtifactNotFound
	}

	written, err := client.DownloadFile(normalizedRemote, normalizedLocal)
	if err != nil {
		logger.Error(
			"download failed",
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
		"download success",
		"server_name", server.Name,
		"server_ip", server.IP,
		"bytes", written,
		"cost_ms", result.Cost.Milliseconds(),
		"source_path", result.SourcePath,
		"target_path", result.TargetPath,
	)
	return result, nil
}

func (s *SSHArtifactTransferService) SearchRemoteFileInDefaultOtherRoots(fileName, serverName string) (RemoteArtifactSearchResult, error) {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "SearchRemoteFileInDefaultOtherRoots")
	start := time.Now()

	logger.Info(
		"search remote file begin",
		"server_name", strings.TrimSpace(serverName),
		"file_name", strings.TrimSpace(fileName),
		"weights_root", strings.TrimSpace(safeRoot(s, ArtifactCategoryWeights)),
		"datasets_root", strings.TrimSpace(safeRoot(s, ArtifactCategoryDatasets)),
	)

	if s.PathService == nil {
		logger.Warn("search remote file failed: artifact path service is nil")
		return RemoteArtifactSearchResult{}, ErrArtifactPathServiceNil
	}
	if s.clientFactory == nil {
		logger.Warn("search remote file failed: ssh client factory is nil")
		return RemoteArtifactSearchResult{}, ErrSSHClientFactoryNil
	}

	name, err := normalizeArtifactFileName(fileName)
	if err != nil {
		logger.Warn("search remote file failed: invalid file name", "file_name", fileName, "error", err)
		return RemoteArtifactSearchResult{}, err
	}

	weightsPath, err := s.PathService.BuildPath(ArtifactCategoryWeights, StorageTargetOtherLocal, name)
	if err != nil {
		logger.Error("search remote file failed: build weights path failed", "file_name", name, "error", err)
		return RemoteArtifactSearchResult{}, err
	}
	datasetsPath, err := s.PathService.BuildPath(ArtifactCategoryDatasets, StorageTargetOtherLocal, name)
	if err != nil {
		logger.Error("search remote file failed: build datasets path failed", "file_name", name, "error", err)
		return RemoteArtifactSearchResult{}, err
	}

	server, err := s.resolveServer(serverName)
	if err != nil {
		logger.Error("search remote file failed: resolve server failed", "server_name", serverName, "error", err)
		return RemoteArtifactSearchResult{}, err
	}

	client, err := s.clientFactory.New(server)
	if err != nil {
		logger.Error("search remote file failed: create ssh client failed", "server_name", server.Name, "server_ip", server.IP, "error", err)
		return RemoteArtifactSearchResult{}, err
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			logger.Error("search remote file close client failed", "server_name", server.Name, "error", closeErr)
		}
	}()

	weightsExists, err := client.FileExists(weightsPath)
	if err != nil {
		logger.Error("search remote file failed: stat weights path failed", "remote_path", weightsPath, "error", err)
		return RemoteArtifactSearchResult{}, err
	}
	datasetsExists, err := client.FileExists(datasetsPath)
	if err != nil {
		logger.Error("search remote file failed: stat datasets path failed", "remote_path", datasetsPath, "error", err)
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
		"search remote file success",
		"server_name", server.Name,
		"server_ip", server.IP,
		"file_name", name,
		"exists_in_weights", weightsExists,
		"exists_in_datasets", datasetsExists,
		"any_exists", result.AnyExists,
		"matched_remote_path", result.MatchedRemotePath,
		"cost_ms", time.Since(start).Milliseconds(),
	)
	return result, nil
}

func (s *SSHArtifactTransferService) UploadArtifactByName(fileName, serverName string) (SSHTransferResult, error) {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "UploadArtifactByName")
	start := time.Now()

	logger.Info("upload artifact by name begin", "server_name", strings.TrimSpace(serverName), "file_name", strings.TrimSpace(fileName))

	if s.PathService == nil {
		logger.Warn("upload artifact by name failed: artifact path service is nil")
		return SSHTransferResult{}, ErrArtifactPathServiceNil
	}

	name, err := normalizeArtifactFileName(fileName)
	if err != nil {
		logger.Warn("upload artifact by name failed: invalid file name", "file_name", fileName, "error", err)
		return SSHTransferResult{}, err
	}

	localPath, category, err := s.resolveLocalBackendFile(name)
	if err != nil {
		logger.Warn("upload artifact by name failed: resolve local backend file failed", "file_name", name, "error", err)
		return SSHTransferResult{}, err
	}

	searchResult, err := s.SearchRemoteFileInDefaultOtherRoots(name, serverName)
	if err != nil {
		logger.Error("upload artifact by name failed: search remote file failed", "file_name", name, "server_name", serverName, "error", err)
		return SSHTransferResult{}, err
	}

	switch category {
	case ArtifactCategoryWeights:
		if searchResult.ExistsInWeights {
			logger.Warn("upload artifact by name failed: remote file already exists in weights", "remote_path", searchResult.WeightsPath)
			return SSHTransferResult{}, ErrRemoteArtifactAlreadyExists
		}
	case ArtifactCategoryDatasets:
		if searchResult.ExistsInDatasets {
			logger.Warn("upload artifact by name failed: remote file already exists in datasets", "remote_path", searchResult.DatasetsPath)
			return SSHTransferResult{}, ErrRemoteArtifactAlreadyExists
		}
	}

	remotePath, err := s.PathService.BuildPath(category, StorageTargetOtherLocal, name)
	if err != nil {
		logger.Error("upload artifact by name failed: build remote target path failed", "category", category, "file_name", name, "error", err)
		return SSHTransferResult{}, err
	}

	result, err := s.UploadFileByPath(localPath, remotePath, serverName)
	if err != nil {
		logger.Error("upload artifact by name failed: upload by path failed", "error", err)
		return SSHTransferResult{}, err
	}

	result.FileName = name
	result.Category = category
	result.Cost = time.Since(start)

	logger.Info(
		"upload artifact by name success",
		"server_name", result.ServerName,
		"server_ip", result.ServerIP,
		"file_name", result.FileName,
		"category", result.Category,
		"bytes", result.Bytes,
		"source_path", result.SourcePath,
		"target_path", result.TargetPath,
		"cost_ms", result.Cost.Milliseconds(),
	)
	return result, nil
}

func (s *SSHArtifactTransferService) DownloadArtifactByName(fileName, serverName string) (SSHTransferResult, error) {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "DownloadArtifactByName")
	start := time.Now()

	logger.Info("download artifact by name begin", "server_name", strings.TrimSpace(serverName), "file_name", strings.TrimSpace(fileName))

	if s.PathService == nil {
		logger.Warn("download artifact by name failed: artifact path service is nil")
		return SSHTransferResult{}, ErrArtifactPathServiceNil
	}

	name, err := normalizeArtifactFileName(fileName)
	if err != nil {
		logger.Warn("download artifact by name failed: invalid file name", "file_name", fileName, "error", err)
		return SSHTransferResult{}, err
	}

	searchResult, err := s.SearchRemoteFileInDefaultOtherRoots(name, serverName)
	if err != nil {
		logger.Error("download artifact by name failed: search remote file failed", "file_name", name, "server_name", serverName, "error", err)
		return SSHTransferResult{}, err
	}
	if !searchResult.AnyExists {
		logger.Warn(
			"download artifact by name failed: file not found in remote roots",
			"file_name", name,
			"weights_path", searchResult.WeightsPath,
			"datasets_path", searchResult.DatasetsPath,
		)
		return SSHTransferResult{}, ErrArtifactNotFoundInRemoteOtherRoot
	}
	if searchResult.ExistsInWeights && searchResult.ExistsInDatasets {
		logger.Warn(
			"download artifact by name failed: file exists in both remote roots",
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
		logger.Error("download artifact by name failed: build local target path failed", "category", category, "file_name", name, "error", err)
		return SSHTransferResult{}, err
	}

	result, err := s.DownloadFileByPath(remotePath, localPath, serverName)
	if err != nil {
		logger.Error("download artifact by name failed: download by path failed", "error", err)
		return SSHTransferResult{}, err
	}

	result.FileName = name
	result.Category = category
	result.Cost = time.Since(start)

	logger.Info(
		"download artifact by name success",
		"server_name", result.ServerName,
		"server_ip", result.ServerIP,
		"file_name", result.FileName,
		"category", result.Category,
		"bytes", result.Bytes,
		"source_path", result.SourcePath,
		"target_path", result.TargetPath,
		"cost_ms", result.Cost.Milliseconds(),
	)
	return result, nil
}

func (s *SSHArtifactTransferService) resolveServer(serverName string) (SSHServerConfig, error) {
	logger := serviceLogger().With("service", "SSHArtifactTransferService", "method", "resolveServer")

	name := strings.TrimSpace(serverName)
	if name == "" {
		logger.Warn("resolve server failed: server name is empty")
		return SSHServerConfig{}, ErrSSHServerNameRequired
	}

	if s.serverConfigs != nil {
		if cfg, ok := s.serverConfigs[name]; ok {
			normalized, err := normalizeServerConfig(cfg)
			if err != nil {
				logger.Error("resolve server failed: invalid mapped config", "server_name", name, "error", err)
				return SSHServerConfig{}, err
			}
			normalized.Name = name
			logger.Info(
				"resolve server from static mapping",
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
		logger.Error("resolve server failed: invalid default server config", "error", err)
		return SSHServerConfig{}, err
	}
	fallback.Name = name
	logger.Warn(
		"server not found in static mapping, use default static ip",
		"server_name", name,
		"server_ip", fallback.IP,
		"port", fallback.Port,
		"user", fallback.User,
		"private_key_path", fallback.PrivateKeyPath,
	)
	return fallback, nil
}

func normalizeServerConfig(cfg SSHServerConfig) (SSHServerConfig, error) {
	normalized := cfg
	normalized.IP = strings.TrimSpace(normalized.IP)
	normalized.User = strings.TrimSpace(normalized.User)
	normalized.PrivateKeyPath = strings.TrimSpace(normalized.PrivateKeyPath)
	if normalized.Port == 0 {
		normalized.Port = DefaultSSHServerPort
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

type sshSFTPClientFactory struct{}

func (f *sshSFTPClientFactory) New(server SSHServerConfig) (remoteFileClient, error) {
	return newSSHSFTPClient(server)
}

type sshSFTPClient struct {
	server     SSHServerConfig
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

func newSSHSFTPClient(server SSHServerConfig) (*sshSFTPClient, error) {
	normalized, err := normalizeServerConfig(server)
	if err != nil {
		return nil, err
	}

	keyBytes, err := os.ReadFile(normalized.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key failed: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key failed: %w", err)
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
		return nil, fmt.Errorf("dial ssh failed: %w", err)
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, fmt.Errorf("create sftp client failed: %w", err)
	}

	return &sshSFTPClient{
		server:     normalized,
		sshClient:  sshClient,
		sftpClient: sftpClient,
	}, nil
}

func (c *sshSFTPClient) UploadFile(localPath, remotePath string) (int64, error) {
	normalizedRemote, err := normalizeRemoteFilePath(remotePath)
	if err != nil {
		return 0, err
	}

	src, err := os.Open(filepath.Clean(localPath))
	if err != nil {
		return 0, fmt.Errorf("open local file failed: %w", err)
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat local file failed: %w", err)
	}
	if !info.Mode().IsRegular() {
		return 0, ErrLocalSourcePathNotRegularFile
	}

	remoteDir := path.Dir(normalizedRemote)
	if err := c.sftpClient.MkdirAll(remoteDir); err != nil {
		return 0, fmt.Errorf("create remote directory failed: %w", err)
	}

	dst, err := c.sftpClient.Create(normalizedRemote)
	if err != nil {
		return 0, fmt.Errorf("create remote file failed: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, src)
	if err != nil {
		return 0, fmt.Errorf("write remote file failed: %w", err)
	}

	return written, nil
}

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
		return 0, fmt.Errorf("open remote file failed: %w", err)
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(normalizedLocal), 0o755); err != nil {
		return 0, fmt.Errorf("create local directory failed: %w", err)
	}

	dst, err := os.Create(normalizedLocal)
	if err != nil {
		return 0, fmt.Errorf("create local file failed: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, src)
	if err != nil {
		return 0, fmt.Errorf("write local file failed: %w", err)
	}

	return written, nil
}

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
		return false, fmt.Errorf("stat remote file failed: %w", err)
	}
	return true, nil
}

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
