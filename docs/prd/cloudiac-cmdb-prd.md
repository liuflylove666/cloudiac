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
| AWS 云采集 | 部分完成 | 已支持 EC2、VPC、Subnet、SecurityGroup、EBS、EKS、RDS、ElastiCache Redis/Valkey；LB、S3、公网 IP、Route Table 等仍待补 |
| OCI 云采集 | 已完成 | 已支持 Compute、VCN/Subnet、Security List/NSG、Public IP、Block Volume、Load Balancer、Bucket、OKE、DB System、Autonomous DB、Redis |
| AliCloud 云采集 | 待开发 | 已支持账号识别和资产类型声明，collector 仍为 placeholder |
| Azure/GCP/腾讯云/华为云 | 待开发 | 尚未实现 |
| 云采集任务后台化 | 已完成 | `StartCmdbSyncTask` 创建任务后使用 goroutine 后台执行 |
| 云采集任务状态/统计 | 部分完成 | 有状态、错误、stats、StartedAt/EndedAt；缺更细日志、最近成功同步摘要和范围可视化 |
| 环境资源跳 CMDB | 已完成 | 环境资源列表/详情可跳转 CMDB 资产详情 |
| CMDB 返回来源环境资源 | 部分完成 | CMDB 详情展示项目、环境、IaC 地址；缺“一键返回环境资源详情”的显式入口 |
| 资产关系图视图 | 已完成 | 资产详情有关系图和表格兜底 |
| 应用依赖视图 | 已完成 | 应用列表、详情、应用关系图、上下游表格、近期变更和风险等级已实现 |
| 应用依赖维护 | 已完成 | 支持人工维护应用上游/下游依赖，写入 `iac_cmdb_application_relation` |
| 变更风险感知 | 部分完成 | 应用近 7 天变更、上下游影响和风险等级已实现；缺更完整风险规则配置 |
| Kubernetes 集群信息 | 部分完成 | EKS/OKE 已采集为 `kubernetes_cluster`，资产详情已新增 K8S 信息页签展示版本、Endpoint、网络、安全组和节点组/节点池；GKE/AKS 节点池和工作负载层仍待扩展 |
| 资产搜索 DSL | 已完成 | 支持 provider、type、source、status、owner、application、tag.*、attr.* 等 DSL |
| 导入/导出 | 已完成 | 支持 JSON 导入、CSV/JSON 导出、选中导出、归属覆盖 |
| Webhook/事件推送 | 待开发 | 尚未实现 CMDB 变更事件推送 |
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

状态：部分完成。AWS/OCI 已具备，AliCloud 和更多云厂商待开发，任务日志和统计展示需增强。

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
4. 应用详情展示绑定资源、上下游应用、近 7 天变更、影响应用和风险等级。
5. 当某应用绑定资源近期有变更，系统结合上下游依赖生成风险提示。

状态：已完成核心能力，风险规则配置化待开发。

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

## 8. API 设计

| 方法 | 路径 | 说明 | 状态 |
| --- | --- | --- | --- |
| GET | `/api/v1/cmdb/assets` | 资产列表、关键词、筛选、DSL、分页排序 | 已完成 |
| GET | `/api/v1/cmdb/assets/filters` | 资产筛选项 | 已完成 |
| GET | `/api/v1/cmdb/assets/export` | 资产 CSV/JSON 导出 | 已完成 |
| POST | `/api/v1/cmdb/assets/import` | 资产 JSON 导入 | 已完成 |
| GET | `/api/v1/cmdb/assets/:id` | 资产详情、关系、变更 | 已完成 |
| PUT | `/api/v1/cmdb/assets/:id/ownership` | 更新资产归属/生命周期/风险 | 已完成 |
| POST | `/api/v1/cmdb/backfill/iac-resources` | 从 IaC 资源回填 CMDB | 已完成 |
| GET | `/api/v1/cmdb/cloud/accounts` | 查询可采集云账号 | 已完成 |
| GET | `/api/v1/cmdb/sync-tasks` | 查询云采集任务 | 已完成 |
| POST | `/api/v1/cmdb/sync-tasks` | 启动云采集任务 | 已完成 |
| GET | `/api/v1/cmdb/applications` | 应用依赖列表和风险 | 已完成 |
| GET | `/api/v1/cmdb/applications/detail` | 应用依赖详情 | 已完成 |
| PUT | `/api/v1/cmdb/applications/relations` | 维护应用上下游依赖 | 已完成 |
| POST | `/api/v1/cmdb/events/webhook` | CMDB 变更事件推送配置 | 待开发 |

## 9. 云采集范围

### 9.1 AWS

已完成：EC2 Instance、VPC、Subnet、Security Group、EBS Volume、EKS Cluster、EKS NodeGroup 信息、RDS DB Instance、ElastiCache Redis/Valkey Cluster、ElastiCache Replication Group。

待开发：Route Table、Public IP/EIP 独立资产、ELB/ALB/NLB、S3 Bucket、NAT Gateway、Internet Gateway、更完整的标签/成本/合规映射。

状态：部分完成。

### 9.2 OCI

已完成：Compute Instance、VCN、Subnet、Security List、Network Security Group、Public IP、Block Volume、Load Balancer、Object Storage Bucket、OKE Cluster、OKE NodePool 信息、DB System、Autonomous Database、Redis Cluster。

待开发：Route Table 独立资产、NAT Gateway、Internet Gateway、Service Gateway、DRG、完整 compartment/identity 归属映射。

状态：已完成核心范围。

### 9.3 AliCloud

已完成：账号识别、凭证键识别、区域识别、支持资产类型声明。

待开发：ECS、VPC、VSwitch、安全组、SLB、RDS、Redis、OSS collector。

状态：待开发。

### 9.4 Azure/GCP/腾讯云/华为云

状态：待开发。

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

能力：展示可采集云账号、账号来源、provider、ready 状态、缺失凭证；支持选择区域和资产类型；启动云采集任务；查看任务状态、错误、统计和时间。

状态：部分完成。缺更清晰的任务日志、最近成功同步时间聚合展示和失败重试入口。

## 12. 环境资源集成

需求：环境资源列表显示 CMDB 入口；环境资源详情显示“查看 CMDB 资产”；CMDB 资产详情展示来源项目、环境、IaC 地址；后续增加从 CMDB 详情一键返回环境资源详情。

状态：环境资源跳 CMDB 已完成；CMDB 返回环境资源显式入口部分完成。

## 13. 风险感知规则

当前规则：统计应用绑定资产的近 7 天变更；统计上游/下游应用数量；根据近期变更和上下游影响计算风险等级与风险原因。

待增强：风险规则配置化；支持业务线/生命周期/合规风险权重；支持关键应用、核心链路、生产环境加权；支持变更窗口和发布计划联动；支持风险说明可追溯到具体资产变更和关系路径。

状态：部分完成。

## 14. 导入导出

已完成：JSON 导入资产；导入时可选择覆盖归属字段；CSV 导出；JSON 导出；选中资产导出；导出数量上限保护。

待增强：导入模板下载；导入预校验和差异预览；异步大批量导入任务。

状态：已完成核心能力。

## 15. Webhook 与事件

需求：当资产创建、更新、归属变更、应用依赖变更时产生事件；支持配置 Webhook endpoint；支持事件重试、签名、投递日志；支持发送到外部 ITSM、告警、数据湖或消息系统。

状态：待开发。

## 16. 非功能需求

### 16.1 性能

要求：资产列表分页查询；导出默认限制 10000 条；资产 upsert 使用增量 diff；云采集任务后台执行。

状态：部分完成。大规模关系图和应用聚合仍需压测与索引优化。

### 16.2 安全

要求：CMDB API 使用组织权限；云账号凭证从变量组/资源账号读取，敏感值解密后仅用于采集；不在 UI 暴露凭证明文。

状态：部分完成。后续需细化 CMDB 编辑权限、导出权限和敏感字段脱敏策略。

### 16.3 可观测性

要求：采集任务记录 status、errorMessage、stats、startedAt、endedAt；panic 会记录任务失败和堆栈日志。

状态：部分完成。后续需补任务级详细日志、provider API 调用耗时和错误分类。

## 17. 验收标准

已完成能力验收：

- Docker Compose build `iac-portal` 和 `iac-web` 通过。
- Docker Compose up 后 `iac-portal` healthy，`iac-web` 可提供静态资源。
- 运行前端 bundle 包含 `manageRelations`、`updateApplicationRelations`、`keywordSearch`。
- 资产列表、云采集、资产详情、关系、变更页签已通过上一轮浏览器验证。
- 应用依赖核心代码已进入运行镜像，待 MCP 浏览器恢复后补完整点击回归。

待补验收：

- 使用 MCP 内置浏览器验证应用依赖维护保存链路。
- 使用真实或模拟 OCI 凭证跑一次 OCI 全类型采集。
- 使用 AWS 凭证补测 EKS/RDS/ElastiCache 等 API 返回映射。
- 大规模资产数据下验证列表、DSL、应用聚合和关系图性能。

## 18. 后续路线图

### P0

- 恢复 MCP 内置浏览器能力，完成应用依赖维护点击回归。
- 修复/优化 CMDB 到环境资源详情的返回入口。
- 补 AWS 未覆盖的 LB、S3、EIP、Route Table。
- 完善云采集任务日志、失败原因和最近成功同步展示。

### P1

- AliCloud collector 第一版。
- 风险规则配置化。
- 导入模板、导入预校验和差异预览。
- Webhook/事件推送第一版。
- CMDB 编辑/导出权限细分。

### P2

- Azure/GCP/腾讯云/华为云 collector。
- 大规模关系图优化。
- 成本、合规和生命周期治理报表。
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
- `backend/portal/apps/cmdb_collect_aws.go`
- `backend/portal/apps/cmdb_collect_oci.go`
- `backend/portal/apps/cmdb_sync.go`
- `backend/portal/web/api/v1/handlers/cmdb.go`
- `backend/portal/web/api/v1/route.go`

前端：

- `frontend/app/services/cmdb.js`
- `frontend/app/containers/org/resource-query/index.jsx`
- `frontend/app/containers/org/resource-query/styles.less`
- `frontend/app/containers/org/m-project/env/detail/components/resource/table-layout/index.jsx`
- `frontend/app/containers/org/m-project/env/detail/components/resource/components/detail-drawer/index.jsx`

## 20. 待开发清单

| 优先级 | 事项 | 状态 |
| --- | --- | --- |
| P0 | MCP 浏览器恢复后补应用依赖维护 UI 点击回归 | 待验证 |
| P0 | CMDB 详情增加返回来源环境资源详情入口 | 待开发 |
| P0 | AWS LB/S3/EIP/Route Table 采集 | 待开发 |
| P0 | 云采集任务最近同步、范围、失败日志增强 | 待开发 |
| P1 | AliCloud collector 第一版 | 待开发 |
| P1 | Webhook/事件推送 | 待开发 |
| P1 | 风险规则配置化 | 待开发 |
| P1 | 导入模板、预校验、差异预览 | 待开发 |
| P1 | 编辑/导出权限细分 | 待开发 |
| P1 | GKE/AKS 节点池和 K8S 工作负载层信息 | 待开发 |
| P2 | Azure/GCP/腾讯云/华为云 collector | 待开发 |
| P2 | 图谱性能优化和大规模资产压测 | 待开发 |
| P2 | 成本、合规、生命周期报表 | 待开发 |
