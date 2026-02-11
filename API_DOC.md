# Lucky Project API 接口文档

本文档基于当前代码实现与数据库映射（`models` / `datasets` / `lucky_model_training_result`）整理。

## 1. 基本信息
- Base URL: `http://localhost:8080/v1`
- Content-Type: `application/json`

---

## 2. 通用查询参数
所有 `GET` 列表接口支持：

| 参数名 | 类型 | 说明 |
| :--- | :--- | :--- |
| page | int | 页码，默认 1 |
| page_size | int | 每页条数，默认 10 |
| name | string | 按名称精确匹配 |
| keyword | string | 关键字模糊匹配 |

---

## 3. 模型接口 (Models)

### 3.1 创建模型
- 接口: `POST /models`
- 必填字段（与 `models` 表一致）:
  - `name`
  - `version`（decimal，如 `1.00`）
  - `base_model_id`（可传 `0`）
  - `task_type`
  - `weight_size_mb`
  - `weight_name`
  - `storage_server`
- 可选字段:
  - `algorithm_id`
  - `framework`
  - `description`
  - `paper`
  - `params_url`
  - `file_name`（兼容旧字段，内部会映射到 `weight_name`）
  - `model_path`（兼容旧字段，用于兜底提取文件名）
- 规则:
  - 幂等键为 `(name, version)`，冲突时按该唯一键 upsert。
  - 当 `weight_name` 为空时，会尝试从 `file_name` 或 `model_path` 提取 basename 回填（兼容旧客户端）。

示例：
```json
{
  "name": "YOLOv8_det",
  "version": 1.00,
  "base_model_id": 0,
  "algorithm_id": "yolo_ultralytics",
  "task_type": "detect",
  "framework": "pytorch",
  "weight_size_mb": 95.5,
  "weight_name": "yolov8_det_7a1b2c3d4e5f.pt",
  "storage_server": "[\"backend\"]",
  "description": "demo model"
}
```

### 3.2 查询模型列表
- 接口: `GET /models`
- 过滤参数:
  - `storage_server`
  - `task_type`
  - `algorithm_id`
  - `framework`
  - `version`
  - `base_model_id`
- 排序参数:
  - `size_sort=asc|desc`（推荐）
  - `weight_sort=asc|desc`（兼容参数，内部映射到 `weight_size_mb`）
- 兼容参数:
  - `algorithm` / `impl_type`（内部映射到 `algorithm_id`）
  - `dataset_id` / `train_task_id`（旧字段，当前 schema 中会忽略）

示例：
`/v1/models?algorithm_id=yolo_ultralytics&task_type=detect&size_sort=desc`

### 3.3 更新模型元信息
- 接口: `PATCH /models/{id}`
- Content-Type: `application/json`
- 说明:
  - 该接口用于修改模型元信息，支持部分字段更新（只传要更新的字段）。
  - 不允许修改：`id`、`create_time`。
  - `storage_server` 可传字符串、JSON 字符串或数组；也可传 `storage_servers` 数组。
- 可更新字段:
  - `name`
  - `version`
  - `base_model_id`
  - `algorithm_id`（可空）
  - `task_type`
  - `description`（可空）
  - `framework`（可空）
  - `weight_size_mb`
  - `paper`（可空）
  - `params_url`（可空）
  - `storage_server` / `storage_servers`
  - `weight_name`
- 返回:
  - 更新后的完整模型记录。

示例：
```bash
curl -X PATCH "http://localhost:8080/v1/models/1" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "YOLOv8_det_optimized",
    "version": 1.101,
    "algorithm_id": "yolo_ultralytics_v2",
    "framework": "pytorch",
    "weight_size_mb": 123.456,
    "storage_servers": ["backend", "baidu_netdisk"],
    "weight_name": "yolov8_det_optimized.pt"
  }'
```

### 3.4 上传模型文件
- 接口: `POST /models/upload`
- Content-Type: `multipart/form-data`
- 表单字段:
  - `file` (必填): 待上传模型文件
  - `artifact_name` (可选): 用户自定义文档名（用于生成标准文件名）
  - `storage_target` (可选): 存储目标，支持 `backend|baidu_netdisk|other_local`，默认 `backend`
  - `storage_server` (可选): 记录层的存储标识（兼容字段，不参与路径计算），默认 `backend`
  - `upload_to_baidu` (可选): 是否上传到百度网盘，布尔值，默认 `false`。支持 `true/false/1/0/t/f`。
  - `subdir` (可选): 兼容字段，已废弃；固定目录模式下忽略。
- 返回:
  - `file_name`: 标准文件名，格式 `artifact_name_哈希uuid.后缀`
  - `storage_target`: 最终存储目标
  - `resolved_path`: 本次后端实际写入路径
  - `saved_path`: 兼容字段，同 `resolved_path`
  - `paths`: 三类固定路径（`backend_path` / `baidu_path` / `other_local_path`）
  - `storage_server`: 记录层存储标识（兼容）
  - `upload_to_baidu`: 本次请求是否要求上传百度网盘
  - `baidu_uploaded`: 百度网盘是否上传成功
  - `baidu_path`: 百度网盘目标路径（仅在 `baidu_uploaded=true` 时有值）
  - 固定目录:
    - 后端: `/Users/wenzhengfeng/code/go/lucky_project/weights`
    - 百度网盘: `/project/luckyProject/weights`
    - 其他本地: `/project/luckyProject/weights`
  - 当目标为 `baidu_netdisk` 时，后端会先写入后端固定目录，再上传到百度网盘固定目录。

示例：
```bash
curl -X POST "http://localhost:8080/v1/models/upload" \
  -F "file=@/path/to/model.pt" \
  -F "artifact_name=yolov7_HRW_4.2k" \
  -F "storage_target=baidu_netdisk" \
  -F "upload_to_baidu=true"
```

### 3.5 扩展模型存储服务字段
- 查询接口: `GET /models/{id}/storage-server`
- 更新接口: `PATCH /models/{id}/storage-server`
- 说明:
  - `storage_server` 字段支持按“数组语义”管理，服务端兼容旧单值。
  - `PATCH` 请求体支持：
    - `action`: `set`(默认) / `add` / `remove`
    - `storage_server`: 单个值（可选）
    - `storage_servers`: 多个值数组（可选）
- 返回:
  - `id`
  - `storage_server`: 兼容字段，返回数组第一个值（无值时为空字符串）
  - `storage_servers`: 完整数组

示例（追加一个存储服务）：
```bash
curl -X PATCH "http://localhost:8080/v1/models/1/storage-server" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "add",
    "storage_servers": ["baidu"]
  }'
```

---

## 4. 数据集接口 (Datasets)

### 4.1 创建数据集
- 接口: `POST /datasets`
- 必填字段（与 `datasets` 表一致）:
  - `name`
  - `storage_server`
  - `task_type`
  - `dataset_format`
  - `dataset_path`
  - `version`
  - `size_mb`
- 可选字段:
  - `file_name`（推荐，标准文件名，不含目录）
  - `description`
  - `config_path`
  - `num_classes`
  - `class_names` (JSON)
  - `train_count`
  - `val_count`
  - `test_count`
- 规则:
  - 当 `file_name` 为空时，会自动从 `dataset_path` 提取 basename 回填（兼容旧客户端）。

示例：
```json
{
  "name": "Traffic_Signs_v1",
  "storage_server": "nas-01",
  "task_type": "detect",
  "dataset_format": "yolo",
  "dataset_path": "/data/datasets/traffic_signs",
  "config_path": "data.yaml",
  "version": "v1.0.0",
  "num_classes": 3,
  "class_names": ["stop", "limit", "yield"],
  "train_count": 5000,
  "val_count": 1000,
  "test_count": 800,
  "size_mb": 2048.125
}
```

### 4.2 查询数据集列表
- 接口: `GET /datasets`
- 过滤参数:
  - `storage_server`
  - `task_type`
  - `dataset_format`
  - `config_path`
  - `version`
  - `num_classes`
- 排序参数:
  - `size_sort=asc|desc`（推荐）
  - `weight_sort=asc|desc`（兼容参数）
- 兼容参数:
  - `dataset_type`（映射到 `task_type`: 1->detect, 2->segment, 3->classify, 4->pose, 5->obb）

示例：
`/v1/datasets?task_type=detect&dataset_format=yolo&size_sort=desc`

### 4.3 上传数据集文件
- 接口: `POST /datasets/upload`
- Content-Type: `multipart/form-data`
- 表单字段:
  - `file` (必填): 待上传数据集文件（例如 zip）
  - `artifact_name` (可选): 用户自定义文档名（用于生成标准文件名）
  - `storage_target` (可选): 存储目标，支持 `backend|baidu_netdisk|other_local`，默认 `backend`
  - `storage_server` (可选): 记录层的存储标识（兼容字段，不参与路径计算），默认 `backend`
  - `upload_to_baidu` (可选): 是否上传到百度网盘，布尔值，默认 `false`。支持 `true/false/1/0/t/f`。
  - `subdir` (可选): 兼容字段，已废弃；固定目录模式下忽略。
- 返回:
  - `file_name`: 标准文件名，格式 `artifact_name_哈希uuid.后缀`
  - `storage_target`: 最终存储目标
  - `resolved_path`: 本次后端实际写入路径
  - `saved_path`: 兼容字段，同 `resolved_path`
  - `paths`: 三类固定路径（`backend_path` / `baidu_path` / `other_local_path`）
  - `storage_server`: 记录层存储标识（兼容）
  - `upload_to_baidu`: 本次请求是否要求上传百度网盘
  - `baidu_uploaded`: 百度网盘是否上传成功
  - `baidu_path`: 百度网盘目标路径（仅在 `baidu_uploaded=true` 时有值）
  - 固定目录:
    - 后端: `/Users/wenzhengfeng/code/go/lucky_project/datasets`
    - 百度网盘: `/project/luckyProject/datasets`
    - 其他本地: `/project/luckyProject/datasets`
  - 当目标为 `baidu_netdisk` 时，后端会先写入后端固定目录，再上传到百度网盘固定目录。

示例：
```bash
curl -X POST "http://localhost:8080/v1/datasets/upload" \
  -F "file=@/path/to/dataset.zip" \
  -F "artifact_name=traffic_dataset" \
  -F "storage_target=backend"
```

### 4.4 扩展数据集存储服务字段
- 查询接口: `GET /datasets/{id}/storage-server`
- 更新接口: `PATCH /datasets/{id}/storage-server`
- 说明:
  - `storage_server` 字段支持按“数组语义”管理，服务端兼容旧单值。
  - `PATCH` 请求体支持：
    - `action`: `set`(默认) / `add` / `remove`
    - `storage_server`: 单个值（可选）
    - `storage_servers`: 多个值数组（可选）
- 返回:
  - `id`
  - `storage_server`: 兼容字段，返回数组第一个值（无值时为空字符串）
  - `storage_servers`: 完整数组

示例（替换为多个存储服务）：
```bash
curl -X PATCH "http://localhost:8080/v1/datasets/1/storage-server" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "set",
    "storage_servers": ["nas-01", "baidu"]
  }'
```

---

## 5. 训练结果接口 (Training Results)

> 当前代码映射表：`lucky_model_training_result`

### 5.1 创建训练结果
- 接口: `POST /training-results`

示例：
```json
{
  "model_id": 1,
  "dataset_id": 1,
  "dataset_version": 1.0,
  "training_status": 2,
  "metric_detail": {
    "mAP50": 0.92,
    "mAP50-95": 0.75,
    "recall": 0.88
  },
  "weight_path": "/data/train/best.pt",
  "comet_log_url": "https://comet.com/exp/123"
}
```

### 5.2 查询训练结果列表
- 接口: `GET /training-results`
- 过滤参数:
  - `training_model_id`
  - `training_dataset_id`
  - `training_status`

---

## 6. 百度网盘下载接口

### 6.1 下载网盘文件到本地
- 接口: `POST /baidu/download`
- Content-Type: `application/json`
- 请求体:
  - `remote_path` (可选): 百度网盘文件路径（例如 `/project/luckyProject/weights/yolo.pt`）
  - `category` (可选): 下载目标目录类别，支持 `weights|models|datasets|dataset`，默认 `weights`
  - `storage_target` (可选): 记录驱动模式下的源存储目标，当前要求 `baidu_netdisk`
  - `file_name` (可选): 本地保存文件名；不传则使用 `remote_path` 的文件名
  - `model_id` (可选): 下载成功后要同步的模型 ID（二选一：`model_id` 或 `model_name`）
  - `model_name` (可选): 下载成功后按名称同步模型记录
  - `dataset_id` (可选): 下载成功后要同步的数据集 ID（二选一：`dataset_id` 或 `dataset_name`）
  - `dataset_name` (可选): 下载成功后按名称同步数据集记录
  - `local_storage_server` (可选): 同步到记录时追加的本地存储标识，默认 `backend`
- 记录驱动模式:
  - 当 `remote_path` 为空时，必须提供 `model_id` 或 `dataset_id`，并传 `storage_target=baidu_netdisk`
  - 服务端会用记录中的文件名字段自动拼接远端路径：
    - 模型: `/project/luckyProject/weights/{weight_name}`
    - 数据集: `/project/luckyProject/datasets/{file_name}`
- 返回:
  - `message`: `download success`
  - `remote_path`: 网盘源路径
  - `local_path`: 本地落盘路径
  - `file_name`: 本地文件名
  - `category`: 实际目标类别（`weights` 或 `datasets`）
  - `size`: 下载文件大小（字节）
  - `record_synced`: 是否已自动同步 `storage_server` 到记录
  - 当 `record_synced=true` 时额外返回:
    - `record_type`: `model` 或 `dataset`
    - `record_id`: 记录 ID
    - `storage_server`: 同步后数组首元素（兼容字段）
    - `storage_servers`: 同步后完整数组

示例：
```bash
curl -X POST "http://localhost:8080/v1/baidu/download" \
  -H "Content-Type: application/json" \
  -d '{
    "model_id": 65,
    "storage_target": "baidu_netdisk",
    "local_storage_server": "backend"
  }'
```

---

## 7. 响应格式

错误响应：
```json
{
  "error": "错误信息"
}
```

分页响应：
```json
{
  "total": 100,
  "list": []
}
```
