# CloudIaC CMDB 产品需求文档

版本：v1.0  
日期：2026-06-20  
范围：CloudIaC 组织级资产 CMDB、云资产采集、资产关系、应用依赖、变更风险感知

## 1. 背景

CloudIaC 当前以 IaC 环境、模板、任务、资源漂移为核心，已经沉淀了 `iac_resource` 等运行态资源数据。但这些数据更偏 IaC 执行视角，无法完整回答以下问题：

- 当前组织有哪些云资产，分别来自 IaC、云采集还是人工导入。
- 资产属于哪个负责人、应用、业务线、生命周期阶段和合规风险等级。
- 资产之间、应用之间有哪些上下游依赖。
- 某个资源或应用发生变更时，会影响哪些上游调用方和下游依赖方。
- 云上真实资产与 IaC 资源之间如何统一检索、追踪和治理。

因此需要建设 CloudIaC CMDB 能力，将 IaC 资源、云账号采集、人工维护、导入导出和依赖关系统一到一个组织级资产视图。

## 2. 目标

### 2.1 产品目标

- 建立组织级资产 CMDB，统一呈现 IaC 资源、云采集资源和人工导入资源。
- 支持资产属性、标签、原始数据、归属字段、关系和变更记录查询。
- 支持从现有 IaC 资源回填资产，并持续维护 IaC dependency 到 CMDB 资产关系。
- 支持云账号采集，优先覆盖 AWS 与 OCI 常用资产类型。
- 支持应用维度上下游依赖，结合应用绑定资源和变更记录感知风险。
- 打通环境资源详情与 CMDB 资产详情，降低从部署视角到资产治理视角的跳转成本。

### 2.2 非目标

- 不替代云厂商控制台的全量资源管理功能。
- 不在本阶段实现完整 ITSM、工单审批、告警事件中心。
- 不在本阶段引入独立图数据库；关系图先基于当前关系表和前端图形/表格呈现。
- 不在本阶段覆盖所有云厂商全资源类型，采用分阶段扩展。

## 3. 状态总览

状态说明：

- 已完成：代码已实现并完成 Docker Compose build/up 验证或已进入运行镜像。
- 部分完成：核心能力已具备，但仍缺关键体验、覆盖面或验证。
- 待开发：尚未实现或当前只有占位。

| 模块 | 状态 | 当前说明 |
| --- | --- | --- |
| CMDB 数据模型 | 已完成 | 已有资产、资产关系、资产变更、同步任务、应用关系表模型 |
| 资产列表/筛选/详情 UI | 已完成 | 资产列表、筛选、详情抽屉、归属/关系/变更/原始数据页签已实现 |
| IaC 资源回填 | 已完成 | 从 `iac_resource` 回填 CMDB 资产，列表/详情会触发回填 |
| IaC dependency 回填资产关系 | 已完成 | IaC dependencies 会写入 `iac_cmdb_asset_relation` |
| 增量 upsert/diff | 已完成 | 资产 upsert 避免无变化重复更新，并记录创建/更新变更 |
| 资产归属字段 | 已完成 | owner、application、businessLine、lifecycle、cost、complianceRisk 已建模并支持编辑 |
| AWS 云采集 | 部分完成 | 已支持 EC2、VPC、Subnet、Route Table、NAT Gateway、Internet Gateway、SecurityGroup、EIP、EBS、ELB/ALB/NLB、LB Listener/Rule/Auth Action/Target Group/Target Health、S3 Bucket 深层配置和 S3 风险规则映射、EKS、RDS、ElastiCache Redis/Valkey；真实云账号端到端联调、跨账号/跨 VPC target 和更完整成本/合规映射仍待继续 |
| OCI 云采集 | 已完成第一阶段 | 已支持 Compute、VCN/Subnet、Route Table、NAT Gateway、Internet Gateway、Service Gateway、DRG、Security List/NSG、Public IP、Block Volume、Load Balancer、Bucket、OKE、DB System、Autonomous DB、Redis；已补充 compartment/identity 归属映射、provider 原生错误码解析、429/5xx 短重试、API 子调用耗时指标、任务详情 API 调用可视化和失败 scope 自动局部重试第一阶段；真实账号端到端联调、更细 identity 权限漂移、分页和 API 子调用级局部补偿仍待继续 |
| AliCloud 云采集 | 已完成第一阶段 | 已支持 ECS、VPC、VSwitch、SecurityGroup、EIP、Disk、SLB、RDS、Redis、OSS、ACK 第一阶段采集；真实账号端到端联调和更细成本/合规映射仍待继续 |
| Azure/GCP/腾讯云/华为云 | 已完成第一阶段 | Azure/GCP 已接入轻量真实云 API collector；腾讯云/华为云已接入真实云 API collector 并保留离线 inventory 回退路径 |
| 云采集任务后台化 | 已完成 | `StartCmdbSyncTask` 创建任务后使用 goroutine 后台执行 |
| 云采集任务状态/统计 | 已完成第一阶段 | 已有状态、错误、stats、StartedAt/EndedAt、阶段日志、最近成功/失败摘要、范围明细、失败明细、API 调用耗时页签、API 调用筛选、任务详情慢调用告警、筛选结果 CSV 导出、API 维度趋势、事件中心慢 API 告警、同步策略级慢调用阈值持久化、慢 API 告警静默窗口和失败 scope 自动局部重试；真实 provider 样本仍待继续 |
| 环境资源跳 CMDB | 已完成 | 环境资源列表/详情可跳转 CMDB 资产详情 |
| CMDB 返回来源环境资源 | 已完成第一阶段 | CMDB 详情展示项目、环境、IaC 地址，并在存在 `iacResourceId` 时提供“返回环境资源详情”入口，可深链到环境资源页并自动打开资源详情抽屉 |
| 资产关系图视图 | 已完成 | 资产详情有关系图和表格兜底 |
| 应用依赖视图 | 已完成 | 应用列表、详情、应用关系图、上下游表格、近期变更和风险等级已实现 |
| 应用依赖维护 | 已完成 | 支持人工维护应用上游/下游依赖，写入 `iac_cmdb_application_relation` |
| 变更风险感知 | 已完成第一阶段 | 应用近期变更、上下游影响、合规风险、生命周期和跨业务线权重已接入组织级风险规则配置；发布计划联动仍待继续 |
| Kubernetes 集群信息 | 已完成第一阶段 | EKS/OKE/GKE/AKS 已采集为 `kubernetes_cluster`，资产详情已新增 K8S 信息页签展示版本、Endpoint、网络、安全组、节点组/节点池，以及基于资产属性/导入数据的 Namespace、Node、Pod、Workload、Service、Ingress 工作负载层摘要和明细；云资产首页已新增 Kubernetes 概览和 EKS/OKE/AKS/GKE 快速入口；K8S 详情已兼容旧数据、导入数据、原生字段大小写变体、EKS `resourcesVpcConfig` 和 OKE `endpoints`；kubeconfig/Agent 实时采集仍待后续阶段 |
| 资产搜索 DSL | 已完成 | 支持 provider、type、source、status、owner、application、tag.*、attr.* 等 DSL |
| 导入/导出 | 已完成第一阶段 | 支持 JSON 导入、导入模板下载、导入预检、差异预览、确认导入、CSV/JSON 导出、选中导出、归属覆盖 |
| Webhook/事件推送 | 已完成第一阶段 | CMDB 资产创建/更新/归属变更和应用依赖变更已写入统一事件中心，并复用平台级 Webhook 配置、签名、重试和投递日志 |
| 浏览器回归验证 | 部分完成 | Docker build/up 与运行 bundle 校验完成；MCP browser 工具未暴露时无法完成点击回归 |

## 4. 用户角色

| 角色 | 核心诉求 |
| --- | --- |
| 运维/SRE | 快速查资产、查关系、查云采集结果、判断变更影响 |
| 应用负责人 | 查看本应用绑定资源、上下游调用、近期变更风险 |
| 平台管理员 | 管理云账号采集、导入导出、补齐资产归属和合规风险 |
| 安全/合规 | 识别高风险资产、未归属资产、生命周期异常资产 |
| 开发/项目成员 | 从环境资源跳到 CMDB，定位资源属性、标签、原始云数据 |

## 5. 信息架构

CMDB 页面入口：组织级资源查询/资产中心页面。

一级页签：

- 资产列表
- 应用依赖
- 云采集

资产详情抽屉：

- 归属
- 关系
- 变更
- 原始数据

应用依赖详情抽屉：

- 关系
- 依赖维护
- 绑定资源
- 近期变更

## 6. 核心业务流程

### 6.1 IaC 资源进入 CMDB

1. CloudIaC 执行任务产生或更新 `iac_resource`。
2. 用户打开资产列表/详情，或触发同步 IaC 资源。
3. 系统读取 `iac_resource`，转换为 `iac_cmdb_asset`。
4. 根据 org、source、provider、account、region、nativeId 做唯一 upsert。
5. 对比关键字段、标签、属性和原始数据：新资产记录 created 变更；有变化记录 updated 变更；无变化跳过。
6. 将 IaC dependencies 写入 `iac_cmdb_asset_relation`。

状态：已完成。

### 6.2 云采集进入 CMDB

1. 系统从环境变量组和资源账号识别云账号。
2. 页面展示云账号、provider、regions、ready 状态和缺失凭证。
3. 用户选择账号、区域、资产类型，启动云采集任务。
4. 后端创建 `iac_cmdb_sync_task`，状态为 running。
5. 后台 goroutine 调用 provider collector。
6. collector 返回资产列表和统计数据。
7. 系统按统一 upsert 逻辑写入 CMDB。
8. 任务结束后写入 complete/failed、errorMessage、stats、endedAt。

状态：已完成第一阶段。AWS/OCI/AliCloud/Azure/GCP/腾讯云/华为云均已具备第一阶段采集路径；任务日志、统计摘要、范围明细、失败明细、API 调用耗时页签、API 调用筛选、任务详情慢调用告警、筛选结果 CSV 导出、API 维度趋势、事件中心慢 API 告警和策略级静默窗口已增强，真实 provider 端到端联调、更多资源类型和成本/合规映射仍待继续。

### 6.3 资产归属维护

1. 用户打开资产详情抽屉。
2. 在归属页签编辑负责人、应用、业务线、生命周期、成本、合规风险。
3. 后端更新资产字段。
4. 系统记录人工维护变更，变更来源为 `manual_edit`。
5. 应用依赖视图基于 application 字段重新聚合绑定资源。

状态：已完成。

### 6.4 应用依赖和变更风险

1. 资产通过 application 字段绑定到应用。
2. 系统根据资产关系推导应用上下游依赖。
3. 用户可在应用详情的依赖维护页签维护人工应用关系。
4. 应用详情展示绑定资源、上下游应用、配置窗口内近期变更、影响应用和风险等级。
5. 当某应用绑定资源近期有变更，系统结合上下游依赖生成风险提示。

状态：已完成第一阶段。已支持组织级应用风险规则配置，包括变更窗口、近期变更/变更资产/调用方/调用应用权重、合规风险权重、生命周期权重、跨业务线权重、风险阈值和近期变更加分。

## 7. 数据模型

### 7.1 `iac_cmdb_asset`

用途：统一资产主表。

关键字段：orgId、projectId、envId、taskId、source、provider、accountId、region、zone、assetType、nativeType、nativeId、name、status、address、module、owner、application、businessLine、lifecycle、cost、complianceRisk、publicIp、privateIp、iacResourceId、iacAddress、tags、attributes、rawData、lastSyncAt。

状态：已完成。

### 7.2 `iac_cmdb_asset_relation`

用途：资产之间的依赖/包含关系。

关键字段：orgId、sourceAssetId、targetAssetId、relationType、source、metadata。

状态：已完成。

### 7.3 `iac_cmdb_application_relation`

用途：人工维护应用之间的上下游依赖。

关键字段：orgId、sourceApplication、targetApplication、relationType、source、metadata。

状态：已完成。

### 7.4 `iac_cmdb_asset_change`

用途：资产变更记录。

关键字段：orgId、assetId、changeType、source、summary、diff。

状态：已完成。

### 7.5 `iac_cmdb_sync_task`

用途：云采集任务状态。

关键字段：orgId、accountSource、accountId、accountName、provider、regions、assetTypes、status、errorMessage、stats、startedAt、endedAt。

状态：已完成。

### 7.6 `iac_cmdb_risk_rule_config`

用途：组织级应用变更风险规则配置。

关键字段：orgId、changeWindowDays、recentChangeWeight、changedAssetWeight、incomingAppWeight、outgoingAppWeight、highComplianceRiskWeight、criticalComplianceRiskWeight、maintenanceLifecycleWeight、retiredLifecycleWeight、crossBusinessLineWeight、criticalIncomingThreshold、mediumIncomingThreshold、mediumOutgoingThreshold、criticalScoreThreshold、highScoreThreshold、mediumScoreThreshold、recentCriticalBoost、recentHighBoost、recentMediumBoost、wideDependencyBoost。

状态：已完成第一阶段。

## 8. API 设计

| 方法 | 路径 | 说明 | 状态 |
| --- | --- | --- | --- |
| GET | `/api/v1/cmdb/assets` | 资产列表、关键词、筛选、DSL、分页排序 | 已完成 |
| GET | `/api/v1/cmdb/assets/filters` | 资产筛选项 | 已完成 |
| GET | `/api/v1/cmdb/assets/export` | 资产 CSV/JSON 导出 | 已完成 |
| POST | `/api/v1/cmdb/assets/import` | 资产 JSON 导入；`dryRun=true` 时只预检并返回差异 | 已完成第一阶段 |
| GET | `/api/v1/cmdb/assets/import-template` | 下载 JSON 导入模板 | 已完成第一阶段 |
| GET | `/api/v1/cmdb/assets/:id` | 资产详情、关系、变更 | 已完成 |
| GET | `/api/v1/cmdb/assets/:id/relations` | 资产关系明细、应用推演关系和关系摘要 | 已完成第一阶段 |
| PUT | `/api/v1/cmdb/assets/:id/ownership` | 更新资产归属/生命周期/风险 | 已完成 |
| POST | `/api/v1/cmdb/backfill/iac-resources` | 从 IaC 资源回填 CMDB | 已完成 |
| GET | `/api/v1/cmdb/cloud/accounts` | 查询可采集云账号 | 已完成 |
| GET | `/api/v1/cmdb/sync-tasks` | 查询云采集任务 | 已完成 |
| POST | `/api/v1/cmdb/sync-tasks` | 启动云采集任务 | 已完成 |
| GET | `/api/v1/cmdb/applications` | 应用依赖列表和风险 | 已完成 |
| GET | `/api/v1/cmdb/applications/detail` | 应用依赖详情 | 已完成 |
| PUT | `/api/v1/cmdb/applications/relations` | 维护应用上下游依赖 | 已完成 |
| GET | `/api/v1/cmdb/risk-rules` | 查询 CMDB 应用风险规则配置 | 已完成第一阶段 |
| PUT | `/api/v1/cmdb/risk-rules` | 更新 CMDB 应用风险规则配置 | 已完成第一阶段 |
| POST | `/api/v1/cloud/webhooks` | 平台级 Webhook 配置，支持订阅 CMDB 事件来源和 `cmdb.*` 事件类型 | 已完成第一阶段 |

## 9. 云采集范围

### 9.1 AWS

已完成：EC2 Instance、VPC、Subnet、Route Table、NAT Gateway、Internet Gateway、Security Group、Elastic IP、公网 IP 独立资产、EBS Volume、Classic ELB、ALB/NLB、LB Listener、Listener Rule、Auth Action、Target Group、Target Health、S3 Bucket 深层配置、EKS Cluster、EKS NodeGroup 信息、RDS DB Instance、ElastiCache Redis/Valkey Cluster、ElastiCache Replication Group。

待开发：真实 AWS 账号端到端联调、更完整的成本/合规映射、跨账号/跨 VPC target 解析；S3 风险规则映射已完成第一阶段。

状态：部分完成。

### 9.2 OCI

已完成：Compute Instance、VCN、Subnet、Route Table、NAT Gateway、Internet Gateway、Service Gateway、DRG、Security List、Network Security Group、Public IP、Block Volume、Load Balancer、Object Storage Bucket、OKE Cluster、OKE NodePool 信息、DB System、Autonomous Database、Redis Cluster。

待开发：真实 OCI 账号端到端联调、更细 identity 权限漂移识别、provider 原生分页、API 子调用级局部补偿、更多真实错误码样本补测和更完整成本/合规映射；region/assetType 失败 scope 自动局部重试已完成第一阶段。

状态：已完成第一阶段。

### 9.3 AliCloud

已完成：账号识别、凭证键识别、区域识别、支持资产类型声明；collector 支持 ECS、VPC、VSwitch、安全组、EIP、Disk、SLB、RDS、Redis、OSS、ACK。

待开发：真实账号端到端联调；更细的 SLB 监听器/后端服务器、ACK 节点池、OSS bucket 配置、成本/合规映射和 provider 原生错误码。

状态：已完成第一阶段。

### 9.4 Azure/GCP/腾讯云/华为云

已完成：Azure/GCP 轻量真实云 API collector；腾讯云/华为云真实云 API collector；腾讯云/华为云离线 inventory 回退 collector。

待开发：真实生产账号端到端联调；更多资源类型、分页/错误码细化、成本/合规映射和区域级权限矩阵。

状态：已完成第一阶段。

## 10. 搜索与筛选

普通筛选支持：关键词、云厂商、资产类型、项目、环境、来源、状态。

DSL 示例：

```text
provider:oci
type:compute_instance
source:cloud_collect
status:RUNNING
owner:alice
application:"Payment Service"
tag.env:prod
attr.vpcId:vpc-xxx
```

DSL 支持：

- 精确匹配：provider、account、region、zone、type、source、status、project、env、lifecycle、cost、risk。
- 模糊匹配：id、name、nativeId、nativeType、address、module、owner、application、businessLine、publicIp、privateIp、iacAddress。
- JSON 匹配：tag.*、attr.*。

状态：已完成。

## 11. 前端页面需求

### 11.1 资产列表

能力：单一关键词搜索框、DSL 搜索框、多维筛选、资产表格、批量选择导出、导入 JSON、同步 IaC 资源、点击资产打开详情抽屉。

状态：已完成。

### 11.2 资产详情

能力：展示资产核心信息；归属页签编辑 owner、application、businessLine、lifecycle、cost、complianceRisk；关系页签展示关系图和表格；变更页签展示变更和 diff；原始数据页签展示 rawData；展示 IaC 地址和最近同步时间。

状态：已完成。

### 11.3 应用依赖

能力：应用列表展示应用、维护依赖、绑定资源、调用方、调用应用、近 7 天变更、风险等级、负责人、业务线；支持搜索和风险筛选；应用详情展示关系图、上下游表格、绑定资源、近期变更；依赖维护表单支持编辑调用本应用和本应用调用。

状态：已完成。

### 11.4 云采集

能力：展示可采集云账号、账号来源、provider、ready 状态、缺失凭证；支持选择区域和资产类型；启动云采集任务；查看任务状态、错误、统计、时间、采集范围、失败范围、最近成功/失败任务和重试建议。

状态：已完成第一阶段。任务列表和详情已展示范围摘要、范围明细、失败分类、失败明细、重试建议、最近成功/失败任务入口和 API 调用耗时页签；API 调用页签已支持 region、service、status、慢调用阈值过滤、任务详情慢调用告警和筛选结果 CSV 导出；云采集统计区已支持 API 维度趋势 Top 10；任务完成后可写入事件中心慢 API 告警并复用 Webhook/通知策略；同步策略已支持慢调用阈值持久化、慢 API 告警静默窗口、失败 scope 自动局部重试、通知负责人、通知路由和分派对象元数据并写入任务/事件 stats；OCI provider 原生错误码解析、429/5xx 短重试、单 API 子调用耗时指标和前端 API 调用可视化已完成第一阶段，更细的 provider 原生分页、按错误类型/服务动态升级和 API 子调用级局部补偿待继续。

## 12. 环境资源集成

需求：环境资源列表显示 CMDB 入口；环境资源详情显示“查看 CMDB 资产”；CMDB 资产详情展示来源项目、环境、IaC 地址；从 CMDB 详情一键返回环境资源详情。

状态：已完成第一阶段。CMDB 详情在存在 `projectId/envId/iacResourceId` 时展示“返回环境资源详情”，跳转到环境资源页并自动打开资源详情抽屉。

## 13. 风险感知规则

当前规则：统计应用绑定资产在配置窗口内的近期变更；统计上游/下游应用数量；按组织级配置的权重和阈值计算风险分数与风险等级。

已完成第一阶段：

- 新增 `iac_cmdb_risk_rule_config`，按组织保存应用风险规则。
- 支持配置变更窗口天数。
- 支持配置近期变更、变更资产、调用方、调用应用、高/严重合规风险、维护期/退役期和跨业务线权重。
- 支持配置严重/高/中评分阈值，以及严重调用方、中风险调用方、中风险调用应用阈值。
- 支持配置近期变更加分和依赖面加分，保持默认值兼容原有风险判断。
- 前端“应用依赖”页签新增“风险规则”弹窗，保存后刷新应用列表并联动“近 N 天变更”列名。

待增强：支持关键应用、核心链路、生产环境加权；支持变更窗口和发布计划联动；支持风险说明可追溯到具体资产变更和关系路径。

状态：已完成第一阶段。

## 14. 导入导出

已完成：JSON 导入资产；导入模板下载；导入时先 dryRun 预检，返回新增/更新/归属更新/跳过/异常统计和逐条差异；前端展示预检弹窗，确认后再真实写入；导入时可选择覆盖归属字段；CSV 导出；JSON 导出；选中资产导出；导出数量上限保护。

待增强：CSV/Excel 导入；字段级映射；异步大批量导入任务；预检结果下载。

状态：已完成第一阶段。

## 15. Webhook 与事件

需求：当资产创建、更新、归属变更、应用依赖变更时产生事件；支持配置 Webhook endpoint；支持事件重试、签名、投递日志；支持发送到外部 ITSM、告警、数据湖或消息系统。

状态：已完成第一阶段。CMDB 事件复用平台级事件中心和 Webhook 能力，不单独维护 `/cmdb/events/webhook` 配置入口：

- 资产创建、云采集 upsert、人工归属维护、批量归属维护、导入覆盖归属和云操作回写会记录 `iac_cmdb_asset_change`，并同步产生 `cmdb.asset.created` 或 `cmdb.asset.updated` 事件。
- 应用依赖维护保存后会对比旧上游/下游与新上游/下游，真实变化时产生 `cmdb.application.relations_updated` 事件。
- Webhook 配置、签名、密钥轮换、投递记录、失败重试、到期自动重试、死信重放和测试发送复用 `/api/v1/cloud/webhooks` 系列接口。
- 事件查询复用 `/api/v1/cloud/events`，可按 `source=cmdb`、`eventType=cmdb.*`、资产 ID 等条件过滤。

待增强：补专用的 CMDB 事件模板、ITSM 事件字段映射、数据湖批量投递和按业务线/应用的订阅策略。

## 16. 非功能需求

### 16.1 性能

要求：资产列表分页查询；导出默认限制 10000 条；资产 upsert 使用增量 diff；云采集任务后台执行。

状态：已完成第一阶段。列表分页、导出上限、增量 upsert 和后台采集已具备；大规模关系图已补充专用关系查询接口、默认 200 条返回上限、最大 1000 条查询保护、关系摘要、应用推演关系局部聚合、关系表组合索引、按来源/类型/方向/关键词服务端筛选，以及 cursor/offset 分页和前端“加载更多”渐进加载第一阶段。

待增强：真实大规模资产数据压测、稳定排序游标、图谱视图按视口渐进展开和前端虚拟化；关系搜索、按关系来源/类型/方向筛选、分页查询和加载更多已完成前后端第一阶段。

### 16.2 安全

要求：CMDB API 使用组织权限；云账号凭证从变量组/资源账号读取，敏感值解密后仅用于采集；不在 UI 暴露凭证明文。

状态：已完成第一阶段。CMDB/云资产已增加编辑、导出、导入、批量治理、应用依赖维护、风险规则配置、IaC 同步的权限预检和响应层敏感字段递归脱敏；后续需继续补标签级授权和真实组织角色矩阵回归。

### 16.3 可观测性

要求：采集任务记录 status、errorMessage、stats、startedAt、endedAt；panic 会记录任务失败和堆栈日志；任务响应提供范围摘要、失败摘要、最近成功/失败任务和阶段日志。

状态：已完成第一阶段。OCI provider 原生错误码映射、限流/临时错误短重试、API 子调用耗时指标和任务详情 API 调用可视化已补第一阶段，API 调用页签已支持筛选、慢调用阈值过滤、任务详情慢调用告警和筛选结果 CSV 导出，云采集统计区已支持 API 维度趋势，任务完成后可写入事件中心慢 API 告警，同步策略可持久化慢调用阈值、静默窗口、失败 scope 自动局部重试、通知负责人、通知路由和分派对象元数据；后续需补更多真实错误样本、按错误类型/服务动态升级和 API 子调用级局部补偿。

## 17. 验收标准

已完成能力验收：

- Docker Compose build `iac-portal` 和 `iac-web` 通过。
- Docker Compose up 后 `iac-portal` healthy，`iac-web` 可提供静态资源。
- 运行前端 bundle 包含 `manageRelations`、`updateApplicationRelations`、`keywordSearch`。
- 资产列表、云采集、资产详情、关系、变更页签已通过上一轮浏览器验证。
- 应用依赖核心代码已进入运行镜像，待 MCP 浏览器恢复后补完整点击回归。
- 资产关系查询已提供独立接口，返回直接关系、应用推演关系、截断状态和来源/类型/方向摘要；独立接口支持按来源、关系类型、方向和关键词进行第一阶段服务端筛选；后端单元测试覆盖关系查询 limit 保护、摘要排序和筛选匹配。

待补验收：

- 使用 MCP 内置浏览器验证应用依赖维护保存链路。
- 使用真实或模拟 OCI 凭证跑一次 OCI 全类型采集。
- 使用 AWS 凭证补测 EKS/RDS/ElastiCache 等 API 返回映射。
- 真实大规模资产数据下验证列表、DSL、应用聚合和关系图性能。

## 18. 后续路线图

### P0

- 恢复 MCP 内置浏览器能力，完成应用依赖维护点击回归。
- CMDB 到环境资源详情的返回入口已完成第一阶段；后续可补资源不存在或权限不足时的友好提示。
- 补 AWS 未覆盖的 LB、S3、EIP、Route Table。
- 云采集任务日志、失败原因、最近成功同步展示、API 调用耗时页签、API 调用筛选、任务详情慢调用告警、筛选结果 CSV 导出、API 维度趋势、事件中心慢 API 告警、同步策略级慢调用阈值持久化、慢 API 告警静默窗口、失败 scope 自动局部重试、通知负责人、通知路由和分派对象元数据已完成第一阶段；OCI provider 原生错误码解析和 429/5xx 短重试已完成第一阶段，后续补更多 provider 错误样本、按错误类型/服务动态升级和 API 子调用级局部补偿。

### P1

- AliCloud collector 第一版已完成第一阶段；后续补真实账号联调和细分资源映射。
- 风险规则配置化已完成第一阶段；后续补关键应用、核心链路、生产环境和发布计划联动。
- 导入模板、导入预校验和差异预览已完成第一阶段；后续补 CSV/Excel、字段映射和异步大批量导入。
- Webhook/事件推送第一版已完成；后续补 CMDB 事件模板和 ITSM/数据湖适配。
- CMDB 编辑/导出权限细分和响应层敏感字段脱敏已完成第一阶段；后续补标签级授权和更完整角色矩阵回归。

### P2

- Azure/GCP/腾讯云/华为云 collector 已完成第一阶段；后续补真实生产账号联调和更多资源类型。
- 大规模关系图优化已完成第一阶段；关系筛选、分页查询和加载更多已完成第一阶段，后续补真实压测、稳定排序游标、视口渐进展开和前端虚拟化。
- 成本、合规和生命周期治理报表已完成第一阶段；后续补筛选维度、导出、趋势和大规模资产性能优化。
- 资产变更事件与发布/审批/告警联动。

## 19. 当前文件映射

后端：

- `backend/portal/models/cmdb.go`
- `backend/portal/models/forms/cmdb.go`
- `backend/portal/models/resps/cmdb.go`
- `backend/portal/apps/cmdb.go`
- `backend/portal/apps/cmdb_dsl.go`
- `backend/portal/apps/cmdb_application.go`
- `backend/portal/apps/cmdb_asset_upsert.go`
- `backend/portal/apps/cmdb_collect.go`
- `backend/portal/apps/cmdb_collect_alicloud.go`
- `backend/portal/apps/cmdb_collect_aws.go`
- `backend/portal/apps/cmdb_collect_azure.go`
- `backend/portal/apps/cmdb_collect_gcp.go`
- `backend/portal/apps/cmdb_collect_huawei.go`
- `backend/portal/apps/cmdb_collect_inventory.go`
- `backend/portal/apps/cmdb_collect_oci.go`
- `backend/portal/apps/cmdb_collect_tencent.go`
- `backend/portal/apps/cmdb_governance_report.go`
- `backend/portal/apps/cmdb_relation_perf_test.go`
- `backend/portal/apps/cmdb_risk_rule.go`
- `backend/portal/apps/cmdb_sync.go`
- `backend/portal/web/api/v1/handlers/cloud_asset.go`
- `backend/portal/web/api/v1/handlers/cmdb.go`
- `backend/portal/web/api/v1/route.go`

前端：

- `frontend/app/services/cloud-asset.js`
- `frontend/app/services/cmdb.js`
- `frontend/app/containers/org/resource-query/index.jsx`
- `frontend/app/containers/org/resource-query/styles.less`
- `frontend/app/containers/org/m-project/env/detail/components/resource/table-layout/index.jsx`
- `frontend/app/containers/org/m-project/env/detail/components/resource/components/detail-drawer/index.jsx`

## 20. 待开发清单

| 优先级 | 事项 | 状态 |
| --- | --- | --- |
| P0 | MCP 浏览器恢复后补应用依赖维护 UI 点击回归 | 待验证 |
| P0 | CMDB 详情增加返回来源环境资源详情入口 | 已完成第一阶段 |
| P0 | AWS LB/S3/EIP/Route Table/NAT/IGW 采集 | 已完成第一阶段 |
| P0 | OCI Route Table/NAT/IGW/Service Gateway/DRG 采集 | 已完成第一阶段 |
| P0 | OCI compartment/identity 归属映射 | 已完成第一阶段 |
| P0 | OCI provider 原生错误码解析与失败分类 | 已完成第一阶段 |
| P0 | OCI API 限流与临时错误短重试 | 已完成第一阶段 |
| P0 | OCI API 子调用耗时与尝试次数指标 | 已完成第一阶段 |
| P0 | 云采集任务 API 调用耗时页签 | 已完成第一阶段，含 region/service/status、慢调用过滤、任务详情慢调用告警、CSV 导出、API 维度趋势、事件中心慢 API 告警、同步策略级阈值持久化和慢 API 告警静默窗口 |
| P0 | 云采集任务最近同步、范围、失败日志增强 | 已完成第一阶段 |
| P1 | AliCloud collector 第一版 | 已完成第一阶段 |
| P1 | Webhook/事件推送 | 已完成第一阶段 |
| P1 | 风险规则配置化 | 已完成第一阶段 |
| P1 | 导入模板、预校验、差异预览 | 已完成第一阶段 |
| P1 | 编辑/导出权限细分 | 已完成第一阶段 |
| P1 | GKE/AKS 节点池信息 | 已完成第一阶段 |
| P1 | K8S 工作负载层信息 | 已完成第一阶段，可见性已增强，兼容旧数据、导入数据和原生字段变体 |
| P2 | Azure/GCP/腾讯云/华为云 collector | 已完成第一阶段 |
| P2 | 图谱性能优化和大规模资产压测 | 已完成查询保护和关系筛选第一阶段，真实压测待验证 |
| P2 | 成本、合规、生命周期报表 | 已完成第一阶段 |
