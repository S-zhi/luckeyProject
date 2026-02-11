# Lucky Project 后端启动文档

本文档说明如何在本地启动当前后端服务（Gin + GORM + MySQL）。

## 1. 前置条件

1. 已安装 Go（建议与项目一致版本）。
2. 可访问 MySQL 数据库。
3. 在项目根目录执行命令。

项目根目录：

```bash
cd /Users/wenzhengfeng/code/go/lucky_project
```

## 2. 配置数据库

编辑配置文件：

`config/config.yaml`

最少需要正确配置：

```yaml
server:
  port: 8080
db:
  driver: mysql
  host: 127.0.0.1
  port: 3306
  user: your_user
  password: your_password
  dbname: your_db
```

注意：

1. `driver` 当前仅支持 `mysql`。
2. 程序会在启动时连接数据库并做连接校验。
3. 请从项目根目录启动，否则可能读不到 `config/config.yaml`。

## 3. 安装依赖

首次拉代码后建议执行：

```bash
go mod tidy
```

## 4. 启动后端服务

开发启动：

```bash
go run main.go
```

正常启动后会看到类似输出：

```text
Server is running on port 8080...
```

## 5. 验证服务是否可用

### 5.1 快速验证（推荐）

项目自带脚本：

```bash
./test_api.sh
```

该脚本会依次调用：

1. 创建模型
2. 创建数据集
3. 查询模型
4. 查询数据集
5. 创建训练结果
6. 查询训练结果

### 5.2 手动验证

检查模型列表接口：

```bash
curl "http://localhost:8080/v1/models?page=1&page_size=1"
```

返回包含 `total` 和 `list` 即表示路由与数据库查询正常。

## 6. 常见问题排查

### 6.1 启动失败：数据库连接错误

表现：

1. `Init database failed`
2. `mysql ping failed`

排查：

1. 检查 `config/config.yaml` 中 `host/port/user/password/dbname`。
2. 确认数据库服务可达（网络、防火墙、白名单）。
3. 确认数据库里存在项目使用的表结构。

### 6.2 端口占用

表现：

1. `listen tcp :8080: bind: address already in use`

处理：

1. 修改 `config/config.yaml` 中 `server.port`。
2. 或停止占用 8080 的进程后重试。

### 6.3 请求 404

检查：

1. 接口是否带 `/v1` 前缀。
2. 路径是否正确（如 `/v1/models`、`/v1/datasets`、`/v1/training-results`）。

## 7. 生产建议

1. 使用环境隔离的配置，不要提交生产密码。
2. 建议通过进程管理器（systemd/supervisor/容器）运行。
3. 启动前先做一次 `go test ./...` 回归。
