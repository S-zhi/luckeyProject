package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeRemoteFileClientFactory struct {
	client      *fakeRemoteFileClient
	serverCalls []SSHServerConfig
	newErr      error
}

func (f *fakeRemoteFileClientFactory) New(server SSHServerConfig) (remoteFileClient, error) {
	if f.newErr != nil {
		return nil, f.newErr
	}
	f.serverCalls = append(f.serverCalls, server)
	return f.client, nil
}

type fakeRemoteFileClient struct {
	remoteFiles    map[string][]byte
	uploadErr      error
	downloadErr    error
	existsErr      error
	closeErr       error
	uploadedPaths  []string
	downloadedPath []string
}

func (f *fakeRemoteFileClient) UploadFile(localPath, remotePath string) (int64, error) {
	if f.uploadErr != nil {
		return 0, f.uploadErr
	}

	content, err := os.ReadFile(localPath)
	if err != nil {
		return 0, err
	}
	if f.remoteFiles == nil {
		f.remoteFiles = make(map[string][]byte)
	}
	f.remoteFiles[remotePath] = content
	f.uploadedPaths = append(f.uploadedPaths, remotePath)
	return int64(len(content)), nil
}

func (f *fakeRemoteFileClient) DownloadFile(remotePath, localPath string) (int64, error) {
	if f.downloadErr != nil {
		return 0, f.downloadErr
	}
	content, ok := f.remoteFiles[remotePath]
	if !ok {
		return 0, ErrRemoteArtifactNotFound
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(localPath, content, 0o644); err != nil {
		return 0, err
	}
	f.downloadedPath = append(f.downloadedPath, remotePath)
	return int64(len(content)), nil
}

func (f *fakeRemoteFileClient) FileExists(remotePath string) (bool, error) {
	if f.existsErr != nil {
		return false, f.existsErr
	}
	_, ok := f.remoteFiles[remotePath]
	return ok, nil
}

func (f *fakeRemoteFileClient) Close() error {
	return f.closeErr
}

func TestSSHArtifactTransferServiceUploadArtifactByNameWeights(t *testing.T) {
	tmpDir := t.TempDir()
	pathService := &ArtifactPathService{
		BackendWeightsRoot:  filepath.Join(tmpDir, "backend", "weights"),
		BackendDatasetsRoot: filepath.Join(tmpDir, "backend", "datasets"),
		BaiduWeightsRoot:    filepath.Join(tmpDir, "baidu", "weights"),
		BaiduDatasetsRoot:   filepath.Join(tmpDir, "baidu", "datasets"),
		OtherWeightsRoot:    "/project/luckyProject/weights",
		OtherDatasetsRoot:   "/project/luckyProject/datasets",
	}

	localFile := filepath.Join(pathService.BackendWeightsRoot, "demo.pt")
	err := os.MkdirAll(filepath.Dir(localFile), 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(localFile, []byte("model-content"), 0o644)
	assert.NoError(t, err)

	client := &fakeRemoteFileClient{
		remoteFiles: map[string][]byte{},
	}
	factory := &fakeRemoteFileClientFactory{client: client}

	svc := &SSHArtifactTransferService{
		PathService: pathService,
		serverConfigs: map[string]SSHServerConfig{
			"dev-server": {
				Name:           "dev-server",
				IP:             "10.0.0.7",
				Port:           22,
				User:           "root",
				PrivateKeyPath: "/tmp/id_rsa",
				Timeout:        10 * time.Second,
			},
		},
		defaultServer: SSHServerConfig{
			IP:             "192.168.1.100",
			Port:           22,
			User:           "root",
			PrivateKeyPath: "/tmp/id_rsa",
			Timeout:        10 * time.Second,
		},
		clientFactory: factory,
	}

	result, err := svc.UploadArtifactByName("demo.pt", "dev-server")
	assert.NoError(t, err)
	assert.Equal(t, "upload", result.Direction)
	assert.Equal(t, ArtifactCategoryWeights, result.Category)
	assert.Equal(t, "demo.pt", result.FileName)
	assert.Equal(t, "/project/luckyProject/weights/demo.pt", result.TargetPath)
	assert.Equal(t, "dev-server", result.ServerName)
	assert.Equal(t, "10.0.0.7", result.ServerIP)
	assert.EqualValues(t, len("model-content"), result.Bytes)

	uploadedContent, ok := client.remoteFiles["/project/luckyProject/weights/demo.pt"]
	assert.True(t, ok)
	assert.Equal(t, "model-content", string(uploadedContent))
	assert.Len(t, factory.serverCalls, 2)
}

func TestSSHArtifactTransferServiceDownloadArtifactByNameDatasets(t *testing.T) {
	tmpDir := t.TempDir()
	pathService := &ArtifactPathService{
		BackendWeightsRoot:  filepath.Join(tmpDir, "backend", "weights"),
		BackendDatasetsRoot: filepath.Join(tmpDir, "backend", "datasets"),
		BaiduWeightsRoot:    filepath.Join(tmpDir, "baidu", "weights"),
		BaiduDatasetsRoot:   filepath.Join(tmpDir, "baidu", "datasets"),
		OtherWeightsRoot:    "/project/luckyProject/weights",
		OtherDatasetsRoot:   "/project/luckyProject/datasets",
	}

	client := &fakeRemoteFileClient{
		remoteFiles: map[string][]byte{
			"/project/luckyProject/datasets/train.zip": []byte("dataset-content"),
		},
	}
	factory := &fakeRemoteFileClientFactory{client: client}

	svc := &SSHArtifactTransferService{
		PathService: pathService,
		serverConfigs: map[string]SSHServerConfig{
			"dev-server": {
				Name:           "dev-server",
				IP:             "10.0.0.8",
				Port:           22,
				User:           "root",
				PrivateKeyPath: "/tmp/id_rsa",
				Timeout:        10 * time.Second,
			},
		},
		defaultServer: SSHServerConfig{
			IP:             "192.168.1.100",
			Port:           22,
			User:           "root",
			PrivateKeyPath: "/tmp/id_rsa",
			Timeout:        10 * time.Second,
		},
		clientFactory: factory,
	}

	result, err := svc.DownloadArtifactByName("train.zip", "dev-server")
	assert.NoError(t, err)
	assert.Equal(t, "download", result.Direction)
	assert.Equal(t, ArtifactCategoryDatasets, result.Category)
	assert.Equal(t, "train.zip", result.FileName)
	assert.Equal(t, "/project/luckyProject/datasets/train.zip", result.SourcePath)
	assert.Equal(t, filepath.ToSlash(filepath.Join(pathService.BackendDatasetsRoot, "train.zip")), result.TargetPath)
	assert.EqualValues(t, len("dataset-content"), result.Bytes)

	localContent, err := os.ReadFile(filepath.Join(pathService.BackendDatasetsRoot, "train.zip"))
	assert.NoError(t, err)
	assert.Equal(t, "dataset-content", string(localContent))
	assert.Len(t, factory.serverCalls, 2)
}

func TestSSHArtifactTransferServiceSearchRemoteFileInDefaultOtherRoots(t *testing.T) {
	pathService := &ArtifactPathService{
		BackendWeightsRoot:  "/tmp/backend/weights",
		BackendDatasetsRoot: "/tmp/backend/datasets",
		BaiduWeightsRoot:    "/tmp/baidu/weights",
		BaiduDatasetsRoot:   "/tmp/baidu/datasets",
		OtherWeightsRoot:    "/project/luckyProject/weights",
		OtherDatasetsRoot:   "/project/luckyProject/datasets",
	}

	client := &fakeRemoteFileClient{
		remoteFiles: map[string][]byte{
			"/project/luckyProject/weights/x.pt": []byte("x"),
		},
	}
	factory := &fakeRemoteFileClientFactory{client: client}

	svc := &SSHArtifactTransferService{
		PathService: pathService,
		serverConfigs: map[string]SSHServerConfig{
			"dev-server": {
				Name:           "dev-server",
				IP:             "10.0.0.9",
				Port:           22,
				User:           "root",
				PrivateKeyPath: "/tmp/id_rsa",
				Timeout:        10 * time.Second,
			},
		},
		defaultServer: SSHServerConfig{
			IP:             "192.168.1.100",
			Port:           22,
			User:           "root",
			PrivateKeyPath: "/tmp/id_rsa",
			Timeout:        10 * time.Second,
		},
		clientFactory: factory,
	}

	result, err := svc.SearchRemoteFileInDefaultOtherRoots("x.pt", "dev-server")
	assert.NoError(t, err)
	assert.True(t, result.ExistsInWeights)
	assert.False(t, result.ExistsInDatasets)
	assert.True(t, result.AnyExists)
	assert.Equal(t, "/project/luckyProject/weights/x.pt", result.MatchedRemotePath)
	assert.Equal(t, "dev-server", result.ServerName)
	assert.Equal(t, "10.0.0.9", result.ServerIP)
	assert.Len(t, factory.serverCalls, 1)
}

func TestSSHArtifactTransferServiceUploadArtifactByNameRemoteAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	pathService := &ArtifactPathService{
		BackendWeightsRoot:  filepath.Join(tmpDir, "backend", "weights"),
		BackendDatasetsRoot: filepath.Join(tmpDir, "backend", "datasets"),
		BaiduWeightsRoot:    filepath.Join(tmpDir, "baidu", "weights"),
		BaiduDatasetsRoot:   filepath.Join(tmpDir, "baidu", "datasets"),
		OtherWeightsRoot:    "/project/luckyProject/weights",
		OtherDatasetsRoot:   "/project/luckyProject/datasets",
	}

	localFile := filepath.Join(pathService.BackendWeightsRoot, "exists.pt")
	err := os.MkdirAll(filepath.Dir(localFile), 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(localFile, []byte("model-content"), 0o644)
	assert.NoError(t, err)

	client := &fakeRemoteFileClient{
		remoteFiles: map[string][]byte{
			"/project/luckyProject/weights/exists.pt": []byte("remote"),
		},
	}
	factory := &fakeRemoteFileClientFactory{client: client}

	svc := &SSHArtifactTransferService{
		PathService: pathService,
		defaultServer: SSHServerConfig{
			IP:             "192.168.1.100",
			Port:           22,
			User:           "root",
			PrivateKeyPath: "/tmp/id_rsa",
			Timeout:        10 * time.Second,
		},
		clientFactory: factory,
	}

	_, err = svc.UploadArtifactByName("exists.pt", "any-server")
	assert.True(t, errors.Is(err, ErrRemoteArtifactAlreadyExists))
}

func TestSSHArtifactTransferServiceUploadFileByPathWithPortOverride(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "demo.bin")
	err := os.WriteFile(localFile, []byte("abc"), 0o644)
	assert.NoError(t, err)

	client := &fakeRemoteFileClient{
		remoteFiles: map[string][]byte{},
	}
	factory := &fakeRemoteFileClientFactory{client: client}

	svc := &SSHArtifactTransferService{
		PathService: NewArtifactPathService(),
		serverConfigs: map[string]SSHServerConfig{
			"dev-server": {
				Name:           "dev-server",
				IP:             "10.0.0.10",
				Port:           22,
				User:           "root",
				PrivateKeyPath: "/tmp/id_rsa",
				Timeout:        10 * time.Second,
			},
		},
		defaultServer: SSHServerConfig{
			IP:             "192.168.1.100",
			Port:           22,
			User:           "root",
			PrivateKeyPath: "/tmp/id_rsa",
			Timeout:        10 * time.Second,
		},
		clientFactory: factory,
	}

	result, err := svc.UploadFileByPathWithPort(localFile, "/project/luckyProject/weights/demo.bin", "dev-server", 10022)
	assert.NoError(t, err)
	assert.Equal(t, "upload", result.Direction)
	assert.Equal(t, int64(3), result.Bytes)
	assert.Len(t, factory.serverCalls, 1)
	assert.Equal(t, 10022, factory.serverCalls[0].Port)
}

func TestSSHArtifactTransferServiceUploadFileByPathWithInvalidPort(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "demo.bin")
	err := os.WriteFile(localFile, []byte("abc"), 0o644)
	assert.NoError(t, err)

	client := &fakeRemoteFileClient{
		remoteFiles: map[string][]byte{},
	}
	factory := &fakeRemoteFileClientFactory{client: client}

	svc := &SSHArtifactTransferService{
		PathService: NewArtifactPathService(),
		defaultServer: SSHServerConfig{
			IP:             "192.168.1.100",
			Port:           22,
			User:           "root",
			PrivateKeyPath: "/tmp/id_rsa",
			Timeout:        10 * time.Second,
		},
		clientFactory: factory,
	}

	_, err = svc.UploadFileByPathWithPort(localFile, "/project/luckyProject/weights/demo.bin", "dev-server", 70000)
	assert.True(t, errors.Is(err, ErrSSHServerPortInvalid))
}
