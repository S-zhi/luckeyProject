# Lucky Project - Gin 四层架构示例项目

本项目是一个基于 Go 语言和 Gin 框架开发的 RESTful API 示例项目，采用了标准的四层架构设计（Handler, Service, DAO, Entity），支持远程数据库配置。

## 1. 项目架构

项目严格遵循以下四层架构，以实现逻辑解耦和高可维护性：

- **Handler (处理层)**: 负责处理 HTTP 请求，绑定 URL 参数，并调用 Service 层。
- **Service (业务逻辑层)**: 存放核心业务逻辑，封装分页结果。
- **DAO (数据访问层)**: 负责与数据库交互，执行分页、过滤和模糊搜索。
- **Entity (实体层)**: 定义数据库模型及通用查询 DTO。

## 2. 数据模型抽象 (Data Models)

### 模型实体 (Model) - 表名: lucky_model_information
| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | uint | 主键 ID |
| model_name | string | 基座模型名称_数据集 |
| model_type | int8 | 1:检测｜2:分割｜3:分类｜4:姿态估计｜5:OBB |
| model_version| decimal| 模型版本 |
| is_latest | bool | 是否最新 |
| is_basic_model| bool | 是否基础模型 |
| algorithm | string | 模型使用的算法 |
| framework | string | 模型框架 |
| weight_size_mb| float | 权重大小(MB) |
| weight_path | string | 权重文件路径 (本地) |
| dataset_id | uint | 数据集主键 ID |
| create_time | time | 创建时间 |

### 数据集实体 (Dataset) - 表名: lucky_dataset_information
| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | uint | 主键 ID |
| dataset_name | string | 数据集名称 |
| dataset_type | int8 | 1:检测｜2:分割｜3:分类｜4:姿态估计｜5:OBB |
| dataset_version| decimal| 数据集版本 |
| is_latest | bool | 是否最新版本 |
| sample_count | uint | 样本数量 |
| label_count | uint | 标注数量 |
| storage_type | int8 | 存储类型 1:本地｜2:OSS｜3:S3 |
| dataset_path | string | 数据集存储路径 (本地) |
| annotation_type| int8 | 标注格式 1:YOLO｜2:COCO｜3:VOC |
| description | string | 数据集描述 |
| create_time | time | 创建时间 |
| update_time | time | 更新时间 |

### 训练结果实体 (ModelTrainingResult) - 表名: lucky_model_training_result
| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | uint | 主键 ID |
| model_id | uint | 关联的模型 ID |
| dataset_id | uint | 训练使用的数据集 ID |
| dataset_version| decimal| 数据集版本 |
| training_status| int8 | 1:训练中, 2:成功, 3:失败, 4:中断 |
| metric_detail | json | 评估指标 (mAP, recall 等) |
| weight_path | string | 产出权重文件路径 |
| comet_log_url | string | Comet 实验日志 URL |
| train_start_time| time | 训练开始时间 |
| train_end_time | time | 训练结束时间 |
| create_time | time | 记录创建时间 |

## 3. 接口清单 (RESTful API)

### 通用查询参数 (Query Parameters)
所有 `GET` 列表接口均支持以下参数：
- `page`: 页码 (默认 1)
- `page_size`: 每页条数 (默认 10)
- `name`: 按名称精确过滤 (对应 `model_name` 或 `dataset_name`)
- `keyword`: 全局关键字搜索 (模糊匹配名称/描述)

#### 模型特有高级查询 (仅 Model 列表支持)
支持多个指标组合过滤：
- `model_type`: 模型类型 (1:检测, 2:分割, 3:分类, 4:姿态估计, 5:OBB)
- `is_latest`: 是否最新 (true/false)
- `is_basic_model`: 是否基础模型 (true/false)
- `algorithm`: 使用的算法 (如 yolov8)
- `framework`: 模型框架 (如 pytorch)
- `dataset_id`: 关联的数据集 ID

#### 数据集特有高级查询 (仅 Dataset 列表支持)
支持多个指标组合过滤：
- `dataset_type`: 数据集类型 (1:检测, 2:分割等)
- `storage_type`: 存储类型 (1:本地, 2:OSS, 3:S3)
- `annotation_type`: 标注格式 (1:YOLO, 2:COCO, 3:VOC)
- `is_latest`: 是否最新 (true/false)

#### 训练结果高级查询 (仅 TrainingResult 列表支持)
- `training_model_id`: 关联的模型 ID
- `training_dataset_id`: 关联的数据集 ID
- `training_status`: 训练状态 (1-4)

#### 排序规则 (Order By)
- `weight_sort`: 权重大小 (WeightSizeMB) 排序。可选值: `asc` (升序), `desc` (降序)。默认按 ID 降序。

### 模型 (Models)
- **POST `/v1/models`**: 保存模型实体
- **GET `/v1/models`**: 分页查询模型列表
    - 示例 (组合查询): `/v1/models?algorithm=yolov8&framework=pytorch&weight_sort=desc`

### 数据集 (Datasets)
- **POST `/v1/datasets`**: 保存数据集实体
- **GET `/v1/datasets`**: 分页查询数据集列表

### 训练结果 (Training Results)
- **POST `/v1/training-results`**: 保存训练结果
- **GET `/v1/training-results`**: 分页查询训练结果

## 4. 响应格式

### 分页查询返回
```json
{
  "total": 100,
  "list": [
    { "id": 1, "name": "...", ... },
    { "id": 2, "name": "...", ... }
  ]
}
```

## 5. 快速启动

### 配置数据库
修改 `config/config.yaml`：
```yaml
db:
  host: 127.0.0.1
  port: 3306
  user: root
  password: password
  dbname: lucky_db
```

### 运行
```bash
go mod tidy
go run main.go
```

## 6. 接口测试

### 自动化测试 (Go Test)
执行以下命令运行全量接口集成测试：
```bash
go test -v ./internal/handler/v1/...
```

### 手动快速测试 (Shell Script)
项目根目录下提供了 `test_api.sh` 脚本，可快速验证核心接口：
```bash
./test_api.sh
```
