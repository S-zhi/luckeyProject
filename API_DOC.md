# Lucky Project API 接口文档

本文档详细描述了 Lucky Project 后端服务的 RESTful API 接口。

## 1. 基本信息
- **Base URL**: `http://localhost:8080/v1`
- **Content-Type**: `application/json`

---

## 2. 通用分页查询参数
所有 `GET` 列表接口均支持以下基础参数：

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 10 |
| name | string | 否 | 按名称精确匹配 |
| keyword | string | 否 | 全局关键字模糊搜索 (名称/描述) |

---

## 3. 模型接口 (Models)

### 3.1 创建模型
- **接口**: `POST /models`
- **说明**: 保存模型实体信息到数据库。
- **请求体**:
```json
{
  "model_name": "ResNet50_Traffic",
  "model_type": 3,
  "model_version": 1.2,
  "is_latest": true,
  "is_basic_model": false,
  "algorithm": "ResNet",
  "framework": "PyTorch",
  "weight_size_mb": 95.5,
  "weight_path": "/data/weights/resnet50.pt"
}
```

### 3.2 查询模型列表
- **接口**: `GET /models`
- **高级过滤参数**:
  - `model_type`: 模型类型 (1:检测, 2:分割, 3:分类, 4:姿态估计, 5:OBB)
  - `is_latest`: 是否最新 (true/false)
  - `is_basic_model`: 是否基础模型 (true/false)
  - `algorithm`: 使用的算法
  - `framework`: 模型框架
  - `dataset_id`: 关联的数据集 ID
- **排序规则**:
  - `weight_sort`: 权重大小排序 (`asc`: 升序, `desc`: 降序)
- **示例**: `/v1/models?algorithm=YOLOv8&weight_sort=desc`

---

## 4. 数据集接口 (Datasets)

### 4.1 创建数据集
- **接口**: `POST /datasets`
- **请求体**:
```json
{
  "dataset_name": "Traffic_Signs_V1",
  "dataset_type": 1,
  "sample_count": 5000,
  "storage_type": 1,
  "dataset_path": "/data/datasets/traffic",
  "annotation_type": 1,
  "description": "交通标志检测数据集"
}
```

### 4.2 查询数据集列表
- **接口**: `GET /datasets`
- **高级过滤参数**:
  - `dataset_type`: 数据集类型 (1-5)
  - `storage_type`: 存储类型 (1:本地, 2:OSS, 3:S3)
  - `annotation_type`: 标注格式 (1:YOLO, 2:COCO, 3:VOC)
  - `is_latest`: 是否最新 (true/false)

---

## 5. 训练结果接口 (Training Results)

### 5.1 保存训练结果
- **接口**: `POST /training-results`
- **请求体**:
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
- **接口**: `GET /training-results`
- **高级过滤参数**:
  - `training_model_id`: 关联的模型 ID
  - `training_dataset_id`: 关联的数据集 ID
  - `training_status`: 训练状态 (1:训练中, 2:成功, 3:失败, 4:中断)

---

## 6. 响应规范
所有接口成功时返回对应的实体或列表，失败时返回统一的错误格式：
```json
{
  "error": "错误详细信息"
}
```
列表接口返回格式：
```json
{
  "total": 100,
  "list": [...]
}
```
