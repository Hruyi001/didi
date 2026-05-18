# 类滴滴打车平台

这是一个类滴滴打车系统的第一版实用型 MVP，包含乘客端 H5、司机端 H5、管理后台、Go 后端 API、MySQL schema、Redis/Kafka/Docker Compose 基础设施配置。

## 功能范围

- 乘客端：手机号验证码登录、地图模拟选点、呼叫快车、订单状态查看、司机实时位置查看、取消、模拟支付、评价。
- 司机端：司机登录、司机信息、司机侧订单查询、上线/下线、接单/拒单、到达、开始行程、结束行程、接单统计、模拟位置上报。
- 管理后台：管理员登录、司机列表、订单查询、异常订单、基础运营统计。
- 后端：基于 token 的基础会话鉴权、订单主状态机、自动派单与超时改派、派单记录、支付模拟、基础短信频控、WebSocket 订单/司机位置实时广播。

## 技术栈

- 前端：Vue 3 + TypeScript + Vite
- 后端：Go 1.25+，当前采用双轨迁移形态：对外入口为 gateway，旧 `api` 单体入口保留在内部网络，新 auth/user/driver/location/order/dispatch/payment/realtime/admin 服务已提供独立二进制骨架
- 数据与基础设施：MySQL、Redis、Kafka、Docker Compose

## 启动后端

如果只验证旧 `api` 单体入口，可直接运行：

```bash
cd backend
go mod tidy
go run ./cmd/api
```

也可以从项目外部直接运行：

```bash
go -C /root/didi/backend run ./cmd/api
```

此模式下后端默认监听：`http://localhost:8080`

健康检查：

```bash
curl http://localhost:8080/health
```

## 启动前端

```bash
cd frontend
npm install
npm run dev
```

访问：

- 乘客端：`http://localhost:5173/?app=passenger`
- 司机端：`http://localhost:5173/?app=driver`
- 管理后台：`http://localhost:5173/?app=admin`

演示验证码固定为：`123456`

说明：三端登录成功后会把 access token 保存到浏览器本地存储，并自动用于后续 HTTP 请求和 WebSocket 实时连接；页面登录成功后会立即建立实时连接，页面离开时会自动关闭连接。乘客、司机、管理员接口都会按当前登录手机号解析真实身份，司机不能访问乘客专属订单接口，乘客也只会看到自己的订单。

## Docker Compose

当前推荐启动完整双轨演示环境：gateway 对外暴露 `8080`，旧 `api` 保留在 Compose 内部网络，前端容器统一通过 gateway 访问 `/api/*` 和 `/ws`。

```bash
docker compose -f infra/docker-compose.yml up --build -d
```

或在任意目录显式指定绝对路径：

```bash
docker compose -f /root/didi/infra/docker-compose.yml up --build -d
```

如果当前 Ubuntu 运行在 Docker 容器中，默认 dockerd 可能因为 overlayfs 无法正常 build/run。此时可先在当前 Ubuntu 实例内部启动一个 `vfs` daemon，再显式指定 `DOCKER_HOST`：

```bash
nohup dockerd \
  --host=unix:///tmp/dockerd-vfs.sock \
  --data-root=/tmp/dockerd-vfs-data \
  --exec-root=/tmp/dockerd-vfs-exec \
  --pidfile=/tmp/dockerd-vfs.pid \
  --storage-driver=vfs \
  >/tmp/dockerd-vfs.log 2>&1 &

DOCKER_HOST=unix:///tmp/dockerd-vfs.sock docker compose -f /root/didi/infra/docker-compose.yml up --build -d
```

启动后可直接访问：

- 乘客端：`http://localhost:8081/?app=passenger`
- 司机端：`http://localhost:8081/?app=driver`
- 管理后台：`http://localhost:8081/?app=admin`
- gateway 健康检查：`http://localhost:8080/health`

当前 Compose 会同时启动 MySQL、Redis、Kafka、旧 `api`、gateway 和前端静态站点。浏览器只需要访问前端入口，前端容器会把 `/api/*` 与 `/ws` 代理到 Compose 网络里的 `gateway:8080`，gateway 再把请求转发到内部 `api:8080`。

如需仅做配置校验：

```bash
docker compose -f infra/docker-compose.yml config
```

如需本地开发模式，仍可单独启动前后端：后端使用 `go run ./cmd/api`，前端使用 `npm run dev`。

## 演示流程

1. 打开乘客端，使用任意手机号和验证码 `123456` 登录。
2. 点击“呼叫快车”。
3. 打开司机端，若首位司机拒单或长时间不处理，系统会自动改派到下一位在线司机。
4. 司机端接单后会自动定时上报模拟位置，也可点击“立即上报位置”；回到乘客端可看到司机实时位置与速度。
5. 依次点击“已到达”、“开始行程”、“结束行程”。
6. 回到乘客端，点击“模拟支付”。
7. 点击“评价”。
8. 打开管理后台查看订单状态和运营统计。

## 关键文档

- 微服务架构设计：`docs/superpowers/specs/2026-05-18-ride-hailing-microservices-architecture-design.md`
- 微服务迁移实施计划：`docs/superpowers/plans/2026-05-18-ride-hailing-microservices-architecture.md`
- 实施过程记录：`docs/implementation/progress.md`

## 验证命令

```bash
go -C /root/didi/backend test ./...
npm --prefix /root/didi/frontend install
npm --prefix /root/didi/frontend run build
docker compose -f /root/didi/infra/docker-compose.yml config
DOCKER_HOST=unix:///tmp/dockerd-vfs.sock docker compose -f /root/didi/infra/docker-compose.yml up --build -d
curl http://localhost:8080/health
curl -I http://localhost:8081/
python3 - <<'PY'
import json, urllib.request
req = urllib.request.Request(
    'http://localhost:8080/api/auth/login',
    data=json.dumps({'phone':'13800000001','code':'123456','role':'PASSENGER'}).encode(),
    headers={'Content-Type':'application/json'},
    method='POST'
)
with urllib.request.urlopen(req) as r:
    print(r.status)
    print(r.read().decode())
PY
DOCKER_HOST=unix:///tmp/dockerd-vfs.sock docker compose -f /root/didi/infra/docker-compose.yml ps
```

## 第一版限制

- 当前双轨版本对外入口已经切到 gateway，但业务请求仍主要由内部旧 `api` 处理，新拆分服务目前提供独立二进制与模块骨架，尚未全部接入真实流量。
- 当前运行时主要使用内存存储，`infra/schema.sql` 提供 MySQL 生产形态 schema。
- Kafka 和 Redis 已进入部署与架构边界，第一版核心链路仍以进程内同步调用跑通，后续可逐步切到事件驱动。
- 高德地图在第一版前端中以地图模拟区域呈现，后续可接入真实高德 JS SDK 和后端地图适配层。
- WebSocket 当前已支持订单状态、司机位置实时广播与心跳；更细粒度的轨迹回放、按角色定向推送和 Kafka 驱动事件扇出可作为下一迭代补充。
- 司机端第一版使用种子司机，完整司机认证表单可在下一迭代增强。
