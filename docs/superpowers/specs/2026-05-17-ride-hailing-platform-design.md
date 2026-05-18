# 类滴滴打车平台第一版设计规格

## 1. 目标与范围

第一版定位为“实用型微服务 MVP”，不是玩具 Demo。目标是在本地或单机环境中运行出一个较完整、可试用的打车系统，覆盖乘客叫车、司机接单、自动派单、行程状态流转、模拟支付、评价、司机审核和后台运营处理。

第一版只支持快车业务，不支持专车、拼车、出租车等复杂车型。系统需要保留后续升级空间，包括真实支付、高并发派单、复杂风控、实时定位优化、监控链路追踪和 Kubernetes 部署。

## 2. 技术栈

### 前端

- Vue 3 + TypeScript + Vite
- 乘客端 H5：移动端优先
- 司机端 H5：移动端优先
- 管理后台：桌面端优先

### 后端

- Go + Kratos
- gRPC/HTTP 服务接口
- API Gateway/BFF 作为统一入口
- WebSocket 用于实时通信

### 基础设施

- MySQL：核心业务数据
- Redis：验证码、会话、司机位置、派单锁、在线状态
- Kafka：订单、派单、行程、支付、通知等业务事件
- Docker Compose：本地和单机部署
- 高德地图：地图展示、点位选择、地理编码、逆地理编码、路线和距离能力

## 3. 总体架构

系统分为三端前端、网关层、业务微服务和基础设施层。

三端前端包括乘客端、司机端和管理后台。乘客端和司机端面向移动端浏览器，管理后台面向桌面浏览器。

网关层包括 API Gateway/BFF 和 WebSocket Gateway。API Gateway/BFF 负责鉴权、路由、限流和接口聚合；WebSocket Gateway 负责维护客户端连接并推送订单状态、派单通知和司机位置。

业务微服务包括：

- Auth Service：手机号短信、JWT、Refresh Token、角色权限、设备会话
- User Service：乘客资料、常用地址
- Driver Service：司机资料、车辆信息、司机审核、司机工作状态
- Order Service：订单主状态、费用、取消、评价
- Dispatch Service：候选司机、自动派单、超时改派、派单历史
- Location/Map Service：司机位置、高德地图封装、距离和路线能力
- Payment Service：模拟支付单、支付流水、退款状态
- Notify/Realtime Service：通知事件、WebSocket 消息投递
- Admin Service：后台聚合接口、司机审核、异常订单处理

同步查询和关键命令通过 HTTP/gRPC 调用；异步业务事件通过 Kafka 传播；客户端实时更新通过 WebSocket 推送。

## 4. 核心业务流程

### 4.1 乘客流程

1. 乘客使用手机号短信登录。
2. 在地图上选择起点和终点。
3. 系统调用地图能力获取距离、时间，并生成预估价格。
4. 乘客确认呼叫快车。
5. Order Service 创建订单，进入派单流程。
6. Dispatch Service 按距离选择附近可用司机，并向司机端推送派单通知。
7. 司机接单后，乘客端展示司机信息、车辆信息和司机位置。
8. 司机到达上车点后，乘客端收到状态更新。
9. 司机开始行程，订单进入行程中。
10. 司机结束行程，系统生成最终金额。
11. 乘客完成模拟支付。
12. 订单完成，乘客可评价司机和行程。

### 4.2 司机流程

1. 司机使用手机号短信登录。
2. 司机提交身份资料和车辆资料。
3. 管理后台审核通过后，司机可上线接单。
4. 司机上线后持续上报位置。
5. 系统向司机推送派单通知，司机可接单或拒单。
6. 司机接单后前往乘客上车点。
7. 司机标记到达、开始行程、结束行程。
8. 系统记录司机收入和订单完成情况。

### 4.3 管理后台流程

1. 管理员账号密码登录。
2. 审核司机和车辆资料。
3. 查询乘客、司机、订单和支付流水。
4. 查看异常订单并进行人工处理。
5. 查看基础运营统计和后台操作日志。

## 5. 状态机设计

状态机拆分为订单主状态、派单状态、支付状态、评价状态和司机工作状态，避免一个字段承载过多业务含义。

### 5.1 订单主状态 OrderStatus

- CREATED：订单已创建
- DISPATCHING：派单中
- WAITING_PICKUP：司机已接单，等待接驾
- DRIVER_ARRIVED：司机已到达上车点
- IN_PROGRESS：行程中
- WAITING_PAYMENT：行程已结束，待支付
- COMPLETED：订单已完成
- CANCELED：订单已取消
- DISPATCH_FAILED：派单失败

最终订单态只包括 COMPLETED、CANCELED、DISPATCH_FAILED。

### 5.2 派单状态 DispatchStatus

派单状态记录在 DispatchTask 和 DispatchAttempt 中，用于保留每次派单尝试的历史。

- PENDING：待派单
- OFFERED：已派给某个司机，等待响应
- ACCEPTED：司机已接单
- REJECTED：司机已拒单
- TIMEOUT：司机响应超时
- CANCELED：派单取消
- FAILED：无可用司机或达到最大派单次数

派单流程为最近司机优先。司机拒单或超时后，系统继续派给下一个候选司机。每一次派单尝试都记录司机、距离、发起时间、响应时间、结果和序号。

### 5.3 支付状态 PaymentStatus

- UNPAID：未支付
- PAYING：支付中
- PAID：已支付
- FAILED：支付失败
- REFUNDING：退款中
- REFUNDED：已退款
- CLOSED：支付关闭

第一版为模拟支付，但仍保留支付单和支付流水，便于后续接入真实支付渠道。

### 5.4 评价状态 ReviewStatus

- NOT_REVIEWED：未评价
- REVIEWED：已评价
- EXPIRED：评价过期

评价是订单完成后的附加行为，不影响订单主状态是否完成。

### 5.5 司机工作状态 DriverWorkStatus

- OFFLINE：离线
- ONLINE_IDLE：在线空闲
- DISPATCHED：被派单中
- SERVING：服务中
- SUSPENDED：暂停接单

司机未审核通过时不能上线。司机在 DISPATCHED 或 SERVING 状态下不能收到新的订单。

## 6. 关键业务规则

### 6.1 派单规则

- 只派给审核通过、在线空闲、位置有效的司机。
- 候选司机按距离上车点从近到远排序。
- 每次只向一个司机发起派单。
- 司机在倒计时内未响应则记为 TIMEOUT。
- 司机拒单则记为 REJECTED。
- 超时或拒单后继续派给下一个候选司机。
- 达到最大尝试次数或无候选司机时，订单变为 DISPATCH_FAILED。

### 6.2 取消规则

- 派单成功前，乘客可取消订单，不计入接单后取消统计。
- 司机接单后，乘客仍可取消，但记录接单后取消次数。
- 行程中不允许普通取消，只能由后台异常处理。
- 司机接单后取消服务需要记录司机异常行为。

### 6.3 并发和幂等

- 订单表使用 version 字段做乐观锁。
- 司机接单、拒单、超时改派、乘客取消必须使用条件更新。
- 派单使用 Redis 短锁，避免同一司机同时收到多个订单。
- Kafka 消费者必须幂等，重复事件不能重复修改业务状态。
- 司机接单和系统超时可能同时发生，以数据库状态条件更新结果为准。

## 7. 服务拆分与数据边界

### 7.1 服务职责

API Gateway/BFF 不保存业务真相，只负责入口、安全、路由和聚合。

Auth Service 拥有账号、角色、权限、设备会话和 Token 生命周期。

User Service 拥有乘客资料和常用地址。

Driver Service 拥有司机资料、车辆资料、审核状态和司机工作状态。

Order Service 拥有订单主状态、起终点、费用、取消记录和评价关联。

Dispatch Service 拥有派单任务、派单尝试、候选司机排序和改派逻辑。

Location/Map Service 拥有司机实时位置写入能力，并封装高德地图接口。

Payment Service 拥有支付单、支付流水和退款状态。

Notify/Realtime Service 不拥有业务真相，只消费业务事件并推送客户端消息。

Admin Service 提供后台聚合查询、审核和异常处理能力，并记录后台操作日志。

### 7.2 数据边界

第一版可共用一个 MySQL 实例，但各服务按库或表前缀隔离。服务只能直接写自己的业务表，跨服务数据通过 API 或事件获得。

Redis 用于：

- 短信验证码
- Refresh Token 会话或黑名单
- 司机实时位置
- 派单锁
- WebSocket 在线状态
- 基础限流计数

Kafka 事件包括：

- OrderCreated
- DispatchRequested
- DispatchOffered
- DriverAccepted
- DriverRejected
- DispatchTimedOut
- TripStarted
- TripEnded
- PaymentPaid
- NotificationRequested

## 8. 关键数据模型

### 8.1 认证与用户

- Account：账号、手机号、角色、状态
- AuthSession：Refresh Token、过期时间、撤销状态
- DeviceSession：设备标识、登录 IP、最近活跃时间
- PassengerProfile：乘客昵称、头像、状态
- PassengerAddress：常用地址

### 8.2 司机与车辆

- DriverProfile：司机姓名、手机号、证件信息、审核状态、工作状态
- Vehicle：车牌号、车型、颜色、车辆证件、审核状态
- DriverAuditRecord：审核结果、原因、操作人

### 8.3 订单

RideOrder 关键字段：

- order_id
- passenger_id
- driver_id
- pickup_name、pickup_lng、pickup_lat
- dropoff_name、dropoff_lng、dropoff_lat
- status
- estimated_distance
- estimated_duration
- estimated_amount
- final_amount
- cancel_reason
- created_at、accepted_at、arrived_at、started_at、ended_at、paid_at
- version

### 8.4 派单

DispatchTask 关键字段：

- dispatch_task_id
- order_id
- status
- candidate_count
- current_attempt_no
- max_attempts
- created_at
- updated_at

DispatchAttempt 关键字段：

- attempt_id
- dispatch_task_id
- order_id
- driver_id
- status
- distance_to_pickup
- offered_at
- responded_at
- timeout_at
- reject_reason
- sequence_no

### 8.5 支付与评价

PaymentOrder 记录订单金额、支付状态、支付方式、支付时间。

PaymentTxn 记录每一次模拟支付流水、流水号、金额、结果和失败原因。

Review 记录订单评价、乘客评分、司机评分、评价内容和评价状态。

### 8.6 管理后台

AdminUser 记录管理员账号、密码摘要、角色和状态。

AdminActionLog 记录管理员操作对象、操作类型、操作前后状态和时间。

## 9. 前端功能设计

### 9.1 乘客端 H5

- 手机号短信登录
- 地图定位和选点
- 起终点搜索
- 价格、距离、时间预估
- 呼叫快车
- 派单等待页
- 司机信息和车辆信息展示
- 司机实时位置展示
- 取消订单
- 行程状态展示
- 模拟支付
- 评价
- 历史订单
- 常用地址

### 9.2 司机端 H5

- 手机号短信登录
- 司机认证资料提交
- 车辆资料提交
- 上线和下线
- 位置上报
- 派单弹窗和倒计时
- 接单和拒单
- 到达上车点
- 开始行程
- 结束行程
- 收入记录
- 接单、拒单、超时统计

### 9.3 管理后台

- 管理员登录
- 司机审核
- 用户列表
- 司机列表
- 订单查询
- 异常订单处理
- 支付流水查询
- 基础运营统计
- 后台操作日志

## 10. 地图、实时通信与通知

高德地图能力通过 Location/Map Service 和前端地图 SDK 封装。前端负责地图展示和交互，后端负责地理编码、逆地理编码、路线距离、价格预估需要的距离和时间数据。

WebSocket 用于：

- 乘客端订单状态推送
- 乘客端司机位置推送
- 司机端派单通知
- 司机端订单状态推送
- 管理后台异常订单提示

Notify/Realtime Service 消费 Kafka 事件后，根据用户、司机或管理员在线状态推送 WebSocket 消息。

## 11. 支付模拟

第一版不接入真实支付宝或微信支付，但支付模型按真实支付流程设计。

支付流程：

1. 行程结束后，Order Service 生成最终金额并进入 WAITING_PAYMENT。
2. Payment Service 创建支付单。
3. 乘客发起模拟支付。
4. Payment Service 生成支付流水。
5. 支付成功后发布 PaymentPaid 事件。
6. Order Service 消费事件后将订单变为 COMPLETED。

支付失败时，订单保持 WAITING_PAYMENT，乘客可重试支付。

## 12. 基础风控

第一版实现基础风控，不引入完整规则引擎。

规则包括：

- 短信验证码发送频率限制
- 短信验证码每日上限
- 登录失败次数限制
- 异常设备会话记录
- 乘客接单后取消次数统计
- 司机拒单次数统计
- 司机超时次数统计
- 未审核通过司机禁止上线
- 被暂停司机禁止接单

风控数据保持结构化，便于后续升级为独立风控服务。

## 13. 错误处理

系统使用统一错误码和统一响应结构。

错误类型包括：

- 认证失败
- Token 过期
- 权限不足
- 参数错误
- 资源不存在
- 业务状态冲突
- 司机不可接单
- 订单不可取消
- 支付状态错误
- 地图服务失败
- 系统内部错误

状态冲突必须返回明确原因。例如订单已取消、司机响应已超时、订单不是待支付状态。

高德地图调用失败时，前端提示用户重试，不静默生成错误路线或错误价格。

## 14. 可观测性

第一版只实现基础结构化日志，不引入 Prometheus、Grafana、Jaeger 或日志采集系统。

每个服务至少记录：

- 请求 ID
- 用户或司机 ID
- 订单 ID
- 错误码
- 接口耗时
- 关键状态变更
- Kafka 事件消费结果

日志格式保持一致，便于后续接入完整可观测性系统。

## 15. 测试策略

### 15.1 单元测试

覆盖：

- 订单状态机
- 派单候选排序
- 计价规则
- 取消规则
- 支付状态流转
- 风控规则

### 15.2 服务测试

覆盖：

- 认证接口
- 司机审核接口
- 订单创建接口
- 派单接口
- 接单和拒单接口
- 行程开始和结束接口
- 支付接口
- 管理后台异常处理接口

### 15.3 集成测试

覆盖 MySQL、Redis 和 Kafka 的真实交互，尤其是派单锁、订单状态条件更新、Kafka 重复消费幂等。

### 15.4 端到端测试

必须覆盖：

- 正常叫车、接单、行程、支付、评价闭环
- 无司机导致派单失败
- 司机拒单后改派
- 司机超时后改派
- 乘客派单前取消
- 乘客接单后取消
- 行程结束后支付失败再重试
- 管理后台审核司机
- 未审核司机无法上线

## 16. 部署方式

第一版使用 Docker Compose 启动：

- 乘客端前端
- 司机端前端
- 管理后台前端
- API Gateway/BFF
- WebSocket Gateway
- 各 Go Kratos 微服务
- MySQL
- Redis
- Kafka

本地开发环境和单机演示环境使用同一套 Compose 编排。后续如需生产化，再补充 Kubernetes、服务注册发现、监控和日志采集。

## 17. 第一版不包含的内容

第一版不包含：

- 真实支付宝或微信支付接入
- 拼车、专车、出租车等多业务类型
- 完整风控规则引擎
- Prometheus、Grafana、Jaeger 等完整可观测性基础设施
- Kubernetes 部署
- 原生 iOS/Android App
- 复杂优惠券、营销、会员体系
- 客服工单系统

这些能力可在第一版稳定后逐步迭代。
