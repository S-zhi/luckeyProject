# Lucky Project

基于 Go + Gin + GORM 的模型/数据集管理后端，采用四层结构：`handler -> service -> dao -> entity`。

## 功能概览
- 模型与数据集元信息管理（`models` / `datasets`）
- 训练结果管理（`lucky_model_training_result`）
- 文件上传（模型、数据集）
- 百度网盘下载到本地
- `storage_server` 多值管理（JSON 数组语义）
- 基于固定目录 + 文件名的路径解析（避免依赖 DB 中历史路径字符串）

## 固定存储路径策略
系统统一按 `weights/datasets` 分类，并通过 `storage_target` 选择根目录：

- `backend`
  - weights: `/Users/wenzhengfeng/code/go/lucky_project/weights`
  - datasets: `/Users/wenzhengfeng/code/go/lucky_project/datasets`
- `baidu_netdisk`
  - weights: `/project/luckyProject/weights`
  - datasets: `/project/luckyProject/datasets`
- `other_local`
  - weights: `/project/luckyProject/weights`
  - datasets: `/project/luckyProject/datasets`

说明：
- 模型文件路径由 `weight_name + storage_target + weights` 解析。
- 数据集路径由 `file_name + storage_target + datasets` 解析。

## 关键数据字段
### models（当前生效）
- `name` + `version` 作为唯一键
- `storage_server`（JSON）
- `base_model_id`
- `algorithm_id`
- `task_type`
- `framework`
- `weight_size_mb`
- `weight_name`（标准文件名，不含目录）

### datasets
- `name`（唯一）
- `storage_server`（JSON）
- `dataset_path`（兼容展示字段）
- `file_name`（标准文件名，不含目录）
- `task_type` / `dataset_format` / `version` / `size_mb` 等

## 文件命名规则
上传时生成标准文件名：
- 不再追加哈希后缀
- 若未传 `artifact_name`，则使用原始文件名
- 若传入 `artifact_name` 且不含扩展名，则沿用原始文件扩展名

示例：`yolov7_HRW_4.2k.pt`

## 主要接口
Base URL: `http://localhost:8080/v1`

### 模型
- `POST /models`
- `GET /models`
- `GET /models/:id/download`（浏览器下载模型文件）
- `PATCH /models/:id`（更新模型元信息）
- `GET /models/:id/storage-server`
- `PATCH /models/:id/storage-server`
- `POST /models/upload`
- `DELETE /models/by-filename?file_name=...`

### 数据集
- `POST /datasets`
- `GET /datasets`
- `GET /datasets/:id/storage-server`
- `PATCH /datasets/:id/storage-server`
- `POST /datasets/upload`

### 训练结果
- `POST /training-results`
- `GET /training-results`

### 百度网盘
- `POST /baidu/download`

## 上传接口字段
`POST /models/upload` 与 `POST /datasets/upload`（`multipart/form-data`）支持：
- `file`（必填）
- `artifact_name`（可选）
- `storage_target`（可选，默认 `backend`）
- `storage_server`（可选，记录层字段）
- `upload_to_baidu`（可选）
- `subdir`（可选，兼容字段，固定路径模式下已忽略）

返回包含：
- `file_name`
- `weight_size_mb`（MB）
- `mysql_updated` / `mysql_affected`（上传后按 `weight_name` 同步更新 models 表文件大小）
- `storage_target`
- `resolved_path`
- `paths.backend_path / paths.baidu_path / paths.other_local_path`
- `baidu_uploaded` / `baidu_path`

## 更新模型元信息
接口：`PATCH /v1/models/:id`

支持更新字段（部分更新）：
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
- `storage_server` 或 `storage_servers`（list 语义）
- `weight_name`

限制：
- `id`、`create_time` 不可修改。

## 下载模型文件
接口：`GET /v1/models/:id/download`

行为：
- 后端先读取该模型的 `storage_server` 列表与 `weight_name`。
- 若后端本地目录已存在文件：直接以附件流返回给浏览器下载。
- 若本地不存在且 `storage_server` 包含 `baidu_netdisk`：先从百度网盘固定路径下载到后端本地目录，再返回附件流。
- 下载成功后会自动把 `backend` 追加到 `storage_server`（数组语义）。

## 百度下载（记录驱动模式）
`POST /baidu/download` 支持两种方式：

1. 显式传 `remote_path`
2. 记录驱动（推荐）
   - 传 `model_id` 或 `dataset_id`
   - 传 `storage_target=baidu_netdisk`
   - 不传 `remote_path`
   - 服务端自动拼路径：
     - 模型：`/project/luckyProject/weights/{weight_name}`
     - 数据集：`/project/luckyProject/datasets/{file_name}`

下载成功后会把 `backend` 追加到对应记录的 `storage_server`（数组语义）。

## 数据库（models 表）
当前后端按以下 `models` 结构工作：

```sql
create table models
(
    id bigint unsigned auto_increment comment '主键id'
        primary key,
    name varchar(128) not null comment '模型名称（可修改）',
    version decimal(5,2) not null comment '模型版本（1.00开始递增）',
    base_model_id bigint unsigned not null default 0 comment '基础模型id（0代表未知，可自引用）',
    algorithm_id varchar(128) null comment '算法标识（可空）',
    task_type varchar(32) not null comment '任务类型（detect/segment/classify/pose/obb等）',
    description text null comment '描述（可空）',
    framework varchar(64) null comment '模型框架（如pytorch/ultralytics/sklearn等）',
    weight_size_mb decimal(10,3) not null comment '模型权重大小（MB）',
    create_time timestamp(3) default current_timestamp(3) not null comment '创建时间',
    paper varchar(1024) null comment '相关论文（URL/DOI等，可空）',
    params_url varchar(1024) null comment '模型参数URL（可空）',
    storage_server json null comment '存储位置列表（如["baidu_netdisk","backend"]）',
    weight_name varchar(128) not null comment '权重文件名称',
    constraint uk_model_name_version
        unique (name, version)
)
comment '模型资产表';
```

## 快速启动
### 1. 配置
编辑 `config/config.yaml`：
```yaml
server:
  port: 8080

db:
  driver: mysql
  host: 127.0.0.1
  port: 3306
  user: root
  password: your_password
  dbname: luckydb

baidu_pan:
  access_token: ""
  is_svip: true
  log_path: "logs/baiduPanSDK.log"

log:
  path: "logs/server.log"
```

### 2. 启动服务
```bash
go run main.go
```

### 3. 运行测试
```bash
go test ./...
```

## 说明
- 更完整字段说明与响应示例请查看 `API_DOC.md`。
- 日志文件：
  - 服务日志：`logs/server.log`
  - 百度 SDK 日志：`logs/baiduPanSDK.log`







## 二、数据库资源表

a. 未声明可以修改的就是不可修改。
b. 未声明可以为空的就是不可为空。


| 序号 |      字段      | 类型         |                             属性                             | 说明                   | 附加内容                                                     |
| :--: | :------------: | ------------ | :----------------------------------------------------------: | ---------------------- | ------------------------------------------------------------ |
|  1   |       id       | uint         |       <mark style="background: #FFB8EBA6;">标识</mark>       | 主键id，用来标识record |                                                              |
|  2   |      name      | string       |  <mark style="background: #FFB8EBA6;">标识，可以修改</mark>  | 模型名称               | {基座模型名称}-{优化算法}-{数据集名称/分类名称}              |
|  3   |    version     | decimal(5,3) |  <mark style="background: #FFB8EBA6;">标识，可以修改</mark>  | 模型版本               | 1.0开始递增；<br>1.x递增的为同模型，算法下的训练；<br>x.0 大批次更新是优化算法出现了更新； |
|  4   | base_model_id  | bool         |  <mark style="background: #6598f0;">属性<br>可以改变</mark>  | 基础模型id             | 0代表未知 ，可以自引用，默认为0                              |
|  5   |  algorithm_id  | string       | <mark style="background: #6598f0;">属性<br>可以改变<br>可以为空</mark> | 算法                   | <br>                                                         |
|  6   |   task_type    | string       |        <mark style="background: #6598f0;">属性</mark>        | 任务类型               | 任务类型（detect/segment/classify/pose/obb等）               |
|  7   |  description   | string       | <mark style="background: #6598f0;">属性<br>可以改变<br>可以为空</mark> | 描述                   |                                                              |
|  8   |   framework    | string       | <mark style="background: #6598f0;">属性<br>可以改变<br>可以为空</mark> | 模型框架               |                                                              |
|  9   | weight_size_mb | float        |        <mark style="background: #6598f0;">属性</mark>        | 模型权重大小           |                                                              |
|  10  |  create_time   | time         |        <mark style="background: #6598f0;">属性</mark>        | 创建时间               |                                                              |
|  11  |     paper      | string       | <mark style="background: #6598f0;">属性<br>可以改变<br>可以为空</mark> | 相关论文               | （URL/DOI/引用信息，可空）                                   |
|  12  |   params_url   | string       | <mark style="background: #6598f0;">属性<br>可以改变<br>可以为空</mark> | 模型参数URL            | （如config/args/yaml等，可空）                               |
|  13  | storage_server | string       | <mark style="background: #f374f7;">存储字段<br>可以改变<br>可以为空</mark> | 训练结果存储服务器标识 | 本身是一个list(string) ， ["baidu_netdisk" ，"backend" ]     |
|  14  |  weight_name   | string       | <mark style="background: #f374f7;">存储字段<br>可以改变<br></mark> | 权重模型名称           |                                                              |
