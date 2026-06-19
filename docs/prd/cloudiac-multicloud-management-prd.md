# CloudIaC 多云管理平台二次开发 PRD

版本：v1.0  
日期：2026-06-20  
范围：在现有 CloudIaC IaC 自动化、合规扫描、资源回写和 CMDB 能力之上，建设面向多云账号、资产、操作、成本、风险治理的一体化管理平台。

## 1. 背景

CloudIaC 当前定位是基础设施即代码管理平台，核心链路围绕组织、项目、环境、模板、变量、任务、Runner、合规策略和资源回写展开。平台已经可以通过 Terraform/Ansible 等模板驱动云资源变更，并将任务结果、IaC 资源、漂移信息、合规扫描结果和部分账单数据沉淀到 Portal。

近期新增的 CMDB 能力进一步把 `iac_resource`、云采集资产、人工导入资产和应用依赖统一到组织级资产视图，已经具备资产列表、资产详情、资产关系、应用依赖、云采集任务和导入导出等基础能力。

如果要把平台完善为真正的多云管理平台，需要从“执行 IaC 和查看资源”升级为“统一接入多云账号、持续采集真实云资产、执行受控云操作、治理风险与成本、形成可审计闭环”。本 PRD 梳理现有功能、能力缺口和二次开发需求。

## 2. 产品目标

### 2.1 总体目标

- 建设统一多云账号中心，管理 AWS、OCI、AliCloud、Azure、GCP、腾讯云、华为云等账号、区域、凭证、权限和健康状态。
- 建设云资产中心 2.0，将 IaC 资源、云原生采集资源、人工导入资源和应用关系统一到 CMDB。
- 建设多云资源操作能力，支持常用资源的安全操作、审批、异步执行、审计和回滚提示。
- 建设多云成本中心，覆盖账单采集、成本分摊、预算、趋势、异常和闲置资源识别。
- 建设多云风险治理能力，围绕合规、暴露面、漂移、未纳管资产、无归属资产和高风险变更形成治理闭环。
- 保留 CloudIaC 现有 IaC 自动化优势，将模板、环境、任务、合规和 CMDB 连接成完整平台体验。

### 2.2 非目标

- 不在第一阶段替代所有云厂商控制台的全量管理能力。
- 不在第一阶段覆盖每个云厂商的全部资源类型和全部操作。
- 不在第一阶段引入独立图数据库；资产关系和拓扑先基于现有关系表和查询服务实现。
- 不允许绕过权限、审批和审计直接执行生产破坏性操作。
- 不改变现有 Terraform/Ansible 执行平面架构，Runner/Worker 仍作为受控执行侧。

## 3. 现有功能梳理

### 3.1 组织、用户和权限

| 能力 | 当前状态 | 说明 |
| --- | --- | --- |
| 用户登录、注册、激活、重置密码 | 已有 | `/auth/*`、用户自服务接口已存在 |
| 组织、组织用户、角色关系 | 已有 | 支持组织管理、邀请、移除和角色更新 |
| 系统配置、系统状态 | 已有 | 支持系统开关、Registry 地址、Provider 缓存清理、健康状态 |
| 操作日志 | 已有 | 平台概览下已有用户操作日志查询 |
| 细粒度云操作权限 | 待开发 | 当前权限主要围绕组织、项目、模板、策略、环境等对象，缺少云账号/资源类型/操作动作级权限 |

### 3.2 IaC 自动化管理

| 能力 | 当前状态 | 说明 |
| --- | --- | --- |
| 项目与环境管理 | 已有 | `Env` 是核心业务对象，绑定项目、模板、变量、Runner、任务状态和资源状态 |
| 云模板管理 | 已有 | 支持 VCS/Registry、本地模板、Terraform 版本、工作目录、触发器和策略开关 |
| 变量与变量组 | 已有 | 支持组织/项目维度变量、变量组和云账号参数承载 |
| 密钥管理 | 已有 | 支持 SSH Key 等执行凭证 |
| Runner 与标签 | 已有 | 支持 Runner 查询、标签查询、Consul 标签更新 |
| 任务执行 | 已有 | 支持 Terraform plan/apply/destroy、Ansible play、日志、步骤、状态回传 |
| 审批和评论 | 已有 | 环境任务具备审批、评论、重试、自动审批等能力 |
| 自动部署和定时漂移检测 | 已有 | 环境模型包含 cron deploy、cron drift、自动修复等字段 |
| 回调和事件 | 部分已有 | 任务回调、VCS webhook 已存在；平台级云资源事件中心待开发 |

### 3.3 合规治理

| 能力 | 当前状态 | 说明 |
| --- | --- | --- |
| 策略、策略模板、策略组 | 已有 | 支持策略配置、模板关系、策略组关系和报告 |
| 模板扫描、环境扫描 | 已有 | 支持扫描任务、扫描结果、策略状态 |
| 在线测试与解析 | 已有 | 支持策略 parse/test |
| 策略抑制 | 已有 | 支持 suppress 来源查询、更新和删除 |
| 多云资产合规映射 | 待增强 | 当前偏 IaC/扫描结果视角，缺少跨云资产类型、风险分类和整改工作流 |

### 3.4 资源、漂移和平台概览

| 能力 | 当前状态 | 说明 |
| --- | --- | --- |
| IaC 资源回写 | 已有 | 任务结果写入 `iac_resource`，包含 provider、type、address、attrs、dependencies 等 |
| 资源敏感字段处理 | 已有 | 资源字段支持敏感字段识别和隐藏 |
| 资源漂移 | 已有 | 支持 drift task、resource drift 和漂移详情 |
| 组织/项目资源搜索 | 已有 | 支持组织和项目维度资源搜索及筛选项 |
| 平台统计 | 已有 | 支持基础数据、provider 环境/资源、资源类型、近 7 天变化、活跃资源、合规统计、今日统计 |
| 云原生真实资产覆盖 | 部分已有 | 依赖 CMDB 云采集，目前仅 AWS/OCI 较可用，AliCloud 和更多云待补齐 |

### 3.5 账单和成本

| 能力 | 当前状态 | 说明 |
| --- | --- | --- |
| 资源账单模型 | 已有 | 已有 `Bill`、`BillData` 等模型 |
| AliCloud 账单采集 | 部分已有 | `billcollect` 当前主要支持 AliCloud |
| 环境/项目成本统计 | 部分已有 | 项目资源增长和费用趋势已有接口基础 |
| 多云账单统一 | 待开发 | AWS/Azure/GCP/OCI/腾讯云/华为云账单采集、汇率、摊销、预算和异常待建设 |

### 3.6 CMDB 和云资产

| 能力 | 当前状态 | 说明 |
| --- | --- | --- |
| CMDB 数据模型 | 已有 | 资产、关系、变更、同步任务、应用关系模型已存在 |
| 资产列表、筛选、详情 | 已有 | 支持关键字、筛选、DSL、详情抽屉、原始数据、变更和关系 |
| IaC 资源回填 | 已有 | 可从 `iac_resource` 回填 CMDB 资产 |
| IaC dependency 资产关系 | 已有 | 依赖关系会写入资产关系表 |
| 应用依赖 | 已有 | 支持应用列表、详情、上下游关系、人工维护关系和风险信息 |
| 应用推演资产关联 | 已有 | 资产详情关系中已增加基于应用依赖的推演关联 |
| 导入导出 | 已有 | 支持 JSON 导入、CSV/JSON 导出和归属字段覆盖 |
| 云账号识别 | 部分已有 | 从变量组、资源账号识别可采集账号，但尚未统一账号中心 |
| AWS 云采集 | 部分已有 | 已支持 EC2、VPC、Subnet、SecurityGroup、EBS、EKS、RDS、ElastiCache 等 |
| OCI 云采集 | 已有 | 已支持 Compute、VCN/Subnet、安全列表/NSG、公网 IP、块存储、LB、Bucket、OKE、DB、Redis 等 |
| AliCloud 云采集 | 待开发 | 目前只有账号识别和类型声明，collector 仍是 placeholder |
| Azure/GCP/腾讯云/华为云 | 待开发 | 尚无 collector |
| 云资源生命周期操作 | 待开发 | 当前以查询、采集、导入、回填为主，不支持云资源开停、扩缩、标签、网络安全组等操作 |

## 4. 主要差距

| 方向 | 当前问题 | 二开方向 |
| --- | --- | --- |
| 云账号 | 变量组、资源账号、CMDB 云账号分散，缺少统一模型、权限校验和健康检查 | 建设统一云账号中心 |
| Provider 适配 | IaC 侧可通过 Terraform 泛化，但云资产采集和操作没有统一 adapter | 建设 provider adapter/plugin 接口 |
| 资产覆盖 | AWS/OCI 部分可采集，AliCloud/Azure/GCP/腾讯云/华为云缺口明显 | 分阶段补齐 collector 和资源标准模型 |
| 资源操作 | 只有 IaC 任务变更，缺少控制台式安全操作能力 | 建设云操作目录、异步任务、审批和审计 |
| 成本 | AliCloud 有账单基础，多云 FinOps 能力不足 | 建设多云账单、预算、分摊、异常和优化建议 |
| 合规 | 已有 IaC 扫描，缺少基于真实云资产的跨云风险治理 | 将策略结果、CMDB 资产和云配置检查打通 |
| 拓扑 | 已有资产关系和应用关系，缺少跨账号、跨区域、网络安全拓扑 | 建设网络/安全/应用拓扑视图 |
| 权限 | 现有权限不足以约束云账号、资源类型、资源动作和敏感字段 | 增加云资源动作级 RBAC 和审计 |
| 任务 | IaC task 完整，但云原生操作任务模型缺失 | 新增 cloud operation task 及步骤结果模型 |
| 体验 | 多云能力分散在环境、资源、CMDB、合规、账单中 | 新增多云管理一级信息架构 |

## 5. 目标用户

| 角色 | 诉求 |
| --- | --- |
| 平台管理员 | 接入云账号、配置权限、查看平台覆盖率、治理未纳管资产 |
| 云运维/SRE | 查询跨云资产、执行安全操作、定位关系影响、处理漂移和风险 |
| 应用负责人 | 查看应用绑定资源、上下游依赖、成本、风险和变更影响 |
| 安全/合规人员 | 查看高危资产、暴露面、策略违规、整改状态和审计证据 |
| FinOps/财务 | 查看多云账单、预算、成本趋势、分摊和异常 |
| 开发/项目成员 | 通过环境、模板和资源视图完成自助申请、变更和问题定位 |

## 6. 二次开发功能需求

### 6.1 多云账号中心

#### 6.1.1 需求说明

建设统一云账号模型，替代当前变量组、资源账号、CMDB 临时账号识别之间的割裂体验。账号中心负责账号接入、凭证引用、区域配置、权限验证、采集策略、默认 Runner 绑定和审计。

#### 6.1.2 功能清单

| 功能 | 优先级 | 说明 |
| --- | --- | --- |
| 云账号列表 | P0 | 展示 provider、账号 ID、账号名、区域数、状态、最近验证时间、最近同步时间 |
| 云账号创建/编辑 | P0 | 支持 AWS、OCI、AliCloud，后续扩展 Azure/GCP/腾讯云/华为云 |
| 凭证安全存储 | P0 | 凭证只保存引用或加密密文，页面不回显敏感值 |
| 账号权限验证 | P0 | 校验凭证有效性、可访问区域、基础只读权限和操作权限 |
| 区域管理 | P0 | 支持启用/禁用区域、默认同步区域、按区域配置资源类型 |
| Runner 绑定 | P1 | 指定账号同步和操作使用的 Runner/tag |
| 账号健康检查 | P1 | 定期检查凭证过期、权限变化、API 限流和最近同步状态 |
| 账号迁移兼容 | P1 | 兼容变量组和资源账号，提供迁移/关联能力 |

#### 6.1.3 验收标准

- 用户可在组织下新增 AWS/OCI/AliCloud 账号并完成验证。
- 凭证敏感字段不会在接口响应和页面中明文展示。
- 账号验证失败时能显示明确错误、缺失权限和建议处理方式。
- CMDB 云采集任务可直接选择统一云账号，而不再依赖临时识别结果。

### 6.2 云资产中心 2.0

#### 6.2.1 需求说明

在现有 CMDB 基础上补齐 provider adapter、资产标准模型、同步策略、增量采集、采集日志、资产覆盖率和未纳管资产识别。

#### 6.2.2 功能清单

| 功能 | 优先级 | 说明 |
| --- | --- | --- |
| Provider adapter 接口 | P0 | 标准化 `ListAssets`、`GetAsset`、`NormalizeAsset`、`ListRelations`、`ValidateAccount` |
| AWS collector 补齐 | P0 | 补齐 EIP、ELB/ALB/NLB、S3、Route Table、NAT Gateway、Internet Gateway |
| OCI collector 稳定化 | P0 | 增加错误分类、分页、限流重试和更多关系映射 |
| AliCloud collector | P1 | ECS、VPC、VSwitch、SecurityGroup、EIP、SLB、RDS、Redis、OSS、ACK |
| Azure collector | P2 | VM、VNet、Subnet、NSG、Public IP、Disk、LB、AKS、SQL、Storage |
| GCP collector | P2 | Compute、VPC、Subnet、Firewall、Disk、LB、GKE、Cloud SQL、Bucket |
| 腾讯云/华为云 collector | P2 | 覆盖计算、网络、存储、数据库和 Kubernetes 基础资源 |
| 同步策略 | P1 | 支持定时同步、手动同步、按账号/区域/类型同步、失败重试 |
| 同步日志 | P1 | 展示阶段、区域、资源类型、成功数、失败数、错误详情和耗时 |
| 未纳管资产识别 | P1 | 对比 IaC 资源与云采集资产，标记 IaC managed、cloud-only、manual |
| 资产覆盖率 | P1 | 按账号、区域、资源类型展示最近同步覆盖和失败范围 |

#### 6.2.3 验收标准

- AWS/OCI/AliCloud 至少各能完成一次真实或 mock 账号的同步任务。
- 同步任务失败时不会影响已成功采集的区域和类型，页面能看到失败明细。
- 同一资产重复同步不会重复入库，无变化时计入 skipped。
- 云采集资产能与 IaC 资源通过 provider、account、region、nativeId/address 等信息建立关联。

### 6.3 多云资源操作中心

#### 6.3.1 需求说明

建设可控的云原生资源操作能力。操作必须经过权限校验、参数校验、可选审批、异步执行、结果回写和审计记录。第一阶段只开放低风险和可恢复操作。

#### 6.3.2 功能清单

| 功能 | 优先级 | 说明 |
| --- | --- | --- |
| 操作目录 | P1 | 按 provider、assetType、status 返回可执行动作 |
| 操作参数表单 | P1 | 根据动作动态渲染参数，如标签、开停机、重启、磁盘扩容 |
| 权限预检查 | P1 | 检查用户权限、账号权限、资源状态和操作风险等级 |
| Dry run | P1 | 对支持的 provider 执行预检查，不支持时给出模拟检查结果 |
| 异步操作任务 | P1 | 创建 cloud operation task，展示步骤、日志、结果和错误 |
| 审批接入 | P1 | 高风险动作复用现有审批能力 |
| 操作审计 | P1 | 记录谁在何时对哪个资源执行了什么动作、参数和结果 |
| CMDB 回写 | P1 | 操作成功后触发资产刷新或状态修正 |
| 首批动作 | P1 | VM start/stop/reboot、标签更新、磁盘扩容、公网 IP 绑定/解绑查询态确认 |
| 高风险动作 | P2 | 删除、释放、网络 ACL/安全组写操作默认需要审批和二次确认 |

#### 6.3.3 验收标准

- 用户在资产详情中能看到当前资源可执行动作。
- 执行动作后产生操作任务，任务有 running/success/failed/canceled 状态。
- 无权限用户无法执行操作，接口返回明确权限错误。
- 操作成功后 CMDB 资产状态在下一次刷新或任务回写后更新。

### 6.4 网络、安全和拓扑视图

#### 6.4.1 需求说明

基于 CMDB 资产关系，建设跨云网络拓扑、安全暴露面和应用影响关系视图，帮助用户判断资源连接、访问路径和变更影响。

#### 6.4.2 功能清单

| 功能 | 优先级 | 说明 |
| --- | --- | --- |
| 网络拓扑 | P1 | 展示 VPC/VNet/VCN、Subnet、Route Table、NAT、IGW、LB、Instance、DB |
| 安全组/规则视图 | P1 | 展示安全组、NSG、安全列表、入站/出站规则 |
| 公网暴露识别 | P1 | 标识公网 IP、0.0.0.0/0、开放高危端口 |
| 应用拓扑 | P1 | 基于应用依赖和资产关系展示应用到资源、资源到资源 |
| 影响分析 | P1 | 资源变更时推演影响的应用、上游、下游和关键链路 |
| 跨账号/跨区域过滤 | P2 | 支持按账号、区域、业务线、应用筛选拓扑 |

#### 6.4.3 验收标准

- 用户可从资产详情进入关系图，并看到 IaC 依赖和应用推演关系。
- 用户可按应用查看绑定资产、上下游应用和近期变更影响。
- 用户可识别公网暴露资源和高风险安全规则。

### 6.5 多云成本中心

#### 6.5.1 需求说明

把现有 AliCloud 账单基础扩展为多云 FinOps 能力，支持账单采集、成本分摊、预算、趋势、异常检测和优化建议。

#### 6.5.2 功能清单

| 功能 | 优先级 | 说明 |
| --- | --- | --- |
| 账单 provider adapter | P1 | 标准化账单拉取、解析、币种、周期和资源 ID 映射 |
| AliCloud 账单完善 | P1 | 补齐日账单、资源匹配率、失败重试和账单同步任务 |
| AWS 成本接入 | P1 | 支持 CUR 或 Cost Explorer，按账号/服务/资源/标签聚合 |
| OCI 成本接入 | P2 | 支持 Usage/Cost 数据同步 |
| Azure/GCP 成本接入 | P2 | 支持 Cost Management、Billing Export |
| 成本分摊 | P1 | 按组织、项目、环境、应用、业务线、owner、标签分摊 |
| 预算管理 | P1 | 支持月度预算、阈值预警、超预算记录 |
| 成本异常 | P2 | 检测环比/同比/突增、闲置实例、未绑定磁盘、未使用公网 IP |
| 成本优化建议 | P2 | 输出资源关停、规格调整、预留实例/节省计划建议 |

#### 6.5.3 验收标准

- 至少支持 AliCloud 与 AWS 两个 provider 的月度成本导入或 API 同步。
- 成本能按项目、环境、应用和业务线聚合展示。
- 未能匹配 CMDB/IaC 资产的账单项会进入未匹配列表。
- 预算超阈值后生成告警/事件记录。

### 6.6 合规和风险治理中心

#### 6.6.1 需求说明

在现有策略扫描基础上，将 IaC 合规、云原生配置检查、CMDB 风险字段、暴露面、未纳管资产和漂移统一成风险治理视图。

#### 6.6.2 功能清单

| 功能 | 优先级 | 说明 |
| --- | --- | --- |
| 风险发现模型 | P1 | 统一记录风险来源、资源、规则、等级、状态、证据和修复建议 |
| 云资产配置检查 | P1 | 针对公网暴露、未加密磁盘、弱安全组、无备份数据库等规则 |
| IaC 扫描映射 CMDB | P1 | 将策略违规结果关联到 CMDB 资产和应用 |
| 漂移风险 | P1 | 将 drift 结果纳入风险列表 |
| 未纳管资产风险 | P1 | cloud-only 资产可标记为治理项 |
| 风险例外 | P1 | 复用/扩展 suppress，支持到期时间和审批记录 |
| 整改工作流 | P2 | 支持创建整改任务、Webhook/ITSM 推送、状态回写 |
| 风险评分 | P2 | 按资产暴露、重要性、应用依赖、违规数量计算综合风险 |

#### 6.6.3 验收标准

- 用户可按 provider、账号、区域、应用、风险等级筛选风险。
- 单个风险能查看证据、受影响资产、应用影响和修复建议。
- 已抑制风险不计入未处理高危统计，且能看到到期时间。

### 6.7 自动化、事件和集成

#### 6.7.1 需求说明

把云采集、云操作、合规风险、成本预算和 CMDB 变更统一纳入事件体系，支持外部系统集成。

#### 6.7.2 功能清单

| 功能 | 优先级 | 说明 |
| --- | --- | --- |
| 云操作任务模型 | P1 | 新增 cloud operation task、step、result |
| 事件中心 | P1 | 统一记录账号验证、采集完成、操作完成、风险发现、预算超限 |
| Webhook | P1 | 支持按事件类型推送到外部系统 |
| ITSM 集成 | P2 | 风险整改、操作审批可接入外部工单 |
| 告警集成 | P2 | 支持企业微信、钉钉、Slack、邮件等通知 |
| OpenAPI/API Token | P2 | 为外部平台提供多云资产、成本、风险查询 API |

#### 6.7.3 验收标准

- CMDB 资产变更、云操作完成、风险发现均能生成事件。
- Webhook 失败有重试和失败记录。
- 外部系统可通过 API 查询资产和操作任务状态。

## 7. 建议数据模型

| 模型 | 类型 | 说明 |
| --- | --- | --- |
| `iac_cloud_account` | 新增 | 统一云账号主表，保存 provider、accountId、name、status、owner、runnerTag、lastValidatedAt、lastSyncAt |
| `iac_cloud_account_credential` | 新增 | 云账号凭证引用或加密密文，按 provider 存储 key schema |
| `iac_cloud_account_region` | 新增 | 账号启用区域、默认区域、同步开关和区域状态 |
| `iac_cloud_account_permission` | 新增 | 最近权限验证结果、缺失权限、支持动作 |
| `iac_cloud_sync_policy` | 新增 | 定时同步策略、资源类型范围、重试策略 |
| `iac_cloud_operation` | 新增 | 云资源操作任务主表，记录资源、动作、状态、风险等级、审批 ID |
| `iac_cloud_operation_step` | 新增 | 操作步骤、日志、错误、耗时 |
| `iac_cloud_operation_audit` | 新增 | 操作审计记录，包含用户、参数摘要、结果、IP、UserAgent |
| `iac_cloud_cost_record` | 新增 | 标准化成本明细，包含 provider、account、resourceId、service、amount、currency、period |
| `iac_cloud_budget` | 新增 | 预算配置，支持组织/项目/环境/应用/业务线维度 |
| `iac_cloud_risk_finding` | 新增 | 风险发现表，关联 CMDB 资产、策略、证据和整改状态 |
| `iac_cloud_event` | 新增 | 平台事件表，统一记录同步、操作、风险、成本和 CMDB 变更 |
| `iac_cmdb_asset` | 扩展 | 增加 managedBy、cloudAccountId、syncPolicyId、lastOperationId、riskScore、costCenter 等字段 |

## 8. 建议 API

### 8.1 云账号

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/cloud/accounts` | 云账号列表 |
| POST | `/api/v1/cloud/accounts` | 创建云账号 |
| GET | `/api/v1/cloud/accounts/:id` | 云账号详情 |
| PUT | `/api/v1/cloud/accounts/:id` | 更新云账号 |
| DELETE | `/api/v1/cloud/accounts/:id` | 删除/禁用云账号 |
| POST | `/api/v1/cloud/accounts/:id/validate` | 验证账号凭证和权限 |
| GET | `/api/v1/cloud/accounts/:id/regions` | 查询账号区域 |
| PUT | `/api/v1/cloud/accounts/:id/regions` | 更新启用区域 |
| GET | `/api/v1/cloud/accounts/:id/permissions` | 查看权限验证结果 |

### 8.2 云资产和同步

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/cloud/assets` | 多云资产列表，可复用/包装 CMDB 查询 |
| GET | `/api/v1/cloud/assets/:id` | 云资产详情 |
| GET | `/api/v1/cloud/assets/:id/relations` | 资产关系、应用推演关系、网络关系 |
| POST | `/api/v1/cloud/sync-tasks` | 创建同步任务 |
| GET | `/api/v1/cloud/sync-tasks` | 同步任务列表 |
| GET | `/api/v1/cloud/sync-tasks/:id` | 同步任务详情、日志和统计 |
| POST | `/api/v1/cloud/sync-policies` | 创建定时同步策略 |
| PUT | `/api/v1/cloud/sync-policies/:id` | 更新同步策略 |

### 8.3 云操作

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/cloud/assets/:id/actions` | 查询资源可执行动作 |
| POST | `/api/v1/cloud/assets/:id/actions/:action/dry-run` | 操作预检查 |
| POST | `/api/v1/cloud/assets/:id/actions/:action` | 创建操作任务 |
| GET | `/api/v1/cloud/operations` | 操作任务列表 |
| GET | `/api/v1/cloud/operations/:id` | 操作任务详情 |
| POST | `/api/v1/cloud/operations/:id/cancel` | 取消操作任务 |
| GET | `/api/v1/cloud/operations/:id/audits` | 操作审计 |

### 8.4 成本、风险和事件

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/cloud/cost/summary` | 成本概览 |
| GET | `/api/v1/cloud/cost/trends` | 成本趋势 |
| GET | `/api/v1/cloud/cost/records` | 成本明细 |
| GET | `/api/v1/cloud/cost/unmatched` | 未匹配资产账单 |
| GET | `/api/v1/cloud/budgets` | 预算列表 |
| POST | `/api/v1/cloud/budgets` | 创建预算 |
| GET | `/api/v1/cloud/risks` | 风险发现列表 |
| PUT | `/api/v1/cloud/risks/:id/status` | 更新风险状态 |
| POST | `/api/v1/cloud/risks/:id/suppress` | 风险例外 |
| GET | `/api/v1/cloud/events` | 事件列表 |
| POST | `/api/v1/cloud/webhooks` | Webhook 配置 |

## 9. 页面和信息架构

建议新增组织级一级入口：多云管理。

| 一级页面 | 二级页面 | 说明 |
| --- | --- | --- |
| 总览 | 多云总览 | 账号数、资产数、成本、风险、未纳管、同步健康、操作成功率 |
| 云账号 | 账号列表、账号详情、权限验证、区域配置 | 管理账号、凭证、区域、同步策略 |
| 云资产 | 资产列表、资产详情、资产关系、原始数据 | 承接现有 CMDB 资产中心 |
| 应用视图 | 应用列表、应用详情、上下游、近期变更 | 承接现有应用依赖能力 |
| 拓扑 | 网络拓扑、安全拓扑、应用拓扑 | 基于 CMDB 关系和云网络关系 |
| 操作任务 | 操作列表、任务详情、审计 | 云原生资源操作入口 |
| 成本 | 成本概览、趋势、预算、异常、未匹配账单 | 多云 FinOps |
| 风险合规 | 风险列表、规则、证据、整改、例外 | 合规和风险治理 |
| 自动化 | 同步策略、事件、Webhook | 定时任务和外部集成 |
| 设置 | Provider 配置、字段映射、资源类型、权限策略 | 平台级配置 |

现有菜单中的“资产 CMDB”可迁移或聚合到“多云管理 - 云资产”，环境详情中的资源入口继续保留，用于从 IaC 视角跳转到多云资产视角。

## 10. Provider 分阶段覆盖

| 阶段 | Provider | 范围 |
| --- | --- | --- |
| P0 | AWS、OCI | 统一账号中心、只读采集、资产标准化、同步日志、CMDB 资产详情和总览 |
| P1 | AliCloud | 补齐资产采集、账单采集完善、安全操作框架首批动作 |
| P2 | Azure、GCP | 资产采集、成本接入、网络/安全拓扑、基础操作 |
| P2 | 腾讯云、华为云 | 资产采集、成本接入、基础风险规则 |
| P3 | 全部已接入 provider | 插件 SDK、规则市场、自动整改、ITSM 深度集成 |

## 11. 版本规划

### 11.1 V1.0 多云只读治理底座

目标：把账号、资产、同步和总览打通，形成可靠只读治理平台。

- 新增统一云账号模型和账号中心页面。
- 改造 CMDB 云采集使用统一云账号。
- 稳定 AWS/OCI collector，补齐 AWS 常用资产类型。
- 同步任务增加日志、统计、区域/类型明细。
- 多云总览展示账号、资产、风险、未纳管和同步健康。
- CMDB 资产增加 managed 状态和账号关联。

### 11.2 V1.1 AliCloud 和操作任务框架

目标：补齐国内常用云和安全操作闭环。

- 实现 AliCloud collector。
- 抽象 provider adapter 和 operation adapter。
- 新增 cloud operation task、step、audit 数据模型。
- 首批支持 VM start/stop/reboot、标签更新。
- 高风险动作接入现有审批能力。
- 操作成功后回写 CMDB 状态和事件。

### 11.3 V1.2 FinOps 和风险治理

目标：让多云管理进入成本和安全治理闭环。

- 扩展 AliCloud/AWS/OCI 成本采集。
- 成本按项目、环境、应用、业务线、owner 分摊。
- 新增预算、成本异常和未匹配账单。
- 新增风险发现模型，打通 IaC 扫描、云配置检查、漂移和 CMDB。
- 支持风险例外、整改状态和 Webhook 推送。

### 11.4 V1.3 多云扩展和开放集成

目标：扩大云厂商覆盖并形成开放平台能力。

- 接入 Azure、GCP、腾讯云、华为云基础资产和成本。
- 建设网络拓扑、安全拓扑和应用拓扑。
- 提供 provider plugin SDK。
- 支持 ITSM、告警平台和外部数据湖集成。

## 12. 验收标准

| 类别 | 验收项 |
| --- | --- |
| 账号 | 至少 AWS/OCI/AliCloud 支持账号创建、凭证校验、区域配置和状态展示 |
| 资产 | 至少 AWS/OCI/AliCloud 常用计算、网络、存储、数据库资源可同步入 CMDB |
| 同步 | 同步任务有状态、日志、统计、失败明细和最近成功时间 |
| CMDB | 资产可按 provider、账号、区域、类型、来源、应用、owner 查询 |
| 关系 | 资产关系同时支持 IaC dependency、云原生关系和应用依赖推演 |
| 操作 | 首批云操作有权限校验、异步任务、审批可选、审计和结果回写 |
| 成本 | 至少 AliCloud/AWS 成本可导入或同步，并能按项目/环境/应用聚合 |
| 风险 | 能展示高危暴露、漂移、未纳管、无归属和策略违规资产 |
| 权限 | 非授权用户无法查看敏感凭证或执行云操作 |
| 审计 | 账号变更、同步任务、云操作、风险状态变更均可追溯 |
| 兼容 | 现有环境、模板、任务、合规、CMDB 页面不出现回归 |

## 13. 当前代码落点

| 能力 | 现有位置 | 二开建议 |
| --- | --- | --- |
| 路由注册 | `backend/portal/web/api/v1/route.go` | 新增 `/cloud/*` 路由组，CMDB 可逐步迁移 |
| 环境和任务 | `backend/portal/models/env.go`、`backend/portal/models/task.go`、`backend/portal/services/task.go` | 云操作任务可复用任务状态和审批思想，但建议独立模型 |
| IaC 资源 | `backend/portal/models/resource.go`、`backend/portal/services/task.go` | 继续作为 IaC managed 资产来源 |
| 资源账号 | `backend/portal/models/resource_account.go`、`backend/portal/services/resource_account.go` | 迁移或关联到统一云账号 |
| 变量组 | `backend/portal/services/variables_group.go` | 保留变量能力，云账号凭证不再完全依赖变量组 |
| 账单 | `backend/portal/services/bill.go`、`backend/portal/services/billcollect/` | 抽象多云 billing adapter |
| 合规 | `backend/portal/services/policy*.go`、`backend/portal/models/policy*.go` | 风险发现关联策略结果和 CMDB 资产 |
| CMDB | `backend/portal/apps/cmdb*.go`、`backend/portal/models/cmdb.go`、`backend/portal/models/forms/cmdb.go` | 扩展账号关联、同步策略、云关系、风险和操作入口 |
| 前端菜单 | `iac-web/src/components/layouts/components/SideMenu/index.tsx` | 新增“多云管理”一级入口 |
| 前端路由 | `iac-web/src/router/` | 新增 cloud accounts/assets/topology/operations/cost/risks routes |

## 14. 风险和约束

- 凭证安全：多云账号凭证必须加密存储、脱敏返回、限制导出，并记录访问审计。
- API 限流：云采集和成本同步需要分页、重试、退避和区域级失败隔离。
- 数据规模：云资产、账单和事件会快速增长，需要分页、索引、归档和异步导出。
- Provider 差异：不同云厂商资源模型差异大，必须先建立标准字段和 provider 原始字段并存机制。
- 操作风险：删除、释放、网络安全组写入等动作必须默认高风险，强制审批和二次确认。
- 兼容迁移：当前变量组、资源账号、CMDB 云账号识别不能一次性废弃，需要灰度迁移。
- 执行边界：云操作应优先复用受控 Runner，避免 Portal 直接持有过宽网络和云权限。
- 现有功能回归：多云管理新增入口不能破坏现有 IaC 环境、任务、合规和 CMDB 工作流。

## 15. 待确认问题

- 第一批必须支持的云厂商顺序是否为 AWS、OCI、AliCloud，还是优先 AliCloud/腾讯云/华为云。
- 云资源操作第一阶段是否允许生产账号执行，还是只开放测试账号/只读和标签类操作。
- 成本中心是否需要对接企业现有成本中心、财务编码、预算审批和汇率规则。
- 风险整改是否需要直接接入现有 ITSM/工单系统。
- 是否需要纳入 Kubernetes 多集群管理，还是先聚焦 IaaS/PaaS 云资源。
- 是否需要按私有云/OpenStack/vSphere 设计 provider adapter 扩展点。
