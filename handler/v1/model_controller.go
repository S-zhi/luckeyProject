package v1

import (
	"errors"
	"fmt"
	"lucky_project/dao"
	entity2 "lucky_project/entity"
	"lucky_project/service"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ModelController struct {
	modelService    *service.ModelService
	uploadService   *service.UploadService
	downloadService *service.BaiduDownloadService
	sshUploadSvc    *service.SSHArtifactTransferService
}

func NewModelController() *ModelController {
	return &ModelController{
		modelService:    service.NewModelService(),
		uploadService:   service.NewUploadService(),
		downloadService: service.NewBaiduDownloadService(),
		sshUploadSvc:    service.NewSSHArtifactTransferService(),
	}
}

// CreateModel handles POST /v1/models
func (c *ModelController) CreateModel(ctx *gin.Context) {
	var model entity2.Model
	if err := ctx.ShouldBindJSON(&model); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.modelService.CreateModel(ctx.Request.Context(), &model); err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, model)
}

// GetAllModels handles GET /v1/models
func (c *ModelController) GetAllModels(ctx *gin.Context) {
	var params entity2.QueryParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.modelService.GetAllModels(ctx.Request.Context(), params)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// GetModelStorageServers handles GET /v1/models/:id/storage-server
func (c *ModelController) GetModelStorageServers(ctx *gin.Context) {
	id, err := parseUintPathParam(ctx, "id")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	servers, err := c.modelService.GetStorageServersByID(ctx.Request.Context(), id)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, buildStorageServerResponse(id, servers))
}

// UpdateModelStorageServers handles PATCH /v1/models/:id/storage-server
func (c *ModelController) UpdateModelStorageServers(ctx *gin.Context) {
	logger := handlerLogger().With("controller", "ModelController", "method", "UpdateModelStorageServers")
	id, err := parseUintPathParam(ctx, "id")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var payload storageServerUpdatePayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action, servers := normalizeStorageServerPayload(payload)
	logger.Info(
		"更新模型存储服务请求",
		"model_id", id,
		"action", action,
		"servers", servers,
	)
	updated, err := c.modelService.UpdateStorageServersByID(ctx.Request.Context(), id, action, servers)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	logger.Info(
		"更新模型存储服务成功（仅更新元数据）",
		"model_id", id,
		"updated_servers", updated,
		"note", "该接口仅更新元数据，不会向远程服务器上传文件",
	)
	ctx.JSON(http.StatusOK, buildStorageServerResponse(id, updated))
}

// UploadModelFile handles POST /v1/models/upload
func (c *ModelController) UploadModelFile(ctx *gin.Context) {
	logger := handlerLogger().With("controller", "ModelController", "method", "UploadModelFile")
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	subdir := ctx.PostForm("subdir")
	_ = subdir // deprecated: fixed root strategy no longer uses subdir for path resolution.
	artifactName := ctx.PostForm("artifact_name")
	storageTarget := ctx.PostForm("storage_target")
	storageServer := ctx.PostForm("storage_server")
	coreServerKey := pickFirstNonEmpty(
		ctx.PostForm("core_server_key"),
		ctx.PostForm("core_server_name"),
	)
	uploadToBaidu, err := parseOptionalBoolForm(ctx, "upload_to_baidu", false)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info(
		"上传模型请求",
		"original_file", file.Filename,
		"artifact_name", artifactName,
		"storage_target", storageTarget,
		"storage_server", storageServer,
		"upload_to_baidu", uploadToBaidu,
		"core_server_key", coreServerKey,
	)

	if coreServerKey == "" && shouldAutoUploadToCoreByStorageServer(storageServer) {
		coreServerKey = strings.TrimSpace(storageServer)
		logger.Info(
			"已从存储服务自动选择核心上传键",
			"storage_server", storageServer,
			"core_server_key", coreServerKey,
		)
	}

	result, err := c.uploadService.SaveModelFile(file, artifactName, storageTarget, storageServer, uploadToBaidu)
	if err != nil {
		logger.Error("保存模型文件失败", "error", err)
		switch {
		case errors.Is(err, service.ErrInvalidUploadFile),
			errors.Is(err, service.ErrInvalidUploadSubdir),
			errors.Is(err, service.ErrInvalidStorageTarget):
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	logger.Info(
		"保存模型文件成功",
		"file_name", result.FileName,
		"resolved_path", result.ResolvedPath,
		"saved_path", result.SavedPath,
		"size", result.Size,
		"storage_target", result.StorageTarget,
		"storage_server", result.StorageServer,
		"baidu_uploaded", result.BaiduUploaded,
		"baidu_path", result.BaiduPath,
	)

	var coreTransfer *service.SSHTransferResult
	var coreServer *service.CoreServer
	if coreServerKey != "" {
		logger.Info("请求上传到核心服务器", "core_server_key", coreServerKey)
		server, transfer, err := c.uploadModelToCoreServer(ctx, coreServerKey, result.FileName, result.ResolvedPath)
		if err != nil {
			logger.Error("上传到核心服务器失败", "core_server_key", coreServerKey, "error", err)
			switch {
			case errors.Is(err, service.ErrCoreServerKeyRequired),
				errors.Is(err, service.ErrCoreServerNotFound),
				errors.Is(err, service.ErrSSHServerPortInvalid),
				errors.Is(err, service.ErrSSHFilePathRequired),
				errors.Is(err, service.ErrInvalidStorageTarget):
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			case errors.Is(err, service.ErrRedisNotInitialized):
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			default:
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		coreServer = &server
		coreTransfer = &transfer
		logger.Info(
			"上传到核心服务器成功",
			"core_server_key", server.Key,
			"core_server_ip", server.IP,
			"core_server_port", server.Port,
			"remote_path", transfer.TargetPath,
			"bytes", transfer.Bytes,
		)
	} else {
		logger.Info("已跳过上传到核心服务器", "reason", "core_server_key 为空")
	}

	affectedRows, weightSizeMB, err := c.modelService.SyncWeightSizeByFileName(ctx.Request.Context(), result.FileName, result.Size)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}
	if affectedRows == 0 {
		logger.Info(
			"同步模型权重大小已跳过",
			"weight_name", result.FileName,
			"affected_rows", affectedRows,
			"note", "没有模型记录匹配权重文件名；如果先上传文件后创建 /v1/models 记录，该情况属于预期",
		)
	} else {
		logger.Info(
			"同步模型权重大小成功",
			"weight_name", result.FileName,
			"affected_rows", affectedRows,
			"weight_size_mb", weightSizeMB,
		)
	}

	resp := gin.H{
		"message":         "upload success",
		"file_name":       result.FileName,
		"resolved_path":   result.ResolvedPath,
		"saved_path":      result.SavedPath,
		"paths":           result.Paths,
		"size":            result.Size,
		"storage_server":  result.StorageServer,
		"storage_target":  result.StorageTarget,
		"upload_to_baidu": result.UploadToBaidu,
		"baidu_uploaded":  result.BaiduUploaded,
		"baidu_path":      result.BaiduPath,
		"weight_size_mb":  weightSizeMB,
		"mysql_updated":   affectedRows > 0,
		"mysql_affected":  affectedRows,
	}

	if coreTransfer != nil && coreServer != nil {
		resp["core_uploaded"] = true
		resp["core_server_key"] = coreServer.Key
		resp["core_server_ip"] = coreServer.IP
		resp["core_server_port"] = coreServer.Port
		resp["core_remote_path"] = coreTransfer.TargetPath
	} else {
		resp["core_uploaded"] = false
	}

	ctx.JSON(http.StatusCreated, resp)
}

func (c *ModelController) uploadModelToCoreServer(ctx *gin.Context, coreServerKey, fileName, localPath string) (service.CoreServer, service.SSHTransferResult, error) {
	logger := handlerLogger().With("controller", "ModelController", "method", "uploadModelToCoreServer")
	if c.sshUploadSvc == nil {
		return service.CoreServer{}, service.SSHTransferResult{}, service.ErrSSHClientFactoryNil
	}

	logger.Info(
		"为上传解析核心服务器",
		"core_server_key", coreServerKey,
		"file_name", fileName,
		"local_path", localPath,
	)
	coreServer, err := service.GetCoreServerByKey(ctx.Request.Context(), coreServerKey)
	if err != nil {
		logger.Error("解析核心服务器失败", "core_server_key", coreServerKey, "error", err)
		return service.CoreServer{}, service.SSHTransferResult{}, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return service.CoreServer{}, service.SSHTransferResult{}, err
	}
	privateKeyPath := strings.TrimSpace(ctx.PostForm("ssh_private_key_path"))
	if privateKeyPath == "" {
		privateKeyPath, err = resolveDefaultSSHPrivateKeyPath(homeDir)
		if err != nil {
			logger.Error("解析SSH私钥路径失败", "error", err)
			return service.CoreServer{}, service.SSHTransferResult{}, err
		}
	}
	sshUser := strings.TrimSpace(ctx.PostForm("ssh_user"))
	if sshUser == "" {
		sshUser = service.DefaultSSHServerUser
	}

	if err := c.sshUploadSvc.SetServerConfig(coreServer.Key, service.SSHServerConfig{
		Name:           coreServer.Key,
		IP:             coreServer.IP,
		Port:           coreServer.Port,
		User:           sshUser,
		PrivateKeyPath: privateKeyPath,
	}); err != nil {
		logger.Error("设置SSH服务器配置失败", "core_server_key", coreServer.Key, "error", err)
		return service.CoreServer{}, service.SSHTransferResult{}, err
	}

	pathService := service.NewArtifactPathService()
	remotePath, err := pathService.BuildPath(service.ArtifactCategoryWeights, service.StorageTargetOtherLocal, fileName)
	if err != nil {
		logger.Error("构建远程路径失败", "file_name", fileName, "error", err)
		return service.CoreServer{}, service.SSHTransferResult{}, err
	}

	logger.Info(
		"开始通过SSH上传到核心服务器",
		"core_server_key", coreServer.Key,
		"core_server_ip", coreServer.IP,
		"core_server_port", coreServer.Port,
		"ssh_user", sshUser,
		"private_key_path", privateKeyPath,
		"remote_path", remotePath,
	)
	transfer, err := c.sshUploadSvc.UploadFileByPathWithPort(localPath, remotePath, coreServer.Key, coreServer.Port)
	if err != nil {
		logger.Error("SSH上传失败", "core_server_key", coreServer.Key, "error", err)
		return service.CoreServer{}, service.SSHTransferResult{}, err
	}
	logger.Info(
		"SSH上传完成",
		"core_server_key", coreServer.Key,
		"remote_path", transfer.TargetPath,
		"bytes", transfer.Bytes,
		"cost", transfer.Cost.String(),
	)
	return coreServer, transfer, nil
}

// DeleteModelByFileName handles DELETE /v1/models/by-filename?file_name=xxx
func (c *ModelController) DeleteModelByFileName(ctx *gin.Context) {
	fileName := strings.TrimSpace(ctx.Query("file_name"))
	if fileName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file_name is required"})
		return
	}

	result, err := c.modelService.DeleteByFileName(ctx.Request.Context(), fileName)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":            "delete success",
		"file_name":          result.FileName,
		"deleted_records":    result.DeletedRecords,
		"local_file_deleted": result.LocalFileDeleted,
	})
}

// DownloadModelFile handles GET /v1/models/:id/download.
func (c *ModelController) DownloadModelFile(ctx *gin.Context) {
	id, err := parseUintPathParam(ctx, "id")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	model, err := c.modelService.GetByID(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
			return
		}
		writeHTTPError(ctx, err)
		return
	}

	fileName := strings.TrimSpace(model.WeightName)
	if fileName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "model weight_name is empty"})
		return
	}
	fileName = filepath.Base(fileName)

	localPath, err := c.modelService.ResolveFilePathByID(ctx.Request.Context(), id, service.StorageTargetBackend)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	if fileExists(localPath) {
		ctx.FileAttachment(localPath, fileName)
		return
	}

	if c.downloadService == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "download service is nil"})
		return
	}

	remotePath, err := c.modelService.ResolveFilePathByID(ctx.Request.Context(), id, service.StorageTargetBaiduNetdisk)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	result, err := c.downloadService.DownloadToLocal(remotePath, service.ArtifactCategoryWeights, fileName)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidDownloadCategory),
			errors.Is(err, service.ErrInvalidBaiduDownloadPath),
			errors.Is(err, service.ErrInvalidBaiduDownloadFile),
			errors.Is(err, service.ErrInvalidLocalDownloadFile),
			errors.Is(err, service.ErrBaiduDownloadTargetRequired),
			errors.Is(err, service.ErrBaiduPanAccessTokenRequired),
			errors.Is(err, service.ErrInvalidStorageTarget):
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	localPath = result.LocalPath
	if !fileExists(localPath) {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "downloaded file not found in backend storage"})
		return
	}

	if _, err := c.modelService.UpdateStorageServersByID(
		ctx.Request.Context(),
		id,
		dao.StorageActionAdd,
		[]string{service.StorageTargetBackend},
	); err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.FileAttachment(localPath, fileName)
}

// UpdateModelMetadata handles PATCH /v1/models/:id
func (c *ModelController) UpdateModelMetadata(ctx *gin.Context) {
	id, err := parseUintPathParam(ctx, "id")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var payload map[string]interface{}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates, err := parseModelMetadataUpdates(payload)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	model, err := c.modelService.UpdateModelMetadata(ctx.Request.Context(), id, updates)
	if err != nil {
		writeHTTPError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, model)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func containsBaiduStorage(servers []string) bool {
	for _, server := range servers {
		switch strings.ToLower(strings.TrimSpace(server)) {
		case service.StorageTargetBaiduNetdisk, "baidu", "baidu-pan", "baidu_pan", "baidupan", "pan.baidu", "百度网盘":
			return true
		}
	}
	return false
}

func isLocalStorageServer(server string) bool {
	switch strings.ToLower(strings.TrimSpace(server)) {
	case "", service.StorageTargetBackend, "local", "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func shouldAutoUploadToCoreByStorageServer(storageServer string) bool {
	if isLocalStorageServer(storageServer) {
		return false
	}
	if containsBaiduStorage([]string{storageServer}) {
		return false
	}
	return true
}

func resolveDefaultSSHPrivateKeyPath(homeDir string) (string, error) {
	sshDir := filepath.Join(homeDir, ".ssh")
	candidates := []string{
		filepath.Join(sshDir, "id_rsa"),
		filepath.Join(sshDir, "id_ed25519"),
	}

	for _, path := range candidates {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		return path, nil
	}

	return "", fmt.Errorf("no usable ssh private key found under %s (tried id_rsa, id_ed25519)", sshDir)
}

func pickFirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
