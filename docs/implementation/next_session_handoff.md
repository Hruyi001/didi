# 下一会话接力说明

## 当前分支
- `feat/microservices-migration`

## 最近已完成提交
- `3175e4f` `feat: route auth send-code through auth-service`
- `2849199` `feat: cut auth login over to auth-service`
- `cd7b417` `Add dual-track microservices MVP`

## 当前已完成状态
目前双轨迁移已经完成到 **第四阶段共享状态迁移与 admin 真实只读切流**：

1. `POST /api/auth/login` 已切到 `auth-service`
2. `POST /api/auth/send-code` 已切到 `auth-service`
3. 旧 `api` 在 Compose 中已使用 `STORE_BACKEND=mysql`，业务状态落到共享 MySQL
4. `admin-service` 已接入共享 MySQL 只读查询模型
5. gateway 仅将 `GET /api/admin/drivers` 切到 `admin-service`
6. 其他 `/api/*` 与 `/ws` 仍走旧 `api`
7. 旧 `api`、`auth-service`、`admin-service` 共享同一演示 token secret
8. Docker Compose 中 `gateway + api + auth-service + admin-service + frontend + mysql + redis + kafka` 已跑通

## 第四阶段关键交付

### 共享状态边界
- 新增 `store.Store` 合同测试，约束 `MemoryStore` 与 `MySQLStore` 一致行为。
- 新增 `backend/internal/store/mysql_store.go` 与 `mysql_store_test.go`。
- MySQL 版 store 已覆盖乘客、司机、车辆、位置、订单、派单、支付、评价。
- `infra/schema.sql` 已补充 driver stats、driver_locations、vehicles/reviews 唯一约束。

### 旧 api 切到共享 MySQL
- `backend/cmd/api/main.go` 支持：
  - `STORE_BACKEND=memory` 或空：继续使用内存存储。
  - `STORE_BACKEND=mysql`：要求 `MYSQL_DSN`，使用 `MySQLStore`。
  - `SEED_DEMO_DATA=true`：在 MySQL 模式下种入演示乘客与司机。
- `SeedDemoDataForStore` 会返回种子状态错误，避免静默失败。

### admin-service 真实只读切流
- 新增 `backend/internal/admin/repo.go`：从 MySQL 读取司机与车辆快照。
- 新增 `backend/internal/admin/auth.go`：服务自身校验 ADMIN token，不依赖 gateway 代鉴权。
- `backend/internal/admin/http.go` 对 repository 错误只返回通用错误，不泄露 raw DB error。
- `backend/cmd/admin-service/main.go` 要求 `AUTH_TOKEN_SECRET` 与 `ADMIN_MYSQL_DSN` 非空。
- `backend/internal/gateway/router.go` 只将 `GET /api/admin/drivers` 转发到 `ADMIN_BASE`，admin 写接口仍在旧 `api`。

### 最终审查修复
- `DispatchService` 的 accept/reject/timeout/redispatch 历史记录会保留原 `DispatchTaskID`。
- `CreatePayment` 按 `order_id` 幂等，重复调用返回已有支付单，不再因唯一键冲突 panic。
- `AddDispatchAttempt` 改为返回 `(DispatchAttempt, error)`，司机并发不可抢占时返回错误而不是 panic。

## 已验证结论

### 单元与集成测试
已验证通过：
```bash
go -C /root/didi/backend test ./...
npm --prefix /root/didi/frontend run build
docker compose -f /root/didi/infra/docker-compose.yml config
```

### 容器联调
已验证通过：
```bash
DOCKER_HOST=unix:///tmp/dockerd-vfs.sock docker compose -f /root/didi/infra/docker-compose.yml up --build -d
DOCKER_HOST=unix:///tmp/dockerd-vfs.sock docker compose -f /root/didi/infra/docker-compose.yml ps
curl http://localhost:8080/health
curl -I http://localhost:8081/
go -C /root/didi/backend test ./integration
```

当前服务健康状态已确认：`api`、`auth-service`、`admin-service`、`gateway`、`frontend`、`mysql`、`redis`、`kafka` 均运行。

## 当前运行拓扑
- `frontend` 对外：`8081`
- `gateway` 对外：`8080`
- `api` 仅容器内，使用 MySQL store
- `auth-service` 仅容器内
- `admin-service` 仅容器内，读取 MySQL
- `mysql`、`redis`、`kafka` 仍作为基础设施服务

## 当前最重要的判断
**第四阶段已经解决“假迁移”的核心风险：旧 `api` 与新 `admin-service` 现在通过 MySQL 共享业务状态。**

后续不要再切没有共享状态或真实读写模型支撑的路由。下一个真实可切流切片应继续遵循：
1. 先把旧 `api` 的相关状态落到共享边界。
2. 新服务直接读取或写入同一共享边界。
3. gateway 只切一小段能端到端验证的真实流量。

## 下一阶段建议优先做的事情

建议规划 **第五阶段：司机侧真实读切流或订单读模型切流**，二选一：

### 方向 A：driver-service 司机只读/状态切片
优先切相对独立的司机读接口，例如：
- `GET /api/driver/profile`
- `GET /api/driver/orders`

需要先评估司机上线/下线、位置上报、接单动作是否仍留在旧 `api`，避免写流量过早拆分。

### 方向 B：order-service 乘客订单只读切片
优先切乘客订单只读查询，例如：
- `GET /api/passenger/orders`

需要确保订单列表语义、鉴权身份解析、派单可见性与旧 `api` 完全一致。

当前更推荐 **方向 A 的司机只读切片**，因为 driver profile/orders 已经主要依赖共享 MySQL 中的 driver/order/dispatch 状态，且比乘客下单/支付/评价写链路风险更低。

## 新会话建议的第一步问题
新会话里建议直接让 Claude 做这件事：

> 基于当前第四阶段成果，规划第五阶段迁移：优先评估 `driver-service` 是否可以真实承接司机只读接口（profile/orders），并继续遵循共享 MySQL 状态边界与小流量切片原则。

## 建议重点查看的文件

### 第四阶段共享状态与 admin 切流
- `backend/internal/store/store.go`
- `backend/internal/store/store_contract_test.go`
- `backend/internal/store/memory.go`
- `backend/internal/store/mysql_store.go`
- `backend/internal/store/mysql_store_test.go`
- `backend/internal/store/seed.go`
- `backend/cmd/api/main.go`
- `backend/internal/admin/auth.go`
- `backend/internal/admin/repo.go`
- `backend/internal/admin/http.go`
- `backend/internal/admin/service.go`
- `backend/cmd/admin-service/main.go`
- `backend/internal/gateway/router.go`
- `backend/internal/gateway/router_test.go`
- `backend/integration/order_dispatch_flow_test.go`
- `infra/schema.sql`
- `infra/docker-compose.yml`

### 下一阶段 driver/order 切片候选
- `backend/internal/http/handlers_driver.go`
- `backend/internal/http/handlers_order.go`
- `backend/internal/services/dispatch.go`
- `backend/internal/services/order.go`
- `backend/internal/driver/*`
- `backend/internal/order/*`
- `backend/cmd/driver-service/main.go`
- `backend/cmd/order-service/main.go`

## 当前未提交文件
工作区仍有一个早于本阶段存在的未跟踪文件：
- `GATEWAY_TEST_COVERAGE_REPORT.md`

本阶段新增的设计、计划和代码文件也仍未提交。不要自动提交，除非用户明确要求。

## 如果要重启联调环境
在当前 Ubuntu-in-Docker 环境里继续使用：
```bash
DOCKER_HOST=unix:///tmp/dockerd-vfs.sock docker compose -f /root/didi/infra/docker-compose.yml up --build -d
```

查看状态：
```bash
DOCKER_HOST=unix:///tmp/dockerd-vfs.sock docker compose -f /root/didi/infra/docker-compose.yml ps
```

查看日志：
```bash
DOCKER_HOST=unix:///tmp/dockerd-vfs.sock docker compose -f /root/didi/infra/docker-compose.yml logs gateway auth-service admin-service api --tail 100
```

## 注意事项
- 如果本地 MySQL 复用旧数据目录，`infra/schema.sql` 不会重新执行。缺少 `driver_profiles.accepted_count/rejected_count/timeout_count` 时按 README 中的 `ALTER TABLE` 处理，除非用户明确允许，否则不要执行 `down -v` 删除数据卷。
- `admin-service` 启动时必须提供 `AUTH_TOKEN_SECRET`，不能回退到默认 secret。
- gateway 目前只切 `GET /api/admin/drivers` 到 `admin-service`，不要误把 admin approve 等写接口切过去。
