# 类滴滴平台微服务架构设计

## 1. 目标

当前项目已经跑通了乘客、司机、管理后台、订单状态机、自动派单、模拟支付和实时推送的演示链路，但后端运行形态仍是单个 `api` 进程。该形态不符合打车业务在订单、派单、司机、支付、实时推送等领域的天然边界，也不利于后续高并发扩展和多人协作。

本设计明确将后端目标调整为：第一版正式后端不再继续沿用单体运行形态，而是直接落地为多个独立微服务。设计重点是业务解耦和可扩展性，要求服务边界清晰、数据归属明确、跨服务通信可控，并能支撑后续业务增长。

## 2. 范围

本设计覆盖以下内容：

- 服务拆分清单与职责边界
- 服务间同步调用与 Kafka 事件边界
- 数据归属与数据库拆分原则
- 下单、派单、接单、行程、支付、实时推送等核心链路
- 幂等、最终一致性、故障处理、降级与测试原则
- 从当前单体实现演进到目标微服务架构的落地原则

本设计不展开具体 proto、表结构、部署 YAML、CI/CD 细节和单个服务内部代码结构，这些在后续实施计划中单独细化。

## 3. 架构原则

### 3.1 业务边界优先

服务按业务域拆分，而不是按技术层拆分。订单、派单、司机、支付、位置、实时通知等核心域必须独立，避免再次形成新的大而全核心服务。

### 3.2 真相单点归属

每一种核心业务状态只允许一个服务拥有真相。其他服务可以消费事件形成读模型或缓存，但不能越权直接修改该状态。

### 3.3 写入隔离

每个服务只直写自己的数据库对象。跨服务不能直连对方数据库做状态修改，只能通过同步 API 或异步事件交互。

### 3.4 同步做命令与查询，异步做状态传播

需要立即返回结果的登录、查询、核心命令请求走同步调用；跨域状态传播、实时通知、后台聚合和异步补偿走 Kafka 事件。

### 3.5 可降级但不能状态错乱

局部能力允许退化，例如实时推送退化到轮询、地图能力退化到缓存或粗略估价，但订单、派单、支付等核心真相不能出现双写、乱序覆盖或越权推进。

## 4. 目标服务拆分

### 4.1 gateway / bff

统一外部入口，面向乘客端、司机端和管理后台提供 HTTP 接口与协议聚合能力。职责包括鉴权透传、路由、接口聚合、限流、灰度和错误转换。

`gateway` 不保存订单、派单、支付、司机状态等业务真相，也不承载核心业务状态机。

### 4.2 auth-service

负责手机号验证码、登录、Token、Refresh Token、会话、设备登录态和角色身份校验。该服务只解决“你是谁”和“你是否有权限”的问题，不管理乘客资料和司机资料。

### 4.3 user-service

负责乘客资料、常用地址、乘客偏好等乘客域主数据。

### 4.4 driver-service

负责司机资料、车辆资料、审核状态、司机工作状态、接单开关和司机可服务能力。该服务是“司机是否具备接单资格”的真相源。

### 4.5 order-service

负责订单主状态机、起终点、预估金额、实付金额、取消记录和评价。该服务拥有订单主状态真相，但不拥有派单过程真相，也不拥有支付状态真相。

### 4.6 dispatch-service

负责候选司机筛选、派单任务、派单尝试、司机响应超时、拒单与自动改派。该服务拥有派单过程真相，但不拥有订单主状态和支付状态。

### 4.7 location-service

负责司机实时位置、位置轨迹、地理索引和高德地图能力封装，包括距离、时长、路线和附近司机能力。它为下单估价、派单排序和乘客查看司机位置提供基础能力。

### 4.8 payment-service

负责支付单、支付流水、退款单和支付状态。第一版即使使用模拟支付，也必须单独成服务，避免后续接入真实支付渠道时再次大拆。

### 4.9 realtime-service

负责 WebSocket 连接管理、订阅关系、在线连接元数据和消息投递。该服务消费订单、派单、支付、位置等业务事件，把结果推送给乘客端、司机端和后台。它不拥有业务真相。

### 4.10 admin-service

负责管理后台聚合查询、司机审核操作、异常订单处理、基础运营统计和后台审计日志。该服务偏运营编排与聚合读，不拥有订单、司机、支付的底层真相。

## 5. 服务通信设计

### 5.1 同步调用

适合同步调用的链路如下：

- `gateway -> auth-service`：登录、验 token、刷新会话
- `gateway -> user-service / driver-service / order-service / admin-service`：前端直接业务请求
- `order-service -> location-service`：距离、时长、预估价计算
- `dispatch-service -> driver-service`：司机可接单状态和资料摘要查询
- `dispatch-service -> location-service`：附近司机检索与距离排序
- `admin-service -> driver-service / order-service / payment-service`：后台查询和操作

这些链路需要立即得到结果，适合使用 HTTP/gRPC 同步调用。

### 5.2 Kafka 事件

必须事件化的跨域状态传播如下：

- `order-created`：`order-service` 发布，`dispatch-service` 消费并启动派单
- `dispatch-accepted`、`dispatch-timeout`、`dispatch-failed`：`dispatch-service` 发布，`order-service` 消费以推进订单状态
- `trip-started`、`trip-ended`：`order-service` 发布，供 `payment-service`、`realtime-service`、`admin-service` 订阅
- `payment-paid`、`payment-failed`、`refund-completed`：`payment-service` 发布，`order-service` 消费更新订单支付结果
- `driver-location-updated`：`location-service` 发布，`realtime-service` 订阅并推送位置变化
- `driver-approved`、`driver-suspended`：`driver-service` 发布，供 `dispatch-service` 和 `admin-service` 感知

### 5.3 真相约束

- `order-service` 拥有订单主状态真相
- `dispatch-service` 拥有派单过程真相
- `payment-service` 拥有支付状态真相
- `realtime-service` 只负责消息投递
- `admin-service` 只负责后台聚合与编排

任何服务都不得绕过这些边界直接改写别人的核心状态。

## 6. 数据边界与存储拆分

### 6.1 服务数据归属

#### auth-service

拥有验证码记录、账号凭证、Token、Refresh Token、会话和角色绑定。

#### user-service

拥有乘客资料、常用地址和乘客偏好。

#### driver-service

拥有司机资料、车辆资料、审核状态、工作状态和接单能力。

#### order-service

拥有订单主表、状态流转、起终点、金额、取消记录和评价。

#### dispatch-service

拥有 dispatch task、dispatch attempt、候选司机快照、超时和改派记录。

#### location-service

拥有司机实时位置、轨迹、地理索引和地图调用缓存。

#### payment-service

拥有支付单、支付流水、退款单和支付结果。

#### realtime-service

拥有连接、订阅关系、在线连接元数据和投递状态。

#### admin-service

拥有后台操作日志、审计记录和运营聚合读模型。

### 6.2 数据库形态

第一版允许多个服务共用同一个 MySQL 实例，但必须满足以下约束：

- 每个服务独立 schema
- 每个服务独立建表和迁移
- 服务只写自己的 schema
- 跨服务数据通过 API、事件或聚合读模型获得

Redis 和 Kafka 可以共用集群，但 key、topic、consumer group 命名必须按服务隔离。

### 6.3 禁止事项

以下行为明确禁止：

- `dispatch-service` 直接更新 `order-service` 的订单表
- `payment-service` 直接把订单改成已完成
- `admin-service` 绕过业务服务直接修订单状态
- `driver-service` 越权写派单或订单核心状态

## 7. 核心业务链路

### 7.1 下单

1. 前端通过 `gateway` 调用 `order-service` 创建订单。
2. `order-service` 同步调用 `location-service` 计算距离、时长和预估价。
3. `order-service` 先落订单真相，再发布 `order-created`。
4. `dispatch-service` 消费该事件并创建派单任务。

订单必须先落真相后触发派单，不能先派单后补订单。

### 7.2 派单

1. `dispatch-service` 调 `location-service` 获取附近司机。
2. `dispatch-service` 调 `driver-service` 过滤审核通过、在线空闲和可接单司机。
3. 生成候选司机快照并排序。
4. 每次只向一个司机发起派单，写入 `dispatch-attempt`。
5. 发布事件给 `realtime-service`，向司机端推送派单通知。
6. 若司机拒单或超时，则继续下一轮派单，直到成功或失败。

派单过程真相只存在于 `dispatch-service`。

### 7.3 接单

1. 司机端经 `gateway` 向 `dispatch-service` 发起接单命令。
2. `dispatch-service` 对当前 attempt 做条件更新，保证同一单只有一个司机成功接单。
3. 成功后发布 `dispatch-accepted`。
4. `order-service` 消费事件并把订单推进到 `WAITING_PICKUP`。
5. `driver-service` 消费事件并把司机切为忙碌态。
6. `realtime-service` 消费事件并推送乘客端和司机端。

“谁接单成功”由 `dispatch-service` 判定，“订单状态推进”由 `order-service` 判定。

### 7.4 行程与支付

1. 司机到达、开始行程、结束行程等命令进入 `order-service`。
2. `order-service` 负责校验状态机合法性。
3. 行程结束后，`order-service` 产出应付金额并发布 `trip-ended`。
4. `payment-service` 创建支付单。
5. 支付成功后，`payment-service` 发布 `payment-paid`。
6. `order-service` 消费后将订单推进到 `COMPLETED`。

支付成功只能通过事件驱动订单完成，不能由支付服务直接修改订单表。

### 7.5 实时推送

`realtime-service` 统一维护 WebSocket 连接，订阅订单、派单、支付和位置事件，并向对应用户推送订单状态、司机位置、派单通知和后台异常事件。它是消息分发层，不做业务决策。

## 8. 幂等与最终一致性

### 8.1 幂等要求

- 创建订单：客户端携带幂等键，避免重复下单
- 司机接单：attempt 条件更新，避免多人同时成功
- 超时改派：定时器触发和司机晚到接单竞争时，以条件更新结果为准
- 支付回调：`payment-service` 必须幂等处理重复通知
- 事件消费：所有消费者按业务主键或事件键去重
- WebSocket 推送：允许重复投递，客户端按状态版本或消息版本去重

### 8.2 一致性机制

系统统一采用以下机制：

- 业务主键 + 幂等键
- Outbox / Event Log，保证落库与发事件的一致性
- 状态版本号，解决乱序和覆盖问题

## 9. 故障处理与降级

### 9.1 基本策略

- 核心真相服务优先保证状态正确性，而不是优先追求所有链路都成功
- 跨服务调用失败时，优先保持状态不推进，避免进入不确定状态
- 每个核心状态变更都带条件更新或版本约束

### 9.2 典型降级

- `location-service` 异常时，下单可退化为缓存距离或简化估价，派单可退化为粗粒度附近司机筛选
- `realtime-service` 异常时，不影响订单和派单真相，前端退化为轮询
- `payment-service` 异常时，订单停留在 `WAITING_PAYMENT`
- `dispatch-service` 异常时，新订单停留在 `DISPATCHING` 或进入明确失败态，不能由其他服务代替修改订单
- `admin-service` 异常时，只影响后台运营，不影响主交易链路

## 10. 测试策略

### 10.1 领域单元测试

覆盖订单状态机、派单规则、取消规则、支付状态推进和司机状态切换。

### 10.2 服务级集成测试

每个服务验证自身 API、数据库访问、事件生产消费和幂等逻辑。

### 10.3 跨服务流程测试

至少覆盖以下场景：

- 下单 -> 派单 -> 接单 -> 到达 -> 开始 -> 结束 -> 支付 -> 完成
- 司机拒单 -> 自动改派
- 派单超时 -> 自动改派
- 支付重复回调
- 司机晚接单与系统超时竞争

### 10.4 端到端冒烟测试

联调乘客端、司机端和管理后台，验证主链路可跑通。

## 11. 可观测性

第一版至少具备以下能力：

- 统一 request id / trace id
- 订单 id、司机 id、派单 task id 贯穿日志
- 关键事件日志：下单、派单、接单、改派、支付、推送失败
- Kafka 消费失败重试与死信记录

微服务拆分后如果缺少这些最小可观测性，排障成本会显著上升。

## 12. 落地与迁移原则

虽然目标是直接多个微服务，但迁移必须遵守以下原则：

1. 先定义服务契约、表归属和事件，再做代码搬迁。
2. 禁止双写真相，不能同时让旧单体表和新服务表都承担正式写入。
3. 优先落地 `gateway`、`auth-service`、`driver-service`、`order-service`、`dispatch-service`、`location-service`、`payment-service`、`realtime-service`，`admin-service` 可稍后补齐。
4. 可以共享 MySQL、Redis、Kafka 实例，但不能共享写权限边界。
5. 每个服务拆出后都必须独立可启动、可测试、可发布。

## 13. 结论

后端第一版正式目标架构应为以下微服务拓扑：

- `gateway`
- `auth-service`
- `user-service`
- `driver-service`
- `order-service`
- `dispatch-service`
- `location-service`
- `payment-service`
- `realtime-service`
- `admin-service`

配套基础设施为 MySQL、Redis 和 Kafka。该方案相比继续维持单体运行形态，初期工程复杂度更高，但换来的是正确的业务边界、更清晰的多人协作边界和可持续的后续扩展能力。