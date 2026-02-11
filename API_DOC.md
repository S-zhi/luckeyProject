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
  - `storage_server`
  - `model_path`
  - `impl_type`
  - `dataset_id`
  - `size_mb`
  - `version`
  - `task_type`
- 可选字段:
  - `paper`
  - `params_url`
  - `base_model_id`
  - `train_task_id`
  - `description`
- 规则:
  - 后端会将最终模型名标准化为 `name_version`（例如 `yolo26n_v1.0.0`）。
  - 如果标准化后名称已存在，按 `name` 执行幂等覆盖更新（不再因重复名报错）。

示例：
```json
{
  "name": "YOLOv8_det_v1",
  "storage_server": "nas-01",
  "model_path": "/data/models/yolov8_det_v1.pt",
  "impl_type": "yolo_ultralytics",
  "dataset_id": 1,
  "size_mb": 95.5,
  "version": "v1.0.0",
  "task_type": "detect",
  "description": "demo model"
}
```

### 3.2 查询模型列表
- 接口: `GET /models`
- 过滤参数:
  - `storage_server`
  - `task_type`
  - `impl_type`
  - `version`
  - `dataset_id`
  - `train_task_id`
  - `base_model_id`
- 排序参数:
  - `size_sort=asc|desc`（推荐）
  - `weight_sort=asc|desc`（兼容参数，内部映射到 `size_mb`）
- 兼容参数:
  - `algorithm`（内部映射到 `impl_type`）

示例：
`/v1/models?impl_type=yolo_ultralytics&task_type=detect&size_sort=desc`

### 3.3 上传模型文件
- 接口: `POST /models/upload`
- Content-Type: `multipart/form-data`
- 表单字段:
  - `file` (必填): 待上传模型文件
  - `subdir` (可选): 上传子目录（相对目录）
  - `storage_server` (可选): 上传目标服务器标识。默认 `backend`。当前版本无论传什么值，都会先落到后端本地存储。
  - `upload_to_baidu` (可选): 是否上传到百度网盘，布尔值，默认 `false`。支持 `true/false/1/0/t/f`。
- 返回:
  - `saved_path` 可直接用于 `model_path`
  - `storage_server` 为最终记录的服务器标识（默认 `backend`）
  - 文件命名规则：优先使用原始文件名（清洗非法字符后），同目录重名时自动追加 `_1`、`_2` 递增后缀。
  - `upload_to_baidu`: 本次请求是否要求上传百度网盘
  - `baidu_uploaded`: 百度网盘是否上传成功
  - `baidu_path`: 百度网盘目标路径（仅在 `baidu_uploaded=true` 时有值）
  - 百度网盘模型固定目录常量：`/project/luckyProject/weights`
  - 当 `upload_to_baidu=true` 或 `storage_server=baidu` 时，后端会先将文件落盘到本地目录 `/Users/wenzhengfeng/code/go/lucky_project/weights`（可拼接 `subdir`），再调用百度网盘上传。

示例：
```bash
curl -X POST "http://localhost:8080/v1/models/upload" \
  -F "file=@/path/to/model.pt" \
  -F "subdir=demo" \
  -F "storage_server=nas-01" \
  -F "upload_to_baidu=true"
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
  - `description`
  - `config_path`
  - `num_classes`
  - `class_names` (JSON)
  - `train_count`
  - `val_count`
  - `test_count`

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
  - `subdir` (可选): 上传子目录（相对目录）
  - `storage_server` (可选): 上传目标服务器标识。默认 `backend`。当前版本无论传什么值，都会先落到后端本地存储。
  - `upload_to_baidu` (可选): 是否上传到百度网盘，布尔值，默认 `false`。支持 `true/false/1/0/t/f`。
- 返回:
  - `saved_path` 可直接用于 `dataset_path`
  - `storage_server` 为最终记录的服务器标识（默认 `backend`）
  - 文件命名规则：优先使用原始文件名（清洗非法字符后），同目录重名时自动追加 `_1`、`_2` 递增后缀。
  - `upload_to_baidu`: 本次请求是否要求上传百度网盘
  - `baidu_uploaded`: 百度网盘是否上传成功
  - `baidu_path`: 百度网盘目标路径（仅在 `baidu_uploaded=true` 时有值）
  - 百度网盘数据集固定目录常量：`/project/luckyProject/datasets`
  - 当 `upload_to_baidu=true` 或 `storage_server=baidu` 时，后端会先将文件落盘到本地目录 `/Users/wenzhengfeng/code/go/lucky_project/datasets`（可拼接 `subdir`），再调用百度网盘上传。

示例：
```bash
curl -X POST "http://localhost:8080/v1/datasets/upload" \
  -F "file=@/path/to/dataset.zip" \
  -F "subdir=demo" \
  -F "storage_server=nas-01" \
  -F "upload_to_baidu=true"
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

## 6. 响应格式

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
