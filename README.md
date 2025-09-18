# DataX Web Go

[![Go Version](https://img.shields.io/badge/Go-1.20+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/fangx1999/datax-web-go.svg)](https://github.com/fangx1999/datax-web-go/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/fangx1999/datax-web-go.svg)](https://github.com/fangx1999/datax-web-go/network)

一个基于 Go 语言开发的 DataX 数据同步任务管理平台，提供 Web 界面来管理和监控 DataX 数据同步任务。

## 快速开始

```bash
# 克隆项目
git clone https://github.com/fangx1999/datax-web-go.git
cd datax-web-go

# 安装依赖
go mod download

# 配置数据库
mysql -u root -p < init.sql

# 修改配置文件
cp config.yaml config.yaml.local
# 编辑 config.yaml.local 中的数据库配置

# 运行项目
go run cmd/main.go config.yaml.local
```

访问 http://localhost:8000，默认账户：admin/admin

## 项目功能

### 核心功能

- **数据源管理**: 支持多种数据源类型（MySQL、HDFS、OFS、COSN）
- **任务管理**: 创建、编辑、执行和监控 DataX 数据同步任务
- **任务流管理**: 支持定时调度的任务流，可配置多个任务按顺序执行
- **用户管理**: 支持管理员和普通用户角色，提供用户认证和授权
- **日志监控**: 提供任务执行日志和任务流执行日志的查看和管理
- **实时监控**: 支持任务的实时状态监控和手动终止

### 详细功能列表

#### 1. 认证与授权
- 用户登录/登出
- 基于角色的访问控制（管理员/普通用户）
- 会话管理

#### 2. 数据源管理
- 支持 MySQL 数据库连接
- 支持 HDFS 分布式文件系统
- 支持 OFS 对象存储
- 支持 COSN 腾讯云对象存储
- 数据源连接测试
- 元数据获取（MySQL 表结构）

#### 3. 任务管理
- 创建和编辑 DataX 任务配置
- 手动执行任务
- 任务配置预览
- 任务删除
- 支持日期占位符替换（${yyyy-mm-dd}, ${yyyy_mm_dd}）

#### 4. 任务流管理
- 创建任务流，支持多个任务按顺序执行
- 基于 Cron 表达式的定时调度
- 任务流启用/禁用
- 手动执行任务流
- 任务流步骤管理（添加、删除、重排序）
- 任务流执行监控和终止

#### 5. 日志与监控
- 任务执行日志查看
- 任务流执行日志查看
- 实时日志显示
- 执行状态跟踪（pending, running, success, failed, killed, skipped）

#### 6. 工具功能
- JSON 格式化工具
- DataX 配置预览

## 技术栈

- **后端**: Go 1.20
- **Web 框架**: Gin
- **数据库**: MySQL 8.0+
- **任务调度**: Cron v3
- **会话管理**: Gorilla Sessions
- **前端**: HTML Templates + CSS + JavaScript

## 部署方式

### 环境要求

- Go 1.20 或更高版本
- MySQL 8.0 或更高版本
- DataX 已安装并配置
- Python 环境（用于执行 DataX）

### 1. 克隆项目

```bash
git clone <repository-url>
cd datax-web-go
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 配置数据库

创建数据库并执行初始化脚本：

```bash
mysql -u root -p < init.sql
```

### 4. 配置文件

复制并修改配置文件：

```bash
cp config.yaml config.yaml.local
```

编辑 `config.yaml.local`：

```yaml
# 数据库配置
db:
  host: 127.0.0.1
  port: 3306
  user: root
  pass: your_password
  name: datax_web

# 会话密钥（请修改为随机字符串）
session_key: your-secret-key-here

# 服务器端口
port: 8000

# DataX 安装目录
datax_home: /opt/datax

# 临时文件目录（DataX 作业配置文件存放位置）
temp_dir: /tmp/datax-web
```

### 5. 编译和运行

#### 开发环境

```bash
go run cmd/main.go config.yaml.local
```

#### 生产环境

```bash
# 编译
go build -o datax-web-go cmd/main.go

# 运行
./datax-web-go config.yaml.local
```

### 6. 使用 Docker 部署（推荐）

创建 `Dockerfile`：

```dockerfile
FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o datax-web-go cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates mysql-client python3 py3-pip
WORKDIR /root/

# 安装 DataX（需要根据实际情况调整）
# COPY datax /opt/datax

COPY --from=builder /app/datax-web-go .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/config.yaml .

EXPOSE 8000
CMD ["./datax-web-go"]
```

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  datax-web:
    build: .
    ports:
      - "8000:8000"
    environment:
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=root
      - DB_PASS=password
      - DB_NAME=datax_web
    depends_on:
      - mysql
    volumes:
      - ./config.yaml:/root/config.yaml
      - /opt/datax:/opt/datax  # DataX 安装目录
      - /tmp/datax-web:/tmp/datax-web

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: password
      MYSQL_DATABASE: datax_web
    volumes:
      - mysql_data:/var/lib/mysql
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "3306:3306"

volumes:
  mysql_data:
```

启动服务：

```bash
docker-compose up -d
```

### 7. 访问应用

打开浏览器访问：http://localhost:8000

默认管理员账户：
- 用户名：admin
- 密码：admin

**注意**: 首次登录后请立即修改默认密码。

## 项目结构

```
datax-web-go/
├── cmd/
│   └── main.go                 # 应用程序入口
├── internal/
│   ├── controllers/           # 控制器层
│   │   ├── auth.go           # 认证控制器
│   │   ├── controller.go     # 主控制器
│   │   ├── data_source.go    # 数据源控制器
│   │   ├── datax.go         # DataX 相关控制器
│   │   ├── ds_meta.go       # 数据源元数据控制器
│   │   ├── helpers.go       # 辅助函数
│   │   ├── log.go           # 日志控制器
│   │   ├── task_flow.go     # 任务流控制器
│   │   └── task.go          # 任务控制器
│   ├── models/
│   │   └── models.go        # 数据模型
│   ├── services/            # 服务层
│   │   ├── auth.go         # 认证服务
│   │   ├── scheduler.go    # 任务调度服务
│   │   └── datax/          # DataX 相关服务
│   │       ├── builder.go  # 配置构建器
│   │       ├── data_source.go
│   │       ├── service.go
│   │       ├── types.go
│   │       └── validator.go
│   └── util/
│       └── config.go       # 配置工具
├── static/                 # 静态资源
│   ├── css/
│   └── js/
├── templates/              # HTML 模板
│   ├── common/
│   ├── data_source/
│   ├── flow_log/
│   ├── task/
│   ├── task_log/
│   ├── taskflow/
│   ├── tools/
│   └── user/
├── config.yaml            # 配置文件
├── init.sql              # 数据库初始化脚本
├── go.mod               # Go 模块文件
└── go.sum              # Go 依赖锁定文件
```

## API 接口

### 认证相关
- `GET /login` - 显示登录页面
- `POST /login` - 处理登录
- `GET /logout` - 登出

### 任务管理
- `GET /tasks` - 任务列表
- `GET /tasks/new` - 新建任务页面
- `POST /tasks` - 创建任务
- `GET /tasks/:id` - 任务详情
- `POST /tasks/:id` - 更新任务
- `DELETE /tasks/:id` - 删除任务
- `POST /tasks/:id/run` - 执行任务

### 任务流管理
- `GET /task-flows` - 任务流列表
- `GET /task-flows/new` - 新建任务流页面
- `POST /task-flows` - 创建任务流
- `GET /task-flows/:id` - 任务流详情
- `POST /task-flows/:id` - 更新任务流
- `DELETE /task-flows/:id` - 删除任务流
- `POST /task-flows/:id/run` - 执行任务流
- `POST /task-flows/:id/toggle` - 启用/禁用任务流
- `POST /task-flows/:id/kill` - 终止任务流

### 数据源管理
- `GET /data-sources` - 数据源列表
- `POST /data-sources` - 创建数据源
- `GET /data-sources/:id` - 数据源详情
- `POST /data-sources/:id` - 更新数据源
- `DELETE /data-sources/:id` - 删除数据源
- `POST /data-sources/test` - 测试数据源连接

### 日志管理
- `GET /task-logs` - 任务日志列表
- `GET /task-logs/:id` - 任务日志详情
- `GET /flow-logs` - 任务流日志列表
- `GET /api/flow-logs` - 获取任务流日志（API）
- `GET /api/flow-logs/:id` - 获取任务流日志详情（API）

## 配置说明

### 数据库配置
- `db.host`: 数据库主机地址
- `db.port`: 数据库端口
- `db.user`: 数据库用户名
- `db.pass`: 数据库密码
- `db.name`: 数据库名称

### 应用配置
- `session_key`: 会话加密密钥
- `port`: Web 服务端口
- `datax_home`: DataX 安装目录
- `temp_dir`: 临时文件目录

## 开发指南

### 添加新的数据源类型

1. 在 `internal/services/datax/types.go` 中定义新的数据源类型
2. 在 `internal/services/datax/builder.go` 中实现配置构建逻辑
3. 在 `internal/controllers/data_source.go` 中添加相应的处理逻辑
4. 更新前端模板以支持新的数据源类型

### 添加新的任务类型

1. 在 `internal/services/datax/types.go` 中定义新的任务类型
2. 在 `internal/services/datax/builder.go` 中实现配置构建逻辑
3. 在 `internal/controllers/task.go` 中添加相应的处理逻辑

## 故障排除

### 常见问题

1. **数据库连接失败**
   - 检查数据库配置是否正确
   - 确认数据库服务是否运行
   - 检查网络连接

2. **DataX 执行失败**
   - 检查 DataX 是否正确安装
   - 确认 Python 环境是否可用
   - 检查临时目录权限

3. **任务调度不工作**
   - 检查 Cron 表达式是否正确
   - 确认任务流是否启用
   - 查看调度器日志

### 日志查看

应用日志会输出到标准输出，可以通过以下方式查看：

```bash
# 直接运行
go run cmd/main.go config.yaml.local

# 后台运行并记录日志
nohup ./datax-web-go config.yaml.local > app.log 2>&1 &
```

## 许可证

本项目采用 MIT 许可证。

## 贡献指南

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 联系方式

如有问题或建议，请通过以下方式联系：

- 提交 [Issue](https://github.com/fangx1999/datax-web-go/issues)
- 发送邮件至项目维护者
- 项目地址：[https://github.com/fangx1999/datax-web-go](https://github.com/fangx1999/datax-web-go)

---

## TODO List

### 功能增强
- [ ] 支持更多数据源类型（PostgreSQL, Oracle, MongoDB 等）
- [ ] 添加任务依赖关系管理
- [ ] 实现任务执行历史统计和报表
- [ ] 添加邮件通知功能
- [ ] 支持任务执行结果的数据质量检查
- [ ] 添加任务模板功能
- [ ] 实现任务配置的版本管理
- [ ] 添加任务执行性能监控

### 用户体验
- [ ] 优化前端界面，使用现代 UI 框架
- [ ] 添加任务执行进度条
- [ ] 实现实时日志流式显示
- [ ] 添加任务配置的图形化编辑器
- [ ] 支持任务配置的导入/导出
- [ ] 添加移动端适配

### 系统优化
- [ ] 添加 Redis 缓存支持
- [ ] 实现分布式任务调度
- [ ] 添加任务执行队列管理
- [ ] 优化数据库查询性能
- [ ] 添加系统监控和告警
- [ ] 实现配置热更新

### 安全增强
- [ ] 添加 API 访问限制
- [ ] 实现操作审计日志
- [ ] 添加数据源连接加密
- [ ] 支持 LDAP 认证集成
- [ ] 添加多租户支持

### 运维支持
- [ ] 添加健康检查接口
- [ ] 实现优雅关闭
- [ ] 添加 Prometheus 监控指标
- [ ] 支持 Kubernetes 部署
- [ ] 添加备份和恢复功能
