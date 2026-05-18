# Docker Compose 一键演示环境设计

## 背景

当前项目已经具备可运行的 Go 后端、Vue 三端前端、基础设施编排和文档，但 `infra/docker-compose.yml` 仍只覆盖 MySQL、Redis、Kafka 和 API。前端仍需要在宿主机手动执行 `npm run dev`，这使交付体验停留在“开发者可运行”，还没有达到“用户一条命令即可打开完整系统”的目标。

本设计聚焦于把现有 Compose 提升到完整演示环境：执行 `docker compose -f infra/docker-compose.yml up --build` 后，用户即可直接访问乘客端、司机端、管理后台和后端 API，不需要额外手动启动前端开发服务。

## 目标

- 把前端纳入 Docker Compose，形成完整演示环境。
- 保持现有三端入口形式：`/?app=passenger`、`/?app=driver`、`/?app=admin`。
- 让浏览器只面向一个前端入口地址访问。
- 让前端继续使用相对路径 `/api/*` 和 `/ws`，不把宿主机地址写死到构建产物里。
- 保留 API 容器对外暴露，便于健康检查、调试和 curl 验证。
- 更新 README 和验证命令，使“一键启动”成为文档主路径。

## 非目标

- 不把运行时从内存存储切换到 MySQL 持久化。
- 不在本次设计中把 Kafka 或 Redis 接入业务主链路。
- 不把三端前端拆成三个独立项目或三个独立镜像。
- 不引入 Kubernetes、Traefik、Ingress 或复杂服务发现机制。
- 不在本次设计中加入 HTTPS、域名、证书或生产级安全加固。

## 推荐方案

采用“一个前端静态容器 + 一个 API 容器 + 基础设施容器”的结构。

前端通过多阶段 Dockerfile 先构建 Vue 静态资源，再使用 Nginx 提供站点和反向代理。浏览器只访问前端容器；前端容器内部把 `/api/` 和 `/ws` 转发到 Compose 网络中的 `api:8080`。

这是当前最适合项目阶段的方案，因为它兼顾三点：

1. **交付感强**：用户只需要起 Compose 并打开浏览器。
2. **对现有代码侵入小**：前端仍使用相对路径，不需要重写 API 地址解析逻辑。
3. **部署形态合理**：静态资源与 API 边界清晰，比把 Vite dev server 直接放进 Compose 更接近真实可交付模式。

## 备选方案与取舍

### 方案 A：前端静态容器 + Nginx 反代 API/WS（推荐）

优点：
- 最接近真实部署形态
- 用户入口单一
- 兼容当前前端相对路径调用方式
- 镜像职责清晰，运行时更轻

缺点：
- 需要新增前端 Dockerfile 和 Nginx 配置
- 需要额外维护一个静态站点容器

### 方案 B：把 Vite dev server 放进 Compose

优点：
- 改造更快
- 开发体验接近本地模式

缺点：
- 更像开发环境而不是交付环境
- 容器更重，启动链路更脆弱
- 对“一键可运行交付”的观感较差

### 方案 C：为乘客/司机/管理端拆成三个前端容器

优点：
- 入口语义最直接
- 每端可以独立部署

缺点：
- 当前阶段明显过度设计
- 增加镜像、路由和文档复杂度
- 与现有 `?app=` 切换模式不匹配

## 架构设计

### 服务拓扑

Compose 中保留以下服务：

- `mysql`
- `redis`
- `kafka`
- `api`
- `frontend`

职责如下：

- `frontend`：托管 Vue 构建产物，对外暴露浏览器入口；代理 `/api/*` 和 `/ws`
- `api`：Go 后端，提供 HTTP API、WebSocket、健康检查
- `mysql` / `redis` / `kafka`：保留为当前架构边界中的基础设施

### 浏览器访问路径

用户统一访问前端容器，例如：

- `http://localhost:8081/?app=passenger`
- `http://localhost:8081/?app=driver`
- `http://localhost:8081/?app=admin`

是否继续使用 `8081` 还是改为 `80`，实现时可根据本地权限和冲突风险决定。默认建议继续使用非特权端口，减少本地环境摩擦。

### 请求流向

- 浏览器访问 `frontend`
- `frontend` 处理静态资源与单页应用路由
- `frontend` 将 `/api/*` 转发到 `api:8080`
- `frontend` 将 `/ws` 转发到 `api:8080`

这样前端构建产物不需要感知宿主机地址，也不需要按环境注入不同 API Base URL。

## 容器与配置设计

### 前端 Dockerfile

新增前端多阶段 Dockerfile：

1. `node` 阶段
   - 安装依赖
   - 执行 `npm run build`
   - 产出 `dist/`
2. `nginx` 阶段
   - 复制 `dist/`
   - 复制自定义 Nginx 配置
   - 暴露前端端口

### Nginx 配置

需要一份最小配置，完成三件事：

1. `/` 提供静态文件
2. 未命中静态文件时回退 `index.html`，兼容单页应用
3. `/api/` 与 `/ws` 代理到 `api:8080`

WebSocket 代理必须显式透传升级头，以保证当前登录后实时连接链路继续工作。

### API Dockerfile

保留当前 `backend/Dockerfile` 的基础结构即可。为了配合 Compose 可用性增强，建议在 Compose 层补健康检查，而不是大幅改动镜像构建逻辑。

### Compose 依赖关系

- `api` 依赖 `mysql`、`redis`、`kafka`
- `frontend` 依赖 `api`

本次只使用 Compose 级 `depends_on` 和健康检查，不引入复杂等待脚本。目标是做到“完整演示环境可启动”，而不是实现生产级启动编排器语义。

## 就绪性与可用性设计

### API 健康检查

为 `api` 添加基于 `/health` 的健康检查，用于：

- 提升 Compose 状态可读性
- 让验证命令有稳定入口
- 为后续如果要用 `depends_on.condition: service_healthy` 留出空间

### 前端对后端的容错

前端不直接硬编码后端地址，只依赖容器内代理目标 `api:8080`。这样即使 API 稍晚启动，前端容器本身仍可成功启动，用户刷新页面后即可正常访问。

### 基础设施服务

MySQL、Redis、Kafka 暂时继续作为已编排依赖存在。虽然当前业务主链路没有真正依赖它们的运行时数据，但保留它们符合项目既定边界，也让 Compose 更接近目标系统结构。

## 文档与验证链路

### README 调整

README 的运行主路径应从“本地分别启动前后端”升级为：

1. 一条命令启动完整演示环境
2. 浏览器访问前端入口
3. 保留独立启动前后端作为开发模式说明

### 验证命令

文档中的推荐验证链路应至少包含：

- `docker compose -f infra/docker-compose.yml config`
- `docker compose -f infra/docker-compose.yml up --build -d`
- `curl http://localhost:8080/health`
- 访问 `http://localhost:<frontend-port>/?app=passenger`
- 访问 `http://localhost:<frontend-port>/?app=driver`
- 访问 `http://localhost:<frontend-port>/?app=admin`

如果环境允许，还应补充：

- `docker compose -f infra/docker-compose.yml ps`
- `docker compose -f infra/docker-compose.yml logs api frontend --tail=100`

### 用户体验结果

完成后，交付体验将变成：

1. `docker compose up --build`
2. 打开浏览器
3. 在三端中完成演示流程

这就是本次设计最核心的成功标准。

## 测试策略

### 配置验证

- `docker compose config` 必须通过
- 前端镜像必须可构建
- 后端镜像必须可构建

### 基础可用性验证

- API 健康检查返回正常
- 前端首页可访问
- `?app=passenger`、`?app=driver`、`?app=admin` 三端入口可加载
- `/api/*` 调用通过前端代理可到达 API
- `/ws` 可通过前端代理建立连接

### 回归验证

继续保留：

- `go -C /root/didi/backend test ./...`
- `npm --prefix /root/didi/frontend run build`

Compose 增强不能破坏当前已经完成的后端测试和前端构建链路。

## 风险与约束

### 端口选择

若前端使用 80/443，某些本地环境会遇到权限或冲突问题。默认使用高位端口更稳妥。

### 基础设施“存在但未深度使用”

当前 Compose 仍会启动 MySQL、Redis、Kafka，但业务主链路主要还是内存存储。这是当前版本已知约束，不应在本次范围内扩展为真正持久化改造。

### 浏览器自动化限制

如果本地缺少 Playwright 或浏览器二进制，最终验证可能仍然只能依靠构建、健康检查和人工访问步骤。文档需要明确把这视为环境限制，而不是代码缺陷。

## 实施范围边界

本设计适合单次实现计划，不需要再拆分为多个子规格。实现改动预计集中在：

- `infra/docker-compose.yml`
- `frontend/` 下新增 Dockerfile 与 Nginx 配置
- `README.md`
- `docs/implementation/progress.md`

不应在本次实现中顺手引入生产级持久化、真实域名、TLS 或复杂反向代理拓扑。
