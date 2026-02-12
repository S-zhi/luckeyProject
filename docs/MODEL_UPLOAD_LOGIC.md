# Model 文件上传逻辑说明

本文档说明接口 `POST /v1/models/upload` 的完整执行链路与关键行为。

## 1. 接口入口

- 路由注册：
  - `/Users/wenzhengfeng/code/go/lucky_project/router/http_router.go`
  - `models.POST("/upload", modelController.UploadModelFile)`
- Controller 实现：
  - `/Users/wenzhengfeng/code/go/lucky_project/handler/v1/model_controller.go`
  - `func (c *ModelController) UploadModelFile(ctx *gin.Context)`

请求方式：
- Method: `POST`
- Path: `/v1/models/upload`
- Content-Type: `multipart/form-data`

请求字段：
- `file`：必填，模型文件
- `artifact_name`：可选，期望保存文件名（可不带扩展名）
- `storage_target`：可选，`backend | baidu_netdisk | other_local`
- `storage_server`：可选，记录层标记
- `upload_to_baidu`：可选，布尔值（`true/false/1/0/t/f`）
- `core_server_key`：可选，核心服务器标识（示例：`rtx3090`）
- `core_server_name`：可选，`core_server_key` 的别名字段
- `ssh_user`：可选，SSH 用户（默认 `root`）
- `ssh_private_key_path`：可选，SSH 私钥路径（默认 `~/.ssh/id_rsa`）
- `subdir`：兼容字段，已弃用

## 2. 总体流程

1. Controller 读取表单和文件，校验 `file` 是否存在。
2. 解析 `upload_to_baidu`（空值用默认 false，非法值直接 400）。
3. 调用 `UploadService.SaveModelFile(...)` 执行实际落盘/上传。
4. 若传入 `core_server_key/core_server_name`，调用 Redis 接口查询目标服务器（IP/Port），然后使用 SSH 传输服务把文件上传到远程 `other_local` 路径。
5. 上传完成后，调用 `ModelService.SyncWeightSizeByFileName(...)` 同步 MySQL 的 `weight_size_mb`。
6. 返回上传结果 JSON（包含本地路径、目标路径、文件大小、数据库更新状态等）。

## 3. Service 详细逻辑

核心实现文件：
- `/Users/wenzhengfeng/code/go/lucky_project/service/upload_service.go`
- `/Users/wenzhengfeng/code/go/lucky_project/service/artifact_path_service.go`

### 3.1 预校验

`UploadService.save(...)` 会先检查：
- 上传文件是否为空
- `PathService` 是否可用

### 3.2 存储目标解析

`resolveUploadTarget(...)` 规则：
- 如果 `upload_to_baidu=true`，强制目标 `baidu_netdisk`
- 否则优先使用 `storage_target`
- 若 `storage_target` 为空且 `storage_server` 表示百度，则目标也为 `baidu_netdisk`
- 都没有则默认 `backend`

### 3.3 文件名生成规则（当前版本）

`ArtifactPathService.GenerateStoredFileName(...)` 当前行为：
- 不再追加 hash 后缀
- 若没传 `artifact_name`：使用原始文件名
- 若传了 `artifact_name` 但无扩展名：沿用原始文件扩展名
- 若传了同扩展名：使用传入文件名
- 会对 base name 做安全字符清洗（非法字符替换为 `_`）

示例：
- 原文件 `yolo26n_v6.0.onnx`，未传 `artifact_name` -> `yolo26n_v6.0.onnx`
- 原文件 `origin.pt`，`artifact_name=yolov7_HRW_4.2k` -> `yolov7_HRW_4.2k.pt`

### 3.4 路径计算与落盘

- 先根据分类 `weights` 计算三类路径（backend/baidu/other）
- 若目标是 `baidu_netdisk`，会先写入 backend 本地文件，再上传百度
- `os.MkdirAll` 创建目录
- `io.Copy` 将 multipart 内容写到目标文件
- 返回 `size`（字节）

默认模型根路径：
- backend: `/Users/wenzhengfeng/code/go/lucky_project/weights`
- baidu: `/project/luckyProject/weights`
- other: `/project/luckyProject/weights`

### 3.5 百度上传分支

当最终需要上传百度时：
- 取百度根目录
- 调用 `BaiduUploader.Upload(localPath, remoteDir)`
- 返回 `baidu_uploaded=true` 与 `baidu_path`

## 4. MySQL 文件大小同步逻辑

关键文件：
- `/Users/wenzhengfeng/code/go/lucky_project/handler/v1/model_controller.go`
- `/Users/wenzhengfeng/code/go/lucky_project/service/model_service.go`
- `/Users/wenzhengfeng/code/go/lucky_project/dao/model_dao.go`

步骤：
1. Controller 在上传完成后调用：
   - `SyncWeightSizeByFileName(result.FileName, result.Size)`
2. Service 将字节转换为 MB（保留 3 位小数）。
3. DAO 执行：
   - `UPDATE models SET weight_size_mb=? WHERE weight_name=?`

注意：
- 上传成功不代表一定更新到 DB。
- 如果 `models` 表里没有匹配该 `weight_name`，则 `rows_affected=0`，接口仍返回上传成功。

## 5. Redis + SSH 核心服务器上传逻辑

关键文件：
- `/Users/wenzhengfeng/code/go/lucky_project/handler/v1/model_controller.go`
- `/Users/wenzhengfeng/code/go/lucky_project/service/core_server_redis_service.go`
- `/Users/wenzhengfeng/code/go/lucky_project/service/ssh_artifact_transfer_service.go`

步骤：
1. Controller 从 `core_server_key`（或 `core_server_name`）读取核心服务器标识。
2. 调用 `GetCoreServerByKey(ctx, key)` 从 Redis `hash=core-servers` 读取 JSON 配置，解析出 `ip/port`。
3. 使用 `SetServerConfig(...)` 动态写入 SSH 连接信息（服务器名、IP、端口、用户、私钥）。
4. 计算远程目标路径（`other_local` 下的 `weights` 路径）。
5. 调用 `UploadFileByPathWithPort(localPath, remotePath, serverName, port)` 完成 SSH 上传。
6. 出错时将错误原文返回前端，便于联调定位。

## 6. 返回字段说明（上传接口）

- `file_name`：最终保存文件名
- `resolved_path` / `saved_path`：后端实际写入路径
- `paths`：三类存储路径
- `size`：上传文件字节数
- `storage_server`：记录层标记
- `storage_target`：最终上传目标
- `upload_to_baidu`：是否触发百度上传流程
- `baidu_uploaded` / `baidu_path`：百度上传结果
- `weight_size_mb`：用于 DB 同步的 MB 值
- `mysql_updated`：`rows_affected > 0`
- `mysql_affected`：MySQL 更新影响行数
- `core_uploaded`：是否触发并完成核心服务器上传
- `core_server_key` / `core_server_ip` / `core_server_port`：核心服务器信息
- `core_remote_path`：核心服务器上的目标路径

## 7. 常见返回与错误

常见成功：
- `201 Created`

常见失败：
- `400`: 缺少 `file`、`upload_to_baidu` 非法、`storage_target` 非法
- `400`: `core_server_key` 为空、Redis 中找不到该 key、SSH 端口非法等参数错误
- `500`: 本地写文件失败、百度上传失败、Redis 未初始化、SSH 上传失败、数据库更新异常

## 8. 调试建议

1. 先看 `file_name` 与 `resolved_path` 是否符合预期。
2. 若 `mysql_updated=false`，确认 `models.weight_name` 是否与上传文件名完全一致。
3. 若走百度流程失败，检查 access token 与远程目录权限。
4. 若核心服务器上传失败，优先检查 Redis `core-servers` 配置值、SSH 用户、私钥路径与端口可达性。
