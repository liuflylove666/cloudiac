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
| 细粒度云操作权限 | 部分完成 | 云资产动作已按操作动作和项目角色收敛权限，并接入云操作审批第一阶段；云账号级授权和标签级授权第一阶段已接入 provider 动作预检 |

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
| 回调和事件 | 部分完成 | 任务回调、VCS webhook 已存在；已新增平台级云事件模型、事件列表 API、事件中心页面和 Webhook 第一阶段，云账号验证/健康检查、云采集慢 API 告警、风险状态、云操作、CMDB 变更和成本预算可写入/推送事件 |

### 3.3 合规治理

| 能力 | 当前状态 | 说明 |
| --- | --- | --- |
| 策略、策略模板、策略组 | 已有 | 支持策略配置、模板关系、策略组关系和报告 |
| 模板扫描、环境扫描 | 已有 | 支持扫描任务、扫描结果、策略状态 |
| 在线测试与解析 | 已有 | 支持策略 parse/test |
| 策略抑制 | 已有 | 支持 suppress 来源查询、更新和删除 |
| 多云资产合规映射 | 部分完成 | 已新增云风险发现模型、风险列表 API 和风险合规页面，支持公网安全规则、未纳管、无负责人、高合规风险、漂移风险和 AWS S3 Bucket 配置风险派生 |

### 3.4 资源、漂移和平台概览

| 能力 | 当前状态 | 说明 |
| --- | --- | --- |
| IaC 资源回写 | 已有 | 任务结果写入 `iac_resource`，包含 provider、type、address、attrs、dependencies 等 |
| 资源敏感字段处理 | 已有 | 资源字段支持敏感字段识别和隐藏 |
| 资源漂移 | 已有 | 支持 drift task、resource drift 和漂移详情 |
| 组织/项目资源搜索 | 已有 | 支持组织和项目维度资源搜索及筛选项 |
| 平台统计 | 已有 | 支持基础数据、provider 环境/资源、资源类型、近 7 天变化、活跃资源、合规统计、今日统计 |
| 云原生真实资产覆盖 | 部分完成 | 依赖 CMDB 云采集，目前 AWS/OCI/AliCloud/Azure/GCP/腾讯云/华为云均已接入第一阶段采集器，腾讯云 COS 与华为云 OBS 已完成对象存储桶采集第一阶段，离线 inventory 仍作为回退路径 |

### 3.5 账单和成本

| 能力 | 当前状态 | 说明 |
| --- | --- | --- |
| 资源账单模型 | 已有 | 已有 `Bill`、`BillData` 等模型 |
| AliCloud 账单采集 | 部分已有 | `billcollect` 当前主要支持 AliCloud |
| 环境/项目成本统计 | 部分已有 | 项目资源增长和费用趋势已有接口基础 |
| 多云账单统一 | 部分完成 | 已新增统一成本明细、成本汇总、趋势、明细、未匹配账单视图，支持 AWS CUR/AWS Cost Explorer、OCI Usage/Cost、Azure/GCP/TencentCloud/Huawei Billing Export 导入型数据源，Azure Cost Management / GCP Billing Export / 多云 JSON/CSV/TSV URL 拉取、gzip/zip 压缩账单、导出文件索引 URL、增量游标，以及 S3/GCS/Azure Blob/OCI Object Storage 对象存储原生列表 API、签名授权、分页、文件元数据审计、重复文件跳过和前端文件审计展示第一阶段；成本同步任务、日志、失败重试、同步计划、后台 worker、失败退避、自动暂停、拉取事件审计和前端配置入口已完成第一阶段；汇率、摊销、真实云环境联调和复杂财务规则仍待建设 |
| 预算管理 | 部分完成 | 已新增月度预算配置、预算评估、到期评估入口、后台定时评估 worker、超阈值事件和成本中心预算视图；预算审批、专用告警渠道和复杂财务编码仍待建设 |

### 3.6 CMDB 和云资产

| 能力 | 当前状态 | 说明 |
| --- | --- | --- |
| CMDB 数据模型 | 已有 | 资产、关系、变更、同步任务、应用关系模型已存在 |
| 资产列表、筛选、详情 | 已有 | 支持关键字、筛选、DSL、详情抽屉、原始数据、变更和关系 |
| IaC 资源回填 | 已有 | 可从 `iac_resource` 回填 CMDB 资产 |
| IaC dependency 资产关系 | 已有 | 依赖关系会写入资产关系表 |
| 应用依赖 | 已有 | 支持应用列表、详情、上下游关系、人工维护关系和风险信息 |
| 应用推演资产关联 | 已有 | 资产详情关系中已增加基于应用依赖的推演关联 |
| 导入导出 | 已有 | 支持 JSON 导入、导入模板、预检差异预览、CSV/JSON 导出、归属字段覆盖和权限门禁 |
| 云账号识别 | 部分已有 | 从变量组、资源账号识别可采集账号，但尚未统一账号中心 |
| AWS 云采集 | 部分完成 | 已支持 EC2、VPC、Subnet、Route Table、NAT Gateway、Internet Gateway、SecurityGroup、EIP、EBS、ELB/ALB/NLB、LB Listener/Rule/Auth Action/Target Group/Target Health、S3 Bucket 深层配置和 S3 风险规则映射、EKS、RDS、ElastiCache 等；真实 AWS 账号端到端联调、跨账号/跨 VPC target 和更完整成本/合规映射仍待继续 |
| OCI 云采集 | 已完成第一阶段 | 已支持 Compute、VCN/Subnet、Route Table、NAT Gateway、Internet Gateway、Service Gateway、DRG、安全列表/NSG、公网 IP、块存储、LB、Bucket、OKE、DB、Redis 等；已补充 compartment/identity 归属映射、provider 原生错误码解析、限流/临时错误短重试和失败 scope 自动局部重试第一阶段；真实账号端到端联调、更细 identity 权限漂移、分页和 API 子调用级局部补偿仍待继续 |
| AliCloud 云采集 | 部分完成 | 已支持 ECS、VPC、VSwitch、SecurityGroup、EIP、SLB、RDS、Redis、OSS、ACK 第一阶段采集 |
| Azure/GCP 云采集 | 部分完成 | Azure 已接入 ARM 资源列表归一化与 VM 网卡/子网/公网 IP/磁盘引用增强；GCP 已接入 Compute/GKE/SQL/Storage 基础采集和网络/磁盘引用增强 |
| 腾讯云/华为云 | 部分完成 | 已支持真实 API collector、COS/OBS 对象存储桶采集和离线 inventory JSON 回退；成本账单导出 URL 导入第一阶段已完成，真实云账单环境联调和复杂财务规则仍待继续 |
| Kubernetes 集群信息 | 已完成第一阶段 | AWS EKS、OCI OKE、Azure AKS 与 GCP GKE 已作为 `kubernetes_cluster` 资产采集；已补充 EKS NodeGroup、OKE NodePool、AKS NodePool、GKE NodePool、资产详情 K8S 信息页签，以及基于资产属性/导入数据的 Namespace、Node、Pod、Workload、Service、Ingress 工作负载层展示和关系推演；K8S 详情已兼容旧数据、导入数据、`rawData.response/properties`、大小写变体和 OKE `endpoints`；kubeconfig/Agent 实时采集仍待后续阶段 |
| 资产治理报表 | 已完成第一阶段 | 云资产页已提供成本、合规风险、生命周期和归属缺口治理报表，支持总成本、平均风险分、负责人/应用缺口、高/严重风险、生命周期/合规分布、高成本资产和高风险资产入口 |
| 云资源生命周期操作 | 部分完成 | 已建立开停重启动作目录、dry-run、审批、异步任务、最终态轮询、取消/重试和审计；AWS/OCI/AliCloud 已覆盖多类生命周期动作，Azure/GCP 已接入计算实例 live read 与启停重启 adapter，真实写操作默认关闭 |
| 安全组/规则视图 | 部分完成 | 已支持从云资产属性解析安全组/安全列表规则并标识公网暴露；AWS 安全组规则采集字段已展开，安全组写操作待继续 |

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
| 权限 | CMDB/云资产编辑、导出、导入、同步权限细分和响应层敏感字段脱敏已完成第一阶段；云账号、资源类型和标签级授权仍需增强 | 增加云资源动作级 RBAC、标签级授权和审计 |
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
| 凭证安全存储 | P0 | 凭证只保存引用或加密密文，页面不回显敏感值；响应层已按字段名强制脱敏私钥、Token、Password、Secret、AccessKey 等敏感凭证 |
| 账号权限验证 | P0 | 校验凭证有效性、可访问区域、基础只读权限和操作权限；权限验证结果已落入 `iac_cloud_account_permission` 快照表，支持最近检查时间和来源展示 |
| 区域管理 | P0 | 已完成第一阶段：支持启用/禁用区域、默认同步区域、区域同步开关、状态和资源类型范围持久化到 `iac_cloud_account_region`；真实云区域健康探测待 provider adapter 增强 |
| Runner 绑定 | P1 | 指定账号同步和操作使用的 Runner/tag |
| 账号健康检查 | P1 | 第一阶段已支持本地健康状态、最近健康检查、批量检查、事件通知、同步策略阈值、同步策略子周期健康详情、同步失败分类影响、后台周期刷新、健康依据/同步摘要展示、周期配置化、后台重复事件降噪和多实例锁保护；直接云 API 探测待 provider adapter 增强 |
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
| AWS collector 补齐 | P0 | 第一阶段已补齐 EIP、ELB/ALB/NLB、S3 Bucket 深层配置、Route Table、NAT Gateway、Internet Gateway、LB Listener/Rule/Auth Action/Target Group/Target Health；后续继续补真实云联调、跨账号/跨 VPC target 和更细错误映射 |
| OCI collector 稳定化 | P0 | Route Table、NAT Gateway、Internet Gateway、Service Gateway、DRG 已完成第一阶段采集与关系推演，compartment/identity 归属映射、provider 原生错误码解析、429/5xx 短重试和失败 scope 自动局部重试已完成第一阶段；后续继续增强分页、API 子调用级局部补偿、更细 identity 权限漂移和真实云联调 |
| AliCloud collector | P1 | ECS、VPC、VSwitch、SecurityGroup、EIP、SLB、RDS、Redis、OSS、ACK |
| Azure collector | P2 | 已完成第一阶段：VM、VNet、Subnet、NSG、Public IP、Disk、LB、AKS、SQL、Storage |
| GCP collector | P2 | 已完成第一阶段：Compute、VPC、Subnet、Firewall、Disk、LB、GKE、Cloud SQL、Bucket |
| 腾讯云/华为云 collector | P2 | 已完成真实 API collector、COS/OBS 对象存储桶采集和离线 inventory JSON 回退；成本账单导出 URL 导入第一阶段已完成，真实云账单环境联调仍待继续 |
| Kubernetes 集群信息 | P1 | 已完成第一阶段：EKS/OKE/GKE/AKS 集群作为 `kubernetes_cluster` 标准资产展示，呈现版本、API Endpoint、VPC/VNet/VCN、子网、安全组/NSG、节点组/节点池，并支持 Namespace、Node、Pod、Workload、Service、Ingress 工作负载资产类型、详情展示和关系推演；K8S 详情已增强旧数据、导入数据、原生字段、大小写变体和 OKE endpoint 可见性；真实云账号端到端联调和 kubeconfig/Agent 实时采集仍待继续 |
| 同步策略 | P1 | 第一阶段已支持定时同步策略模型、手动运行、到期 worker、按账号/区域/类型同步、按区域/资源类型独立子周期、失败重试、通知事件、自动暂停和资产 `syncPolicyId` 写入 |
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

- 用户可从资产详情进入关系图，并看到 IaC 依赖、应用推演关系和云采集推演的网络/安全/存储关系。
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
| AWS 成本接入 | P1 | 第一阶段已支持 AWS CUR 与 AWS Cost Explorer JSON/CSV/TSV URL 拉取、gzip/zip 压缩账单、S3 对象存储列表、文件元数据审计、重复文件跳过、字段归一化、同步任务、计划来源枚举和成本中心前端入口；真实 CUR/Cost Explorer 导出目录、Parquet/Excel 和财务规则仍待继续 |
| OCI 成本接入 | P2 | 第一阶段已支持 OCI Usage/Cost JSON/CSV/TSV URL 拉取、gzip/zip 压缩账单、OCI Object Storage 原生列表与签名下载、文件元数据审计、重复文件跳过、字段归一化、同步任务、计划来源枚举和成本中心前端入口；真实 Object Storage 导出样本、Parquet/Excel 和复杂财务规则仍待继续 |
| Azure/GCP 成本接入 | P2 | 第一阶段已支持 Azure/GCP Billing Export/Cost Management 导出 JSON 导入统一成本记录，并支持 Azure Cost Management API、GCP Billing Export URL、导出文件索引 URL、增量游标、S3/GCS/Azure Blob/OCI Object Storage 对象存储列表、文件元数据审计、重复文件跳过、同步任务日志、失败重试、同步计划、后台 worker、计划失败退避、自动暂停和成本拉取事件审计，复用成本异常规则 |
| 腾讯云/华为云成本接入 | P2 | 第一阶段已支持 TencentCloud/Huawei Billing Export JSON/CSV/TSV URL 拉取、gzip/zip 压缩账单、字段归一化、同步任务、计划来源枚举和成本中心前端入口；真实云账单环境联调、Parquet/Excel 和复杂财务规则仍待继续 |
| 成本分摊 | P1 | 按组织、项目、环境、应用、业务线、owner、标签分摊 |
| 预算管理 | P1 | 支持月度预算、阈值预警、超预算记录、到期评估和事件通知 |
| 成本异常 | P2 | 第一阶段已支持未匹配成本、缺少 Owner、缺少成本中心和高成本资源识别；环比/同比/突增待接入真实多账期账单 |
| 成本优化建议 | P2 | 第一阶段已支持闲置实例、未绑定磁盘、未使用公网 IP 的规则推演；规格调整、预留实例/节省计划建议待接入用量和计费数据 |

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
| 整改工作流 | P2 | 已完成第一阶段：风险/漂移可一键创建 ITSM 自助整改工单，ITSM 状态可同步风险处理中/已解决/重新打开，外部 ITSM 可通过验签 callback 或周期拉取回写状态；漂移类风险已可在工单解决后触发环境漂移自动修复任务 |
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
| 事件中心 | P1 | 统一记录账号验证/健康检查、采集完成、操作完成、风险发现、预算超限 |
| Webhook | P1 | 支持按事件类型推送到外部系统；第一阶段已支持配置、签名投递、投递记录、投递详情、手动重试、到期自动重试、队列可观测摘要、测试发送、指数退避、重试抖动和最大重试窗口 |
| ITSM 集成 | P2 | 风险整改、操作审批可接入外部工单 |
| 告警集成 | P2 | 支持企业微信、钉钉、Slack、邮件等通知 |
| OpenAPI/API Token | P2 | 为外部平台提供多云资产、成本、风险查询 API |

#### 6.7.3 验收标准

- CMDB 资产变更、云操作完成、风险发现均能生成事件。
- Webhook 失败有失败记录、手动重试、到期自动重试、队列摘要、指数退避、重试抖动和最大重试窗口。
- 外部系统可通过 API 查询资产和操作任务状态。

## 7. 建议数据模型

| 模型 | 类型 | 说明 |
| --- | --- | --- |
| `iac_cloud_account` | 新增 | 统一云账号主表，保存 provider、accountId、name、status、owner、runnerTag、lastValidatedAt、lastSyncAt |
| `iac_cloud_account_credential` | 新增 | 云账号凭证引用或加密密文，按 provider 存储 key schema |
| `iac_cloud_account_region` | 新增 | 已完成第一阶段：账号启用区域、默认区域、同步开关、区域状态、资源类型范围和最近同步时间 |
| `iac_cloud_account_permission` | 新增 | 已完成第一阶段：持久化最近权限验证结果、检查来源、检查时间、资源、动作、状态、说明和证据 |
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
| GET | `/api/v1/cloud/sync-policies` | 查询同步策略列表 |
| GET | `/api/v1/cloud/sync-policies/:id` | 查询同步策略详情 |
| PUT | `/api/v1/cloud/sync-policies/:id` | 更新同步策略 |
| DELETE | `/api/v1/cloud/sync-policies/:id` | 删除同步策略 |
| POST | `/api/v1/cloud/sync-policies/:id/run` | 手动运行同步策略 |
| POST | `/api/v1/cloud/sync-policies/run-due` | 运行到期同步策略 |

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
| GET | `/api/v1/cloud/webhooks/queue/summary` | Webhook 投递队列摘要 |

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
| P0 | AWS、OCI | 统一账号中心、只读采集、资产标准化、同步日志、CMDB 资产详情、总览，以及 AWS CUR/Cost Explorer 与 OCI Usage/Cost 成本 JSON/CSV/TSV URL、gzip/zip 压缩账单和对象存储原生列表/签名接入第一阶段 |
| P1 | AliCloud | 补齐资产采集、账单采集完善、安全操作框架首批动作 |
| P2 | Azure、GCP | 资产采集、成本导入/拉取型数据源、成本同步任务与计划 worker、导出文件索引 URL、增量游标、对象存储原生列表 API、签名授权与分页、计算实例 live read 与启停重启基础操作、网络/安全/存储关系推演第一阶段已完成；真实云环境联调和更细拓扑视图待继续 |
| P2 | 腾讯云、华为云 | 真实 API 资产采集、COS/OBS 对象存储桶采集、离线 inventory 回退和成本账单导出 URL 导入第一阶段已完成；基础风险规则和真实云账单环境联调待继续 |
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
- 建设网络拓扑、安全拓扑和应用拓扑；第一阶段已在 CMDB 资产关系中落地云侧网络/安全/存储关系推演。
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
- Kubernetes 集群和基于资产属性/导入数据的 Namespace、Node、Pod、Workload、Service/Ingress 信息已纳入云资产视图；kubeconfig/Agent 实时采集、事件和多集群管理仍待后续阶段。
- 是否需要按私有云/OpenStack/vSphere 设计 provider adapter 扩展点。

## 16. 开发进展

### 16.1 2026-06-20 V1.0 P0 云账号中心底座

状态：已完成后端 API 和前端组织级入口。

已完成：

- 新增统一云账号模型 `iac_cloud_account`，支持 provider、accountId、tenantId、regions、runnerTags、credentials、status、validationStatus、lastValidatedAt、lastSyncAt、supportedTypes。
- 新增云账号 CRUD API：`/api/v1/cloud/accounts`。
- 新增云账号验证 API：`POST /api/v1/cloud/accounts/:id/validate`。
- 凭证字段按 `isSecret` 加密存储，接口响应会脱敏返回。
- 当前本地验证范围为凭证完整性、区域字段和 provider 支持类型校验，不直接从 Portal 调云厂商 API。
- CMDB 云采集账号来源新增 `cloud_account`，旧的 `variable_group`、`resource_account` 来源保持兼容。
- CMDB 同步任务可选择统一云账号，并在同步完成后回写云账号 `lastSyncAt`。
- 云采集资产的 accountId 优先使用统一云账号的账号 ID。
- 前端新增组织级“多云管理 - 云账号”页面，支持列表、筛选、创建、编辑、验证、启停和删除。

验证：

- 后端包编译验证通过：`go test -vet=off -run "^$" ./portal/models ./portal/services ./portal/apps ./portal/web/api/v1 ./portal/web/api/v1/handlers`。
- 前端新增 JSX、路由和菜单 Babel 解析通过。
- Docker Compose 生产构建已通过；本地 `frontend/node_modules/.bin/webpack` 仍存在断开软链问题，验证以容器构建为准。

### 16.2 2026-06-20 V1.0 P1 云采集同步日志和任务详情

状态：已完成 CMDB 云采集任务详情、阶段日志和前端查看入口。

已完成：

- 新增 CMDB 同步任务日志模型 `iac_cmdb_sync_task_log`，记录 orgId、taskId、level、stage、message、data。
- 新增同步任务详情 API：`GET /api/v1/cmdb/sync-tasks/:id`，返回任务基础信息、统计和阶段日志。
- 云采集执行过程会写入 created、running、collecting、collector、complete、failed、panic 等阶段日志。
- 同步任务统计中增加当前 stage 信息，失败和 panic 会写入错误日志与任务最终状态。
- “资产 CMDB - 云采集”任务列表新增详情操作，可查看任务参数、账号来源、状态、错误、阶段日志和统计 JSON。

验证：

- 后端包编译验证通过：`go test -vet=off -run "^$" ./portal/models ./portal/services ./portal/apps ./portal/web/api/v1 ./portal/web/api/v1/handlers`。
- 前端 `resource-query/index.jsx` Babel 解析通过。
- Docker Compose 重新构建并启动 `iac-portal`、`iac-web` 通过，`/api/v1/check` 返回 200，前端根路径返回 200。
- 鉴权后 `GET /api/v1/cmdb/sync-tasks` 返回 200；当前无同步任务，使用不存在 taskId 调用详情接口返回 404。
- 内置浏览器验证生产包加载正常，“资产 CMDB - 云采集”页签可见账号筛选、启动云采集按钮以及新增的来源、统计、操作等表头，控制台无 error。

待继续：

- 云资产中心包装路由 `/api/v1/cloud/sync-tasks` 和 `/api/v1/cloud/sync-tasks/:id` 已在当前实现中提供；同步任务区域/资源类型级统计、阶段耗时、失败分类和重试提示已在 16.122 完成第一阶段。
- OCI 云厂商错误码映射已在 16.179 完成第一阶段，429/5xx/临时网络错误短重试已在 16.180 完成第一阶段，OCI collector 内部单 API 子调用耗时已在 16.181 完成第一阶段，任务详情 API 调用耗时可视化已在 16.182 完成第一阶段，API 维度趋势已在 16.183 完成第一阶段，事件中心慢调用告警已在 16.184 完成第一阶段，同步策略级慢调用阈值持久化已在 16.185 完成第一阶段，慢 API 告警静默窗口已在 16.187 完成第一阶段，失败 scope 自动局部重试已在 16.188 完成第一阶段，负责人分派和通知路由元数据已在 16.189 完成第一阶段，云采集事件按错误类型/云服务动态路由和失败次数升级已在 16.192 完成第一阶段，成本同步计划通知静默和路由已在 16.190 完成第一阶段；更细的 provider 原生分页游标、真实错误样本和企业级值班升级仍待继续补强。

### 16.3 2026-06-20 V1.0 P0 多云总览

状态：已完成组织级多云总览 API、菜单入口和前端页面。

已完成：

- 新增多云总览 API：`GET /api/v1/cloud/overview`。
- 总览会基于现有 `iac_cloud_account`、`iac_cmdb_asset`、`iac_cmdb_sync_task` 统计账号、资产、治理缺口和同步健康。
- 指标包含云账号总数/启用数/有效数/异常数/未同步数、CMDB 资产数、IaC 纳管资产数、云采集资产数、未纳管资产数、无负责人资产数、高风险资产数、运行中同步任务和近 24 小时失败任务。
- 增加 Provider 覆盖统计、资产来源分布、资产类型 Top 10、风险分布、最近云采集任务和治理事项。
- 前端新增“多云管理 - 总览”菜单和 `/org/:orgId/m-cloud-overview` 路由。

验证：

- 后端包编译验证通过：`go test -vet=off -run "^$" ./portal/models ./portal/services ./portal/apps ./portal/web/api/v1 ./portal/web/api/v1/handlers`。
- 前端 `cloud-overview/index.jsx` Babel 解析通过。
- Docker Compose 重新构建并启动 `iac-portal`、`iac-web` 通过，`/api/v1/check` 返回 200，前端总览路由返回 200。
- 鉴权后 `GET /api/v1/cloud/overview` 返回 200，当前测试数据统计为 8 个 CMDB 资产、8 个未纳管资产、1 个无负责人资产、3 个高风险资产。
- 内置浏览器验证生产包加载正常，“多云管理 - 总览”菜单和总览页面可见，关键指标、Provider 覆盖、治理事项、最近同步任务区域正常展示，控制台无 error。

待继续：

- 当前总览未新增 `managedBy/cloudAccountId` 字段，未纳管资产先按 `cloud_collect` 且无项目/环境/IaC 资源关联统计。
- 多云总览中的成本、操作任务、事件和风险发现模型仍待后续 PRD 阶段接入。

### 16.4 2026-06-20 V1.0 P0 多云云资产入口和包装 API

状态：已完成多云云资产组织级入口和 cloud 包装 API。

已完成：

- 新增多云资产包装 API，先复用现有 CMDB 资产查询、详情、导入导出、归属更新、IaC 回填和云采集任务能力。
- 新增 API：
  - `GET /api/v1/cloud/assets`
  - `GET /api/v1/cloud/assets/filters`
  - `GET /api/v1/cloud/assets/export`
  - `POST /api/v1/cloud/assets/import`
  - `GET /api/v1/cloud/assets/:id`
  - `PUT /api/v1/cloud/assets/:id/ownership`
  - `POST /api/v1/cloud/backfill/iac-resources`
  - `GET /api/v1/cloud/sync-tasks`
  - `GET /api/v1/cloud/sync-tasks/:id`
  - `POST /api/v1/cloud/sync-tasks`
- 前端新增“多云管理 - 云资产”菜单和 `/org/:orgId/m-cloud-assets` 路由。
- 云资产页面复用现有资产 CMDB 组件，但标题切换为“云资产”，资产列表/筛选/详情/导入/导出/同步任务走 `/api/v1/cloud/*` 包装接口。

验证：

- Docker Compose 生产构建 `iac-portal`、`iac-web` 通过，其中前端仅保留既有 bundle size warning。
- Docker Compose 启动 `iac-portal`、`iac-web` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200。
- 前端 `/org/:orgId/m-cloud-assets` 路由返回 200，内置浏览器验证“多云管理 - 云资产”菜单、面包屑、页面标题、资产列表/应用依赖/云采集页签和资产表格可见，控制台无 error。

### 16.5 2026-06-20 V1.0 P0 云账号区域与权限验证

状态：已完成 PRD 8.1 中云账号区域和权限查询/更新接口，以及前端查看入口。

已完成：

- 新增云账号区域查询 API：`GET /api/v1/cloud/accounts/:id/regions`。
- 新增云账号区域更新 API：`PUT /api/v1/cloud/accounts/:id/regions`。
- 新增云账号权限验证结果 API：`GET /api/v1/cloud/accounts/:id/permissions`。
- 区域响应会区分手动配置和自动推断来源，并标记默认区域。
- 权限响应基于当前本地 provider adapter 能力，返回账号状态、凭证完整性、区域范围和资产采集能力检查结果。
- 前端“多云管理 - 云账号”列表新增“区域/权限”操作，可查看和更新区域，并展示权限检查结果与支持采集的资产类型。

验证：

- 后端和前端生产镜像通过 Docker Compose 构建。
- Docker Compose 重新启动 `iac-portal`、`iac-web` 通过，`/api/v1/check` 返回 200。
- 使用临时 AWS 云账号验证 `GET/PUT /api/v1/cloud/accounts/:id/regions` 和 `GET /api/v1/cloud/accounts/:id/permissions` 返回 200，区域更新和权限状态正常，临时账号已删除。
- 内置浏览器验证“多云管理 - 云账号”页面可加载，临时账号行展示“区域/权限”入口，抽屉内“启用区域”“权限验证结果”“支持采集资产类型”正常渲染；验证后浏览器已恢复到“多云管理 - 云资产”页面。

待继续：

- 当前权限验证结果持久化已在 16.169 完成第一阶段，但仍是本地只读预检查；真实云 API 权限校验需在 provider adapter 中继续增强。
- 区域独立表已在 16.170 完成第一阶段；真实云 API 区域可用性、区域级权限漂移和区域级同步失败矩阵仍待 provider adapter 增强。

### 16.6 2026-06-20 V1.0 P1 云资产覆盖率和未纳管识别

状态：已完成多云云资产中心覆盖率 API 和前端展示。

已完成：

- 新增云资产覆盖率 API：`GET /api/v1/cloud/assets/coverage`。
- 覆盖率接口基于现有 `iac_cmdb_asset` 和 `iac_cloud_account` 统计资产总数、IaC 纳管数、云采集数、未纳管资产、已关联云采集资产、无负责人资产、高风险资产、Provider 数、账号数、资产类型数、区域数、最近同步时间。
- 未纳管口径沿用当前 V1.0 数据结构：`source = cloud_collect` 且无 `project_id/env_id/iac_resource_id` 的资产视为 cloud-only 未纳管资产。
- 覆盖率接口按 Provider、云账号、资产类型输出分布，云账号统计会同时包含已接入但尚无资产的统一云账号，以及资产侧存在但未关联统一云账号的账号组。
- 前端“多云管理 - 云资产”资产列表页新增“资产覆盖率”区块，展示资产总数、IaC 纳管、未纳管、无负责人、IaC 纳管覆盖率、资产归属覆盖率。
- 前端新增 Provider 覆盖、账号覆盖、资产类型覆盖三组表格，并可从 Provider/资产类型快速回填资产列表筛选。
- 多云总览中的治理事项跳转目标从旧“资产 CMDB”入口调整为“多云管理 - 云资产”入口。

验证：

- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal`、`iac-web` 通过；前端仅保留既有 bundle size warning。
- Docker Compose 重新启动 `iac-portal`、`iac-web` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200。
- 鉴权后 `GET /api/v1/cloud/assets/coverage` 返回 200，当前测试组织统计为 8 个云采集资产、8 个未纳管资产、1 个无负责人资产、3 个高风险资产。
- 前端 `/org/:orgId/m-cloud-assets` 和 `/org/:orgId/m-cloud-overview` 路由返回 200。
- 内置浏览器验证“多云管理 - 云资产”页面可加载，覆盖率区块、Provider 覆盖、账号覆盖、资产类型覆盖正常渲染，控制台无 error。

待继续：

- 未纳管资产治理还缺少审批流、治理状态细分和账号关联修复入口。

### 16.7 2026-06-20 V1.0 P1 未纳管资产筛选与批量治理

状态：已完成多云云资产中心的账号筛选、纳管状态筛选和批量治理入口。

已完成：

- 资产搜索和筛选接口新增 `accountIds`、`managedBy` 参数，支持按云账号和纳管状态过滤资产列表。
- `managedBy` 包含 `iac`、`cloud_linked`、`cloud_only`、`manual` 四类状态。
- 新增批量资产归属治理 API：`PUT /api/v1/cloud/assets/ownership`、`PUT /api/v1/cmdb/assets/ownership`。
- 批量治理支持一次更新负责人、应用、业务线、生命周期、合规风险，并记录资产变更历史。
- 前端“多云管理 - 云资产”资产列表新增“账号”“纳管状态”列。
- 前端筛选区新增“账号”“纳管状态”筛选项，覆盖率表中的未纳管数量可下钻到 `cloud_only` 资产列表。
- 前端新增“批量治理”操作：未选择资产时禁用，选择资产后可打开治理弹窗批量填写归属信息。

验证：

- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal`、`iac-web` 通过；前端仅保留既有 bundle size warning。
- Docker Compose 重新启动 `iac-portal`、`iac-web` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200。
- 鉴权后 `GET /api/v1/cloud/assets/filters` 返回 200，当前测试组织返回 3 个 `accountIds` 和 `cloud_only` 纳管状态。
- 鉴权后 `GET /api/v1/cloud/assets?managedBy=cloud_only` 返回 200，当前测试组织可筛选出 8 条云上未纳管资产。
- 使用一条未纳管资产验证 `PUT /api/v1/cloud/assets/ownership`，临时更新负责人后立即恢复原值，两次批量请求均返回 `updated=1`、`skipped=0`。
- 内置浏览器验证“多云管理 - 云资产”页面可加载，覆盖率区块、账号/纳管状态筛选、账号/纳管状态列、“批量治理”入口正常渲染；选择资产后可打开批量治理弹窗并取消关闭，控制台无 error。

待继续：

- 批量治理继续支持审批流、异步执行和更细治理状态流转。
- 覆盖率表可继续补充账号维度的统一云账号关联修复入口。

### 16.8 2026-06-20 V1.0 P1 云资产真实治理字段落地

状态：已完成 `iac_cmdb_asset` 真实治理字段扩展、写入和旧数据回填。

已完成：

- `iac_cmdb_asset` 新增真实治理字段：`managedBy`、`cloudAccountId`、`syncPolicyId`、`lastOperationId`、`riskScore`、`costCenter`。
- IaC 回填资产写入 `managedBy=iac`，云采集资产写入 `managedBy=cloud_only`，已关联项目/环境/IaC 资源的云资产会刷新为 `cloud_linked`。
- 云采集资产优先写入统一云账号 `cloudAccountId`；旧资产会按 Provider 和账号 ID 自动回填统一云账号关联。
- 资产列表、详情、筛选和覆盖率统计在查询前会刷新治理字段，兼容历史数据。
- `managedBy` 筛选和覆盖率统计改为优先使用真实字段，保留旧数据推导兜底。
- CSV 导出新增统一云账号 ID、纳管状态、同步策略 ID、最近操作 ID、成本中心、风险评分字段。
- 前端资产列表优先展示后端返回的 `managedBy`，账号列补充展示已关联的统一云账号 ID。

验证：

- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal`、`iac-web` 通过；前端仅保留既有 bundle size warning。
- Docker Compose 重新启动 `iac-portal`、`iac-web` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200。
- 鉴权后 `GET /api/v1/cloud/assets?managedBy=cloud_only` 返回 200，当前测试组织返回 8 条未纳管资产，响应资产包含 `managedBy=cloud_only` 和 `cloudAccountId` 字段。
- 鉴权后 `GET /api/v1/cloud/assets/coverage` 返回 200，当前测试组织覆盖率仍为 8 个资产、8 个未纳管、0 个已关联云采集资产。
- 鉴权后 `GET /api/v1/cloud/assets/export?format=csv&managedBy=cloud_only` 返回 CSV 表头包含统一云账号 ID、纳管状态、同步策略 ID、最近操作 ID、成本中心、风险评分。
- 内置浏览器验证“多云管理 - 云资产”页面可加载，资产覆盖率、账号筛选、纳管状态筛选、账号列、纳管状态列正常渲染，控制台无 error。

待继续：

- 批量治理继续支持审批流、异步执行和更细治理状态流转。
- 云账号覆盖表继续补充“未关联账号修复/合并”入口。
- `lastOperationId` 已接入批量治理操作任务；`syncPolicyId` 已在 16.110 接入同步策略模型、同步任务和云采集资产真实写入。

### 16.9 2026-06-20 V1.0 P1 未纳管资产项目/环境绑定治理

状态：已完成多云云资产批量治理中的项目/环境绑定能力。

已完成：

- 批量资产归属治理 API 支持提交 `projectId`、`envId`，可将云上未纳管资产绑定到 CloudIaC 项目/环境。
- 绑定环境时自动校验环境归属项目；未显式提交项目但提交环境时，后端会按环境反推项目。
- 绑定项目前校验项目属于当前组织，并校验当前用户具备组织管理员、超级管理员或项目成员权限。
- 项目/环境绑定后自动刷新 `managedBy`，云采集资产从 `cloud_only` 切换为 `cloud_linked`；提交空项目/环境时可清空绑定并恢复未纳管状态。
- 批量治理变更历史新增记录 `projectId`、`envId`、`managedBy` 差异。
- 云资产筛选项中的项目/环境候选改为读取当前用户可见的组织项目/环境，不再受当前资产筛选结果限制，避免 `cloud_only` 资产列表下无法选择绑定目标。
- 前端“多云管理 - 云资产”批量治理弹窗新增“绑定项目”“绑定环境”和“清空项目/环境绑定”字段，保留负责人、应用、业务线、生命周期、合规风险批量治理字段。

验证：

- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal`、`iac-web` 通过。
- Docker Compose 重新启动 `iac-portal`、`iac-web` 通过，`iac-portal` healthy。
- 鉴权后 `GET /api/v1/cloud/assets/filters` 返回 200，当前测试组织返回 1 个项目候选和 0 个环境候选。
- 使用一条 `cloud_only` 未纳管资产验证 `PUT /api/v1/cloud/assets/ownership`：绑定项目后资产 `projectId` 更新为目标项目、`managedBy` 变为 `cloud_linked`；随后提交空 `projectId/envId` 恢复原始项目/环境绑定，`managedBy` 恢复 `cloud_only`。
- 内置浏览器验证“多云管理 - 云资产”页面可加载，选择资产后“批量治理”按钮可用，弹窗展示“绑定项目”“绑定环境”“清空项目/环境绑定”“负责人”等字段，取消关闭未提交变更。

待继续：

- 批量治理继续接入审批流、异步执行和更细治理状态流转。
- 云账号覆盖表继续补充“未关联账号修复/合并”入口。
- 环境候选当前随组织测试数据为空；后续创建环境后需补充项目/环境联动选择体验。

### 16.10 2026-06-20 V1.0 P1 云操作任务底座和治理审计入口

状态：已完成云操作任务基础模型、API、前端入口，以及批量治理任务记录。

已完成：

- 新增云操作任务模型：`iac_cloud_operation`、`iac_cloud_operation_step`、`iac_cloud_operation_audit`。
- 新增操作任务 API：
  - `GET /api/v1/cloud/operations`
  - `GET /api/v1/cloud/operations/:id`
  - `POST /api/v1/cloud/operations/:id/cancel`
  - `GET /api/v1/cloud/operations/:id/audits`
- 批量资产归属治理会自动创建 `governance_ownership` 操作任务，记录请求参数、更新结果、步骤和审计。
- 真实发生变更的资产会写入 `lastOperationId`，指向本次治理操作任务。
- 前端新增“多云管理 - 操作任务”菜单和 `/org/:orgId/m-cloud-operations` 路由。
- 操作任务页面支持搜索、状态/动作过滤、分页列表和详情抽屉；详情展示参数、结果、步骤和审计。

验证：

- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal`、`iac-web` 通过；前端仅保留既有 bundle size warning。
- Docker Compose 重新启动 `iac-portal`、`iac-web` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200。
- 使用一条云资产验证 `PUT /api/v1/cloud/assets/ownership`：临时更新负责人后生成操作任务，资产 `lastOperationId` 指向该操作任务；恢复原负责人后再次生成恢复操作任务。
- 鉴权后 `GET /api/v1/cloud/operations` 可查询到 `governance_ownership` 操作任务；`GET /api/v1/cloud/operations/:id` 返回 `complete` 状态、1 条步骤和 2 条审计记录。
- 内置浏览器验证“多云管理 - 操作任务”菜单、列表、详情抽屉、参数、结果、步骤和审计正常渲染，控制台无 error。

待继续：

- 云操作任务继续接入真实 provider operation adapter、异步执行器和审批流。
- 高风险动作需要二次确认、审批和更细粒度权限控制。
- 操作任务取消目前是状态级能力，真实云操作执行器接入后需支持运行中中断和回滚提示。

### 16.11 2026-06-20 V1.0 P1 云资产动作目录、预检查和任务创建入口

状态：已完成云资产详情中的动作目录、dry-run 预检查和低风险操作任务创建闭环。

已完成：

- 新增云资产动作常量和任务类型：`operationType=action`、`refresh_metadata`、`start_instance`、`stop_instance`、`restart_instance`、`delete_resource`。
- 新增云资产动作 API：
  - `GET /api/v1/cloud/assets/:id/actions`
  - `POST /api/v1/cloud/assets/:id/actions/:action/dry-run`
  - `POST /api/v1/cloud/assets/:id/actions/:action`
- 当前版本仅开放低风险 `refresh_metadata` 创建任务；该动作只记录操作任务、刷新本地 `managedBy` 和 `lastOperationId`，不执行云端变更。
- 云主机启动、停止、重启和资源删除动作先进入动作目录，但在 provider operation adapter、审批流和高危权限接入前保持禁用。
- dry-run 输出资产存在、云厂商、云资源 ID、资产来源、操作适配器、审批要求等检查项，前端可直接展示检查状态。
- 创建 `refresh_metadata` 任务时写入 `iac_cloud_operation`、`iac_cloud_operation_step`、`iac_cloud_operation_audit`，步骤名显示为“记录操作结果”。
- 资产详情新增“操作”页签，展示动作目录、风险等级、启用状态、审批/高危要求、预检查结果和创建任务按钮。
- 资产详情基础信息新增“最近操作”入口，可跳转到“多云管理 - 操作任务”页面。
- 操作任务页面新增 `action` 类型和 `refresh_metadata` 等动作的中文展示。

验证：

- `git diff --check` 通过。
- Docker Compose 完整生产构建 `iac-portal`、`iac-web`、`ct-runner` 通过；前端仅保留既有 bundle size warning。
- Docker Compose 重新启动服务通过，`iac-portal`、`ct-runner` healthy，`iac-web` 正常启动，`GET /api/v1/check` 返回 200。
- 鉴权后使用资产 `ci-d8qlhn9jm3jc73bcu290` 验证 `GET /api/v1/cloud/assets/:id/actions` 返回 5 个动作，`refresh_metadata` 为可创建。
- 鉴权后 `POST /api/v1/cloud/assets/:id/actions/refresh_metadata/dry-run` 返回 `executable=true`。
- 鉴权后 `POST /api/v1/cloud/assets/:id/actions/refresh_metadata` 创建任务 `cop-d8r0cdgc5jfs73c9im9g`，任务状态 `complete`，资产 `lastOperationId` 回写成功，详情包含 1 条步骤和 2 条审计。
- 内置浏览器验证资产详情“操作”页签正常渲染：`refresh_metadata` 可创建，启停/重启/删除动作禁用，高危删除展示“需审批/高危”。
- 内置浏览器点击“预检查”后展示可执行、云厂商识别、操作适配器和审批检查结果。
- 内置浏览器点击“创建任务”并确认后出现“操作任务已创建”提示，资产详情“最近操作”更新为 UI 创建的任务 `cop-d8r0ddoc5jfs73c9imd0`；API 回读该任务为 `operationType=action`、`action=refresh_metadata`、`status=complete`、步骤“记录操作结果”、审计 2 条。

待继续：

- 接入真实 provider operation adapter 后再开放实例启停/重启等云端动作。
- `delete_resource` 等高危动作需先接入审批流、二次确认、权限细分和回滚/失败提示。
- 操作任务继续从同步完成模型演进为异步执行模型，支持运行中状态、取消、重试和 provider 原始响应留痕。

### 16.12 2026-06-20 V1.0 P1 云资产动作权限预检查和 adapter 执行边界

状态：已完成云资产动作的权限预检查、adapter 元数据透出和本地安全 adapter 执行框架。

已完成：

- 云资产动作目录新增 `adapterKey`、`adapterMode` 字段，前端可区分“本地安全”和“云端适配器”执行方式。
- `refresh_metadata` 固定走 `local_metadata/local` adapter，只刷新本地 `managedBy` 和 `lastOperationId`，明确不触发云端 API。
- 启动、停止、重启和删除动作继续保留在动作目录，执行方式标记为 `provider`，在真实 provider operation adapter 接入前保持禁用。
- dry-run 新增“用户权限”检查项：
  - 平台管理员、组织管理员可执行云资产操作。
  - 绑定项目的云资产允许项目负责人、操作员、审批员执行。
  - 未绑定项目的云资产仅平台管理员或组织管理员可执行。
- 动作列表会根据权限预检查禁用无权限动作，创建接口也会在权限不满足时返回权限错误。
- 创建任务逻辑统一通过 `executeCloudAssetAction` 分发；当前仅实现 `executeCloudAssetMetadataRefresh`，为后续 provider adapter 接入预留单一执行入口。
- 前端资产详情“操作”表格新增“执行方式”列，dry-run 摘要新增“执行方式”展示，预检查列表展示“用户权限”和“操作适配器”结果。

验证：

- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal`、`iac-web` 通过；前端仅保留既有 bundle size warning。
- Docker Compose 重新启动 `iac-portal`、`iac-web` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200，Web 首页返回 200。
- 鉴权后使用资产 `ci-d8qlhn9jm3jc73bcu290` 验证 `GET /api/v1/cloud/assets/:id/actions` 返回 5 个动作；`refresh_metadata` 为 `enabled=true`、`adapterKey=local_metadata`、`adapterMode=local`，其余云端动作为 `adapterMode=provider` 且保持禁用。
- 鉴权后 `POST /api/v1/cloud/assets/:id/actions/refresh_metadata/dry-run` 返回 `executable=true`，包含“用户权限=通过”和“操作适配器=已接入本地安全适配器 local_metadata，不调用云端 API”。
- 鉴权后 `POST /api/v1/cloud/assets/:id/actions/refresh_metadata` 创建任务 `cop-d8r0ikrfk0vs73cilr40`，任务状态 `complete`，结果包含 `adapter.key=local_metadata`、`adapter.mode=local`、`providerAdapterExecuted=false`。
- 内置浏览器验证资产详情“操作”页签展示“执行方式”列，`refresh_metadata` 显示“本地安全”，禁用的云端动作显示“云端适配器”。
- 内置浏览器点击 `refresh_metadata` 的“预检查”后展示“用户权限”“操作适配器”和 `local_metadata` 本地安全适配器说明；浏览器控制台无 error。

待继续：

- 接入真实 provider operation adapter registry，按 `aws`、`oci`、`alicloud` 分别实现实例启停/重启动作。
- 高风险动作接入审批流、二次确认、权限细分和失败/回滚提示后再开放。
- 操作任务继续演进为异步执行模型，支持运行中状态、取消、重试和 provider 原始响应留痕。

### 16.13 2026-06-20 V1.0 P1 Provider Operation Adapter Registry 和写操作保护

状态：已完成 provider operation adapter registry 的第一阶段能力声明和写操作保护展示，真实云端执行仍保持关闭。

已完成：

- 新增 provider operation adapter 能力矩阵：
  - `aws` 实例启停/重启注册为 `aws_ec2_instance`。
  - `oci` 实例启停/重启注册为 `oci_compute_instance`。
  - `alicloud` 实例启停/重启注册为 `alicloud_ecs_instance`。
- 云资产动作响应新增 `adapterStatus`、`adapterMessage`、`providerAdapter`、`writeEnabled` 字段。
- `start_instance`、`stop_instance`、`restart_instance` 从“未接入 adapter”升级为“provider adapter 已注册，但云端写操作保护未开放”。
- 新增 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED=false` 部署样例开关；默认关闭时仅允许本地安全动作和 dry-run 预检查，不执行云端启停/重启/删除。
- dry-run 将 provider 动作的检查拆分为“适配器注册”和“执行保护”：
  - adapter 已注册时显示具体 provider adapter 名称。
  - 写操作未开放或真实执行器未接入时，执行保护检查失败，动作不可创建。
- `delete_resource` 继续优先展示高危动作需要审批流/二次确认的禁用原因，不因 registry 存在与否误导用户。
- 前端资产详情“操作”表格的“执行方式”列新增 adapter 状态标签和 provider adapter 名称。
- 前端 dry-run 摘要新增“适配器状态”和“Provider Adapter”展示。

验证：

- `git diff --check` 已通过。
- Docker Compose 生产构建 `iac-portal`、`iac-web` 通过；前端仅保留既有 bundle size warning。
- Docker Compose 重新启动 `iac-portal`、`iac-web` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200，Web 首页返回 200。
- 鉴权后使用资产 `ci-d8qlhn9jm3jc73bcu290` 验证 `GET /api/v1/cloud/assets/:id/actions`：实例启动、停止、重启动作返回 `adapterStatus=registered`、`providerAdapter=oci_compute_instance`、`writeEnabled=false`，创建按钮仍不可用；`refresh_metadata` 仍为 `adapterStatus=ready`、`writeEnabled=true`。
- 鉴权后 `POST /api/v1/cloud/assets/:id/actions/start_instance/dry-run` 返回 `executable=false`，包含“适配器注册=通过”和“执行保护=失败”，失败原因指向 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED` 未开启。
- 内置浏览器验证资产详情“操作”页签显示“云端适配器 / 已注册 / oci_compute_instance”，启动实例创建按钮保持禁用。
- 内置浏览器点击“启动实例”的“预检查”后展示“不可执行”“适配器注册”“执行保护”和 Provider Adapter 摘要；浏览器控制台无 error。

待继续：

- 将 registry 从静态矩阵演进为真实 provider adapter 接口，实现 credential/resource preflight。
- 在审批流和二次确认接入后，再考虑受控开放实例启停/重启。
- 操作任务执行器继续演进为异步执行模型，支持运行中状态、取消、重试和 provider 原始响应留痕。

### 16.14 2026-06-20 V1.0 P1 Provider Action Credential/Resource Preflight

状态：已完成 provider 写操作的资源定位和云账号凭证预检，真实云端执行仍保持关闭。

已完成：

- `start_instance`、`stop_instance`、`restart_instance` 的 dry-run 增加 provider action preflight，只有 `provider_operation` 动作触发，本地 `refresh_metadata` 不受影响。
- 新增“资源定位”检查，按动作支持的资产类型、云资源 ID、区域判断 provider 侧资源是否可定位。
- 新增“云账号绑定”检查，资产未绑定统一云账号或账号不存在时直接失败，避免误以为已具备云端写操作条件。
- 云账号存在时复用统一云账号校验逻辑，补充账号厂商、账号身份、账号状态、凭证完整性、区域范围、资产采集能力和区域覆盖检查。
- 保留 provider adapter registry 和写操作保护检查顺序，确保页面同时展示“预检缺口”和“执行保护”两类阻断原因。

验证：

- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal`、`iac-web` 通过；后端 Go 编译通过，前端镜像复用缓存。
- Docker Compose 重新启动 `iac-portal`、`iac-web` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200，Web 首页返回 200。
- 鉴权后使用资产 `ci-d8qlhn9jm3jc73bcu290` 验证 `POST /api/v1/cloud/assets/:id/actions/start_instance/dry-run` 返回 `executable=false`，包含“资源定位=通过”“云账号绑定=失败”“适配器注册=通过”“执行保护=失败”。
- 鉴权后 `POST /api/v1/cloud/assets/:id/actions/refresh_metadata/dry-run` 返回 `executable=true`，且不包含 `cloud_account_binding`，确认本地安全动作未被 provider 预检误伤。
- 内置浏览器验证资产详情“操作”页签点击“启动实例”的“预检查”后，弹窗展示“资源定位”“云账号绑定”“适配器注册”“执行保护”和 `Provider Adapter=oci_compute_instance`；浏览器控制台无 error。

待继续：

- 将 provider action preflight 从 dry-run 扩展到真实 adapter 调用前的统一执行门禁，形成不可绕过的后端保护。
- 设计 AWS/OCI/阿里云实例启停/重启 adapter 的真实执行器接口、幂等策略、错误映射和 provider 原始响应留痕。
- 接入审批流、二次确认、异步任务状态和失败重试后，再受控开放 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED`。

### 16.15 2026-06-20 V1.0 P1 Provider Action Execution Guard 和执行计划审计

状态：已完成云资产动作创建和执行入口的统一执行门禁，并为后续真实 provider adapter 接入预留执行计划审计结构。

已完成：

- 新增 `cloudAssetActionExecutionGuard`，云资产动作创建入口和执行入口都必须通过同一份 dry-run 预检查结果，避免后续执行器绕过 provider preflight。
- 新增阻断检查提取逻辑，优先把 `user_permission` 转为权限错误，其余失败检查统一转为不可执行错误。
- 新增 `executionPlan` 结果结构，记录动作、adapter key/mode/status、provider adapter、写保护开关、执行门禁状态、阻断检查、全部预检查和 provider 资源定位信息。
- `refresh_metadata` 任务结果新增 `executionPlan.guardStatus=pass`，继续保持 `providerAdapterExecuted=false`，明确只执行本地元数据刷新。
- 新增 provider operation 执行分支占位：只有执行门禁通过后才会进入 provider adapter 执行分支；当前真实执行器仍未接入，不会执行云端 API。

验证：

- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal`、`iac-web` 通过；后端 Go 编译通过，前端镜像复用缓存。
- Docker Compose 重新启动 `iac-portal`、`iac-web` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200，Web 首页返回 200。
- 鉴权后使用资产 `ci-d8qlhn9jm3jc73bcu290` 验证 `POST /api/v1/cloud/assets/:id/actions/start_instance/dry-run` 返回 `executable=false`，阻断项包含 `cloud_account_binding` 和 `execution_guard`。
- 鉴权后 `POST /api/v1/cloud/assets/:id/actions/start_instance` 被后端执行门禁拒绝，真实 provider adapter 未执行，也未创建成功操作任务。
- 鉴权后 `POST /api/v1/cloud/assets/:id/actions/refresh_metadata` 创建任务 `cop-d8r0rse6g7es73fmd3u0`，任务状态 `complete`，结果包含 `executionPlan.guardStatus=pass`、`adapterKey=local_metadata`、`adapterMode=local`、`providerAdapterExecuted=false`。
- 内置浏览器验证资产详情“操作”页签仍正常展示 provider adapter 状态，`start_instance` 创建按钮保持禁用；点击“预检查”后展示“资源定位”“云账号绑定”“适配器注册”“执行保护”，浏览器控制台无 error。

待继续：

- 把 provider adapter 占位分支演进为正式接口，定义 `ExecuteStart/Stop/Restart` 的请求体、幂等键、错误映射和 provider 原始响应留痕。
- 为 AWS EC2、OCI Compute、阿里云 ECS 分别实现只读状态确认和变更前/变更后状态校验。
- 接入审批流和二次确认后，再受控开放 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED`。

### 16.16 2026-06-20 V1.0 P1 Provider Operation Adapter 接口骨架和状态校验

状态：已完成 AWS/OCI/阿里云计算实例 provider operation adapter 的统一接口骨架、请求结构、幂等键和缓存状态校验，真实云端执行仍保持关闭。

已完成：

- 新增 `cloudProviderOperationAdapter` 接口，定义 `BuildRequest`、`ReadResourceState`、`ValidateTransition`、`Execute` 四段 provider 操作生命周期。
- 新增 `cloudProviderOperationRequest`，统一记录 `operationId`、`idempotencyKey`、动作、provider、provider adapter、资源类型、资源 ID、区域、账号、当前状态和目标状态。
- AWS EC2、OCI Compute、阿里云 ECS 继续复用 registry，统一落到计算实例 adapter 骨架：
  - `aws_ec2_instance`
  - `oci_compute_instance`
  - `alicloud_ecs_instance`
- dry-run 新增“状态只读确认”，从 CMDB 缓存状态读取实例状态，并归一化 `RUNNING/Running/running` 等状态为统一状态模型。
- dry-run 新增“状态变更校验”，按动作校验实例变更前状态：
  - `start_instance`：已运行时给出幂等提醒，停止态可启动。
  - `stop_instance`：运行态可停止，已停止时给出幂等提醒。
  - `restart_instance`：要求运行态，否则失败。
  - 终止态和变更中状态会失败，未知状态会提醒真实执行前重新确认。
- `executionPlan` 增加 `providerRequest`，为后续真实 SDK 调用和审计留存幂等键、目标状态和状态归一化结果。
- provider adapter `Execute` 当前只返回未实现的原始响应占位，不触发云端 API；如果未来写保护打开但真实执行器未实现，操作任务会落为 `failed`，避免悬挂在 `running`。

验证：

- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal`、`iac-web` 通过；后端 Go 编译通过，前端镜像复用缓存。
- Docker Compose 重新启动 `iac-portal`、`iac-web` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200，Web 首页返回 200。
- 鉴权后使用资产 `ci-d8qlhn9jm3jc73bcu290` 验证 `start_instance` dry-run 包含“状态只读确认=通过”和“状态变更校验=提醒”，提示实例已运行、启动动作将作为幂等空操作处理。
- 鉴权后验证 `stop_instance` dry-run 包含“状态只读确认=通过”和“状态变更校验=通过”，目标状态为 `stopped`。
- 鉴权后验证 `restart_instance` dry-run 包含“状态只读确认=通过”和“状态变更校验=通过”，目标状态为 `running`。
- 鉴权后验证 `refresh_metadata` dry-run 仍为 `executable=true`，不包含 `provider_state_read`，创建任务成功且 `executionPlan.providerRequest=null`，确认本地安全动作未被 provider adapter 逻辑误伤。
- 内置浏览器验证资产详情“操作”页签点击“启动实例”的“预检查”后展示“状态只读确认”“状态变更校验”“云账号绑定”“执行保护”；浏览器控制台无 error。

待继续：

- 将计算实例 adapter 拆成可接 SDK 的 provider 实现文件，分别对接 AWS EC2、OCI Compute、阿里云 ECS 的真实只读状态查询。
- 设计 provider 原始错误映射、重试策略、超时策略和 provider request/response 的脱敏留痕。
- 接入审批流和二次确认后，再受控开放 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED`。

### 16.17 2026-06-20 V1.0 P1 Provider 状态读取模式和只读查询留痕

状态：已完成 provider 状态读取模式配置、AWS/OCI 计算实例只读状态查询通道和 provider 响应脱敏留痕；真实云端写操作仍保持关闭。

已完成：

- 新增 `CLOUD_OPERATION_PROVIDER_READ_MODE` 配置：
  - 默认 `cache`：继续只读取 CMDB 缓存状态，不访问云厂商 API。
  - `live`：通过资产绑定的统一云账号执行 provider 只读状态查询。
- 新增 `CLOUD_OPERATION_PROVIDER_READ_TIMEOUT_SECONDS`，为 provider 只读状态查询设置超时边界，默认 15 秒。
- 将 provider `ReadResourceState` 扩展为可接收统一云账号上下文，dry-run 和 execution plan 均可按读取模式生成状态来源和读取结果。
- 新增 `cloud_operation_provider_state.go`，把 provider 状态读取逻辑从主操作流程中拆出：
  - AWS EC2：复用现有 AWS V4 签名 HTTP 客户端，通过 `DescribeInstances` 查询实例状态。
  - OCI Compute：复用现有 OCI 签名 HTTP 客户端，通过 `GetInstance` 查询实例生命周期状态。
  - 阿里云 ECS：当前显式返回 `provider_reader_not_implemented`，保留缓存回退和审计留痕，等待 ECS 只读客户端接入。
- 状态读取结果新增 `readMode`、`stateSource`、`rawState`、`rawResponse`、`readError`，用于后续操作审计和排障。
- provider 原始响应留痕新增脱敏处理，自动屏蔽 `secret`、`password`、`token`、`privateKey`、`accessKey`、`credential`、`authorization`、`signature` 等敏感字段。
- provider 读取失败时不直接替代执行保护：dry-run 会给出“状态只读确认=提醒”，并回退 CMDB 缓存状态继续做状态变更校验，最终仍由执行保护/云账号/权限检查决定是否可执行。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `cloud_operation.go` 和 `cloud_operation_provider_state.go`。
- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal` 通过，后端 Go 编译通过。
- Docker Compose 重新启动 `iac-portal` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200。
- 鉴权后使用资产 `ci-d8qlhn9jm3jc73bcu290` 验证 `start_instance` dry-run：
  - `providerAdapter=oci_compute_instance`
  - “状态只读确认=通过”，消息为 `CMDB缓存(cache) 状态 RUNNING 已归一化为 running`
  - “状态变更校验=提醒”，提示实例已处于运行中，启动动作将作为幂等空操作处理。
- 鉴权后验证 `stop_instance` dry-run：
  - “状态只读确认=通过”，消息为 `CMDB缓存(cache) 状态 RUNNING 已归一化为 running`
  - “状态变更校验=通过”，目标状态为 `stopped`。
- 鉴权后验证 `restart_instance` dry-run：
  - “状态只读确认=通过”，消息为 `CMDB缓存(cache) 状态 RUNNING 已归一化为 running`
  - “状态变更校验=通过”，目标状态为 `running`。
- 三个 provider 动作仍为 `executable=false`，均被 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED` 写操作保护阻断，确认只读状态查询没有放开云端写操作。

待继续：

- 接入阿里云 ECS 只读状态查询客户端，补齐 `alicloud_ecs_instance` 的 live read。
- 将 AWS/OCI/Aliyun 写操作执行器继续拆分为 provider 专属文件，补充 `Start/Stop/Reboot` 的真实调用、幂等返回和变更后状态轮询。
- 在审批流、二次确认、异步任务状态和失败重试完成后，再受控开放 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED`。

### 16.18 2026-06-20 V1.0 P1 阿里云 ECS Live 状态只读查询

状态：已完成 `alicloud_ecs_instance` 的 live 状态只读查询通道；真实云端写操作仍保持关闭。

已完成：

- `CLOUD_OPERATION_PROVIDER_READ_MODE=live` 时，阿里云计算实例动作可通过绑定云账号执行 ECS `DescribeInstances` 只读查询。
- 新增阿里云 RPC API 签名实现：
  - 使用已有统一云账号字段 `ALICLOUD_ACCESS_KEY`、`ALICLOUD_SECRET_KEY`、`ALICLOUD_REGION`。
  - 使用 HMAC-SHA1 RPC 签名，不新增阿里云 ECS SDK 依赖。
  - Endpoint 按区域生成 `ecs.<region>.aliyuncs.com`。
- 新增阿里云 ECS 响应解析：
  - 读取 `Instances.Instance[].Status` 作为 provider 原始状态。
  - 继续复用统一状态归一化逻辑，将 `Running/Stopped/Starting/Stopping` 等状态归一化后进入状态变更校验。
  - 原始响应留痕记录 `requestId`、`totalCount`、`instanceId`、`instanceName`、`status`、`zoneId`、`regionId`。
- 阿里云 ECS 读取失败、凭证缺失或资源未返回时，统一进入 `provider_read_failed` / `resource_not_found` 等错误映射，并回退 CMDB 缓存状态，不放开写操作。
- AWS/OCI/Aliyun 三类计算实例现在都具备 live read 通道，后续可以在同一执行计划结构里接入真实写操作前/后的状态确认。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `cloud_operation_provider_state.go`。
- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal` 通过，后端 Go 编译通过。
- Docker Compose 重新启动 `iac-portal` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200。
- 鉴权后使用当前 OCI 资产 `ci-d8qlhn9jm3jc73bcu290` 回归 `start_instance`、`stop_instance`、`restart_instance` dry-run：
  - 默认 `cache` 模式仍返回 `CMDB缓存(cache) 状态 RUNNING 已归一化为 running`。
  - `start_instance` 仍提示实例已运行、启动动作将作为幂等空操作处理。
  - `stop_instance`、`restart_instance` 状态变更校验仍通过。
  - 三个 provider 动作仍被 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED` 写操作保护阻断。
- 当前测试数据没有可用阿里云 ECS 资产和真实阿里云云账号，未对 live 阿里云 API 做真实云端联通验证。

待继续：

- 将 provider 写操作执行器继续拆分为 AWS EC2、OCI Compute、阿里云 ECS 专属文件，补充 `Start/Stop/Reboot` 的真实调用占位、幂等返回和变更后状态轮询接口。
- 接入审批流、二次确认、异步任务状态和失败重试后，再受控开放 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED`。

### 16.19 2026-06-20 V1.0 P1 Provider 计算实例写操作执行器

状态：已完成 AWS EC2、OCI Compute、阿里云 ECS 计算实例写操作执行器的受控接入；默认部署仍关闭真实云端写操作。

已完成：

- 新增 `cloud_operation_provider_execute.go`，将 provider 写操作从主操作流程中拆出。
- provider 写操作启用条件调整为双门禁：
  - 必须显式开启 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED=true`。
  - 必须设置 `CLOUD_OPERATION_PROVIDER_READ_MODE=live`，确保写操作前使用 provider 只读状态作为前置校验。
- 新增 `CLOUD_OPERATION_PROVIDER_WRITE_TIMEOUT_SECONDS=30`，为云端写操作设置超时边界。
- AWS EC2 写操作映射：
  - `start_instance` -> `StartInstances`
  - `stop_instance` -> `StopInstances`
  - `restart_instance` -> `RebootInstances`
- OCI Compute 写操作映射：
  - `start_instance` -> `InstanceAction START`
  - `stop_instance` -> `InstanceAction STOP`
  - `restart_instance` -> `InstanceAction RESET`
- 阿里云 ECS 写操作映射：
  - `start_instance` -> `StartInstance`
  - `stop_instance` -> `StopInstance`
  - `restart_instance` -> `RebootInstance`
- 写操作执行前会拒绝以下情况：
  - 写保护开关未开启。
  - 状态读取不是 live 模式。
  - provider live 状态读取失败。
  - 云账号不可用或凭证不完整。
- 新增幂等空操作处理：
  - 启动已运行实例时，不再调用 provider 写 API，直接返回 `idempotentNoop=true`。
  - 停止已停止实例时，不再调用 provider 写 API，直接返回 `idempotentNoop=true`。
- 写操作结果新增审计字段：
  - `preState`
  - `providerWriteRequest`
  - `rawResponse`
  - `postState`
  - `providerAdapterExecuted`
  - `idempotentNoop`
- 写操作提交成功后，会复用 live read 通道读取一次 `postState`，为后续异步轮询和最终态确认预留结构。
- `executeCloudAssetProviderAction` 不再固定写死 `providerAdapterExecuted=false`，改为以 adapter 执行结果为准。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `cloud_operation.go`、`cloud_operation_provider_execute.go`、`cloud_operation_provider_state.go`。
- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal` 通过，后端 Go 编译通过。
- Docker Compose 重新启动 `iac-portal` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200。
- 鉴权后使用当前 OCI 资产 `ci-d8qlhn9jm3jc73bcu290` 验证动作列表：
  - `start_instance`、`stop_instance`、`restart_instance` 仍为 `enabled=false`。
  - `adapterStatus=registered`、`writeEnabled=false`。
  - 禁用原因为 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED` 未开启。
- 鉴权后回归三类动作 dry-run：
  - 默认 `cache` 模式仍返回 `CMDB缓存(cache) 状态 RUNNING 已归一化为 running`。
  - `execution_guard` 仍失败并指向写保护开关未开启。
  - 确认默认部署不会因为写操作执行器接入而开放云端写操作。
- 当前测试环境没有开启 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED=true` 和 `CLOUD_OPERATION_PROVIDER_READ_MODE=live`，未对真实云端写操作做联通验证。

待继续：

- 接入审批流、二次确认、操作原因强校验和风险提示后，再允许生产环境受控打开 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED`。
- 将写操作从同步提交演进为异步执行任务，补充运行中状态、轮询最终态、失败重试和取消/回滚提示。
- 补充真实云账号沙箱验证：AWS EC2、OCI Compute、阿里云 ECS 的 start/stop/reboot 全链路。

### 16.20 2026-06-20 V1.0 P1 云资产动作二次确认和操作原因强校验

状态：已完成 provider 云资产动作创建前的二次确认和操作原因强校验；本地元数据刷新仍保持轻量创建流程。

已完成：

- 创建云资产动作请求新增确认字段：
  - `confirmAction`
  - `confirmResourceId`
- 后端服务层新增 provider 操作确认守卫：
  - provider 动作必须填写不少于 6 个字符的操作原因。
  - provider 动作必须输入当前动作 key 进行确认。
  - provider 动作必须输入当前云资源 ID 进行确认。
  - 本地 `refresh_metadata` 动作不强制二次确认。
- 操作任务审计参数新增 `confirmation` 结构，记录提交时输入的确认动作和资源 ID。
- dry-run 预检查新增“二次确认”提醒项，提前提示创建任务时需要填写操作原因并确认动作/资源 ID。
- 云资产详情页“操作”Tab 的创建任务入口改为受控弹窗：
  - provider 动作展示动作、风险、执行方式和目标资源。
  - provider 动作提交前必须填写原因、确认动作和确认资源 ID。
  - 本地动作默认保留“来自资产详情页”的操作原因。
- 前端创建任务文案从“不会执行云端变更”调整为按动作执行方式展示，避免与 provider 写操作执行器能力冲突。

验证：

- 使用 Docker Go 镜像执行 `gofmt`。
- `git diff --check` 通过。
- Docker Compose 生产构建 `iac-portal` 和 `iac-web` 通过；前端仅保留既有 bundle size warning。
- Docker Compose 重新启动 `iac-portal` 和 `iac-web` 通过，`iac-portal` healthy，`GET /api/v1/check` 返回 200，Web 首页返回 200。
- 鉴权后使用当前 OCI 资产 `ci-d8qlhn9jm3jc73bcu290` 回归动作列表：
  - provider 动作仍为 `enabled=false`。
  - `adapterStatus=registered`、`writeEnabled=false`。
  - 禁用原因仍指向 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED` 未开启。
- 鉴权后回归 `stop_instance` dry-run，返回新增 `operator_confirmation` 检查项，页面展示“二次确认=提醒”。
- 鉴权后创建 `refresh_metadata` 本地动作任务 `cop-d8r1c25djb4c73e3qbj0`，任务状态 `complete`，确认本地安全动作未被二次确认误伤。
- 内置浏览器验证资产详情“操作”Tab 正常渲染；点击 `stop_instance` 预检查后展示“二次确认”提醒；点击 `refresh_metadata` 创建任务弹窗仅展示操作原因，不展示 provider 确认字段；浏览器控制台无 error。
- 默认写操作保护开关仍关闭，本轮未在当前环境执行真实云端写操作。

待继续：

- 将云资产操作从同步完成演进为异步任务，补充运行中状态、最终态轮询和失败重试。
- 接入审批流后开放 critical/destructive 类动作。
- 补充真实云账号沙箱验证：AWS EC2、OCI Compute、阿里云 ECS 的 start/stop/reboot 全链路。

### 16.21 2026-06-20 V1.0 P1 云资产动作失败重试入口

状态：已完成云资产动作失败任务的重试入口第一阶段；重试会创建新的操作任务，原失败任务保留审计。

已完成：

- 新增操作任务重试 API：
  - `POST /api/v1/cloud/operations/:id/retry`
- 重试入口仅支持失败状态的云资产动作任务：
  - 仅允许 `status=failed`。
  - 仅允许 `operationType=action`。
  - 必须具备原始资产 ID 和动作 key。
- 重试时复用原任务参数：
  - 资产 ID
  - 动作 key
  - 操作原因
  - provider 二次确认字段 `confirmAction`、`confirmResourceId`
- 重试任务不会覆盖原失败任务，而是创建新的云资产动作任务，并在新任务参数中写入 `retryFromOperationId`。
- 原失败任务新增审计记录：
  - 重试成功时记录新任务 ID。
  - 重试失败时记录失败原因。
- 前端“多云管理 - 操作任务”详情页新增“重试任务”按钮：
  - 仅在失败的云资产动作任务上展示。
  - 点击后按原参数创建重试任务。
  - 创建成功后自动切换到新重试任务详情，并刷新列表。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，覆盖本次新增/修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；前端生产构建仅保留既有 bundle size warning。
- 使用 Docker Compose 重新启动 `iac-portal` 和 `iac-web`，`iac-portal` 状态为 healthy，`GET /api/v1/check` 返回 `success=true`，首页返回 HTTP 200。
- 鉴权后验证 `GET /api/v1/cloud/operations?currentPage=1&pageSize=10` 返回 9 条操作任务。
- 鉴权后对完成态 action 任务 `cop-d8r1c25djb4c73e3qbj0` 调用 `POST /api/v1/cloud/operations/:id/retry`，后端拒绝并返回 `50010340`，提示当前状态不是失败，不能重试。
- 内置浏览器验证“多云管理 - 操作任务”页面可加载；打开完成态云资产动作任务详情，状态为“完成”，且不展示“重试任务”按钮，避免误操作。
- 2026-06-21 补充失败态重试闭环验证：
  - 插入临时云采集资产 `ci-codex-retry1621`。
  - 插入失败态云资产动作任务 `cop-codex-retry1621-fail`，动作 `refresh_metadata`，状态 `failed`。
  - 鉴权后查询任务详情，确认失败任务返回 `operationType=action`、`status=failed` 和既有失败审计。
  - 鉴权后调用 `POST /api/v1/cloud/operations/cop-codex-retry1621-fail/retry` 成功创建重试任务 `cop-d8rqp0f2g0ms73fnjno0`。
  - 新重试任务返回 `status=complete`，参数中写入 `retryFromOperationId=cop-codex-retry1621-fail`，并复用原失败任务的 `reason`。
  - 原失败任务新增审计记录“已创建重试任务 cop-d8rqp0f2g0ms73fnjno0”。
  - 内置浏览器验证操作任务列表同时展示原失败任务和完成的重试任务；打开原失败任务详情可看到“重试任务”按钮及新增重试审计。

验证限制：

- 当前环境未开启 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED=true` 和 `CLOUD_OPERATION_PROVIDER_READ_MODE=live`，未做真实云端写操作联通验证。

待继续：

- provider 写操作异步后台任务已在 16.22 完成第一阶段。
- provider 写操作最终态轮询、超时和重试退避策略已在 16.23 完成第一阶段。
- 接入审批流后开放 critical/destructive 类动作。

### 16.22 2026-06-20 V1.0 P1 Provider 云资产动作异步执行第一阶段

状态：已完成 provider 云资产动作异步执行骨架；本地安全动作保持同步完成，provider 写动作创建后进入后台执行流转。

已完成：

- 云资产动作创建逻辑按执行方式分流：
  - `local_metadata` 继续同步执行并返回最终状态。
  - `provider_operation` 创建后先写入 `pending` 状态，参数中记录 `executionMode=async`，随后由后台执行器推进。
- 新增后台执行器：
  - 后台执行开始时将操作任务从 `pending` 更新为 `running`。
  - 后台执行成功后写入 `complete`、结果、步骤和审计。
  - 后台执行失败后写入 `failed`、错误结果、步骤和审计。
  - 后台 panic 会落 `failed`，并记录错误与堆栈，避免任务永久卡住。
- 增加取消保护：
  - 任务执行前如果已被取消，不再执行。
  - 执行完成前再次检查 `aborted`，避免后台完成结果覆盖用户取消状态。
- 前端“多云管理 - 操作任务”详情页增加运行态轮询：
  - 当详情抽屉中的任务为 `pending` 或 `running` 时，每 3 秒刷新详情和列表。
  - 任务进入最终态后自动停止轮询。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；前端生产构建仅保留既有 bundle size warning。
- 使用 Docker Compose 重新启动 `iac-portal` 和 `iac-web`，`iac-portal` 状态为 healthy，`GET /api/v1/check` 返回 `success=true`，首页返回 HTTP 200。
- 鉴权后创建本地安全动作 `refresh_metadata`，任务 `cop-d8r1vdqnb22c73a0r0mg` 返回 `status=complete`、`executionMode=sync`、`providerAdapterExecuted=false`，确认本地动作未受异步改造影响。
- 鉴权后查询 provider 动作列表，`start_instance`、`stop_instance`、`restart_instance` 仍为 `enabled=false`、`adapterStatus=registered`、`writeEnabled=false`。
- 鉴权后尝试创建 `start_instance` provider 动作，服务端返回 `50010340` 并被预检查拒绝，确认异步执行未绕过安全门控。
- 内置浏览器验证“多云管理 - 操作任务”页面可加载；打开任务 `cop-d8r1vdqnb22c73a0r0mg` 详情，参数展示 `executionMode=sync`，状态为“完成”，详情无“取消任务”和“重试任务”误展示。

验证限制：

- 当前测试资产未绑定可用于 provider 写操作的云账号，且环境未开启 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED=true` 与 `CLOUD_OPERATION_PROVIDER_READ_MODE=live`，未创建真实 provider `pending -> running -> complete/failed` 任务。

待继续：

- 增加 provider 写操作最终态轮询、超时和重试退避策略。
- 增加后台任务恢复机制，避免 Portal 重启时遗留 `pending/running` provider 操作。
- 接入审批流后开放 critical/destructive 类动作。

### 16.23 2026-06-20 V1.0 P1 Provider 写操作最终态轮询和退避策略

状态：已完成 provider 计算实例写操作提交后的最终态确认机制；真实写操作提交成功后不再只读一次状态，而是按可配置策略轮询直到目标状态、不可重试错误或超时。

已完成：

- provider 写操作执行器新增最终态轮询：
  - 写操作提交成功后调用 provider live 状态读取接口确认最终状态。
  - 目标状态来自动作定义：启动/重启目标为 `running`，停止目标为 `stopped`。
  - 每次轮询记录 `rawState`、`normalizedState`、目标状态、是否命中和读取错误。
- 增加退避和超时控制：
  - 默认轮询 3 次。
  - 默认初始间隔 2 秒。
  - 默认最大退避 10 秒。
  - 读取错误不可重试时立即失败；可重试错误会继续轮询直到次数耗尽。
- 操作结果结构化留痕：
  - `statePolling.attempts`
  - `statePolling.finalState`
  - `statePolling.matched`
  - `statePolling.timedOut`
  - `statePolling.error`
- 执行计划增加 `providerStatePolling`，在审计和结果中可看到最终态确认策略。
- 新增部署配置项：
  - `CLOUD_OPERATION_PROVIDER_POLL_ATTEMPTS`
  - `CLOUD_OPERATION_PROVIDER_POLL_INTERVAL_SECONDS`
  - `CLOUD_OPERATION_PROVIDER_POLL_MAX_DELAY_SECONDS`

验证：

- 使用 Docker Go 镜像执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过。
- 使用 Docker Compose 重新启动 `iac-portal` 和 `iac-web`，`iac-portal` 状态为 healthy，`GET /api/v1/check` 返回 `success=true`，首页返回 HTTP 200。
- 鉴权后创建本地安全动作 `refresh_metadata`，任务 `cop-d8r3n8fn7fbc73ctajqg` 返回 `status=complete`、`executionMode=sync`、`providerAdapterExecuted=false`，确认本地动作未受最终态轮询改造影响。
- 鉴权后查询 provider 动作列表，`start_instance` 仍为 `enabled=false`、`adapterStatus=registered`、`writeEnabled=false`。
- 鉴权后尝试创建 `start_instance` provider 动作，服务端返回 `50010340` 并被预检查拒绝，确认最终态轮询改造未绕过安全门控。
- 内置浏览器验证“多云管理 - 操作任务”页面可加载；打开任务 `cop-d8r3n8fn7fbc73ctajqg` 详情，参数展示 `executionMode=sync`，状态为“完成”。

验证限制：

- 当前测试资产未绑定可用于 provider 写操作的云账号，且环境未开启 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED=true` 与 `CLOUD_OPERATION_PROVIDER_READ_MODE=live`，未触发真实 provider 写操作后的 `statePolling` 结果。

待继续：

- 接入审批流后开放 critical/destructive 类动作。

### 16.24 2026-06-20 V1.0 P1 Provider 异步操作遗留任务恢复机制

状态：已完成 provider 异步云资产动作的遗留任务恢复机制；Portal 重启或后台执行异常后，过期 `pending/running` 异步任务会被安全标记为失败，不自动重放云端写操作。

已完成：

- 新增遗留任务恢复入口：
  - 查询操作任务列表时触发恢复检查。
  - 查询操作任务详情时触发恢复检查。
- 恢复范围收敛到云资产动作异步任务：
  - `operationType=action`
  - `status in pending/running`
  - `params.executionMode=async`
  - `updated_at` 超过配置的遗留阈值
- 恢复策略保持安全优先：
  - 不自动重新执行 provider 写操作。
  - 不覆盖已经进入最终态的任务。
  - 将遗留任务标记为 `failed`，并记录原始状态、恢复原因和超时时长。
- 恢复结果结构化留痕：
  - `recoveredBy=stale_cloud_operation_recovery`
  - `previousStatus`
  - `staleAfter`
  - `reason`
- 恢复审计写入操作任务审计列表，详情页可查看恢复说明。
- 新增部署配置项：
  - `CLOUD_OPERATION_STALE_MINUTES`
  - 默认值 `30`

验证：

- 使用 Docker Go 镜像执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过。
- 使用 Docker Compose 重新启动 `iac-portal` 和 `iac-web`，`iac-portal` 状态为 healthy，`GET /api/v1/check` 返回 `success=true`，首页返回 HTTP 200。
- 在测试库写入一条过期异步 provider 动作任务 `cop-codex-stale-1624`：
  - 初始状态 `running`
  - `executionMode=async`
  - `updated_at` 为 2 小时前
- 鉴权后调用 `GET /api/v1/cloud/operations?currentPage=1&pageSize=10` 触发恢复检查。
- 鉴权后查询 `GET /api/v1/cloud/operations/cop-codex-stale-1624`，任务返回：
  - `status=failed`
  - `message=云资产操作任务超时未完成，已由恢复机制标记失败`
  - `result.recoveredBy=stale_cloud_operation_recovery`
  - `result.previousStatus=running`
  - `audits=1`
- 内置浏览器验证“多云管理 - 操作任务”页面第二页可见测试任务：
  - 任务 `Codex stale async recovery`
  - 动作 `启动实例`
  - 状态 `失败`
  - 风险 `中`
- 内置浏览器打开任务详情，确认详情中展示恢复说明、恢复标识和“平台未自动重放 provider 写操作”的原因。

验证限制：

- 当前环境仍未开启 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED=true` 与 `CLOUD_OPERATION_PROVIDER_READ_MODE=live`，本轮只验证遗留任务恢复机制，没有对真实云资源重放或执行写操作。
- 当前恢复机制按接口访问触发，暂未引入独立定时扫描器。

待继续：

- 接入审批流后开放 critical/destructive 类动作。
- 增加 provider 操作任务的更细粒度权限。

### 16.25 2026-06-20 V1.0 P1 Provider 异步操作取消保护和回滚提示

状态：已完成 provider 异步操作取消能力的正式闭环；用户可取消 `pending/running` 操作任务，后台执行器不会覆盖已取消状态，详情页会提示取消不等于云端回滚。

已完成：

- 后端取消结果新增结构化字段：
  - `previousStatus`
  - `executionMode`
  - `providerOperationMayContinue`
  - `rollbackHint`
- 后端状态迁移增加并发保护：
  - 后台执行开始时只允许 `pending -> running`。
  - 任务完成、失败、取消只允许从 `pending/running` 写入最终态。
  - 如果任务已被取消，后台执行器不会把状态覆盖为 `running/complete/failed`。
- 前端操作任务详情新增“云端回滚提示”：
  - 可取消的异步任务展示取消风险提示。
  - 已取消且结果包含 `rollbackHint` 的任务继续展示回滚提示。
- 前端保持既有“取消任务”入口：
  - 仅 `pending/running` 任务展示。
  - 已进入最终态的任务不展示取消按钮。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；前端生产构建仅保留既有 bundle size warning。
- 使用 Docker Compose 重新启动 `iac-portal` 和 `iac-web`，`iac-portal` 状态为 healthy，`GET /api/v1/check` 返回 `success=true`，首页返回 HTTP 200。
- 在测试库写入一条当前时间的 pending 异步 provider 动作任务 `cop-codex-cancel-1625`：
  - 初始状态 `pending`
  - `executionMode=async`
  - 动作 `stop_instance`
- 鉴权后调用 `POST /api/v1/cloud/operations/cop-codex-cancel-1625/cancel`，任务返回：
  - `status=aborted`
  - `message=用户取消操作任务`
  - `result.previousStatus=pending`
  - `result.executionMode=async`
  - `result.providerOperationMayContinue=true`
  - `result.rollbackHint` 提示 provider 操作不会自动回滚
  - `audits=1`
- 再次取消同一任务返回 `50010340`，确认最终态任务不能重复取消。
- 内置浏览器验证“多云管理 - 操作任务”页面可搜索到测试任务：
  - 任务 `Codex async cancel recovery hint`
  - 动作 `停止实例`
  - 状态 `已取消`
  - 风险 `中`
- 内置浏览器打开任务详情，确认展示“云端回滚提示”和取消结果 JSON。

验证限制：

- 当前环境未开启真实 provider 写操作，本轮验证的是平台任务取消、状态保护和提示，不验证云厂商侧取消或回滚能力。

待继续：

- 接入审批流后开放 critical/destructive 类动作。

### 16.26 2026-06-20 V1.0 P1 云资产动作细粒度权限第一阶段

状态：已完成云资产动作按动作风险和项目角色收敛权限的第一阶段；动作创建、取消和重试共用同一套权限矩阵。

已完成：

- 云资产动作权限从“项目成员可统一操作”细化为动作级矩阵：
  - `refresh_metadata`：项目负责人、操作员、审批员可执行。
  - `start_instance` / `stop_instance`：项目负责人、操作员可执行。
  - `restart_instance`：仅项目负责人、组织管理员、平台管理员可执行。
  - `delete_resource`：保留为组织管理员/平台管理员 + 审批流接入后开放。
- 未绑定项目的云资产继续收紧：
  - 仅组织管理员或平台管理员可操作。
- 动作列表和 dry-run 预检查共用细粒度权限：
  - `user_permission` 会返回动作级通过/失败原因。
  - 动作禁用原因会透出权限不足的具体角色要求。
- 操作任务变更入口复用动作级权限：
  - `POST /api/v1/cloud/operations/:id/cancel`
  - `POST /api/v1/cloud/operations/:id/retry`
- 非云资产动作任务的取消继续收敛到组织管理员或平台管理员。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过。
- 使用 Docker Compose 重新启动 `iac-portal` 和 `iac-web`，`iac-portal` 状态为 healthy，`GET /api/v1/check` 返回 `success=true`，首页返回 HTTP 200。
- 创建本地权限测试数据：
  - 测试用户 `u-codex-perm-1626`，非平台管理员。
  - 用户在组织 `org-d8qk6fsd6t1s73fu2kr0` 中为 `member`。
  - 用户在项目 `p-d8qtv3v9fm0s738ko4o0` 中为 `operator`。
  - 测试云资产 `ci-codex-perm-1626` 绑定该项目，类型为 OCI compute instance。
- 使用测试用户正式登录后验证 dry-run：
  - `refresh_metadata` 返回 `user_permission=pass`，提示项目负责人、操作员或审批员可执行刷新元数据。
  - `start_instance` 返回 `user_permission=pass`，提示项目负责人或操作员可执行实例开停机。
  - `restart_instance` 返回 `user_permission=fail`，提示重启实例需要项目负责人、组织管理员或平台管理员权限。
- 使用测试用户验证操作任务变更：
  - 对 `start_instance` pending 任务 `cop-codex-perm-cancel` 调用取消，返回 `status=aborted`。
  - 对 `restart_instance` failed 任务 `cop-codex-perm-retry` 调用重试，返回 `50010340`，确认 operator 不能重试重启动作。

验证限制：

- 当前阶段仍是动作级和项目角色级权限，尚未引入云账号级授权、资源标签级授权和审批流联动。
- 当前环境未开启真实 provider 写操作，本轮只验证平台权限门禁。

待继续：

- 增加云账号级授权和资源标签级授权。

### 16.27 2026-06-20 V1.0 P1 云资产动作审批流第一阶段

状态：已完成云资产动作审批流第一阶段；高风险云操作可进入待审批状态，审批通过后进入既有执行通道，审批驳回后进入最终驳回态并保留审计。

已完成：

- 云操作任务新增审批状态：
  - `approving`：待审批。
  - `rejected`：已驳回。
- 新增云操作审批 API：
  - `POST /api/v1/cloud/operations/:id/approve`
  - 请求字段 `action=approved|rejected`
  - 可选字段 `comment`
- 审批权限第一阶段复用现有组织/项目角色：
  - 平台管理员、组织管理员可审批。
  - 绑定项目的任务允许项目负责人、审批员审批。
  - 项目操作员不能审批。
  - 未绑定项目的任务仅平台管理员或组织管理员可审批。
- 云资产动作创建支持审批等待：
  - 动作声明 `RequiresApproval=true` 时，创建任务进入 `approving`。
  - 任务参数记录 `approval.required`、`approvalId`、申请人和申请时间。
  - 待审批任务不会启动后台执行器。
- 审批通过后进入既有执行通道：
  - 本地动作会从 `approving -> pending -> running -> complete/failed`。
  - provider 动作会从 `approving -> pending` 后交给后台执行器。
  - 执行前会重新执行 dry-run 门禁，避免审批期间配置或权限变化导致绕过检查。
- 审批驳回后进入最终态：
  - `status=rejected`
  - `result.approval` 记录审批人、动作、时间和备注。
  - 审计列表记录审批结果。
- 前端“多云管理 - 操作任务”支持审批状态和审批操作：
  - 状态标签新增“待审批”“已驳回”。
  - 待审批云资产动作详情展示“审批通过”和“驳回任务”按钮。
  - 保留取消任务入口，待审批任务可取消。
- `restart_instance` 标记为需要审批。
- `delete_resource` 仍保持禁用，继续等待 provider 删除 adapter、审批策略和更强回滚/影响面提示补齐后开放。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；前端生产构建仅保留既有 bundle size warning。
- 使用 Docker Compose 重新启动 `iac-portal` 和 `iac-web`，`iac-portal` 状态为 healthy，`GET /api/v1/check` 返回 `success=true`，首页返回 HTTP 200。
- 写入本地审批通过测试任务 `cop-codex-appr-ok`：
  - 初始状态 `approving`
  - 动作 `refresh_metadata`
  - 使用 admin 调用审批通过后，任务返回 `status=complete`，审计数为 3。
- 写入审批驳回测试任务 `cop-codex-appr-rej`：
  - 初始状态 `approving`
  - 动作 `restart_instance`
  - 使用 admin 调用驳回后，任务返回 `status=rejected`，`result.approval.status=rejected`。
- 写入审批权限反向测试任务 `cop-codex-appr-deny`：
  - 项目 operator 用户调用审批通过返回 `50020113`，提示需要项目负责人、审批员、组织管理员或平台管理员权限。
- 内置浏览器验证“多云管理 - 操作任务”页面：
  - 搜索 `Codex approval deny operator` 可见状态“待审批”。
  - 打开详情可见“审批通过”和“驳回任务”按钮。

验证限制：

- 当前环境未开启真实 provider 写操作，审批通过后的真实云端执行仍受 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED`、provider adapter 和云账号 live 配置共同控制。
- `delete_resource` 仍未开放真实执行。

待继续：

- 补充扩缩容、安全组等更多生命周期动作。

### 16.30 2026-06-20 V1.0 P1 安全组/安全列表规则视图第一阶段

状态：已完成多云资产安全规则视图第一阶段；支持从已采集资产属性解析入站/出站规则并在资产详情展示公网暴露标识。

已完成：

- 新增安全规则 API：
  - `GET /api/v1/cloud/assets/:id/security-rules`
- 新增安全规则响应模型：
  - `ruleCount`
  - `publicRuleCount`
  - `rules[]`
  - 规则字段包含方向、协议、来源、目标、端口范围、说明、公网暴露标识和原始规则。
- 支持解析 OCI security list 已采集字段：
  - `attributes.ingressSecurityRules`
  - `attributes.egressSecurityRules`
- 兼容常见字段别名：
  - `ingressRules`、`egressRules`
  - `ipPermissions`、`ipPermissionsEgress`
  - `sourceCidrBlock`、`destinationCidrBlock`、`cidrBlock`
- 支持解析 OCI `tcpOptions/udpOptions.destinationPortRange` 和通用 `fromPort/toPort`。
- 自动识别 `0.0.0.0/0`、`::/0` 为公网暴露规则。
- 前端资产详情新增“安全规则”页签：
  - 展示规则数、公网规则数和资产类型。
  - 表格展示方向、协议、端口、来源、目标、公网暴露和说明。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，覆盖本次新增/修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；前端生产构建仅保留既有 bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web`，`iac-portal` healthy。
- 写入测试安全列表资产 `ci-codex-sg-1630`，包含 2 条入站规则和 1 条出站规则。
- 鉴权后调用 `GET /api/v1/cloud/assets/ci-codex-sg-1630/security-rules` 返回：
  - `ruleCount=3`
  - `publicRuleCount=2`
  - 入站公网 SSH 规则端口解析为 `22`
  - 内网 HTTPS 规则端口解析为 `443`
  - 出站公网规则标识为公网暴露。
- 前端入口 `http://127.0.0.1/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-assets` 返回 HTTP 200。

验证限制：

- 当前阶段只读展示安全规则，不支持安全组写操作。

待继续：

- 补充安全组写操作的审批、二次确认、影响面提示和 provider adapter。

### 16.31 2026-06-20 V1.0 P1 AWS 安全组规则采集字段展开

状态：已完成 AWS 安全组规则采集字段展开；`DescribeSecurityGroups` 返回的 ingress/egress 权限会转换为统一安全规则字段，供安全规则视图复用。

已完成：

- AWS 安全组采集结构新增：
  - `ipPermissions`
  - `ipPermissionsEgress`
  - IPv4 CIDR、IPv6 CIDR、PrefixList、UserIdGroupPair 目标。
- AWS 安全组资产属性新增：
  - `ingressRules`
  - `egressRules`
- 权限展开规则：
  - ingress 目标写入 `source`。
  - egress 目标写入 `destination`。
  - 保留协议、fromPort、toPort、portRange、description。
  - 无明确目标时仍保留一条规则，避免权限信息丢失。
- 统一安全规则 API 会自动复用 `ingressRules/egressRules` 并识别公网暴露。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal`，健康检查通过。
- 写入 AWS 风格安全组测试资产 `ci-codex-aws-sg-1631`：
  - 入站 `0.0.0.0/0:80`
  - 出站 prefix list `pl-codex:443`
- 鉴权后调用 `GET /api/v1/cloud/assets/ci-codex-aws-sg-1631/security-rules` 返回：
  - `ruleCount=2`
  - `publicRuleCount=1`
  - 入站公网规则端口 `80`
  - 出站 prefix list 规则端口 `443`，不标记公网暴露。

验证限制：

- 当前未连接真实 AWS 账号执行 live `DescribeSecurityGroups`，本轮验证使用本地 AWS 风格规则数据。
- 当前阶段只读展示安全规则，不支持安全组写操作。

待继续：

- 补充安全组写操作的审批、二次确认、影响面提示和 provider adapter。

### 16.29 2026-06-20 V1.0 P1 本地安全资源标签更新动作

状态：已完成本地安全资源标签更新动作；支持在资产详情创建 `update_tags` 操作任务，更新 CMDB 资产标签并纳入云操作审计。

已完成：

- 新增云资产动作 `update_tags`：
  - 动作名称：更新标签。
  - 风险等级：低。
  - 执行方式：`local_metadata/local`。
  - 当前阶段只更新 CMDB 本地资产标签，不调用云厂商标签写入 API。
- 创建云资产动作请求新增 `tags` 参数：
  - 类型为 JSON 对象。
  - 标签值为空字符串时表示删除该标签。
  - 标签 key 会去除首尾空格，空 key 会被忽略。
- `update_tags` dry-run 新增“操作参数”提醒，提示创建任务时需要提交 `tags` JSON。
- `update_tags` 创建任务会：
  - 合并更新资产 `tags`。
  - 回写资产 `lastOperationId`。
  - 写入 `iac_cloud_operation`、step 和 audit。
  - 写入 CMDB 资产变更记录，来源为 `cloud_operation`。
  - 任务结果包含 `beforeTags`、`afterTags`、`changedTags`、`removedTags`。
- 前端资产详情“操作”页签支持 `update_tags`：
  - 动作目录展示“更新标签”。
  - 创建弹窗展示标签 JSON 输入框。
  - 默认带出当前资产标签，提交前校验 JSON 对象和非空标签。
- 前端操作任务页新增 `update_tags` 中文动作名称。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；第一次 web 构建因 npm registry `ECONNRESET` 中断，重试后通过，仍只保留既有 bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web`，`iac-portal` healthy。
- 鉴权后验证测试资产 `ci-codex-auth-1628`：
  - `GET /api/v1/cloud/assets/:id/actions` 返回 `update_tags`，`enabled=true`、`adapterMode=local`。
  - `POST /api/v1/cloud/assets/:id/actions/update_tags/dry-run` 返回 `executable=true`，包含“操作参数”提醒。
  - `POST /api/v1/cloud/assets/:id/actions/update_tags` 创建任务 `cop-d8r4fovnbvoc73dhp550`，返回 `status=complete`、`action=update_tags`。
  - 资产详情回读标签包含 `CodexTag=enabled`，`lastOperationId` 指向该任务。
  - 操作详情回读步骤 1 条、审计 2 条，结果包含 `afterTags`。

验证限制：

- 当前阶段仍为本地安全标签更新，尚未调用 AWS/OCI/AliCloud provider 标签写入 API。
- 磁盘扩容动作目录、参数校验和 dry-run 已在 16.81 补齐；扩缩容、安全组写操作仍待开发。

待继续：

- 补充扩缩容、安全组等更多生命周期动作。

### 16.28 2026-06-20 V1.0 P1 云账号级授权和资源标签级授权第一阶段

状态：已完成 provider 云资产动作的云账号级授权和资源标签级授权第一阶段；授权策略进入 dry-run、创建任务和审批通过后的复检链路。

已完成：

- 云账号创建/更新 API 支持 `metadata` 字段，为后续前端策略配置表单预留统一承载位。
- 在 `iac_cloud_account.metadata.operationPolicy` 下支持云操作授权策略：
  - `projectIds`：允许使用该云账号执行云操作的项目 ID。
  - `actions`：允许执行的动作，如 `start_instance`、`stop_instance`、`restart_instance`。
  - `resourceTypes`：允许操作的资产类型，如 `compute_instance`。
  - `requiredTags`：资源标签约束，支持标签存在校验和标签值白名单。
- 策略字段兼容数组、逗号分隔字符串和常见下划线命名，便于 API、脚本或后续 UI 表单写入。
- provider 云资产动作 dry-run 新增检查项：
  - `cloud_account_authorization`：云账号授权。
  - `resource_tag_authorization`：资源标签授权。
- 未配置授权策略时不阻断老账号，预检查提示“未配置授权限制”，继续由项目角色、审批、云账号凭证、区域、provider adapter 和写保护共同控制。
- 配置授权策略后，项目、动作、资源类型或标签不匹配会返回 `fail`，创建任务被执行门禁阻断。
- 云账号权限验证结果新增“操作授权策略”项，提示账号是否已配置云操作策略。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 后，`GET /api/v1/check` 返回 `success=true`。
- 使用本地测试云账号策略验证：
  - 策略允许当前项目、`start_instance`、`compute_instance` 和标签 `Environment=dev` 时，dry-run 返回 `cloud_account_authorization=pass`、`resource_tag_authorization=pass`。
  - 策略改为要求标签 `Environment=prod` 后，dry-run 返回 `resource_tag_authorization=fail`，创建任务会被执行门禁阻断。
- 鉴权后验证 `GET /api/v1/cloud/accounts/:id/permissions` 返回“操作授权策略”项。

验证限制：

- 当前阶段先复用云账号 `metadata` 存储策略，尚未拆分独立 `iac_cloud_account_permission` 策略表。
- 前端暂未提供云账号策略编辑表单，可通过 API 或初始化脚本写入 `metadata.operationPolicy`。
- 当前环境未开启真实 provider 写操作，本轮验证只覆盖预检、门禁和审计输入链路。

待继续：

- 补充扩缩容、标签、安全组等更多生命周期动作。

### 16.32 2026-06-20 V1.2 P1 风险治理中心第一阶段

状态：已完成风险发现模型、风险列表 API、风险状态/例外接口和前端风险合规页面第一阶段；风险列表会基于当前 CMDB 资产自动派生首批治理项。

已完成：

- 新增模型 `iac_cloud_risk_finding`：
  - 记录组织、项目、环境、资产、云账号、provider、账号、区域、资源类型、资源 ID 和资源名称。
  - 记录风险来源、规则 key、规则名称、风险等级、状态、指纹、证据、修复建议、首次发现、最近发现、解决时间、例外到期和例外原因。
  - 通过 `org_id + fingerprint` 去重，同一风险重复刷新只更新最近发现时间和证据。
- 新增风险刷新逻辑 `RefreshCloudRiskFindings`：
  - 查询风险列表前自动从当前组织 CMDB 资产派生风险。
  - 支持 `public_ingress_security_rule` / `public_egress_security_rule`，复用安全规则解析结果识别公网暴露。
  - 支持 `unmanaged_cloud_asset`，将 `managedBy=cloud_only` 资产纳入治理。
  - 支持 `unowned_cloud_asset`，将缺少 owner 的资产纳入治理。
  - 支持 `cmdb_compliance_risk_high/critical`，将已有高/严重合规风险标记纳入风险列表。
  - 支持 `terraform_drift_detected`，从 `iac_resource_drift` 映射漂移风险。
  - 刷新时会把不再命中的派生风险自动置为 `resolved`。
- 新增风险 API：
  - `GET /api/v1/cloud/risks`：风险列表和摘要统计。
  - `PUT /api/v1/cloud/risks/:id/status`：更新风险状态，支持 `open`、`in_progress`、`resolved`。
  - `POST /api/v1/cloud/risks/:id/suppress`：风险例外，支持例外到期时间和原因。
- 新增前端“多云管理 - 风险合规”页面：
  - 顶部展示未处理、严重、高危、公网暴露、未纳管和已例外风险摘要。
  - 支持按关键字、状态、等级和来源筛选。
  - 表格展示风险、等级、状态、来源、资源、云厂商/区域、项目、最近发现和快捷操作。
  - 详情抽屉展示风险证据、修复建议、首次/最近发现、解决时间和例外信息。
  - 支持标记处理中、标记已解决、重新打开和风险例外。
- 多云管理菜单新增“风险合规”，路由为 `/org/:orgId/m-cloud-risks`。

验证：

- 使用 Docker Go 镜像在 `/private/tmp` 临时目录执行 `gofmt`，覆盖本次新增 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web`，`iac-portal` healthy，`iac-web` started。
- 鉴权后调用 `GET /api/v1/cloud/risks?currentPage=1&pageSize=5` 返回 200：
  - `total=18`
  - `open=18`
  - `critical=1`
  - `high=4`
  - `publicExposure=3`
  - `unmanagedAssets=8`
  - `unownedAssets=4`
  - `complianceRisks=3`
- 验证 `PUT /api/v1/cloud/risks/:id/status`：
  - 风险可更新为 `in_progress`。
  - 测试后可恢复为 `open`。
- 验证 `POST /api/v1/cloud/risks/:id/suppress`：
  - 风险可更新为 `suppressed`。
  - 返回例外到期时间和例外原因。
  - 测试后通过状态接口恢复为 `open`。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-risks`：
  - 菜单出现“风险合规”。
  - 页面展示风险摘要和风险列表。
  - 点击第一条风险可以打开详情抽屉，并显示风险详情、证据和修复建议。

验证限制：

- 风险规则当前为代码内置派生规则，尚未做规则配置 UI 和规则启停。
- 策略扫描结果目前只通过 CMDB `complianceRisk` 汇总映射，尚未逐条关联 `PolicyResult` 证据。
- 风险状态变更已可追踪在风险记录证据字段中，并写入统一事件中心。

待继续：

- 建设平台级事件中心，并让风险发现、状态变更、云操作完成和预算超限写入统一事件。
- 扩展可配置风险规则、策略结果明细映射、风险评分、审批例外和 ITSM/Webhook 推送。

### 16.33 2026-06-20 V1.2 P1 平台级事件中心第一阶段

状态：已完成平台级云事件模型、事件列表 API、事件中心页面和首批事件写入；风险状态变更、云操作任务、CMDB 资产变更已经可以沉淀到统一事件中心。

已完成：

- 新增模型 `iac_cloud_event`：
  - 记录组织、项目、环境、资产、云操作、风险发现、云账号、操作者。
  - 记录来源、事件类型、级别、状态、provider、账号、区域、资源类型、资源 ID、资源名称、标题、消息、载荷、用户 IP 和发生时间。
  - 事件来源覆盖 `account`、`sync`、`operation`、`risk`、`cost`、`cmdb`、`notification`，为后续预算、Webhook 和同步任务事件预留。
- 新增事件 API：
  - `GET /api/v1/cloud/events`：事件列表。
  - 支持按关键字、来源、事件类型、级别、状态、provider、账号、区域、资产、操作、风险、项目和环境筛选。
- 新增事件写入 helper：
  - `recordCloudEvent`：统一事件写入。
  - `recordCloudEventBestEffort`：非阻断写入，事件失败不会中断原业务流程。
- 接入首批事件来源：
  - 云操作任务创建写入 `cloud_operation.created`。
  - 云操作任务完成/失败/取消/驳回写入 `cloud_operation.<status>`。
  - 风险首次发现写入 `risk.detected`。
  - 风险状态更新写入 `risk.status_updated`。
  - 风险例外写入 `risk.suppressed`。
  - CMDB 资产新增/变更写入 `cmdb.asset.created` / `cmdb.asset.updated`。
- 新增前端“多云管理 - 事件中心”页面：
  - 支持按关键字、来源和级别筛选。
  - 表格展示事件、级别、来源、类型、状态、资源、操作者和发生时间。
  - 详情抽屉展示事件上下文和 JSON 载荷。
- 多云管理菜单新增“事件中心”，路由为 `/org/:orgId/m-cloud-events`。

验证：

- 使用 Docker Go 镜像在 `/private/tmp` 临时目录执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web`，`iac-portal` healthy，`iac-web` started。
- 鉴权后通过风险状态变更生成事件：
  - `PUT /api/v1/cloud/risks/:id/status` 更新为 `in_progress`。
  - 再恢复为 `open`。
- 鉴权后调用 `GET /api/v1/cloud/events?source=risk&currentPage=1&pageSize=5` 返回 200：
  - `total=2`
  - 事件类型包含 `risk.status_updated`
  - 事件载荷包含状态变更备注。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`：
  - 菜单出现“事件中心”。
  - 页面展示风险状态更新事件。
  - 点击事件可以打开详情抽屉，并展示事件载荷。

验证限制：

- 当前事件中心为平台内查询能力，尚未接入外部 Webhook/ITSM 推送。
- 云账号验证、同步任务完成、成本预算超限事件仍待补充写入。
- 事件归档、重试推送、签名、投递状态和 API Token 访问控制待后续建设。

待继续：

- 补充 Webhook/ITSM/API Token，并将账号验证、同步任务、成本预算、云操作审批等更多流程接入事件中心。

### 16.34 2026-06-20 V1.2 P1 多云成本中心第一阶段

状态：已完成统一成本明细模型、成本汇总/趋势/明细 API、未匹配账单视图和成本中心前端页面；现阶段可以从既有 `iac_bill` 与 CMDB 资产成本字段归一化成本记录，并按 provider、项目、应用、业务线和成本中心聚合展示。

已完成：

- 新增模型 `iac_cloud_cost_record`：
  - 记录组织、项目、环境、资产、云账号、provider、账号、区域、服务、资源类型、资源 ID、资源名称。
  - 记录金额、币种、账期、成本中心、owner、应用、业务线、匹配资产状态、来源、来源 ID、指纹和原始载荷。
  - 通过 `org_id + fingerprint` 去重，重复刷新时更新金额、归属和载荷。
- 新增成本刷新逻辑 `RefreshCloudCostRecords`：
  - 将既有 `iac_bill` 账单按标准成本明细归一化。
  - 将 CMDB 资产中 `cost > 0` 的资产成本归一化为成本明细。
  - 支持通过云资源原生 ID 和 IaC 资源 ID 自动匹配 CMDB 资产。
- 新增成本 API：
  - `GET /api/v1/cloud/cost/summary`：成本总览和 provider、项目、应用、业务线、成本中心维度聚合。
  - `GET /api/v1/cloud/cost/trends`：成本趋势。
  - `GET /api/v1/cloud/cost/records`：成本明细列表。
  - `GET /api/v1/cloud/cost/unmatched`：未匹配资产的成本记录列表。
- 新增前端“多云管理 - 成本中心”页面：
  - 顶部展示总成本、成本记录、已匹配资产和未匹配账单。
  - 展示 Provider 成本、应用成本和近 6 个月趋势。
  - 支持按关键字、账期、来源和资产匹配状态筛选成本明细。
  - 点击成本记录可打开详情抽屉，查看资源、归属、匹配状态和原始载荷。
- 多云管理菜单新增“成本中心”，路由为 `/org/:orgId/m-cloud-costs`。

验证：

- 使用 Docker Go 镜像在 `/private/tmp` 临时目录执行 `gofmt`，覆盖本次新增 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web`，`iac-portal` healthy，`iac-web` started。
- 鉴权后调用 `GET /api/v1/cloud/cost/summary` 返回 200：
  - `totalAmount=300.85`
  - `recordCount=7`
  - `matchedCount=7`
  - `unmatchedCount=0`
  - provider 聚合包含 `oci`
  - 应用聚合包含 `Billing Core`、`Import App`、`Payment Service` 等。
- 鉴权后调用 `GET /api/v1/cloud/cost/records?currentPage=1&pageSize=5` 返回 200：
  - `total=7`
  - 首条记录来源为 `cmdb_asset`
  - 首条记录资产为 `cmdb-demo-instance`
  - 首条记录金额为 `123.45 CNY`。
- 鉴权后调用 `GET /api/v1/cloud/cost/trends?granularity=month` 返回 200：
  - 返回 1 个账期点。
  - `2026-06` 成本为 `300.85 CNY`。
- 鉴权后调用 `GET /api/v1/cloud/cost/unmatched?currentPage=1&pageSize=5` 返回 200：
  - 当前验证数据未匹配成本为 0 条。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-costs`：
  - 菜单出现“成本中心”。
  - 页面展示总成本、成本记录、Provider 成本、应用成本和趋势。
  - 列表展示 7 条成本记录链接。
  - 点击第一条成本记录可以打开详情抽屉，并展示金额、匹配资产和原始载荷。

验证限制：

- 该阶段成本归一化主要复用既有 `iac_bill` 与 CMDB 资产成本字段；Azure/GCP/TencentCloud/Huawei/AWS/OCI 拉取型账单来源已在后续 16.9x-16.105 阶段补齐第一阶段。
- 预算管理、成本异常和闲置资源优化建议已完成第一阶段；汇率/摊销规则仍待后续建设。
- 成本中心页面已提供查询、分析、预算和成本建议视图；导入任务管理和成本异常工单闭环尚未完成。

待继续：

- 在真实云账单环境继续端到端联调，并扩展汇率、摊销、CSV 字段差异、Parquet/Excel、分区目录和异常工单闭环。

### 16.35 2026-06-20 V1.2 P1 成本预算管理第一阶段

状态：已完成预算模型、预算 CRUD API、预算汇总、成本中心预算视图和超阈值事件写入；预算可按组织、项目、环境、provider、账号、应用、业务线、成本中心和 owner 维度配置，并基于统一成本明细实时评估使用率。

已完成：

- 新增模型 `iac_cloud_budget`：
  - 记录组织、项目、环境、云账号、预算名称、说明、范围、provider、账号、区域、应用、业务线、成本中心、owner。
  - 记录账期、币种、预算金额、预警阈值、启停状态、最近评估金额、最近使用率、最近评估时间和最近超阈值时间。
  - 通过 `org_id + name` 做唯一约束，避免同组织内预算重名。
- 新增预算评估逻辑：
  - 查询预算前刷新统一成本明细。
  - 按预算范围和维度条件聚合当前账期成本。
  - 计算使用率、剩余额度、超阈值和超预算状态。
  - 当预算从未达阈值变成达到阈值时写入成本事件，避免重复刷新刷屏。
- 新增预算 API：
  - `GET /api/v1/cloud/budgets/summary`：预算汇总。
  - `GET /api/v1/cloud/budgets`：预算列表。
  - `POST /api/v1/cloud/budgets`：创建预算。
  - `PUT /api/v1/cloud/budgets/:id`：更新预算。
  - `DELETE /api/v1/cloud/budgets/:id`：删除预算。
- 新增事件写入：
  - 预算超阈值写入 `cost.budget.threshold_exceeded`。
  - 事件来源为 `cost`。
  - 事件载荷包含预算 ID、预算名称、范围、账期、币种、预算金额、当前金额、使用率和阈值。
- 成本中心页面新增“预算管理”：
  - 展示预算总额、当前消耗、超阈值预算和超预算数。
  - 展示预算列表、范围、预算金额、当前成本、使用率、阈值、状态。
  - 支持新建、编辑和删除预算。
  - 预算弹窗支持组织、项目、环境、provider、账号、应用、业务线、成本中心和 owner 范围。

验证：

- 使用 Docker Go 镜像在 `/private/tmp` 临时目录执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web`，`iac-portal` healthy，`iac-web` started。
- 鉴权后创建组织级预算：
  - `POST /api/v1/cloud/budgets` 返回 200。
  - 创建预算金额为 `100 CNY`、阈值为 `80%`。
  - 当前成本为 `300.85 CNY`。
  - 使用率为 `300.85%`。
  - 返回 `thresholdReached=true`、`budgetExceeded=true`。
- 鉴权后调用 `GET /api/v1/cloud/budgets/summary` 返回 200：
  - `budgetCount=1`
  - `thresholdCount=1`
  - `exceededCount=1`
  - `totalLimitAmount=100`
  - `totalCurrentAmount=300.85`
- 鉴权后调用 `GET /api/v1/cloud/budgets` 返回 200：
  - `total=1`
  - 首条预算为本次创建的 `Codex budget 2026-06 ...`。
- 鉴权后调用 `GET /api/v1/cloud/events?source=cost&eventType=cost.budget.threshold_exceeded` 返回 200：
  - `total=1`
  - 事件来源为 `cost`
  - 事件类型为 `cost.budget.threshold_exceeded`。
- 鉴权后调用 `PUT /api/v1/cloud/budgets/:id` 返回 200：
  - 预算金额更新为 `500 CNY`。
  - 使用率更新为 `60.17%`。
  - 返回 `thresholdReached=false`。
- 鉴权后创建并删除临时预算：
  - `DELETE /api/v1/cloud/budgets/:id` 返回 200。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-costs`：
  - 页面展示“预算管理”。
  - 页面展示“新建预算”按钮。
  - 页面展示本次创建的预算。
  - 页面展示“超阈值预算”和“超预算”指标。
  - 点击“新建预算”可打开预算弹窗。
  - 弹窗展示预算名称、预算范围、预算金额和预警阈值字段。

验证限制：

- 当前预算评估支持查询触发、手动到期评估入口和后台定时评估 worker；后台 worker 已在 16.66 完成第一阶段。
- 超阈值事件已写入事件中心并可投递 Webhook，但尚未接入 ITSM 或专用告警渠道。
- 预算暂不支持审批、预算版本、汇率、摊销、财务编码和复杂预算周期。

待继续：

- 预算后台定时评估 worker 已在 16.66 完成第一阶段；继续建设专用告警渠道和预算审批。
- 扩展多账期成本异常、资源使用率驱动的优化建议、Webhook 重试和 ITSM 成本事件推送。

### 16.36 2026-06-20 V1.2 P1 Webhook 集成第一阶段

状态：已完成事件 Webhook 配置、签名投递、投递记录、最近投递状态和事件中心页面入口；平台级事件创建后可按事件来源和事件类型推送到外部系统。

已完成：

- 新增模型 `iac_cloud_webhook`：
  - 记录组织、名称、说明、目标地址、签名密钥、事件类型、事件来源、启停状态和超时时间。
  - 记录最大重试次数和重试间隔。
  - 记录最近投递状态、状态码、消息和投递时间。
  - 通过 `org_id + name` 做唯一约束，避免同组织内配置重名。
- 新增模型 `iac_cloud_webhook_delivery`：
  - 记录组织、Webhook、事件 ID、事件类型、投递状态、目标地址、尝试次数、请求载荷、响应码、响应体、错误消息和投递时间。
  - 记录父投递 ID、投递模式、下一次重试时间和已重试时间，支持区分初始投递、手动重试、自动重试和测试发送。
  - 支持查询单个 Webhook 的投递记录。
- 新增 Webhook API：
  - `GET /api/v1/cloud/webhooks`：Webhook 列表。
  - `POST /api/v1/cloud/webhooks`：创建 Webhook。
  - `PUT /api/v1/cloud/webhooks/:id`：更新 Webhook。
  - `DELETE /api/v1/cloud/webhooks/:id`：删除 Webhook。
  - `GET /api/v1/cloud/webhooks/:id/deliveries`：Webhook 投递记录。
  - `POST /api/v1/cloud/webhooks/:id/deliveries/:deliveryId/retry`：重试失败投递。
  - `POST /api/v1/cloud/webhooks/retry-due`：重试已到期的失败投递。
  - `GET /api/v1/cloud/webhooks/queue/summary`：Webhook 队列摘要。
  - `GET /api/v1/cloud/webhooks/dead-letters`：Webhook 死信列表。
  - `POST /api/v1/cloud/webhooks/dead-letters/:deadLetterId/replay`：重放死信。
  - `POST /api/v1/cloud/webhooks/dead-letters/:deadLetterId/ignore`：忽略死信。
  - `POST /api/v1/cloud/webhooks/:id/secret/rotate`：轮换签名密钥。
  - `POST /api/v1/cloud/webhooks/deliveries/:deliveryId/signature-verification`：回传接收端验签结果。
  - `POST /api/v1/cloud/webhooks/:id/test`：发送 Webhook 测试事件。
- 新增事件投递逻辑：
  - `recordCloudEvent` 保存事件后触发 Webhook 分发。
  - 支持按事件类型和事件来源过滤，留空表示匹配全部。
  - 请求头包含 `X-CloudIaC-Event`、`X-CloudIaC-Delivery`。
  - 配置签名密钥时写入 `X-CloudIaC-Signature: sha256=<hmac>` 和 `X-CloudIaC-Signature-Version`。
  - 旧密钥仍在宽限期时写入上一版本和过期时间提示头。
  - 投递成功/失败都会写入投递记录，并回写 Webhook 最近状态。
  - 支持对失败投递进行手动重试，重试会生成新的投递记录并保留原失败记录。
  - 失败投递会按 Webhook 配置写入下一次重试时间，查询 Webhook/投递记录、触发新事件或调用重试到期接口时会自动补偿到期投递。
  - 测试发送使用 `webhook.test` 事件和 `test` 投递模式，不影响真实业务事件。
- 事件中心页面新增 Webhook 管理区：
  - 展示 Webhook 名称、目标地址、事件类型、启停状态、最近投递、成功/失败计数。
  - 展示最大重试次数、重试间隔和队列摘要。
  - 支持新建、编辑和删除 Webhook。
  - 支持单个 Webhook 测试发送、暂停/恢复、手动触发“重试到期”和查看死信。
  - Webhook 弹窗支持目标地址、事件类型、事件来源、签名密钥、状态、超时时间、最大重试次数和重试间隔。

验证：

- 使用 Docker Go 镜像在 `/private/tmp` 临时目录执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web`，`iac-portal` healthy，`iac-web` started。
- 鉴权后创建 Webhook：
  - `POST /api/v1/cloud/webhooks` 返回 200。
  - 目标地址为 `http://127.0.0.1:9030/api/v1/check`。
  - 事件类型为 `cost.budget.threshold_exceeded`。
  - 事件来源为 `cost`。
  - 返回结果中签名密钥被隐藏。
- 鉴权后更新 Webhook：
  - `PUT /api/v1/cloud/webhooks/:id` 返回 200。
  - 更新说明成功。
  - 未填写密钥时不会覆盖已有密钥，返回结果仍隐藏密钥。
- 鉴权后通过创建低预算触发成本事件：
  - `POST /api/v1/cloud/budgets` 返回 200。
  - 返回 `thresholdReached=true`。
  - Webhook 投递记录生成。
- 鉴权后调用 `GET /api/v1/cloud/webhooks/:id/deliveries` 返回 200：
  - `total=1`
  - 首条投递状态为 `success`
  - 响应码为 `200`
  - 事件类型为 `cost.budget.threshold_exceeded`。
- 鉴权后调用 `GET /api/v1/cloud/webhooks` 返回 200：
  - `total=1`
  - 最近投递状态为 `success`
  - 成功计数为 `1`
  - 失败计数为 `0`。
- 鉴权后验证失败重试：
  - 创建目标地址为不可用端口的 Webhook。
  - 触发成本预算事件后首条投递状态为 `failed`。
  - 将 Webhook 目标地址更新为 `http://127.0.0.1:9030/api/v1/check`。
  - 调用 `POST /api/v1/cloud/webhooks/:id/deliveries/:deliveryId/retry` 返回 200。
  - 重试返回状态为 `success`，响应码为 `200`。
  - 投递列表保留 2 条记录，状态包含 `failed` 和 `success`。
- 鉴权后创建并删除临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/:id` 返回 200。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`：
  - 页面展示“Webhook”管理区。
  - 页面展示“新建 Webhook”按钮。
  - 页面展示本次创建的 Webhook。
  - 页面展示成功投递状态。
  - 页面展示预算超阈值事件。
  - 点击“新建 Webhook”可打开弹窗。
  - 弹窗展示名称、目标地址、事件类型和签名密钥字段。

验证限制：

- 当前投递仍以事件创建时同步投递为主，到期自动重试已接入后台 worker，并保留查询/事件触发的轻量补偿；死信已落表，投递队列仍复用 `iac_cloud_webhook_delivery.next_retry_at`。
- Webhook 支持 HMAC 签名、签名密钥轮换、接收端验签结果回传、失败重试 API、指数退避、重试抖动、最大重试窗口、后台到期自动重试、队列可观测摘要、测试发送、暂停/恢复、页面级单条失败重放和死信重放。
- ITSM、告警平台、企业微信/钉钉/Slack/邮件等渠道仍待后续建设。

待继续：

- 完善队列保留、清理策略和多分片执行器。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 Webhook/ITSM。

### 16.37 2026-06-20 V1.2 P2 成本异常与优化建议第一阶段

状态：已完成成本异常与优化建议规则推演、列表查询、汇总指标、状态流转和成本中心页面入口；成本记录和 CMDB 资产刷新后会自动生成建议，首次发现会写入事件中心。

已完成：

- 新增模型 `iac_cloud_cost_insight`：
  - 记录组织、项目、环境、资产、云账号、类型、规则、等级、状态、provider、账号、区域、资源、账期、币种、金额、预计可节省金额、标题、说明、建议、证据、指纹和首次/最近发现时间。
  - 通过 `org_id + fingerprint` 做唯一约束，避免同一账期、同一规则、同一资源重复生成。
- 新增成本建议推演逻辑：
  - 从 `iac_cloud_cost_record` 推演未匹配资产成本、缺少 Owner、缺少成本中心和高成本资源。
  - 从 `iac_cmdb_asset` 推演闲置计算实例、未挂载块存储和未使用公网 IP。
  - 重新命中的建议更新 `last_seen_at`，不再命中的待处理建议自动转为已解决。
  - 手动标记为已解决或已忽略的建议不会被下一次刷新自动重开；用户可手动重开。
  - 首次发现建议时写入 `cost.insight.detected` 事件，事件来源为 `cost`。
- 新增成本建议 API：
  - `GET /api/v1/cloud/cost/insights/summary`：汇总总数、待处理、已解决、已忽略、高等级、异常、优化建议和预计可节省金额。
  - `GET /api/v1/cloud/cost/insights`：分页查询成本建议。
  - `PUT /api/v1/cloud/cost/insights/:id/status`：更新建议状态，支持 `open`、`resolved`、`ignored`。
- 成本中心页面新增“成本异常与优化建议”区：
  - 展示待处理、高等级、异常、优化建议和预计可节省金额。
  - 支持按关键字、类型和状态筛选。
  - 列表展示建议、类型、等级、资源、金额、预计可节省金额、云厂商/区域、状态和操作。
  - 待处理建议可标记为解决或忽略，已解决/已忽略建议可重开。

验证：

- 使用 Docker Go 镜像在 `/private/tmp` 临时目录执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web`，`iac-portal` healthy，`iac-web` started。
- 鉴权后调用 `GET /api/v1/cloud/cost/insights/summary?period=2026-06&currency=CNY` 返回 200：
  - `totalCount=8`
  - `openCount=8`
  - `highCount=1`
  - `anomalyCount=8`
  - `optimizationCount=0`
  - `potentialSavings=0`
- 鉴权后调用 `GET /api/v1/cloud/cost/insights?period=2026-06&currency=CNY&status=open&pageSize=5` 返回 200：
  - `total=8`
  - 包含 `high_cost_resource` 规则。
  - 包含 `unallocated_cost_center` 规则。
  - 示例资源 `cmdb-demo-instance` 金额为 `123.45 CNY`。
- 鉴权后验证状态流转：
  - 调用 `PUT /api/v1/cloud/cost/insights/:id/status` 设置 `resolved` 返回 200。
  - 再次查询汇总返回 `openCount=7`、`resolvedCount=1`、`highCount=0`。
  - 调用同一接口设置 `open` 返回 200。
  - 再次查询汇总恢复 `openCount=8`、`resolvedCount=0`、`highCount=1`。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-costs`：
  - 页面展示“成本异常与优化建议”区。
  - 页面展示“高成本资源”建议。
  - 页面展示预算管理区。
  - 页面展示“解决”操作按钮。
  - 浏览器控制台无 error 级日志。

验证限制：

- 当前高成本阈值为固定 `100 CNY`，后续应支持按组织、provider、服务或成本中心配置阈值。
- 当前环比/同比/突增检测尚未启用，需接入多账期真实账单后计算。
- 当前规格调整、预留实例和节省计划建议尚未接入资源利用率、规格和计费折扣数据。
- 当前成本建议可写入事件中心，但尚未创建整改工单或接入 ITSM。

待继续：

- 接入 AWS CUR/Cost Explorer、AliCloud 日账单和多账期成本数据。
- 建设成本异常阈值配置、趋势异常、优化建议明细和 ITSM 整改闭环。

### 16.38 2026-06-20 V1.2 P1 Webhook 自动重试与测试发送第一阶段

状态：已完成 Webhook 自动重试补偿、测试发送、重试配置和事件中心页面操作入口；失败投递可按配置写入下一次重试时间，并通过接口或页面按钮触发到期重试。

已完成：

- `iac_cloud_webhook` 新增 `maxRetries`、`retryInterval`：
  - 创建和编辑 Webhook 时可配置最大重试次数和重试间隔。
  - 后端默认值为 `3` 次和 `60` 秒，并限制最大重试次数为 `10`、最大重试间隔为 `3600` 秒。
- `iac_cloud_webhook_delivery` 新增重试链路字段：
  - `parentDeliveryId`：记录由哪一次失败投递派生。
  - `deliveryMode`：区分 `initial`、`manual`、`auto`、`test`。
  - `nextRetryAt`：失败后下一次可自动重试时间。
  - `retriedAt`：到期投递被补偿处理的时间。
- 新增 Webhook 到期重试 API：
  - `POST /api/v1/cloud/webhooks/retry-due`：批量处理当前组织已到期的失败投递，返回 total、success、failed、skipped。
  - 查询 Webhook 列表、查询投递记录和新事件分发前都会轻量触发一次到期重试补偿。
- 新增 Webhook 测试发送 API：
  - `POST /api/v1/cloud/webhooks/:id/test`：构造 `webhook.test` 测试事件并向目标地址投递。
  - 测试投递使用 `deliveryMode=test`，不写入业务事件中心，且失败不进入自动重试。
- 前端事件中心增强：
  - Webhook 表格新增重试配置列。
  - Webhook 行操作新增“测试”。
  - Webhook 管理区新增“重试到期”按钮。
  - Webhook 新建/编辑弹窗新增最大重试次数和重试间隔字段。

验证：

- 使用 Docker Go 镜像在 `/private/tmp` 临时目录执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-portal` 和 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web`，`iac-portal` healthy，`iac-web` started。
- 鉴权后创建目标地址不可达的临时 Webhook：
  - `maxRetries=2`
  - `retryInterval=1`
  - 首次预算事件投递返回 `failed`。
  - 失败投递 `attempt=1`、`deliveryMode=initial`，并写入 `nextRetryAt`。
- 将临时 Webhook 目标地址更新为 `http://127.0.0.1:9030/api/v1/check` 后等待到期：
  - 调用 `POST /api/v1/cloud/webhooks/retry-due` 返回 `total=1`、`success=1`、`failed=0`、`skipped=0`。
  - 自动重试投递状态为 `success`，`attempt=2`，`deliveryMode=auto`，`parentDeliveryId` 指向原失败投递，响应码为 `200`。
- 鉴权后调用 `POST /api/v1/cloud/webhooks/:id/test` 返回成功：
  - 投递状态为 `success`。
  - `deliveryMode=test`。
  - 响应码为 `200`。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`：
  - 页面展示 Webhook 管理区。
  - 页面展示“重试到期”按钮。
  - 页面展示“测试”操作。
  - 页面展示重试配置列。
  - 浏览器控制台无 error 级日志。

验证限制：

- 当前自动重试已接入 portal 后台 worker，同时保留查询/事件触发的轻量补偿机制；尚未建设独立队列表。
- 当前重试已支持基础间隔、指数退避、重试抖动、最大重试窗口和死信表沉淀；队列可观测摘要已建设。
- 当前页面支持测试发送、批量到期重试、单条失败投递重放、投递详情查看、队列摘要、暂停/恢复、死信重放和签名轮换。

待继续：

- 完善队列保留、清理策略和多分片执行器。
- 将账号验证、同步完成、风险整改和成本建议更多事件接入通知/ITSM/告警渠道。

### 16.39 2026-06-20 V1.2 P1 预算到期评估与通知投递第一阶段

状态：已完成预算评估间隔、到期评估 API、成本中心页面“立即评估”入口和超阈值事件投递联动；后台定时评估 worker 已在 16.66 复用本阶段 API 完成第一阶段。

已完成：

- `iac_cloud_budget` 新增 `evaluationInterval`：
  - 创建和编辑预算时可配置评估间隔。
  - 后端默认 `3600` 秒，最大限制为 `86400` 秒。
- 新增预算到期评估 API：
  - `POST /api/v1/cloud/budgets/evaluate-due`。
  - 参数支持 `period`、`currency` 和 `force`。
  - 返回 total、evaluated、skipped、threshold、exceeded、notification 等统计。
  - `force=false` 时只评估未评估或超过评估间隔的预算。
  - `force=true` 时评估当前账期和币种下所有启用预算。
- 预算评估逻辑增强：
  - 单条预算评估返回当前金额、使用率、是否超阈值、是否超预算和是否推送通知。
  - 预算从未超阈值进入超阈值时写入 `cost.budget.threshold_exceeded` 事件。
  - 事件中心和 Webhook 可继续按既有事件投递逻辑推送预算通知。
- 成本中心页面增强：
  - 预算管理区新增“立即评估”按钮。
  - 预算表新增“评估”列，展示评估间隔和最近评估时间。
  - 新建/编辑预算弹窗新增“评估间隔”字段。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web`，`iac-portal` healthy，`iac-web` started。
- 鉴权后创建临时预算：
  - `POST /api/v1/cloud/budgets` 返回 `200`。
  - `evaluationInterval=60`。
  - `currentAmount=300.85`。
  - 初始 `thresholdReached=false`。
- 通过测试数据模拟预算到期并超阈值后调用：
  - `POST /api/v1/cloud/budgets/evaluate-due` 返回 `200`。
  - `total=5`。
  - `evaluated=1`。
  - `skipped=4`。
  - `notifications=1`。
- 鉴权后查询事件中心：
  - `GET /api/v1/cloud/events?source=cost&eventType=cost.budget.threshold_exceeded&q=<临时预算名>` 返回 `200`。
  - `total=1`。
  - 事件类型为 `cost.budget.threshold_exceeded`。
  - 标题为“成本预算超阈值”。
- 鉴权后删除临时预算：
  - `DELETE /api/v1/cloud/budgets/:id` 返回 `200`。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-costs`：
  - 页面展示“预算管理”。
  - 页面展示“立即评估”按钮。
  - 预算表展示“评估”列。
  - 点击“新建预算”可打开弹窗。
  - 弹窗展示“评估间隔”字段。
  - 浏览器控制台无 error 级日志。

验证限制：

- 当前到期评估由页面按钮、API、查询链路和后台常驻 worker 触发；后台 worker 已在 16.66 完成第一阶段。
- 当前通知通道复用事件中心和 Webhook，尚未接入 ITSM、企业微信、钉钉、Slack 或邮件。
- 当前预算仍为单账期模型，预算版本、审批、汇率、摊销和财务编码仍待后续建设。

待继续：

- 后台 cron/worker 调用 `evaluate-due` 已在 16.66 完成第一阶段；继续增加失败重试、调度审计和 worker 指标。
- 建设预算审批、预算版本、财务编码和专用告警渠道。

### 16.40 2026-06-20 V1.2 P1 云账号健康检查与事件接入第一阶段

状态：已完成云账号健康状态持久化、单账号/批量健康检查 API、云账号页面健康展示和账号事件接入；账号验证和健康检查结果可进入事件中心并被 Webhook 订阅。

已完成：

- `iac_cloud_account` 新增健康字段：
  - `healthStatus`：`healthy`、`warning`、`unhealthy`。
  - `healthMessage`：健康检查说明。
  - `lastHealthCheckedAt`：最近健康检查时间。
- 新增健康检查逻辑：
  - 复用当前账号本地验证结果，检查账号状态、必需凭证字段、区域范围和已支持资产类型。
  - 未配置同步或最近同步超过 24 小时标记为 `warning`。
  - 缺少必需凭证或 provider 不支持标记为 `unhealthy`。
  - 账号禁用标记为 `warning`。
  - 本地检查全部通过且最近同步正常时标记为 `healthy`。
- 新增云账号健康检查 API：
  - `POST /api/v1/cloud/accounts/:id/health-check`：检查单个云账号。
  - `POST /api/v1/cloud/accounts/health-check`：批量检查当前组织云账号。
  - 批量接口返回 total、checked、healthy、warning、unhealthy 和每个账号的健康结果。
- 云账号验证和健康检查接入事件中心：
  - 验证账号写入 `cloud_account.validated`。
  - 健康检查写入 `cloud_account.health_checked`。
  - 事件记录 provider、accountId、cloudAccountId、状态、消息和最近检查时间。
  - 事件中心页面新增账号事件中文类型展示。
- 云账号页面增强：
  - 表格新增“健康”和“最近健康检查”列。
  - 行操作新增“健康”。
  - 工具栏新增“健康检查”批量按钮。
  - 概览区新增“健康异常”统计。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web`，`iac-portal` healthy，`iac-web` started。
- 鉴权后调用 `POST /api/v1/cloud/accounts/health-check` 返回 `200`：
  - `total=1`。
  - `checked=1`。
  - `warning=1`。
  - 首个账号 `healthStatus=warning`。
  - `healthMessage=账号尚未完成云采集同步`。
  - `validationStatus=valid`。
- 鉴权后调用 `GET /api/v1/cloud/accounts?pageSize=5` 返回 `200`：
  - 账号列表包含 `healthStatus=warning`。
  - 账号列表包含 `lastHealthCheckedAt=2026-06-20T17:53:22+08:00`。
- 鉴权后调用 `GET /api/v1/cloud/events?source=account&eventType=cloud_account.health_checked&pageSize=5` 返回 `200`：
  - `total=1`。
  - 事件类型为 `cloud_account.health_checked`。
  - 事件等级为 `warning`。
  - 标题为“云账号健康检查”。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-account`：
  - 页面展示“健康检查”按钮。
  - 页面展示“健康”列。
  - 页面展示“最近健康检查”列。
  - 页面展示“健康异常”统计。
  - 当前账号健康状态展示为“提醒”。
  - 浏览器控制台无 error 级日志。

验证限制：

- 当前健康检查仍以 Portal 本地检查为主；同步任务失败分类已在 16.123 接入账号健康推断，但尚未直接调用各云厂商 API 做主动权限、限流和凭证过期探测。
- 最近同步健康已在 16.111 接入账号启用同步策略阈值；无启用策略时仍保留 24 小时兜底。
- 健康检查事件已进入事件中心和 Webhook 体系，但尚未接入 ITSM 或专用告警渠道。

待继续：

- 在 provider adapter 中接入真实云 API 主动权限验证、限流识别和凭证过期检查。
- 账号健康检查后台调度已在 16.112 完成第一阶段，告警渠道联动仍待继续；同步策略阈值联动已在 16.111 完成第一阶段。

### 16.41 2026-06-20 V1.2 P1 Webhook 投递记录与单条失败重放页面第一阶段

状态：已完成事件中心 Webhook 投递记录页面、失败投递单条重放入口和页面级验证；失败投递可在 Webhook 行内打开记录列表，并从页面直接重试生成新的手动投递记录。

已完成：

- 事件中心 Webhook 表格新增“投递”操作。
- Webhook 投递记录以弹窗展示，避免当前 Drawer 动画兼容问题影响可用性。
- 投递记录列表展示状态、模式、事件类型、尝试次数、响应码、下一次重试、投递时间和错误信息。
- 失败投递行新增“重试”按钮，调用既有 `POST /api/v1/cloud/webhooks/:id/deliveries/:deliveryId/retry` 接口。
- 单条重试完成后自动刷新 Webhook 列表和投递记录列表，最近投递状态、成功/失败计数会同步更新。

验证：

- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-web` 通过，依赖的 `mysql`、`consul`、`iac-portal` 均保持 healthy/running。
- 鉴权后创建目标地址不可达的临时 Webhook：
  - 事件类型为 `cloud_account.health_checked`。
  - 目标地址为 `http://127.0.0.1:65534/unreachable`。
  - 触发云账号健康检查后生成失败投递。
- 鉴权后调用 `GET /api/v1/cloud/webhooks/:id/deliveries?pageSize=5` 返回 `200`：
  - `total=1`。
  - 首条投递 `status=failed`。
  - `deliveryMode=initial`。
  - `eventType=cloud_account.health_checked`。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`：
  - 页面展示临时 Webhook 行和“投递”按钮。
  - 点击“投递”打开“Webhook 投递记录”弹窗。
  - 弹窗展示失败投递和“重试”按钮。
- 将临时 Webhook 目标地址更新为 `http://127.0.0.1:9030/api/v1/check` 后，通过页面点击失败投递的“重试”：
  - Webhook 列表最近投递更新为“成功 200”。
  - 投递记录弹窗新增“成功 / 手动 / 200”记录。
  - API 回读首条投递 `status=success`、`deliveryMode=manual`、`attempt=2`、`responseCode=200`。
- 鉴权后删除临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/:id` 返回 `200`。
- 浏览器控制台无 error 级日志。

验证限制：

- 当前页面展示投递错误摘要、请求载荷、响应体、请求头、响应头和签名校验详情。
- 当前单条重放复用同步接口；到期重试已接入后台 worker，队列摘要、死信表和暂停/恢复已建设。
- 签名密钥已支持发送侧轮换、版本化、旧密钥宽限期和接收端验签结果回传。

待继续：

- 建设事件订阅模板和 ITSM/告警渠道适配。

### 16.42 2026-06-20 V1.2 P1 Webhook 投递详情排障视图第一阶段

状态：已完成 Webhook 投递详情弹窗，支持从投递记录列表查看单条投递的排障字段、请求载荷和响应体。

已完成：

- Webhook 投递记录列表新增“详情”操作。
- 新增“投递详情”弹窗，展示投递 ID、Webhook、状态、模式、事件类型、事件 ID、尝试次数、响应码、目标地址、父投递、下一次重试、已重试时间、投递时间和错误信息。
- 投递详情展示请求载荷，便于排查事件内容、预算/风险/账号等业务字段是否正确。
- 投递详情展示响应体，便于排查外部系统返回内容。
- 失败投递仍保留“重试”操作，成功/失败记录均可查看详情。

验证：

- 使用 Docker Compose 构建 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-web` 通过，依赖的 `mysql`、`consul`、`iac-portal` 均保持 healthy/running。
- 鉴权后调用 `GET /api/v1/cloud/webhooks/:id/deliveries?pageSize=1` 返回 `200`：
  - `total=5`。
  - 首条投递 `status=success`。
  - `deliveryMode=initial`。
  - `responseCode=200`。
  - `eventType=cost.budget.threshold_exceeded`。
  - `requestPayload` 和 `responseBody` 字段存在。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`：
  - 点击 Webhook 行“投递”打开投递记录弹窗。
  - 投递记录列表展示“详情”按钮。
  - 点击“详情”打开“投递详情”弹窗。
  - 弹窗展示投递 ID、目标地址、响应码、请求载荷和响应体区块。
  - 浏览器控制台无 error 级日志。

验证限制：

- 当前详情展示请求载荷、响应体、请求头、响应头和签名校验过程。
- 当前投递详情已支持单条详情 API；响应体仍按投递阶段限制保存前 `4096` 字节。

待继续：

- 完善队列保留、清理策略和多分片执行器。

### 16.43 2026-06-20 V1.2 P1 Webhook 请求/响应头与签名校验详情第一阶段

状态：已完成 Webhook 投递请求头、响应头和签名校验信息留存，并在投递详情弹窗展示；签名值会脱敏保存，不落明文密钥。

已完成：

- `iac_cloud_webhook_delivery` 新增排障字段：
  - `requestHeaders`：保存投递请求头。
  - `responseHeaders`：保存外部系统响应头。
  - `signatureInfo`：保存签名启用状态、算法、scheme、签名头名称和载荷字节数。
- 投递请求头留存：
  - 保存 `Content-Type`、`User-Agent`、`X-CloudIaC-Event`、`X-CloudIaC-Delivery`。
  - 启用 secret 时保存 `X-CloudIaC-Signature`，签名值脱敏为 `sha256=<masked>`。
- 投递响应头留存：
  - 保存外部系统返回的响应头，便于排查内容类型、跨域、长度和网关响应。
  - 响应体仍按既有逻辑限制读取前 `4096` 字节。
- 前端投递详情增强：
  - 新增“请求头”“签名信息”“响应头”区块。
  - 保留请求载荷、响应体和基础排障字段。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web` 通过，`iac-portal` healthy，`iac-web` started。
- 鉴权后创建带 secret 的临时 Webhook 并触发测试发送：
  - `POST /api/v1/cloud/webhooks` 返回 `200`。
  - `POST /api/v1/cloud/webhooks/:id/test` 返回 `200`。
- 鉴权后调用 `GET /api/v1/cloud/webhooks/:id/deliveries?pageSize=1` 返回 `200`：
  - 首条投递 `status=success`。
  - `deliveryMode=test`。
  - `responseCode=200`。
  - `eventType=webhook.test`。
  - `requestHeaders.X-Cloudiac-Signature=sha256=<masked>`。
  - `signatureInfo.enabled=true`。
  - `signatureInfo.algorithm=HMAC-SHA256`。
  - `responseHeaders` 和 `responseBody` 字段存在。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`：
  - 临时 Webhook 展示最近投递“成功 200”。
  - 点击“投递”打开投递记录弹窗。
  - 点击“详情”打开投递详情弹窗。
  - 弹窗展示“请求头”“签名信息”“响应头”和“响应体”区块。
  - 页面可见脱敏签名 `sha256=<masked>` 和算法 `HMAC-SHA256`。
  - 浏览器控制台无 error 级日志。
- 鉴权后删除临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/:id` 返回 `200`。

验证限制：

- 当前签名详情用于发送侧排障，尚未支持外部系统回传签名验证结果。
- 当前详情已支持单条详情 API；响应体仍按投递阶段限制保存前 `4096` 字节。

待继续：

- 完善队列保留、清理策略和多分片执行器。

### 16.44 2026-06-20 V1.2 P1 Webhook 单条投递详情 API 第一阶段

状态：已完成 Webhook 单条投递详情 API，并将事件中心“详情”按钮改为按需读取单条详情。

已完成：

- 新增单条投递详情 API：
  - `GET /api/v1/cloud/webhooks/:id/deliveries/:deliveryId`。
  - 校验当前组织、Webhook ID 和投递 ID，越权或不存在返回 `404`。
  - 返回完整 `CloudWebhookDeliveryResp`，包含 Webhook 名称、请求载荷、请求头、签名信息、响应头、响应体和错误信息。
- 前端服务新增 `cloudEventAPI.webhookDelivery`。
- 事件中心投递记录“详情”按钮改为先请求单条详情 API，再打开“投递详情”弹窗。
- 投递记录列表仍保留轻量展示和失败重试入口。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web` 通过，`iac-portal` healthy，`iac-web` started。
- 鉴权后创建带 secret 的临时 Webhook 并触发测试发送：
  - `POST /api/v1/cloud/webhooks` 返回 `200`。
  - `POST /api/v1/cloud/webhooks/:id/test` 返回 `200`。
- 鉴权后调用 `GET /api/v1/cloud/webhooks/:id/deliveries/:deliveryId` 返回 `200`：
  - `status=success`。
  - `deliveryMode=test`。
  - `responseCode=200`。
  - `eventType=webhook.test`。
  - `webhookName` 为临时 Webhook 名称。
  - `requestHeaders`、`signatureInfo` 字段存在。
  - `requestHeaders.X-Cloudiac-Signature=sha256=<masked>`。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`：
  - 点击临时 Webhook 行“投递”打开投递记录弹窗。
  - 点击“详情”后可打开“投递详情”弹窗。
  - 弹窗展示事件类型、请求载荷、请求头、签名信息和响应头。
  - 浏览器控制台无 error 级日志。
- 鉴权后删除临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/:id` 返回 `200`。

验证限制：

- 当前详情 API 是同步读取数据库记录，已加入投递详情访问审计。
- 当前投递记录列表仍返回完整投递字段，后续可进一步瘦身为列表摘要。

待继续：

- 完善队列保留、清理策略和多分片执行器。

### 16.45 2026-06-20 V1.2 P1 Webhook 指数退避重试第一阶段

状态：已完成 Webhook 失败投递指数退避计算；失败投递会基于 Webhook 的基础重试间隔按尝试次数翻倍，并设置 `nextRetryAt`。

已完成：

- 新增 Webhook 重试延迟计算：
  - 第 1 次失败按 `retryInterval` 秒调度下一次重试。
  - 第 2 次失败按 `retryInterval * 2` 秒调度下一次重试。
  - 后续失败继续按尝试次数翻倍。
  - 单次延迟最大限制为 `3600` 秒，避免异常配置造成过长等待。
- 到期自动重试和手动重试复用同一退避计算。
- 不改变现有 API、页面字段和 Webhook 配置模型，保持兼容。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 鉴权后创建目标地址不可达、`retryInterval=2`、`maxRetries=3` 的临时 Webhook。
- 触发云账号健康检查事件后，首条失败投递通过数据库回读：
  - `attempt=1`。
  - `status=failed`。
  - `deliveryMode=initial`。
  - `delivered_at` 到 `next_retry_at` 的间隔为 `2` 秒。
- 等待到期后调用 `POST /api/v1/cloud/webhooks/retry-due`：
  - 返回 `total=1`。
  - 返回 `failed=1`。
  - 自动重试生成第二条失败投递。
- 第二条失败投递通过数据库回读：
  - `attempt=2`。
  - `status=failed`。
  - `deliveryMode=auto`。
  - `delivered_at` 到 `next_retry_at` 的间隔为 `4` 秒。
- 鉴权后删除临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/:id` 返回 `200`。

验证限制：

- 当前已支持指数退避、重试抖动和最大重试窗口。
- 当前重试抖动按投递 ID 稳定计算，避免同一记录重复查询时重试时间漂移。
- 当前到期重试已接入后台 worker，同时保留查询、事件分发和手动接口触发的补偿能力。

待继续：

- 完善队列保留、清理策略和多分片执行器。

### 16.46 2026-06-20 V1.2 P1 Webhook 失败告警事件第一阶段

状态：已完成 Webhook 最终投递失败事件；当失败投递达到最大重试次数后，会写入事件中心 `webhook.delivery_failed`，用于平台内告警和排障。

已完成：

- 新增内部事件写入能力：
  - `recordCloudEventWithoutDispatch` 支持只写入事件中心、不触发 Webhook 分发。
  - 避免 Webhook 失败告警再次触发 Webhook 导致循环投递。
- Webhook 投递最终失败时写入事件：
  - `eventType=webhook.delivery_failed`。
  - `source=notification`。
  - `level=error`。
  - `status=failed`。
  - `resourceType=webhook`。
  - `resourceId/resourceName` 指向失败的 Webhook。
- 事件 payload 记录 Webhook ID、投递 ID、原始事件 ID、事件类型、投递模式、尝试次数、最大重试次数、目标地址、响应码、错误信息和父投递 ID。
- 事件中心页面新增 `webhook.delivery_failed` 中文展示“Webhook 投递失败”。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web` 通过，`iac-portal` healthy，`iac-web` started。
- 鉴权后创建目标地址不可达、`maxRetries=1` 的临时 Webhook。
- 触发云账号健康检查事件后查询事件中心：
  - `GET /api/v1/cloud/events?source=notification&eventType=webhook.delivery_failed&q=<临时 Webhook 名称>` 返回 `200`。
  - `total=1`。
  - 首条事件 `eventType=webhook.delivery_failed`。
  - `source=notification`。
  - `level=error`。
  - `status=failed`。
  - `title=Webhook 投递失败`。
  - `resourceName` 为临时 Webhook 名称。
  - payload 包含 `deliveryId`、`attempt=1`、`maxRetries=1`、`targetUrl` 和 `errorMessage`。
- 同一临时 Webhook 投递记录回读：
  - `status=failed`。
  - `attempt=1`。
  - `nextRetryAt=0001-01-01T00:00:00Z`，确认达到最大重试后不再排队。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`：
  - 事件列表展示“Webhook 投递失败”。
  - 级别展示“错误”。
  - 来源展示“通知”。
  - 资源名为临时 Webhook 名称。
  - 浏览器控制台无 error 级日志。
- 鉴权后删除临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/:id` 返回 `200`。

验证限制：

- 当前失败告警只写入平台事件中心，暂不再分发给 Webhook，避免循环投递。
- 当前尚未接入企业微信、钉钉、Slack、邮件或 ITSM。

待继续：

- 完善队列保留、清理策略和多分片执行器。

### 16.47 2026-06-20 V1.2 P1 Webhook 重试抖动与最大窗口第一阶段

状态：已完成 Webhook 重试抖动和最大重试窗口；失败投递可按配置错开下一次重试时间，超过最大重试窗口后不再排队，并写入最终失败事件。

已完成：

- `iac_cloud_webhook` 新增重试策略字段：
  - `retryJitterPercent`：重试抖动百分比，范围 `0` 到 `100`，默认 `0` 保持旧行为。
  - `maxRetryDuration`：最大重试窗口秒数，范围 `0` 到 `604800`，默认 `0` 表示不限窗口，保持旧行为。
- 创建和更新 Webhook API 支持新增字段，并在服务端做边界归一化。
- 重试调度增强：
  - 保留原有基础间隔和指数退避。
  - 抖动按投递 ID 稳定计算，避免同一投递重复读取时重试时间漂移。
  - 最大重试窗口以同一 Webhook 和事件的首条投递时间为窗口起点。
  - 计算出的 `nextRetryAt` 超过窗口时不再排队。
- 最终失败事件增强：
  - 窗口过期时写入 `webhook.delivery_failed`。
  - payload 新增 `reason=retry_window_expired` 和 `maxRetryDuration`。
  - 失败消息按“达到最大重试次数”和“超过最大重试窗口”区分。
- 事件中心 Webhook 页面增强：
  - Webhook 列表“重试”列展示最大次数、基础间隔、抖动百分比和重试窗口。
  - 新建/编辑 Webhook 弹窗新增“重试抖动(%)”和“最大重试窗口(秒)”配置项。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web` 通过，`iac-portal` healthy，`iac-web` started。
- 数据库迁移验证：
  - `SHOW COLUMNS FROM iac_cloud_webhook LIKE 'retry_jitter_percent'` 返回字段，默认值为 `0`。
  - `SHOW COLUMNS FROM iac_cloud_webhook LIKE 'max_retry_duration'` 返回字段，默认值为 `0`。
- 鉴权后创建目标地址不可达、`retryInterval=10`、`retryJitterPercent=100`、`maxRetryDuration=0` 的临时 Webhook：
  - `POST /api/v1/cloud/webhooks` 返回 `200`。
  - 返回体 `retryJitterPercent=100`。
  - 返回体 `maxRetryDuration=0`。
- 触发云账号健康检查事件后回读投递记录：
  - `status=failed`。
  - `attempt=1`。
  - 数据库回读 `delivered_at` 到 `next_retry_at` 的间隔为 `20` 秒，处于 `retryInterval=10` 和 `retryJitterPercent=100` 的有效范围内。
- 鉴权后创建目标地址不可达、`retryInterval=5`、`retryJitterPercent=0`、`maxRetryDuration=1` 的临时 Webhook：
  - `POST /api/v1/cloud/webhooks` 返回 `200`。
  - 返回体 `maxRetryDuration=1`。
- 触发云账号健康检查事件后回读投递记录：
  - `status=failed`。
  - `attempt=1`。
  - `nextRetryAt=0001-01-01T00:00:00Z`，确认超过窗口后不再排队。
- 查询事件中心：
  - `GET /api/v1/cloud/events?source=notification&eventType=webhook.delivery_failed&q=<临时 Webhook 名称>` 返回 `200`。
  - `total=1`。
  - payload `reason=retry_window_expired`。
  - payload `maxRetryDuration=1`。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`：
  - Webhook 列表展示“抖动0% / 窗口不限”。
  - 点击“新建 Webhook”打开弹窗。
  - 弹窗展示“重试抖动(%)”和“最大重试窗口(秒)”。
  - 浏览器控制台无 error 级日志。
- 鉴权后删除两组临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/:id` 返回 `200`。

验证限制：

- 当前到期重试已接入后台 worker，同时保留查询、事件分发和手动接口触发的补偿能力。
- 当前最大重试窗口以首条投递记录为起点，死信已落表；仍未拆分独立队列表。

待继续：

- 完善队列保留、清理策略和多分片执行器。

### 16.48 2026-06-20 V1.2 P1 Webhook 投递详情访问审计第一阶段

状态：已完成 Webhook 单条投递详情访问审计；用户读取投递详情时会写入平台用户操作日志，便于追踪排障数据访问行为。

已完成：

- 新增用户操作日志对象类型：
  - `webhook_delivery`。
- 新增用户操作日志动作：
  - `webhook_delivery.view`：查看 Webhook 投递详情。
- `GET /api/v1/cloud/webhooks/:id/deliveries/:deliveryId` 成功读取详情后写入审计日志：
  - `operatorId` 为当前用户。
  - `orgId` 为当前组织。
  - `objectId` 为投递 ID。
  - `objectName` 为 Webhook 名称。
  - `attribute` 记录 Webhook ID、事件 ID、事件类型、投递状态、投递模式、尝试次数、响应码和目标地址。
- 审计日志不保存请求载荷、响应体、请求头或响应头，避免扩大敏感数据落库范围。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 鉴权后调用既有 Webhook 投递详情 API：
  - `GET /api/v1/cloud/webhooks/cwh-d8r5d5ucmfuc738bnang/deliveries/cwd-d8r62c7svpbc73arqv1g` 返回 `200`。
  - 返回体 `status=success`。
  - 返回体 `webhookName=Codex webhook 2026-06 170007`。
- 数据库回读 `iac_user_operation_log`：
  - `object_type=webhook_delivery`。
  - `action=view`。
  - `object_id=cwd-d8r62c7svpbc73arqv1g`。
  - `object_name=Codex webhook 2026-06 170007`。
  - `attribute.webhookId=cwh-d8r5d5ucmfuc738bnang`。
  - `attribute.eventType=cost.budget.threshold_exceeded`。
  - `attribute.status=success`。

验证限制：

- 当前审计写入平台用户操作日志，尚未单独提供 Webhook 投递详情访问审计页面。
- 当前审计不记录请求载荷和响应体内容，后续如需深度取证可设计受控脱敏快照。

待继续：

- 完善队列保留、清理策略和多分片执行器。

### 16.49 2026-06-20 V1.2 P1 Webhook 后台到期重试 worker 第一阶段

状态：已完成 Webhook 后台到期重试 worker；portal 启动后会周期性扫描数据库中到期的失败投递记录，自动生成重试投递，不再只依赖页面查询、事件分发或手动接口触发。

已完成：

- 新增 `RetryDueCloudWebhookDeliveriesForAllOrgs`：
  - 查询所有组织中 `status=failed` 且 `nextRetryAt` 已到期的投递记录。
  - 按组织构造后台服务上下文。
  - 复用既有 `RetryDueCloudWebhookDeliveries` 执行组织内到期重试。
  - 聚合返回 `orgs`、`total`、`success`、`failed` 和 `skipped`。
- 新增 `StartCloudWebhookRetryWorker`：
  - portal 进程内启动后台 ticker。
  - 默认每 `15` 秒扫描一次到期失败投递。
  - 有到期记录时输出聚合执行结果日志。
- portal 启动流程接入 Webhook 后台 worker：
  - 与既有 task manager 一起在 `cmds/portal/main.go` 中启动。
- 保留既有 `POST /api/v1/cloud/webhooks/retry-due` 手动触发接口和查询/事件分发补偿逻辑，避免后台 worker 异常时完全失去补偿路径。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 鉴权后创建目标地址不可达、`retryInterval=1`、`maxRetries=2` 的临时 Webhook：
  - `POST /api/v1/cloud/webhooks` 返回 `200`。
- 触发云账号健康检查事件生成首条失败投递。
- 不调用 `POST /api/v1/cloud/webhooks/retry-due`。
- 等待后台 worker 周期执行后直接查询数据库：
  - 临时 Webhook 投递记录数为 `2`。
  - 最大尝试次数为 `2`。
  - `delivery_mode=initial` 的记录数为 `1`。
  - `delivery_mode=auto` 的记录数为 `1`。
  - 投递链路为 `1:initial:failed,2:auto:failed`。
- 鉴权后删除临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/:id` 返回 `200`。

验证限制：

- 当前 worker 是 portal 进程内轻量扫描器，已通过组织级 MySQL 命名锁避免跨实例重复消费；独立队列表第一阶段已完成，清理策略和分片执行器待完善。
- 当前投递队列仍复用 `iac_cloud_webhook_delivery.next_retry_at`，死信已落表，独立队列表第一阶段已完成，清理策略和分片执行器待完善。
- 当前已提供事件中心队列摘要、暂停/恢复开关和死信重放入口。

待继续：

- 完善队列保留、清理策略和多分片执行器。

### 16.50 2026-06-20 V1.2 P1 Webhook 队列可观测摘要第一阶段

状态：已完成 Webhook 队列可观测 API 和事件中心页面摘要；组织级可查看投递总量、成功/失败、排队、已到期、未来重试、死信、自动重试和已消费投递。

已完成：

- 新增 `CloudWebhookQueueSummaryResp`：
  - 返回 `totalDeliveries`、`pending`、`success`、`failed`。
  - 返回 `queued`、`due`、`future`、`deadLetter`、`retried`。
  - 返回 `initial`、`manual`、`auto`、`test` 投递模式统计。
  - 返回 `oldestQueuedAt` 和 `lastDeliveredAt`，用于定位队列积压和最近投递时间。
- 新增 Webhook 队列摘要服务：
  - `CloudWebhookQueueSummary` 按当前组织统计 `iac_cloud_webhook_delivery`。
  - `queued` 统计失败且 `nextRetryAt` 已排队的投递。
  - `due` 统计失败且已到重试时间的投递。
  - `future` 统计失败且未来才会重试的投递。
  - `deadLetter` 统计 `iac_cloud_webhook_dead_letter` 中 `status=open` 的死信记录。
  - 查询摘要前会同步历史失败投递到死信表，避免旧失败记录不可见。
  - `retried` 统计已被自动重试 worker 或补偿逻辑消费过的失败父投递。
- 新增 Webhook 队列摘要 API：
  - `GET /api/v1/cloud/webhooks/queue/summary`。
  - 路由位于 `/cloud/webhooks/:id` 之前，避免被动态 ID 路由截获。
- 事件中心 Webhook 管理区新增队列摘要：
  - 展示“排队”“已到期”“未来重试”“死信”“已成功”“已失败”“自动重试”“已消费”。
  - 刷新、测试投递、到期重试、单条重试、创建和删除 Webhook 后会刷新摘要。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；前端仅保留既有 vendor/app bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web` 通过，`iac-portal` healthy，`iac-web` started。
- 数据库基准统计当前组织 Webhook 投递：
  - `total=27`。
  - `queued=0`。
  - `due=0`。
  - `future=0`。
  - `deadLetter=7`。
  - `success=12`。
  - `failed=15`。
  - `auto=5`。
  - `retried=8`。
- 鉴权后调用 `GET /api/v1/cloud/webhooks/queue/summary` 返回 `200`：
  - `totalDeliveries=27`。
  - `queued=0`。
  - `due=0`。
  - `future=0`。
  - `deadLetter=7`。
  - `success=12`。
  - `failed=15`。
  - `auto=5`。
  - `retried=8`。
  - `lastDeliveredAt=2026-06-20T18:54:07+08:00`。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`：
  - 页面展示“Webhook”。
  - 页面展示“排队”“已到期”“未来重试”“死信”“已成功”“已失败”“自动重试”“已消费”。
  - 浏览器控制台无 error 级日志。

验证限制：

- 当前队列可观测仍基于 `iac_cloud_webhook_delivery` 汇总投递状态，并已接入 `iac_cloud_webhook_dead_letter` 统计 open 死信；独立队列表第一阶段已完成，清理策略和分片执行器待完善。
- 当前 worker 是 portal 进程内轻量扫描器，已通过组织级 MySQL 命名锁避免跨实例重复消费；独立队列表第一阶段已完成，清理策略和分片执行器待完善。
- 当前已提供 Webhook 暂停/恢复开关、死信重放入口、签名密钥轮换和接收端验签结果回传。

待继续：

- 完善队列保留、清理策略和多分片执行器。

### 16.51 2026-06-20 V1.2 P1 Webhook 死信表、死信重放和暂停恢复第一阶段

状态：已完成 Webhook 死信表、死信列表 API、死信重放/忽略 API、事件中心死信弹窗和 Webhook 暂停/恢复入口；最终失败或无后续重试的投递会沉淀为可处置的死信记录。

已完成：

- 新增模型 `iac_cloud_webhook_dead_letter`：
  - 记录组织、Webhook、投递、事件、事件类型、状态、原因、目标地址、尝试次数、响应码、错误信息和失败载荷。
  - 记录 `replayedAt`、`replayedDeliveryId`、`ignoredAt` 和备注，支持处置闭环。
  - 通过 `org_id + delivery_id` 做唯一约束，避免同一失败投递重复生成死信。
- 新增死信同步和记录逻辑：
  - 最终失败事件写入时同步生成死信。
  - 查询死信和队列摘要前会补偿同步历史失败投递。
  - `max_retries_reached` 表示达到最大重试次数。
  - `retry_window_expired` 表示超过最大重试窗口。
  - `unscheduled_failure` 表示失败后无后续重试计划。
- 新增死信 API：
  - `GET /api/v1/cloud/webhooks/dead-letters`：死信列表，支持状态、Webhook、事件类型和关键字筛选。
  - `POST /api/v1/cloud/webhooks/dead-letters/:deadLetterId/replay`：重放死信并生成新的手动投递记录，成功后标记为 `replayed`。
  - `POST /api/v1/cloud/webhooks/dead-letters/:deadLetterId/ignore`：忽略死信并标记为 `ignored`。
- 增强队列摘要：
  - `deadLetter` 改为统计死信表中的 `open` 记录。
  - 重放或忽略死信后队列摘要自动反映 open 死信数量变化。
- 事件中心页面增强：
  - Webhook 表格新增“暂停/恢复”操作，复用更新接口切换 `enable/disable`。
  - Webhook 管理区新增“死信”入口。
  - 死信弹窗展示状态、Webhook、事件类型、原因、尝试次数、响应码、目标地址、创建时间和重放时间。
  - open 死信支持“重放”和“忽略”，已处置死信不再显示处置按钮。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web` 通过，`iac-portal` healthy，`iac-web` started。
- 数据库迁移验证：
  - `iac_cloud_webhook_dead_letter` 表已创建。
  - 表字段包含 `webhook_id`、`delivery_id`、`event_id`、`status`、`reason`、`replayed_delivery_id`、`ignored_at` 和 `note`。
- 鉴权后验证 Webhook 暂停/恢复：
  - `PUT /api/v1/cloud/webhooks/:id` 设置 `status=disable` 返回 `200`。
  - 再设置 `status=enable` 返回 `200`。
- 鉴权后验证死信生成：
  - 创建目标不可达且 `maxRetries=1` 的临时 Webhook。
  - 触发云账号健康检查事件后生成 open 死信。
  - 死信状态为 `open`，原因为 `max_retries_reached`。
- 鉴权后验证死信重放：
  - 将临时 Webhook 目标地址更新为 `http://iac-portal:9030/api/v1/check`。
  - 调用 `POST /api/v1/cloud/webhooks/dead-letters/cwdl-d8r7e3vc3drs73dj8fj0/replay` 返回 `200`。
  - 返回投递状态为 `success`，响应码为 `200`，投递模式为 `manual`。
  - 原死信状态变为 `replayed`，`replayedDeliveryId=cwd-d8r7frr3v2rs73cnn2dg`。
- 鉴权后验证死信忽略：
  - 调用 `POST /api/v1/cloud/webhooks/dead-letters/cwdl-d8r7dpnc3drs73dj8f9g/ignore` 返回 `200`。
  - 原死信状态变为 `ignored`，备注为 `codex validation ignore`。
- 鉴权后验证队列摘要联动：
  - 处置前 `deadLetter=9`。
  - 重放 1 条并忽略 1 条后 `deadLetter=7`。
  - `totalDeliveries` 从 `29` 增加到 `30`。
  - `success` 从 `12` 增加到 `13`。
  - `manual` 从 `1` 增加到 `2`。
- 鉴权后删除本次验证使用的临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/cwh-d8r7e3vc3drs73dj8fh0` 返回 `200`。
  - `DELETE /api/v1/cloud/webhooks/cwh-d8r7dpnc3drs73dj8f6g` 返回 `200`。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`：
  - 页面展示“Webhook”“死信”“排队”“已到期”“未来重试”“已成功”“已失败”“自动重试”“已消费”。
  - 页面展示 Webhook 行操作“暂停”。
  - 点击“死信”打开“Webhook 死信”弹窗。
  - 弹窗展示“已重放”“待处理”“重放”“忽略”。
  - 浏览器控制台无 error 级日志。

验证限制：

- 当前已新增 `iac_cloud_webhook_delivery_queue` 独立队列表；`next_retry_at` 仍作为兼容和补偿字段保留。
- 当前后台 worker 是 portal 进程内轻量扫描器，已通过组织级 MySQL 命名锁避免跨实例重复消费；独立队列表第一阶段已完成，清理策略和分片执行器待完善。
- 当前死信重放要求原 Webhook 仍存在且处于启用状态。
- 当前已提供签名密钥轮换、版本化、旧密钥宽限期和接收端验签结果回传。

待继续：

- 完善队列保留、清理策略和多分片执行器。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.52 2026-06-20 V1.2 P1 Webhook 签名密钥轮换第一阶段

状态：已完成 Webhook 发送侧签名密钥轮换、密钥版本号、旧密钥宽限期、投递签名版本头和事件中心页面轮换入口；接收端可以根据签名版本和旧密钥过期时间完成平滑切换。

已完成：

- 增强 `iac_cloud_webhook`：
  - 新增 `secret_version`，记录当前签名密钥版本。
  - 新增 `previous_secret`、`previous_secret_version` 和 `previous_secret_expires_at`，记录上一版密钥和宽限期。
  - 新增 `secret_rotated_at`，记录最近一次轮换时间。
- 新增签名轮换 API：
  - `POST /api/v1/cloud/webhooks/:id/secret/rotate`。
  - 请求参数包含 `secret` 和 `gracePeriodSeconds`。
  - 轮换后当前密钥版本递增，上一版密钥进入宽限期。
  - API 响应继续隐藏 `secret` 和 `previousSecret` 明文。
- 兼容既有更新接口：
  - `PUT /api/v1/cloud/webhooks/:id` 显式携带 `secret` 时复用轮换逻辑。
  - 既有页面编辑留空密钥时仍不修改密钥。
- 增强投递签名头：
  - `X-CloudIaC-Signature` 继续保存当前密钥 HMAC，投递记录中脱敏保存。
  - 新增 `X-CloudIaC-Signature-Version`。
  - 旧密钥仍在宽限期时新增 `X-CloudIaC-Previous-Signature-Version`。
  - 旧密钥仍在宽限期时新增 `X-CloudIaC-Previous-Signature-Expires-At`。
- 增强投递详情签名信息：
  - 记录 `version`、`previousVersion`、`previousExpiresAt`、`previousInGrace` 和 `rotatedAt`。
  - 不记录任何密钥明文。
- 事件中心页面增强：
  - Webhook 表格新增“签名”列，展示当前版本和宽限期状态。
  - Webhook 行操作新增“轮换”。
  - 新增“轮换密钥”弹窗，支持填写新签名密钥和旧密钥宽限期。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；前端仅保留既有 bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web` 通过，`iac-portal` healthy，`iac-web` started。
- 数据库迁移验证：
  - `iac_cloud_webhook` 已新增 `secret_version`。
  - `iac_cloud_webhook` 已新增 `previous_secret`、`previous_secret_version`、`previous_secret_expires_at` 和 `secret_rotated_at`。
- 鉴权后创建临时 Webhook：
  - 目标地址为 `http://iac-portal:9030/api/v1/check`。
  - 初始密钥为 `old-secret-codex`。
  - 返回 `secretVersion=1`。
  - 返回 `secret=null`，不泄露密钥明文。
- 鉴权后调用 `POST /api/v1/cloud/webhooks/cwh-d8r7l9bob2rc73cfa07g/secret/rotate`：
  - 新密钥为 `new-secret-codex`。
  - `gracePeriodSeconds=3600`。
  - 返回 `secretVersion=2`。
  - 返回 `previousSecretVersion=1`。
  - 返回 `previousSecret=null`，不泄露上一版密钥明文。
  - 返回 `previousSecretExpiresAt=2026-06-20T20:33:57+08:00`。
- 鉴权后调用测试投递：
  - `POST /api/v1/cloud/webhooks/cwh-d8r7l9bob2rc73cfa07g/test` 返回 `200`。
  - 投递状态为 `success`。
  - 响应码为 `200`。
  - 请求头包含 `X-Cloudiac-Signature=sha256=<masked>`。
  - 请求头包含 `X-Cloudiac-Signature-Version=2`。
  - 请求头包含 `X-Cloudiac-Previous-Signature-Version=1`。
  - 请求头包含 `X-Cloudiac-Previous-Signature-Expires-At=2026-06-20T20:33:57+08:00`。
  - `signatureInfo.version=2`。
  - `signatureInfo.previousVersion=1`。
  - `signatureInfo.previousInGrace=true`。
- 鉴权后读取投递详情：
  - `GET /api/v1/cloud/webhooks/:id/deliveries/:deliveryId` 返回同样的签名版本头和 `signatureInfo`。
- 鉴权后删除本次验证使用的临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/cwh-d8r7l9bob2rc73cfa07g` 返回 `200`。
- 通过应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`：
  - 页面展示 Webhook 表格“签名”列。
  - 页面展示当前签名版本 `v1`。
  - 页面展示行操作“轮换”。
  - 点击“轮换”打开“轮换密钥”弹窗。
  - 弹窗展示“新签名密钥”和“旧密钥宽限期(秒)”字段。
  - 浏览器控制台无 error 级日志。

验证限制：

- 当前已完成发送侧签名轮换和旧密钥宽限期；接收端验签结果回传在 16.53 补齐。
- 当前宽限期只用于给接收端展示和投递头提示，平台不会主动调用接收端校验接口。
- 当前已新增 `iac_cloud_webhook_delivery_queue` 独立队列表；`next_retry_at` 仍作为兼容和补偿字段保留。

待继续：

- 完善队列保留、清理策略和多分片执行器。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.53 2026-06-20 V1.2 P1 Webhook 接收端验签结果回传第一阶段

状态：已完成 Webhook 接收端验签结果回传 API、投递记录验签结果字段和投递详情页面展示；外部接收端完成 HMAC 验签后可把结果回写到平台，形成签名排障闭环。

已完成：

- 增强 `iac_cloud_webhook_delivery`：
  - 新增 `signature_verify_status`，记录 `valid`、`invalid` 或 `skipped`。
  - 新增 `signature_verify_version`，记录接收端验证的签名版本。
  - 新增 `signature_verify_message`，记录接收端回传的说明。
  - 新增 `signature_verified_at`，记录回传时间。
- 新增验签结果回传 API：
  - `POST /api/v1/cloud/webhooks/deliveries/:deliveryId/signature-verification`。
  - 请求参数包含 `status`、`signatureVersion` 和 `message`。
  - 仅更新当前组织内的投递记录。
  - 返回更新后的 `CloudWebhookDeliveryResp`。
- 事件中心投递详情增强：
  - 新增“验签结果”。
  - 新增“验签版本”。
  - 新增“验签时间”。
  - 新增“验签消息”。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；前端仅保留既有 bundle size warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web` 通过，`iac-portal` healthy，`iac-web` started。
- 数据库迁移验证：
  - `iac_cloud_webhook_delivery` 已新增 `signature_verify_status`。
  - `iac_cloud_webhook_delivery` 已新增 `signature_verify_version`。
  - `iac_cloud_webhook_delivery` 已新增 `signature_verify_message`。
  - `iac_cloud_webhook_delivery` 已新增 `signature_verified_at`。
- 鉴权后创建临时 Webhook 并发送测试投递：
  - Webhook ID 为 `cwh-d8r7pqbaours73edsi00`。
  - 投递 ID 为 `cwd-d8r7pqbaours73edsi1g`。
  - 投递状态为 `success`。
  - `signatureInfo.version=1`。
- 鉴权后调用验签结果回传 API：
  - `POST /api/v1/cloud/webhooks/deliveries/cwd-d8r7pqbaours73edsi1g/signature-verification` 返回 `200`。
  - 请求 `status=valid`。
  - 请求 `signatureVersion=1`。
  - 请求 `message=receiver verified hmac`。
  - 返回 `signatureVerifyStatus=valid`。
  - 返回 `signatureVerifyVersion=1`。
  - 返回 `signatureVerifyMessage=receiver verified hmac`。
  - 返回 `signatureVerifiedAt=2026-06-20T19:43:37+08:00`。
- 鉴权后读取投递详情：
  - `GET /api/v1/cloud/webhooks/:id/deliveries/:deliveryId` 返回同样的验签结果字段。
- 鉴权后删除 API 验证使用的临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/cwh-d8r7pqbaours73edsi00` 返回 `200`。
- 通过应用内浏览器验证投递详情页面：
  - 创建页面验证 Webhook `Codex UI signature 20260620194407`。
  - 投递 ID 为 `cwd-d8r7q1raours73edsi70`。
  - 回传 `signatureVerifyStatus=valid`、`signatureVerifyVersion=1`、`signatureVerifyMessage=ui receiver verified hmac`。
  - 打开事件中心 Webhook 行“投递”。
  - 点击投递记录“详情”。
  - 投递详情展示“验签结果”。
  - 投递详情展示“通过”。
  - 投递详情展示“验签版本 1”。
  - 投递详情展示“验签消息 ui receiver verified hmac”。
  - 浏览器控制台无 error 级日志。
- 鉴权后删除页面验证使用的临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/cwh-d8r7q1raours73edsi5g` 返回 `200`。

验证限制：

- 当前已提供平台鉴权回传接口；面向外部接收端的最小权限回传 token 在 16.54 补齐。
- 当前只记录接收端验签结果，不主动发起接收端验签探测。
- 当前已新增 `iac_cloud_webhook_delivery_queue` 独立队列表；`next_retry_at` 仍作为兼容和补偿字段保留。

待继续：

- 完善队列保留、清理策略和多分片执行器。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.54 2026-06-20 V1.2 P1 Webhook 接收端最小权限验签回传 token 第一阶段

状态：已完成面向外部接收端的投递级验签回传 token；接收端无需平台登录 token 和组织头，即可只针对当前投递回写验签结果。

已完成：

- 增强 `iac_cloud_webhook_delivery`：
  - 新增 `signature_report_token_hash`，保存投递级回传 token 的 SHA-256 哈希。
  - API 响应不返回 token 哈希。
- 增强 Webhook 投递请求：
  - 每次投递生成独立随机 token。
  - 请求头新增 `X-CloudIaC-Signature-Report-Token`。
  - 投递详情保存的请求头会对回传 token 脱敏。
  - `signatureInfo` 新增 `reportTokenHeader` 和 `reportEndpoint`，便于接收端按投递 ID 回传验签结果。
- 新增公开回传 API：
  - `POST /api/v1/cloud/webhooks/deliveries/:deliveryId/signature-verification/report`。
  - 支持通过 `X-CloudIaC-Signature-Report-Token` 请求头或 `token` 参数传入回传 token。
  - 请求参数包含 `status`、`signatureVersion` 和 `message`。
  - 仅在投递 ID 和 token 哈希匹配时更新验签结果。
  - 返回最小响应，只包含投递 ID、验签状态、验签版本、验签消息和验签时间。
- 增强 CORS：
  - 允许 `X-CloudIaC-Signature-Report-Token` 请求头。
  - 保留既有平台鉴权回传接口，供平台内调试和运维使用。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 数据库迁移验证：
  - `iac_cloud_webhook_delivery` 已新增 `signature_report_token_hash`。
  - 新投递记录的 `signature_report_token_hash` 长度为 64，仅保存 SHA-256 十六进制哈希。
- 启动临时 HTTP 接收端并创建临时 Webhook：
  - Webhook ID 为 `cwh-d8r7v8gvhikc73cn4ic0`。
  - 目标地址为 `http://host.docker.internal:18081/webhook`。
  - 测试投递 ID 为 `cwd-d8r7v8gvhikc73cn4idg`。
  - 投递状态为 `success`，响应码为 `200`。
  - 接收端收到 `X-CloudIaC-Signature-Report-Token`、`X-CloudIaC-Delivery`、`X-CloudIaC-Signature` 和 `X-CloudIaC-Signature-Version`。
- 使用错误回传 token 调用公开 API：
  - `POST /api/v1/cloud/webhooks/deliveries/cwd-d8r7v8gvhikc73cn4idg/signature-verification/report` 返回 HTTP `403`。
- 使用正确回传 token 调用公开 API：
  - `POST /api/v1/cloud/webhooks/deliveries/cwd-d8r7v8gvhikc73cn4idg/signature-verification/report` 返回 `200`。
  - 请求 `status=valid`。
  - 请求 `signatureVersion=1`。
  - 请求 `message=public receiver verified hmac`。
  - 返回最小响应 `signatureVerifyStatus=valid`、`signatureVerifyVersion=1`。
- 鉴权后读取投递详情：
  - `signatureVerifyStatus=valid`。
  - `signatureVerifyVersion=1`。
  - `signatureVerifyMessage=public receiver verified hmac`。
  - `requestHeaders.X-Cloudiac-Signature-Report-Token=<masked>`。
  - `signatureInfo.reportTokenHeader=X-CloudIaC-Signature-Report-Token`。
  - `signatureInfo.reportEndpoint=/api/v1/cloud/webhooks/deliveries/cwd-d8r7v8gvhikc73cn4idg/signature-verification/report`。
- 鉴权后删除本次验证使用的临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/cwh-d8r7v8gvhikc73cn4ic0` 返回 `200`。

验证限制：

- 当前已提供投递级回传 token；token 过期、单次使用和审计事件在 16.55 补齐。
- 当前只记录接收端验签结果，不主动发起接收端验签探测。
- 当前已新增 `iac_cloud_webhook_delivery_queue` 独立队列表；`next_retry_at` 仍作为兼容和补偿字段保留。

待继续：

- 完善队列保留、清理策略和多分片执行器。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.55 2026-06-20 V1.2 P1 Webhook 验签回传 token 过期、单次使用和审计第一阶段

状态：已完成 Webhook 验签回传 token 的过期控制、单次使用保护和审计事件；外部接收端只能在有效期内成功回写一次验签结果。

已完成：

- 增强 `iac_cloud_webhook_delivery`：
  - 新增 `signature_report_token_expires_at`，记录回传 token 过期时间。
  - 新增 `signature_report_token_used_at`，记录回传 token 首次成功使用时间。
  - API 响应不直接暴露 token 哈希、过期控制字段和使用时间字段。
- 增强 Webhook 投递请求：
  - 每次投递生成的回传 token 默认 7 天有效。
  - `signatureInfo.reportTokenExpiresAt` 返回回传 token 过期时间。
- 增强公开回传 API：
  - 回传 token 过期时返回 HTTP `403`。
  - 回传 token 已使用时返回 HTTP `409`。
  - 成功回传时通过数据库条件更新原子写入 `signature_report_token_used_at`，避免并发重复使用。
- 新增回传审计事件：
  - 成功回传后记录 `webhook.signature_verification_reported` 事件。
  - 事件来源为 `notification`。
  - `signatureReportSource=token` 表示来自外部最小权限 token。
  - 审计事件不再触发 Webhook 分发，避免回传事件递归触发外部接收端。
- 保留平台鉴权回传接口：
  - 平台内回传同样记录审计事件。
  - `signatureReportSource=platform` 表示来自平台鉴权接口。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 数据库迁移验证：
  - `iac_cloud_webhook_delivery` 已新增 `signature_report_token_expires_at`。
  - `iac_cloud_webhook_delivery` 已新增 `signature_report_token_used_at`。
- 启动临时 HTTP 接收端并创建临时 Webhook：
  - Webhook ID 为 `cwh-d8r8190gnqps73a8d0v0`。
  - 目标地址为 `http://host.docker.internal:18081/webhook`。
  - 测试投递 ID 为 `cwd-d8r8190gnqps73a8d10g`。
  - 投递状态为 `success`，响应码为 `200`。
  - `signatureInfo.reportTokenExpiresAt=2026-06-27T19:59:32.609912051+08:00`。
- 使用正确回传 token 第一次调用公开 API：
  - `POST /api/v1/cloud/webhooks/deliveries/cwd-d8r8190gnqps73a8d10g/signature-verification/report` 返回 `200`。
  - 请求 `status=valid`。
  - 请求 `signatureVersion=1`。
  - 请求 `message=single use receiver verified hmac`。
  - 返回 `signatureVerifyStatus=valid`。
- 使用同一回传 token 第二次调用公开 API：
  - 第二次请求返回 HTTP `409`。
- 鉴权后读取投递详情：
  - `signatureVerifyStatus=valid`。
  - `signatureVerifyMessage=single use receiver verified hmac`。
  - `signatureInfo.reportTokenExpiresAt=2026-06-27T19:59:32.609912051+08:00`。
  - `requestHeaders.X-Cloudiac-Signature-Report-Token=<masked>`。
- 数据库状态验证：
  - 新投递记录的 `signature_report_token_hash` 长度为 64。
  - 新投递记录的 `signature_report_token_expires_at` 非空。
  - 新投递记录的 `signature_report_token_used_at` 非空。
- 审计事件验证：
  - `iac_cloud_event` 已生成 `webhook.signature_verification_reported`。
  - 事件 `level=info`。
  - 事件 `status=valid`。
  - 事件 `resource_id=cwd-d8r8190gnqps73a8d10g`。
  - 事件载荷 `signatureReportSource=token`。
- 鉴权后删除本次验证使用的临时 Webhook：
  - `DELETE /api/v1/cloud/webhooks/cwh-d8r8190gnqps73a8d0v0` 返回 `200`。

验证限制：

- 当前只记录接收端验签结果，不主动发起接收端验签探测。
- 当前已新增 `iac_cloud_webhook_delivery_queue` 独立队列表；`next_retry_at` 仍作为兼容和补偿字段保留。

待继续：

- 完善队列保留、清理策略和多分片执行器。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.56 2026-06-20 V1.2 P1 Webhook 到期重试组织级分布式锁第一阶段

状态：已完成 Webhook 到期重试的组织级 MySQL 命名锁；后台 worker、页面查询补偿、事件触发补偿和手动 `retry-due` 接口共用同一把组织级锁，避免多实例同时消费同一组织的到期投递。

已完成：

- 新增组织级锁名称：
  - `cloudiac:webhook_retry:<orgId>`。
  - 每个组织独立加锁，避免不同组织之间相互阻塞。
- 增强 `RetryDueCloudWebhookDeliveries`：
  - 执行到期重试扫描前先调用 MySQL `GET_LOCK(lockName, 0)`。
  - 未拿到锁时立即返回，不阻塞请求线程。
  - 成功拿锁后再查询并消费到期投递。
  - 执行结束后调用 MySQL `RELEASE_LOCK(lockName)`。
- 锁连接控制：
  - 使用独立事务持有命名锁，保证 `GET_LOCK` 和 `RELEASE_LOCK` 使用同一条数据库连接。
  - 释放失败或事务提交失败会记录 warning 日志。
- API 可观测增强：
  - `POST /api/v1/cloud/webhooks/retry-due` 返回 `locked`。
  - `POST /api/v1/cloud/webhooks/retry-due` 返回 `lockSkipped`。
  - 正常拿锁时 `locked=true`、`lockSkipped=false`。
  - 锁被占用时 `locked=false`、`lockSkipped=true`。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 正常路径验证：
  - 鉴权后调用 `POST /api/v1/cloud/webhooks/retry-due` 返回 `200`。
  - 返回 `locked=true`。
  - 返回 `lockSkipped=false`。
  - 返回 `total=0`、`success=0`、`failed=0`、`skipped=0`。
- 锁占用路径验证：
  - 使用独立 MySQL 连接执行 `GET_LOCK('cloudiac:webhook_retry:org-d8qk6fsd6t1s73fu2kr0', 0)` 并保持连接 `sleep(20)`。
  - 锁占用期间鉴权后调用 `POST /api/v1/cloud/webhooks/retry-due` 返回 `200`。
  - 返回 `locked=false`。
  - 返回 `lockSkipped=true`。
  - 返回 `total=0`、`success=0`、`failed=0`、`skipped=0`。
  - 占锁连接结束后 MySQL 返回 `locked=1`、`sleep(20)=0`，锁自然释放。

验证限制：

- 当前已新增 `iac_cloud_webhook_delivery_queue` 独立队列表；`next_retry_at` 仍作为兼容和补偿字段保留。
- 当前锁粒度为组织级，尚未按 Webhook 或分片做更细粒度并行消费。

待继续：

- 完善队列保留、清理策略和多分片执行器。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.57 2026-06-20 V1.2 P1 Webhook 独立投递队列表第一阶段

状态：已完成 Webhook 到期重试独立队列表第一阶段；失败投递会进入 `iac_cloud_webhook_delivery_queue`，队列摘要和到期重试消费均从独立队列表读取，`next_retry_at` 作为兼容和补偿字段保留。

已完成：

- 新增模型 `iac_cloud_webhook_delivery_queue`：
  - 记录组织、Webhook、原始投递、事件、事件类型、队列状态、尝试次数和下次运行时间。
  - 记录 `lockedAt`、`lockedBy`、`consumedAt`、`resultDeliveryId` 和错误摘要。
  - 通过 `org_id + delivery_id` 做唯一约束，避免同一失败投递重复入队。
  - 状态支持 `queued`、`processing`、`done` 和 `skipped`。
- 失败投递入队：
  - Webhook 投递失败且存在下一次重试时间时立即写入队列表。
  - `delivery_mode=test` 的测试投递不入队。
  - 保留 `iac_cloud_webhook_delivery.next_retry_at` 作为兼容字段。
- 队列补偿同步：
  - 查询队列摘要和执行到期重试前，会把历史 `next_retry_at` 失败投递补偿同步到队列表。
  - 已完成或已消费的队列项不会重复回到 `queued`。
- 到期重试消费：
  - `RetryDueCloudWebhookDeliveries` 改为从 `iac_cloud_webhook_delivery_queue` 读取 due 队列项。
  - 消费前把队列项从 `queued` 标记为 `processing`，并记录 `lockedAt` 和 `lockedBy`。
  - 消费完成后标记为 `done`，写入 `resultDeliveryId` 和错误摘要。
  - 原始失败投递仍会写入 `retried_at` 并清空 `next_retry_at`。
- 队列摘要增强：
  - `queued`、`due`、`future` 和 `oldestQueuedAt` 改为基于独立队列表统计。
  - 投递总数、成功数、失败数、投递模式统计仍基于投递表统计。
- 后台 worker 兼容：
  - 全组织后台扫描会同时查看 due 队列表和旧投递表 `next_retry_at`，避免历史数据漏处理。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次新增和修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 数据库迁移验证：
  - `iac_cloud_webhook_delivery_queue` 表已创建。
  - 表字段包含 `delivery_id`、`status`、`next_run_at` 和 `result_delivery_id`。
- 失败投递入队验证：
  - 创建临时 Webhook `cwh-d8r85h9s580c739rv1r0`，目标地址为 `http://host.docker.internal:19999/webhook`。
  - 触发 `cloud_account.health_checked` 事件后生成失败投递 `cwd-d8r85h9s580c739rv1sg`。
  - 队列记录 `cwq-d8r85h9s580c739rv1t0` 状态为 `queued`。
  - 队列记录 `attempt=1`。
  - 关联投递状态为 `failed`，且 `next_retry_at` 非空。
- 后台 worker 消费验证：
  - 队列记录最终变为 `done`。
  - 队列记录写入结果投递 `cwd-d8r85ips580c739rv1u0`。
  - 结果投递状态为 `failed`、`delivery_mode=auto`、`attempt=2`。
  - 原始失败投递 `retried_at` 非空。
- 手动 retry-due 消费验证：
  - 创建临时 Webhook `cwh-d8r85p9s580c739rv220`，目标地址为 `http://host.docker.internal:19998/webhook`。
  - 触发 `cloud_account.health_checked` 后立即调用 `POST /api/v1/cloud/webhooks/retry-due`。
  - 返回 `locked=true`、`lockSkipped=false`。
  - 返回 `total=2`、`failed=2`、`success=0`、`skipped=0`。
  - 队列记录状态为 `done`。
  - 队列记录写入结果投递 `cwd-d8r85phs580c739rv27g`。
  - 原始失败投递 `retried_at` 非空。
- 队列摘要验证：
  - 鉴权后调用 `GET /api/v1/cloud/webhooks/queue/summary` 返回 `200`。
  - 返回 `queued=0`。
  - 返回 `due=0`。
  - 返回 `future=0`。
  - 返回 `totalDeliveries=41`。
  - 返回 `failed=23`。
  - 返回 `deadLetter=10`。
- 清理验证：
  - `DELETE /api/v1/cloud/webhooks/cwh-d8r85h9s580c739rv1r0` 返回 `200`。
  - `DELETE /api/v1/cloud/webhooks/cwh-d8r85p9s580c739rv220` 返回 `200`。

验证限制：

- 队列保留清理第一阶段已在 16.58 补齐，已完成/跳过的队列项会按 7 天保留期软删除。
- 队列分片消费第一阶段已在 16.59 补齐，默认 retry-due 会按 4 分片聚合消费。

待继续：

- 继续推进 worker 内部真正并发执行、分片指标和后台主动清理。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.58 2026-06-20 V1.2 P1 Webhook 投递队列保留清理第一阶段

状态：已完成 Webhook 独立投递队列的基础保留清理能力，避免已消费队列无限增长。

已完成：

- 新增队列保留周期常量，当前已完成/跳过队列项默认保留 7 天。
- `CloudWebhookQueueSummary` 查询队列摘要前会先清理当前组织过期队列项。
- `RetryDueCloudWebhookDeliveries` 执行到期重试前会先清理当前组织过期队列项。
- `POST /api/v1/cloud/webhooks/retry-due` 返回新增 `queueCleaned` 字段，便于观察本次清理数量。
- 全组织 retry worker 汇总 `queueCleaned`，后续日志和 API 可继续沿用该计数。
- 清理采用模型软删除，保留数据库审计可回溯能力；投递记录和死信记录不被删除。

验证：

- 使用 Docker Compose 重新启动 `iac-portal` 通过，`iac-portal` healthy。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 数据库验证 `iac_cloud_webhook_delivery_queue` 表使用通用软删除字段 `deleted_at_t`。
- 选择已完成队列 `cwq-d8r85p9s580c739rv240`，将 `consumed_at` 回拨到 8 天前且 `deleted_at_t=0`。
- 鉴权后调用 `POST /api/v1/cloud/webhooks/retry-due` 返回 `200`。
- 返回 `locked=true`、`lockSkipped=false`。
- 返回 `queueCleaned=1`。
- 回读队列 `cwq-d8r85p9s580c739rv240`，`deleted_at_t` 已写入非零软删除时间戳。

验证限制：

- 当前清理触发点为队列摘要、手动 retry-due 和有 due 工作的后台 worker；没有 due 工作且不访问摘要的组织不会被后台主动周期清理。
- 队列分片消费第一阶段已在 16.59 补齐，默认 retry-due 会按 4 分片聚合消费。

待继续：

- 继续推进 worker 内部真正并发执行、分片指标和后台主动清理。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.59 2026-06-20 V1.2 P1 Webhook 投递队列分片消费第一阶段

状态：已完成 Webhook 到期重试队列的分片消费第一阶段；默认 retry-due 路径会按 4 个分片聚合执行，显式分片参数可只消费单个分片。

已完成：

- 新增 retry-due 分片参数：
  - `shardIndex`
  - `shardTotal`
- `POST /api/v1/cloud/webhooks/retry-due` 不传分片参数时默认按 4 分片聚合执行，保持原入口兼容。
- 显式提交 `shardIndex/shardTotal` 时只消费对应分片，并返回 `shardIndex`、`shardTotal`。
- 分片消费使用分片级 MySQL 命名锁：
  - `cloudiac:webhook_retry:<orgId>:shard:<shardTotal>:<shardIndex>`。
  - 不同分片可独立拿锁，避免整组织互斥。
- 队列查询使用稳定的 `md5(id)` 分片表达式过滤，避免在 Go 层先取大批数据再过滤。
- 默认聚合结果新增：
  - `shards`
  - `lockSkippedCount`
- 全组织后台 retry worker 复用默认 4 分片聚合逻辑，并汇总 `lockSkippedCount`。
- 参数校验：
  - `shardIndex` 和 `shardTotal` 不能为负数。
  - `shardIndex` 必须小于 `shardTotal`。
  - `shardTotal` 最大为 128。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 默认路径验证：
  - 鉴权后调用 `POST /api/v1/cloud/webhooks/retry-due` 返回 `200`。
  - 返回 `shards=4`。
  - 返回 `lockSkipped=false`、`lockSkippedCount=0`。
  - 返回 `total=0`、`success=0`、`failed=0`、`skipped=0`。
- 显式分片验证：
  - 鉴权后提交 `shardIndex=1`、`shardTotal=4` 调用 `POST /api/v1/cloud/webhooks/retry-due` 返回 `200`。
  - 返回 `shardIndex=1`、`shardTotal=4`。
  - 返回 `locked=true`、`lockSkipped=false`。
- 参数校验验证：
  - 提交 `shardIndex=4`、`shardTotal=4` 返回 HTTP `400`。
  - 返回错误详情 `shardIndex must be less than shardTotal`。
- 分片锁验证：
  - 使用独立 MySQL 连接占用 `cloudiac:webhook_retry:org-d8qk6fsd6t1s73fu2kr0:shard:4:0`。
  - 锁占用期间提交 `shardIndex=0`、`shardTotal=4` 调用 retry-due 返回 `200`。
  - 返回 `locked=false`、`lockSkipped=true`。
  - 返回 `shardIndex=0`、`shardTotal=4`。

验证限制：

- worker 内部并发 fan-out 第一阶段已在 16.60 补齐，默认 4 分片会并发执行并聚合结果。
- 当前清理触发点仍依赖队列摘要、retry-due 和有 due 工作的后台 worker。

待继续：

- 继续推进分片指标、后台主动清理和可配置分片数。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.60 2026-06-20 V1.2 P1 Webhook 投递队列分片并发 fan-out 第一阶段

状态：已完成默认 4 分片 retry-due 的 worker 内部并发 fan-out；单个 portal 进程内会并发消费各分片并聚合结果。

已完成：

- 默认 `RetryDueCloudWebhookDeliveryShards` 从顺序遍历分片升级为 goroutine fan-out。
- 每个分片使用独立 `ServiceContext`，避免多个 goroutine 共享同一个缓存 DB session。
- 队列维护逻辑前置：
  - 补偿同步历史 `next_retry_at` 投递。
  - 清理过期已完成/跳过队列项。
  - 只在 fan-out 前执行一次。
- 新增 maintenance 命名锁：
  - `cloudiac:webhook_retry:<orgId>:maintenance`。
  - 避免多个 portal 实例同时执行历史投递补偿同步和队列清理。
- 分片消费 goroutine 只负责获取对应分片锁、查询分片队列和执行重试，不重复执行队列维护。
- 默认聚合结果继续返回 `shards`、`lockSkippedCount`、`queueCleaned`、`total`、`success`、`failed`、`skipped`。
- 显式单分片调用仍保持同步执行，适合外部调度器按分片独立调用。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 默认 fan-out 验证：
  - 鉴权后调用 `POST /api/v1/cloud/webhooks/retry-due` 返回 `200`。
  - 返回 `shards=4`。
  - 返回 `lockSkipped=false`、`lockSkippedCount=0`。
  - 返回 `total=0`、`success=0`、`failed=0`、`skipped=0`。
- 显式单分片验证：
  - 提交 `shardIndex=2`、`shardTotal=4` 调用 retry-due 返回 `200`。
  - 返回 `shardIndex=2`、`shardTotal=4`。
  - 返回 `locked=true`、`lockSkipped=false`。
- 并发分片锁聚合验证：
  - 使用独立 MySQL 连接占用 `cloudiac:webhook_retry:org-d8qk6fsd6t1s73fu2kr0:shard:4:3`。
  - 锁占用期间调用默认 retry-due 返回 `200`。
  - 返回 `lockSkipped=true`。
  - 返回 `lockSkippedCount=1`。
  - 返回 `shards=4`。

验证限制：

- 当前分片数固定为 4，尚未做系统配置化。
- 当前清理触发点仍依赖队列摘要、retry-due 和有 due 工作的后台 worker。

待继续：

- 分片指标第一阶段已在 16.61 补齐；继续推进后台主动清理和可配置分片数。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.61 2026-06-20 V1.2 P1 Webhook 投递队列分片可观测第一阶段

状态：已完成 Webhook 队列摘要的分片可观测字段，外部调度器和页面可按分片查看 queued/due/future 分布。

已完成：

- 增强 `CloudWebhookQueueSummaryResp`：
  - 新增 `shardTotal`。
  - 新增 `shards`。
- 新增 `CloudWebhookQueueShardSummaryResp`：
  - `shardIndex`
  - `shardTotal`
  - `queued`
  - `due`
  - `future`
  - `oldestQueuedAt`
- `GET /api/v1/cloud/webhooks/queue/summary` 继续保留原有总览字段，新增字段向后兼容。
- 分片摘要复用 retry-due 的稳定 `md5(id)` 分片表达式，保证摘要口径和消费口径一致。
- 当前默认返回 4 个分片，对应默认 retry-due fan-out 分片数。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 鉴权后调用 `GET /api/v1/cloud/webhooks/queue/summary` 返回 `200`。
- 返回原有总览字段：
  - `totalDeliveries=41`
  - `success=18`
  - `failed=23`
  - `deadLetter=10`
  - `queued=0`
  - `due=0`
  - `future=0`
- 返回新增字段：
  - `shardTotal=4`
  - `shards` 长度为 4。
  - 分片索引覆盖 `0`、`1`、`2`、`3`。
  - 每个分片包含 `queued`、`due`、`future` 和 `oldestQueuedAt`。

验证限制：

- 当前分片数固定为 4，尚未做系统配置化。
- 后台主动清理已在 16.62 补齐，过期已消费队列可由 worker 周期清理。
- 当前摘要只返回队列分布，不包含每个分片最近消费耗时和成功率。

待继续：

- 继续推进可配置分片数和分片执行指标。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.62 2026-06-20 V1.2 P1 Webhook 投递队列后台主动清理第一阶段

状态：已完成 Webhook 过期已消费队列的后台主动清理；即使组织没有 due 投递，worker 也能发现并清理过期 done/skipped 队列项。

已完成：

- `RetryDueCloudWebhookDeliveriesForAllOrgs` 新增过期队列组织来源：
  - 查询 `done` 和 `skipped` 队列。
  - `consumed_at` 超过 7 天保留期。
  - `consumed_at` 大于最小有效时间。
- 全组织 worker 现在会合并三类组织：
  - due 队列表组织。
  - 旧投递表 `next_retry_at` 兼容组织。
  - 过期已消费队列组织。
- worker 默认复用 4 分片 fan-out 执行清理和消费。
- worker 日志增强：
  - `total > 0` 时记录。
  - `queueCleaned > 0` 时记录。
  - `lockSkippedCount > 0` 时记录。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 选择已完成队列 `cwq-d8r85p9s580c739rv250`。
- 将 `consumed_at` 回拨到 8 天前，并确认 `deleted_at_t=0`。
- 不调用 queue summary，也不手动调用 retry-due，等待后台 worker 一个周期。
- 回读 `cwq-d8r85p9s580c739rv250`：
  - `deleted_at_t` 已写入非零软删除时间戳。
- `iac-portal` 日志出现：
  - `retry due cloud webhooks result`
  - `orgs:1`
  - `queueCleaned:1`
  - `shards:4`

验证限制：

- 默认分片数配置化已在 16.63 补齐，未配置时保持 4。
- 当前后台清理复用 retry worker 周期，尚未拆成独立队列维护 worker。

待继续：

- 继续推进分片执行指标和独立队列维护 worker。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.63 2026-06-20 V1.2 P1 Webhook retry-due 默认分片数配置化第一阶段

状态：已完成 Webhook retry-due 默认分片数配置化；默认仍为 4，部署可通过环境变量调整。

已完成：

- 新增环境变量 `CLOUDIAC_WEBHOOK_RETRY_SHARDS`。
- 默认分片数读取规则：
  - 未配置时使用默认值 `4`。
  - 非法值、空值或小于等于 0 时回退到 `4`。
  - 大于最大值时截断到 `128`。
- retry-due 默认聚合路径使用配置化分片数。
- queue summary 的 `shardTotal` 和 `shards` 使用同一配置化分片数。
- 全组织后台 worker 汇总结果使用同一配置化分片数。
- 显式 `shardIndex/shardTotal` 调用继续保留，外部调度器可覆盖默认分片数。
- `shardTotal` 参数最大值统一使用 `128`。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 未配置 `CLOUDIAC_WEBHOOK_RETRY_SHARDS` 时，默认 retry-due 返回 `shards=4`。
- 显式提交 `shardIndex=7`、`shardTotal=8` 调用 retry-due 返回 `200`。
- 显式分片返回 `shardIndex=7`、`shardTotal=8`。
- queue summary 返回 `shardTotal=4`，与未配置环境变量时的默认值一致。

验证限制：

- 当前仅支持通过环境变量配置默认分片数，尚未接入系统配置页面。
- retry-due 分片执行明细已在 16.64 补齐，默认响应会返回 `shardResults`。

待继续：

- 继续推进独立队列维护 worker 和分片耗时/成功率指标。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.64 2026-06-20 V1.2 P1 Webhook retry-due 分片执行明细第一阶段

状态：已完成 retry-due 默认聚合响应的分片执行明细；调用方可以直接看到每个分片的锁状态、消费数量和成功/失败结果。

已完成：

- 默认 `POST /api/v1/cloud/webhooks/retry-due` 响应新增 `shardResults`。
- `shardResults` 按 `shardIndex` 稳定排序。
- 每个分片结果包含：
  - `shardIndex`
  - `shardTotal`
  - `locked`
  - `lockSkipped`
  - `queueCleaned`
  - `total`
  - `success`
  - `failed`
  - `skipped`
- 聚合结果继续保留原有字段：
  - `shards`
  - `lockSkippedCount`
  - `queueCleaned`
  - `total`
  - `success`
  - `failed`
  - `skipped`

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 鉴权后调用默认 `POST /api/v1/cloud/webhooks/retry-due` 返回 `200`。
- 返回 `shards=4`。
- 返回 `shardResults` 长度为 4。
- `shardResults` 覆盖 `shardIndex=0`、`1`、`2`、`3`。
- 每个分片返回 `locked=true`、`lockSkipped=false`、`total=0`、`success=0`、`failed=0`、`skipped=0`。

验证限制：

- 当前分片执行明细不包含耗时、最近消费时间和成功率。
- 独立队列维护 worker 已在 16.65 补齐，过期队列清理由 `cloudWebhookQueueMaintenance` worker 负责。

待继续：

- 继续推进分片耗时、最近消费时间和成功率指标。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.65 2026-06-20 V1.2 P1 Webhook 独立队列维护 worker 第一阶段

状态：已完成 Webhook 独立队列维护 worker；过期 done/skipped 队列项由专门 worker 周期清理，不再依赖 retry worker 扫描纯清理组织。

已完成：

- 新增 `StartCloudWebhookQueueMaintenanceWorker`。
- portal 启动时新增后台 goroutine：
  - `go apps.StartCloudWebhookQueueMaintenanceWorker(configs.Get().Consul.ServiceID)`。
- 新增 `CleanupExpiredCloudWebhookDeliveryQueuesForAllOrgs`：
  - 只扫描过期 `done` / `skipped` 队列项。
  - 按组织聚合。
  - 复用 maintenance 命名锁和 `prepareCloudWebhookDeliveryQueues`。
- `RetryDueCloudWebhookDeliveriesForAllOrgs` 回归只扫描 retry 相关组织：
  - due 队列表组织。
  - 旧投递表 `next_retry_at` 兼容组织。
- queue maintenance worker 独立日志：
  - worker 名称为 `cloudWebhookQueueMaintenance`。
  - `queueCleaned > 0` 时输出清理结果。
- 默认维护周期当前为 15 秒，便于与 retry worker 保持同一轻量后台节奏。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 选择已完成队列 `cwq-d8r85h9s580c739rv1t0`。
- 将 `consumed_at` 回拨到 8 天前，并确认 `deleted_at_t=0`。
- 不调用 queue summary，也不手动调用 retry-due，等待独立维护 worker 一个周期。
- 回读 `cwq-d8r85h9s580c739rv1t0`：
  - `deleted_at_t` 已写入非零软删除时间戳。
- `iac-portal` 日志出现：
  - `worker=cloudWebhookQueueMaintenance`
  - `cleanup expired cloud webhook delivery queues result`
  - `orgs:1`
  - `queueCleaned:1`

验证限制：

- 当前维护周期固定为 15 秒，尚未接入系统配置。
- 当前分片执行明细不包含耗时、最近消费时间和成功率。

待继续：

- 继续推进分片耗时、最近消费时间和成功率指标。
- 将云账号验证、同步任务完成、更多成本异常和风险整改事件接入 ITSM/企业微信/钉钉/Slack/邮件。

### 16.66 2026-06-20 V1.2 P1 成本预算后台评估 worker 第一阶段

状态：已完成成本预算后台定时评估 worker；预算超阈值事件不再只依赖成本中心页面查询或手动 evaluate-due 接口触发。

已完成：

- 新增 `EvaluateDueCloudBudgetsForAllOrgs`：
  - 扫描所有启用预算。
  - 按 `orgId + period + currency` 聚合 due 预算。
  - 支持历史账期和多币种预算，不固定只处理当前月 `CNY`。
- 新增预算评估命名锁：
  - 锁名格式为 `cloudiac:budget_evaluation:<orgId>:<period>:<currency>`。
  - 多 Portal 实例并行时，同一组织/账期/币种只会有一个 worker 执行评估。
- 新增 `StartCloudBudgetEvaluationWorker`：
  - Portal 启动时自动运行。
  - 日志 worker 名称为 `cloudBudgetEvaluation`。
  - 有实际评估、事件通知或锁跳过时输出聚合结果。
- Portal 启动入口新增后台 goroutine：
  - `go apps.StartCloudBudgetEvaluationWorker(configs.Get().Consul.ServiceID)`。
- 新增环境变量样例：
  - `CLOUDIAC_BUDGET_EVALUATION_WORKER_INTERVAL_SECONDS=60`。
  - 默认 60 秒，最小 15 秒，最大 86400 秒。
- 预算评估继续复用现有 `EvaluateDueCloudBudgets` 和 `cost.budget.threshold_exceeded` 事件写入逻辑，避免后台 worker 与手动接口出现不同口径。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check -- backend/portal/apps/cloud_budget.go backend/cmds/portal/main.go backend/configs/dotenv.sample` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过，`iac-portal` healthy。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 插入临时预算 `cbd-codex-worker-2042`：
  - `period=2026-06`
  - `currency=CNY`
  - `limitAmount=10`
  - `thresholdPercent=80`
  - `evaluationInterval=15`
- 当前组织已有 `2026-06/CNY` 成本合计 `300.85`，worker 首次启动后日志出现：
  - `worker=cloudBudgetEvaluation`
  - `evaluate due cloud budgets result`
  - `dueCount:5`
  - `evaluatedCount:5`
  - `notificationCount:1`
- 回读临时预算：
  - `last_amount=300.8500`
  - `last_usage_percent=3008.50`
  - `last_evaluated_at=2026-06-20 20:42:51`
  - `last_exceeded_at=2026-06-20 20:42:51`
- 回读事件表确认生成 `cost.budget.threshold_exceeded` 事件，`resource_id=cbd-codex-worker-2042`。
- 验证后已软删除临时预算和对应事件，避免污染后续测试数据。
- 2026-06-22 补充后台调度核心单元测试：
  - `TestCloudBudgetEvaluationDue` 覆盖未评估、到期边界、未到期跳过和默认评估间隔。
  - `TestCloudBudgetEvaluationWorkerInterval` 覆盖环境变量默认值、非法值、最小值、正常值和最大值裁剪。
  - `TestCloudBudgetEvaluateResultCount` 覆盖 worker 聚合计数类型兼容和累加。
  - `TestCloudBudgetRespAmounts` 覆盖预算响应金额、使用率、剩余额度下限和超阈值/超预算状态。
  - `TestCloudBudgetEvaluationLockName` 覆盖预算评估命名锁格式。

验证限制：

- 当前 worker 仍使用固定环境变量周期，尚未接入系统配置页面。
- 预算审批、财务编码、汇率和复杂摊销仍待后续 FinOps 阶段建设。

待继续：

- 将更多成本异常事件和预算事件接入 ITSM/企业微信/钉钉/Slack/邮件。
- 继续推进分片耗时、最近消费时间和成功率指标。

### 16.67 2026-06-20 V1.2 P1 CloudEvent 外部通知集成第一阶段

状态：已完成 CloudEvent 与现有通知渠道的第一阶段打通；事件中心写入后可按事件类型订阅并投递到 Webhook、企业微信、钉钉、Slack 和邮件。

已完成：

- `recordCloudEventWithDispatch` 在原有 Webhook 投递后新增通知渠道分发。
- 新增 CloudEvent 通知分发逻辑：
  - 复用组织设置中的 `iac_notification` 配置。
  - 支持精确事件类型订阅，例如 `cost.budget.threshold_exceeded`。
  - 支持通配订阅，例如 `cloud.*`、`cost.*`、`risk.*`、`operation.*`。
  - 投递失败只记录日志，不阻断事件写入和原业务流程。
- 通知内容统一包含：
  - 事件类型、来源、级别、状态和发生时间。
  - 云厂商、账号、地域、资源、项目和环境等上下文字段。
  - 事件消息和事件中心查看链接。
- 后端通知事件类型校验从仅允许 `task.*` 扩展为允许 CloudEvent 事件族。
- `iac_notification_event.event_type` 字段从枚举扩展为 `varchar(128)`，支持新的多云事件类型。
- 前端通知事件类型选项新增：
  - `cloud.*`
  - `account.*`
  - `operation.*`
  - `risk.*`
  - `cost.*`
  - `cmdb.*`
  - `notification.*`
  - 账号、操作、风险、成本和 Webhook 失败等精确事件。
- 前端通知类型补齐 Webhook 选项和 URL 配置面板。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 覆盖本次后端和前端修改文件，通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- Portal 启动迁移后确认 `iac_notification_event.event_type` 为 `varchar(128)`。
- 启动本地临时 HTTP 接收器，确认 `iac-portal` 容器可访问 `host.docker.internal:19099`。
- 通过 API 创建临时 Webhook 通知：
  - 订阅事件类型为 `cost.*`。
  - URL 为临时接收器 `/cloud-event`。
- 通过 API 创建临时预算并触发 `cost.budget.threshold_exceeded` 事件。
- 临时接收器收到 CloudEvent 通知正文，包含：
  - `事件类型：cost.budget.threshold_exceeded`
  - `来源：cost`
  - `级别：error`
  - `状态：threshold_exceeded`
  - 预算资源名称和事件中心链接。
- 验证后已删除临时通知、临时预算，并软删除对应临时事件。
- 使用 Docker Compose 构建 `iac-web` 通过，仅保留既有 webpack 体积警告。
- 使用 Docker Compose 重启 `iac-web` 通过。
- 调用 `GET /api/v1/check` 和 `HEAD /` 均返回正常。

验证限制：

- 企业微信、钉钉、Slack 和邮件复用现有通知客户端，当前本地验证使用 Webhook 临时接收器完成链路验证。
- ITSM 工单系统尚未接入，后续需要补充外部系统适配器和重试/回执模型。
- 通知模板当前为统一 Markdown 文本，尚未建设可配置模板和多语言模板。

待继续：

- 继续推进 Webhook 分片耗时、最近消费时间和成功率指标。
- 建设 ITSM 工单集成、通知模板管理和通知投递历史查询。

### 16.68 2026-06-20 V1.2 P1 Webhook 分片执行指标第一阶段

状态：已完成 Webhook retry-due 和队列摘要的分片指标增强；页面和 API 均可查看成功率、最近消费时间和执行耗时。

已完成：

- `POST /api/v1/cloud/webhooks/retry-due` 聚合响应新增：
  - `startedAt`
  - `finishedAt`
  - `durationMs`
  - `successRate`
  - `lastConsumedAt`
- `shardResults` 每个分片响应新增：
  - `startedAt`
  - `finishedAt`
  - `durationMs`
  - `successRate`
  - `lastConsumedAt`
- `GET /api/v1/cloud/webhooks/queue/summary` 总览新增：
  - `successRate`
  - `queueConsumed`
  - `queueSuccess`
  - `queueFailed`
  - `queueSkipped`
  - `queueSuccessRate`
  - `lastConsumedAt`
- `GET /api/v1/cloud/webhooks/queue/summary` 分片摘要新增：
  - `consumed`
  - `success`
  - `failed`
  - `skipped`
  - `successRate`
  - `lastConsumedAt`
- 队列成功率按保留期内已消费队列计算：
  - 成功：`done`、存在结果投递且无错误信息。
  - 失败：`done` 且有错误信息。
  - 跳过：`skipped`。
- retry-due 成功率按本次消费的 `success / total` 计算；无本次消费时返回 `0`。
- 事件中心 Webhook 面板新增总览指标：
  - 投递成功率。
  - 队列成功率。
  - 分片数。
  - 最近消费时间。
- 事件中心 Webhook 面板新增分片表，展示每个分片的排队、到期、未来、已消费、成功/失败/跳过、成功率、最早待跑和最近消费。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check -- backend/portal/apps/cloud_webhook.go backend/portal/models/resps/cloud_webhook.go frontend/app/containers/org/cloud-event/index.jsx frontend/app/containers/org/cloud-event/styles.less docs/prd/cloudiac-multicloud-management-prd.md` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 鉴权后调用 `GET /api/v1/cloud/webhooks/queue/summary` 返回新增字段：
  - `successRate=46.51`
  - `queueConsumed=0`
  - `queueSuccessRate=0`
  - `shardTotal=4`
  - 首个分片包含 `consumed/success/failed/skipped/successRate/lastConsumedAt`。
- 鉴权后调用 `POST /api/v1/cloud/webhooks/retry-due` 返回新增字段：
  - `durationMs=5`
  - `successRate=0`
  - `startedAt`
  - `finishedAt`
  - `shardResults[0].durationMs=2`
  - `shardResults[0].successRate=0`
- 使用 Docker Compose 构建 `iac-web` 通过，仅保留既有 webpack 体积警告。
- 使用 Docker Compose 重启 `iac-web` 通过。
- 调用 `GET /api/v1/check` 和 `HEAD /` 均返回正常。
- 内置浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-events`，确认页面包含：
  - `投递成功率`
  - `队列成功率`
  - `分片数`
  - `最近消费`
  - `成功/失败/跳过`
- 浏览器控制台未发现 error。

验证限制：

- 当前成功率为即时计算，未做长期趋势存储。
- 队列成功率受 7 天队列保留周期影响，过期清理后只反映当前保留窗口。
- 当前页面只展示分片摘要，未增加分片历史趋势图。

待继续：

- 建设 ITSM 工单集成、通知模板管理和通知投递历史查询。
- 将更多成本异常、风险整改和账号同步完成事件接入外部集成闭环。

### 16.69 2026-06-20 V1.2 P1 通知投递历史第一阶段

状态：已完成 CloudEvent 通知投递历史第一阶段；外部通知发送后会记录成功/失败状态，组织设置中的通知配置可查看投递历史。

已完成：

- 新增通知投递历史模型 `iac_notification_delivery`：
  - 组织、项目、通知配置、事件 ID、事件类型。
  - 通知类型、目标、标题、消息正文。
  - 投递状态、错误信息、投递时间。
- Portal 启动迁移新增 `NotificationDelivery`。
- CloudEvent 通知分发完成后写入投递历史：
  - 成功投递记录 `status=success`。
  - 失败投递记录 `status=failed` 并保存错误信息。
  - 支持 Webhook、企业微信、钉钉、Slack 和邮件渠道统一留痕。
- 新增通知投递历史查询 API：
  - `GET /api/v1/notifications/:id/deliveries`
  - 支持按事件 ID、事件类型、通知类型、状态筛选。
  - 返回分页列表和通知配置名称。
- 组织设置“通知”页面新增“历史”操作入口。
- 通知历史抽屉展示：
  - 状态。
  - 类型。
  - 事件类型。
  - 目标。
  - 标题。
  - 错误。
  - 投递时间。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 覆盖本次后端和前端修改文件，通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过。
- Portal 启动迁移后确认 `iac_notification_delivery` 表存在。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 启动本地临时 HTTP 接收器。
- 通过 API 创建临时 Webhook 通知：
  - `eventType=["cost.*"]`
  - URL 为临时接收器 `/notification-history`。
- 通过 API 创建临时预算并触发 `cost.budget.threshold_exceeded` 事件。
- 临时接收器收到 CloudEvent 通知。
- 调用 `GET /api/v1/notifications/:id/deliveries` 返回：
  - `total=1`
  - `status=success`
  - `eventType=cost.budget.threshold_exceeded`
  - `notificationType=webhook`
  - `target=http://host.docker.internal:19099/notification-history`
  - `title=成本预算超阈值`
- 验证后已删除临时通知、临时预算，并软删除对应临时事件和临时投递历史。
- 使用 Docker Compose 构建 `iac-web` 通过，仅保留既有 webpack 体积警告。
- 使用 Docker Compose 重启 `iac-web` 通过。
- 调用 `GET /api/v1/check` 和 `HEAD /` 均返回正常。
- 内置浏览器打开组织设置页，确认设置页和通知页签 DOM 可加载；浏览器 role/capture 点击接口在该页签页出现超时，因此本次前端交互验证以生产构建和 API 验证为准。

验证限制：

- 当前投递历史只记录 CloudEvent 通知链路；旧任务通知链路尚未统一写入该表。
- 当前投递历史不做自动清理策略，后续需补保留周期和归档策略。
- ITSM 工单系统、通知模板管理和投递重试仍待后续阶段建设。

待继续：

- 建设 ITSM 工单集成、通知模板管理和通知投递历史清理策略。
- 将旧任务通知链路统一写入通知投递历史。

### 16.70 2026-06-20 V1.2 P1 通知投递历史清理策略第一阶段

状态：已完成通知投递历史保留期清理 worker；CloudEvent 通知历史不再无限增长，默认保留 90 天。

已完成：

- 新增 `StartNotificationDeliveryCleanupWorker`：
  - Portal 启动后自动运行。
  - worker 名称为 `notificationDeliveryCleanup`。
  - 启动后立即执行一轮清理，然后按周期执行。
- 新增 `CleanupExpiredNotificationDeliveries`：
  - 清理 `delivered_at` 超过保留期的 `iac_notification_delivery` 记录。
  - 使用软删除，保留与现有模型一致的数据生命周期语义。
  - 使用 MySQL 命名锁 `cloudiac:notification_delivery:cleanup`，避免多 Portal 实例重复清理。
- Portal 启动入口新增后台 goroutine：
  - `go apps.StartNotificationDeliveryCleanupWorker(configs.Get().Consul.ServiceID)`。
- 新增环境变量样例：
  - `CLOUDIAC_NOTIFICATION_DELIVERY_RETENTION_DAYS=90`。
  - `CLOUDIAC_NOTIFICATION_DELIVERY_CLEANUP_INTERVAL_SECONDS=3600`。
- 配置约束：
  - 保留天数默认 90 天，最小 1 天，最大 3650 天。
  - 清理周期默认 3600 秒，最小 60 秒，最大 86400 秒。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 中转执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check -- backend/portal/apps/notification.go backend/cmds/portal/main.go backend/configs/dotenv.sample docs/prd/cloudiac-multicloud-management-prd.md` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 插入临时旧投递历史 `ndl-codex-cleanup-2124`：
  - `delivered_at=2026-03-21 13:24:05`
  - `deleted_at_t=0`
- 使用 Docker Compose 重启 `iac-portal` 通过。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- worker 启动首轮清理后回读临时记录：
  - `deleted_at_t=1781961881`
  - `delivered_at=2026-03-21 13:24:05`
- `iac-portal` 日志出现：
  - `worker=notificationDeliveryCleanup`
  - `cleanup expired notification deliveries result`
  - `cleaned=1`
  - `retentionDays=90`

验证限制：

- 当前清理策略只做保留期软删除，尚未提供归档到外部数据湖或冷存储。
- 当前投递历史仍只覆盖 CloudEvent 通知链路，旧任务通知链路待后续统一接入。

待继续：

- 建设 ITSM 工单集成和通知模板管理。
- 将旧任务通知链路统一写入通知投递历史。

### 16.71 2026-06-20 V1.2 P1 旧任务通知投递历史统一接入第一阶段

状态：已完成旧任务通知链路投递历史接入；任务状态变化触发的邮件、Webhook、企业微信、钉钉和 Slack 通知会统一写入 `iac_notification_delivery`。

已完成：

- 旧任务通知发送函数改为返回发送错误：
  - `SendDingTalkMessage`
  - `SendWebhookMessage`
  - `SendWechatMessage`
  - `SendSlackMessage`
  - `SendEmailMessage`
- `SyncSendMessage` 在每次任务通知发送后写入投递历史：
  - `event_id` 使用任务 ID。
  - `event_type` 使用任务状态映射事件，如 `task.failed`。
  - `notification_type`、`target`、`title`、`message`、`status`、`error_message`、`delivered_at` 与 CloudEvent 通知历史保持一致。
- 非邮件通道按通知配置逐条记录投递结果。
- 邮件通道保留旧逻辑的全局邮箱去重语义：
  - 同一个邮箱在多个邮件通知配置中只发送一次。
  - 每个实际发送的邮箱写入一条投递历史。
- 发送失败也会写入历史：
  - `status=failed`
  - `error_message` 保存发送错误，便于组织设置通知历史页排查。

验证：

- 使用 Docker Go 镜像经临时容器执行 `gofmt`，覆盖 `backend/portal/services/notificationrc/notificationservice.go`。
- 执行 `git diff --check -- backend/portal/services/notificationrc/notificationservice.go` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 插入临时验证数据：
  - 通知配置 `notif-codex-taskhist-2134`
  - 模板 `tpl-codex-taskhist-2134`
  - 环境 `env-codex-taskhist-2134`
  - 任务 `run-codex-taskhist-2134`
  - 审批步骤 `step-codex-taskhist-2134`
- 鉴权后调用 `POST /api/v1/tasks/run-codex-taskhist-2134/approve`，请求 `action=rejected` 返回 `200`。
- 任务状态从 `approving` 更新为 `rejected`，旧任务通知链路触发 `task.failed`。
- `iac_notification_delivery` 新增投递历史：
  - `notification_id=notif-codex-taskhist-2134`
  - `event_id=run-codex-taskhist-2134`
  - `event_type=task.failed`
  - `notification_type=webhook`
  - `target=http://127.0.0.1:9/codex-taskhist`
  - `status=failed`
  - `error_message` 包含 `connection refused`
- 鉴权后调用 `GET /api/v1/notifications/notif-codex-taskhist-2134/deliveries?pageSize=5` 返回：
  - `total=1`
  - `notificationName=codex-taskhist-webhook-2134`
  - `eventType=task.failed`
  - `eventId=run-codex-taskhist-2134`
  - `status=failed`
- 验证后已清理 `codex-taskhist` 临时通知、模板、环境、任务、任务步骤和投递历史数据。

验证限制：

- 本阶段只统一投递历史留痕，不增加任务通知的重试队列。
- 邮件通道仍沿用既有发送机制，未增加批量投递详情或 SMTP 分段重试。
- ITSM 工单系统和通知模板可视化管理仍待后续阶段建设。

待继续：

- 建设 ITSM 工单集成和通知模板管理。
- 评估是否为旧任务通知链路复用 Webhook 投递队列和重试策略。

### 16.72 2026-06-20 V1.2 P1 通知模板管理后端第一阶段

状态：已完成通知模板管理后端第一阶段；任务通知和 CloudEvent 通知可按事件类型和通知类型使用自定义模板渲染标题与消息内容。

已完成：

- 新增通知模板模型 `iac_notification_template`：
  - 组织、项目、模板名称。
  - 事件类型、通知类型。
  - 标题模板、文本/邮件模板、Markdown 模板。
  - 启用状态、创建人、软删除字段。
- 新增唯一约束：
  - `org_id + project_id + event_type + notification_type + deleted_at_t`
  - 防止同一作用域、同一事件、同一通知类型出现多个活跃模板。
- Portal 启动迁移新增 `NotificationTemplate`。
- 新增通知模板 API：
  - `GET /api/v1/notification-templates`
  - `POST /api/v1/notification-templates`
  - `GET /api/v1/notification-templates/:id`
  - `PUT /api/v1/notification-templates/:id`
  - `DELETE /api/v1/notification-templates/:id`
- 查询 API 支持按事件类型、通知类型和状态筛选。
- 模板查找策略：
  - 项目级模板优先。
  - 未命中项目级模板时回退组织级模板。
  - 未配置模板时回退系统内置模板。
- 旧任务通知链路接入自定义模板：
  - 邮件使用 `content` 优先。
  - Webhook、企业微信、钉钉、Slack 使用 `markdownContent` 优先。
  - 支持 Go template 变量，如 `{{.EnvName}}`、`{{.TaskType}}`、`{{.Message}}`。
- CloudEvent 通知链路接入自定义模板：
  - 支持 CloudEvent 字段变量。
  - 额外提供 `{{.ViewUrl}}` 作为事件中心链接。

验证：

- 使用 Docker Go 镜像经临时容器执行 `gofmt`，覆盖本次修改的 Go 文件。
- 执行 `git diff --check` 覆盖通知模板后端相关文件，通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- Portal 启动迁移后确认 `iac_notification_template` 表存在，字段包含：
  - `event_type`
  - `notification_type`
  - `title`
  - `content`
  - `markdown_content`
  - `status`
- 确认唯一索引 `unique__notification_template_scope` 已创建。
- 鉴权后调用 `POST /api/v1/notification-templates` 创建临时模板：
  - `name=codex-notification-template-2142`
  - `eventType=task.failed`
  - `type=webhook`
  - `title=自定义任务失败 {{.EnvName}}`
  - `markdownContent=自定义模板命中: {{.TaskType}} / {{.Message}} / {{.EnvName}}`
- 鉴权后调用 `GET /api/v1/notification-templates?eventType=task.failed&type=webhook&pageSize=5` 返回：
  - `total=1`
  - `creatorName=admin@example.com`
- 插入临时任务通知验证数据并调用审批驳回 API：
  - `POST /api/v1/tasks/run-codex-tpl-2142/approve`
  - 请求 `action=rejected`
  - 返回 `200`
- 投递历史中确认模板渲染结果：
  - `title=自定义任务失败 codex-template-env-2142`
  - `message=自定义模板命中: plan / rejected / codex-template-env-2142`
  - `event_type=task.failed`
  - `status=failed`
- 鉴权后调用 `GET /api/v1/notifications/notif-codex-tpl-2142/deliveries?pageSize=5` 返回同样的自定义标题和消息。
- 鉴权后调用 `PUT /api/v1/notification-templates/ntpl-d8r9hd06c6kc73e4t5k0` 更新模板返回 `200`。
- 鉴权后调用 `GET /api/v1/notification-templates/ntpl-d8r9hd06c6kc73e4t5k0` 返回更新后的模板详情。
- 鉴权后调用 `DELETE /api/v1/notification-templates/ntpl-d8r9hd06c6kc73e4t5k0` 返回 `200`。
- 验证后已清理临时任务、环境、云模板、通知配置、通知事件和投递历史；临时通知模板已软删除，活跃行数为 `0`。

验证限制：

- 本阶段提供后端 API 和发送链路渲染，尚未建设前端可视化模板管理页面。
- 当前模板使用 Go template 变量，尚未提供变量字典接口、预览接口或模板语法校验 API。
- 当前没有模板版本历史和回滚功能。

待继续：

- 建设通知模板可视化管理页面、模板预览和变量字典。
- 建设 ITSM 工单集成。
- 评估旧任务通知是否复用 Webhook 投递队列和重试策略。

### 16.73 2026-06-20 V1.2 P1 通知模板管理前端第一阶段

状态：已完成组织通知设置页的通知模板可视化管理第一阶段；管理员可以在前端查看、创建、编辑和删除组织级通知模板。

已完成：

- 组织设置的通知页面新增页签：
  - `通知配置`
  - `通知模板`
- 通知配置页签保留原有通知配置列表、添加、编辑和删除能力。
- 通知配置页签新增投递历史入口，可打开抽屉查看单个通知配置的投递记录。
- 通知模板页签新增模板列表：
  - 名称。
  - 事件类型。
  - 通知类型。
  - 启用状态。
  - 创建人。
  - 更新时间。
- 新增通知模板抽屉表单：
  - 模板名称。
  - 事件类型。
  - 通知类型。
  - 启用状态。
  - 标题模板。
  - Markdown 模板。
  - 文本模板。
- 前端服务层新增通知模板 API 调用：
  - 查询模板列表。
  - 创建模板。
  - 查询模板详情。
  - 更新模板。
  - 删除模板。
- 通知模板列表的事件类型、通知类型和启用状态使用中文渲染。

验证：

- 执行 `git diff --check` 覆盖本阶段前端相关文件，通过。
- 使用 Docker Compose 构建 `iac-web` 通过。
- 构建过程中只出现既有依赖废弃提示和 webpack 包体积 warning，未出现 JSX 或业务代码编译错误。
- 使用 Docker Compose 重启 `iac-web` 通过。
- 调用 `HEAD http://127.0.0.1` 返回 `200 OK`。
- 调用 `GET http://127.0.0.1/api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- Docker Compose 服务状态确认：
  - `consul` healthy。
  - `mysql` healthy。
  - `iac-portal` healthy。
  - `iac-web` running。
  - `ct-runner` healthy。

验证限制：

- 本阶段完成模板管理的基础 CRUD 页面，尚未提供模板变量字典、模板预览和语法校验。
- 当前模板作用域仍以组织级为主，项目级模板的前端入口待后续补充。
- 本阶段未增加模板版本历史、复制模板和回滚能力。

待继续：

- 建设通知模板变量字典、预览和语法校验 API/前端。
- 建设项目级通知模板入口。
- 建设 ITSM 工单集成。
- 评估旧任务通知是否复用 Webhook 投递队列和重试策略。

### 16.74 2026-06-20 V1.2 P1 通知模板变量字典与预览校验第一阶段

状态：已完成通知模板变量字典、服务端预览渲染和保存前语法校验第一阶段；管理员可在模板抽屉中查看可用变量并预览当前模板。

已完成：

- 新增通知模板变量字典 API：
  - `GET /api/v1/notification-templates/variables`
- 变量字典第一阶段覆盖：
  - 任务通知变量，如 `{{.EnvName}}`、`{{.TaskType}}`、`{{.Message}}`、`{{.Addr}}`。
  - 多云事件变量，如 `{{.EventType}}`、`{{.Provider}}`、`{{.Region}}`、`{{.ResourceName}}`、`{{.ViewUrl}}`。
- 变量字典支持按事件类型动态裁剪：
  - `task.*` 事件只返回任务通知变量。
  - 其它平台事件返回多云事件变量。
  - 未传事件类型时返回全部变量组。
- 新增通知模板预览 API：
  - `POST /api/v1/notification-templates/preview`
- 预览 API 支持传入标题模板、Markdown 模板和文本模板，并返回服务端使用样例变量渲染后的结果。
- 通知模板创建和更新时增加 Go template 语法校验：
  - 标题模板校验。
  - 文本模板校验。
  - Markdown 模板校验。
- 组织通知模板抽屉新增：
  - 变量列表展示。
  - `预览` 操作。
  - 预览结果展示标题、Markdown 和文本渲染结果。
- 前端会在事件类型或通知类型变化时刷新变量字典。
- 前端服务层新增变量字典和模板预览接口调用。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 临时目录执行 `gofmt`，覆盖本阶段修改的 Go 文件。
- 执行 `git diff --check` 覆盖本阶段后端、前端和 PRD 文件，通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；仅出现既有依赖废弃提示和 webpack 包体积 warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web` 通过。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 鉴权后调用 `GET /api/v1/notification-templates/variables?eventType=task.failed&type=webhook` 返回：
  - `code=200`
  - `groups=["task"]`
  - 样例环境名为 `production`
  - 样例事件类型为 `task.failed`
  - 字典包含 `Message` 变量
- 鉴权后调用 `GET /api/v1/notification-templates/variables?eventType=cloud_operation.failed&type=webhook` 返回：
  - `code=200`
  - `groups=["cloudEvent"]`
  - 样例事件类型为 `cloud_operation.failed`
- 鉴权后调用 `POST /api/v1/notification-templates/preview`，使用：
  - `title=失败 {{.EnvName}}`
  - `markdownContent=消息 {{.Message}} / {{.TaskType}}`
  - `content=文本 {{.ProjectName}}`
- 预览返回：
  - `title=失败 production`
  - `markdownContent=消息 示例通知消息 / plan`
  - `content=文本 示例项目`
- 鉴权后使用错误模板 `{{ if` 调用预览接口，返回参数错误。
- 鉴权后使用错误模板 `{{ if` 调用创建模板接口，返回参数错误。
- 数据库确认错误模板 `codex-invalid-template-save` 活跃记录数为 `0`。
- 调用 `HEAD http://127.0.0.1` 返回 `200 OK`。
- Docker Compose 服务状态确认：
  - `consul` healthy。
  - `mysql` healthy。
  - `iac-portal` healthy。
  - `iac-web` running。
  - `ct-runner` healthy。

验证限制：

- 本阶段变量字典仍为内置字典，只按任务通知和多云事件两大类裁剪；真实事件 schema 动态字典已在 16.80 补齐。
- 预览使用内置样例数据，尚未支持前端自定义样例 payload。
- 保存时校验模板语法，但不校验字段是否一定存在于每一种事件 payload。

待继续：

- 建设项目级通知模板入口。
- 建设 ITSM 工单集成。
- 评估旧任务通知是否复用 Webhook 投递队列和重试策略。

### 16.75 2026-06-20 V1.2 P1 项目级通知模板入口和作用域隔离第一阶段

状态：已完成项目设置中的通知管理入口第一阶段；通知配置和通知模板现在可在组织级、项目级两个作用域内隔离管理。

已完成：

- 项目设置页面新增 `通知` 页签。
- 项目级通知页签复用组织设置中的通知管理能力：
  - `通知配置`
  - `通知模板`
- 前端通知服务层新增可选 `projectId` 透传：
  - 通知配置列表、创建、详情、更新、删除。
  - 通知投递历史。
  - 通知模板列表、创建、详情、更新、删除。
  - 通知模板变量字典和预览。
- 后端通知配置查询增加作用域隔离：
  - 带 `IaC-Project-Id` 时只返回项目级通知配置。
  - 不带 `IaC-Project-Id` 时只返回组织级通知配置。
- 后端通知配置详情、更新、删除和投递历史查询增加 org/project 作用域校验。
- 通知配置删除增加命中行数判断；当前作用域无匹配记录时返回“对象不存在或者无权限”，不会继续清理关联事件。
- 后端通知模板查询增加作用域隔离：
  - 带 `IaC-Project-Id` 时只返回项目级通知模板。
  - 不带 `IaC-Project-Id` 时只返回组织级通知模板。
- 通知模板详情、更新、删除增加 org/project 作用域校验。
- 通知模板列表和详情显式过滤 `deleted_at_t = 0`，避免软删除模板继续出现在页面中。
- 通知模板删除增加命中行数判断；当前作用域无匹配记录时返回“对象不存在或者无权限”。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 临时目录执行 `gofmt`，覆盖本阶段修改的 Go 文件。
- 执行 `git diff --check` 覆盖本阶段后端、前端和 PRD 文件，通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；仅出现既有依赖废弃提示和 webpack 包体积 warning。
- 使用 Docker Compose 重启 `iac-portal` 和 `iac-web` 通过。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- 调用 `HEAD http://127.0.0.1` 返回 `200 OK`。
- 项目级通知配置 API 验证：
  - 创建项目级通知配置返回 `code=200`。
  - 创建结果 `projectId=p-d8qtv3v9fm0s738ko4o0`。
  - 项目级列表包含该通知配置。
  - 组织级列表不包含该通知配置。
  - 组织级详情返回 `50010013`。
  - 项目级详情返回 `200`。
  - 组织级删除返回 `50010013`，项目级详情仍返回 `200`。
  - 项目级删除返回 `200`。
  - 删除后 `iac_notification` 临时记录数为 `0`。
- 项目级通知模板 API 验证：
  - 创建项目级通知模板返回 `code=200`。
  - 创建结果 `projectId=p-d8qtv3v9fm0s738ko4o0`。
  - 项目级列表包含该通知模板。
  - 组织级列表不包含该通知模板。
  - 组织级详情返回 `50010013`。
  - 项目级详情返回 `200`。
  - 组织级删除返回 `50010013`，项目级详情仍返回 `200`。
  - 项目级删除返回 `200`。
  - 删除后项目级详情返回 `50010013`。
  - 删除后项目级列表不再包含该模板。
  - 删除后 `iac_notification_template` 活跃记录数为 `0`。
- 内置浏览器打开项目设置页面：
  - 确认项目设置页存在 `通知` 页签。
  - 点击 `通知` 后确认 `通知配置` 和 `通知模板` 子页签可渲染。
  - 在浏览器验证中发现软删除模板仍显示的问题，并已修复后端活跃记录过滤。
- Docker Compose 服务状态确认：
  - `iac-portal` healthy。
  - `iac-web` running。

验证限制：

- 项目级通知入口当前复用组织级组件，尚未提供“从组织模板复制到项目模板”的专用交互。
- 组织级模板到项目级模板的继承、覆盖关系当前由后端查找策略支持，前端尚未提供可视化说明和覆盖状态展示。
- 模板变量字典在本阶段仍为内置静态定义；真实事件 schema 动态字典已在 16.80 补齐。

待继续：

- 建设模板版本历史、来源模板快照和历史回滚能力；项目级模板批量复制已在 16.78 补齐。
- 建设 ITSM 工单集成。
- 评估旧任务通知是否复用 Webhook 投递队列和重试策略。

### 16.76 2026-06-20 V1.2 P1 项目级通知模板继承复制第一阶段

状态：已完成项目级通知模板继承复制第一阶段；项目设置的通知模板页可以查看组织模板，并将组织模板复制为项目模板。

已完成：

- 项目设置 `通知 -> 通知模板` 页面拆分展示：
  - `项目模板`
  - `组织模板`
- 项目模板列表仍使用项目作用域查询，只展示当前项目自有模板。
- 组织模板列表在项目页面中使用组织作用域查询，不透传 `IaC-Project-Id`。
- 组织模板列表新增 `复制` 操作。
- 复制组织模板时复用已有通知模板创建接口，在项目作用域下创建项目模板。
- 复制字段覆盖：
  - 模板名称。
  - 事件类型。
  - 通知类型。
  - 标题模板。
  - Markdown 模板。
  - 文本模板。
  - 启用状态。
- 组织模板与当前项目模板存在相同事件类型和通知类型时，组织模板行显示 `已覆盖`。
- 点击复制前会按事件类型和通知类型查询项目作用域，避免分页情况下重复创建并撞唯一约束。

验证：

- 使用 Docker Compose 构建 `iac-web` 通过；仅出现既有依赖废弃提示和 webpack 包体积 warning。
- 使用 Docker Compose 重启 `iac-web` 通过。
- 调用 API 创建临时组织级通知模板：
  - `name=codex-org-template-copy-final-1781966913`
  - `eventType=notification.codex_copy_final`
  - `type=webhook`
- 内置浏览器打开项目设置 `通知 -> 通知模板`：
  - 页面显示 `项目模板` 表格。
  - 页面显示 `组织模板` 表格。
  - 组织模板表格显示临时组织模板。
  - 组织模板行显示 `复制` 操作。
- 点击 `复制` 后返回操作成功。
- 复制后项目模板表格显示对应项目级模板。
- 复制后组织模板行显示 `已覆盖`。
- 验证后删除临时项目模板和临时组织模板：
  - 项目模板删除返回 `200`。
  - 组织模板删除返回 `200`。
  - `iac_notification_template` 临时活跃记录数为 `0`。

验证限制：

- 本阶段仅支持单条组织模板复制；批量复制已在 16.78 补齐。
- 本阶段复制后生成独立项目模板，尚未保存组织模板来源 ID 或版本关系；来源 ID 和差异展示已在 16.77 补齐，来源版本快照仍待开发。
- 本阶段覆盖状态以事件类型和通知类型判断；项目模板与组织模板内容差异已在 16.77 补齐。

待继续：

- 建设模板版本历史、来源模板快照和历史回滚能力。
- 建设 ITSM 工单集成。
- 评估旧任务通知是否复用 Webhook 投递队列和重试策略。

### 16.77 2026-06-20 V1.2 P1 项目级通知模板来源追踪与差异回退第一阶段

状态：已完成项目级通知模板来源追踪、差异对比和回退组织模板第一阶段；项目模板可记录来源组织模板，并在项目设置页面查看差异或回退到组织模板。

已完成：

- 通知模板模型新增来源字段：
  - `source_template_id`
- 通知模板创建表单新增可选字段：
  - `sourceTemplateId`
- 项目级模板创建时支持保存来源组织模板 ID。
- 来源模板校验：
  - 仅项目级模板允许传入 `sourceTemplateId`。
  - 来源模板必须属于同一组织。
  - 来源模板必须是组织级活跃模板。
  - 来源模板的事件类型和通知类型必须与新建项目模板一致。
- 项目通知模板列表新增项目上下文专属列：
  - `来源`
  - `差异`
- 项目模板行新增项目上下文专属操作：
  - `差异`
  - `回退`
- 差异抽屉按字段对比项目模板与组织模板：
  - 名称。
  - 标题。
  - Markdown。
  - 文本。
  - 状态。
- 回退组织模板复用删除项目模板能力；删除项目模板后，通知发送链路按既有策略回落到组织级模板。
- 删除/回退项目模板成功后会关闭差异抽屉，避免残留空差异面板。
- 组织级通知模板页面不显示项目专属来源、差异和回退操作。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 临时目录执行 `gofmt`，覆盖本阶段修改的 Go 文件。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过。
- Portal 启动迁移后确认 `iac_notification_template.source_template_id` 字段已创建。
- 调用 `GET /api/v1/check` 返回 `success=true`、`version=v1.3.5`、`build=docker-compose`。
- API 创建来源组织模板返回 `code=200`。
- API 创建带 `sourceTemplateId` 的项目模板返回 `code=200`。
- 项目模板创建结果、详情结果和数据库记录均包含正确 `sourceTemplateId`。
- 使用不匹配事件类型的 `sourceTemplateId` 创建项目模板返回参数错误 `50010340`。
- 验证后删除临时项目模板和组织模板，活跃记录数为 `0`。
- 使用 Docker Compose 构建 `iac-web` 通过；仅出现既有依赖废弃提示和 webpack 包体积 warning。
- 使用 Docker Compose 重启 `iac-web` 通过。
- 内置浏览器打开项目设置 `通知 -> 通知模板`：
  - 项目模板表格显示来源组织模板名称。
  - 项目模板表格显示 `有差异`。
  - 项目模板行显示 `差异` 和 `回退` 操作。
  - 点击 `差异` 后差异抽屉展示字段对比。
  - 差异抽屉中名称和标题显示 `有差异`，Markdown、文本和状态显示 `一致`。
  - 点击 `回退` 并确认后，项目模板表格回到 `暂无数据`。
  - 回退后组织模板行从 `已覆盖` 回到 `复制`。
- 验证后删除最终临时组织模板，活跃记录数为 `0`。

验证限制：

- 本阶段保存来源组织模板 ID；来源模板版本快照和历史回滚已在 16.79 补齐。
- 差异对比以当前组织模板为准；组织模板后续被编辑时，差异会反映最新组织模板状态，历史快照可通过 16.79 的模板历史查看。
- 回退组织模板仍通过删除项目模板实现；回滚到历史项目模板版本已在 16.79 补齐。

待继续：

- 建设 ITSM 工单集成。

### 16.89 2026-06-21 V1.2 P1 ITSM 工单集成第一阶段

状态：已完成 ITSM 工单集成第一阶段；平台新增通用 ITSM 连接器、云操作生成工单、工单状态回写、ITSM 事件沉淀和前端管理页面。当前阶段支持不依赖外网的本地待提交工单模式，也支持通过通用 HTTP POST 对接外部工单系统。

已完成：

- 新增 ITSM 数据模型：
  - `iac_cloud_itsm_config`：ITSM 连接器配置。
  - `iac_cloud_itsm_ticket`：云操作关联的 ITSM 工单。
- 新增 ITSM API：
  - 查询、新建、编辑、删除连接器。
  - 查询工单列表。
  - 从云操作任务创建 ITSM 工单。
  - 回写工单状态、外部单号、外部链接和同步载荷。
- 连接器能力：
  - 支持 `generic/jira/servicenow` provider 预留。
  - 支持 `none/bearer/basic` 认证类型。
  - 支持 `baseUrl`、`createTicketPath`、`createTicketUrl`、`browseTicketPath` 扩展参数。
  - 未配置外部地址时创建本地 `pending` 工单，适配离线环境。
- 工单创建能力：
  - 工单请求载荷包含云操作 ID、动作、状态、风险等级、云厂商、资源类型、资源 ID、资源名、项目、环境、参数和执行结果。
  - 支持 dry-run，不写入数据库，只返回待发送载荷。
  - 外部 HTTP 2xx 响应自动标记 `submitted`；非 2xx 或请求失败标记 `failed` 并保存错误。
  - 解析通用响应中的 `id/key/url/externalId/externalKey/externalUrl` 等字段。
- 事件与审计：
  - 新增事件来源 `itsm`。
  - 新增事件类型：
    - `itsm.ticket.created`
    - `itsm.ticket.submitted`
    - `itsm.ticket.failed`
    - `itsm.ticket.updated`
  - ITSM 事件自动进入事件中心，并可继续被 Webhook/通知订阅。
- 前端页面：
  - 多云管理菜单新增“ITSM工单”。
  - 新增 `/org/:orgId/m-cloud-itsm` 页面。
  - 支持连接器管理、工单查询、工单详情、状态回写。
  - 事件中心识别 ITSM 来源和事件类型。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 执行 `gofmt`，覆盖新增和修改的 Go 文件。
- `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；仅保留既有 webpack 包体积警告。
- 使用 Docker Compose `up -d --force-recreate iac-portal iac-web` 重建服务通过。
- API `/api/v1/check` 返回：
  - `success=true`
  - `build=docker-compose`
- 数据库迁移验证：
  - `iac_cloud_itsm_config` 已创建。
  - `iac_cloud_itsm_ticket` 已创建。
- API 验证：
  - 创建临时通用 ITSM 连接器成功。
  - 从已有云操作任务 dry-run 创建工单成功，返回 `dryRunPayload.operation.id`。
  - 从已有云操作任务创建本地待提交工单成功，状态为 `pending`。
  - 回写工单状态为 `in_progress` 成功，外部单号 `CODX-1001` 已写入。
  - 查询工单列表成功，返回临时工单状态 `in_progress`。
  - 数据库验证 `source='itsm'` 的事件写入数为 `2`。
- 验证后已清理临时 ITSM 连接器、临时工单和对应 ITSM 事件：
  - 剩余临时连接器数为 `0`
  - 剩余临时工单数为 `0`
  - 剩余临时 ITSM 事件数为 `0`

验证限制：

- 当前阶段未对接真实 Jira 或 ServiceNow 专用字段映射，仅预留 provider 类型并实现通用 HTTP 对接。
- 当前本地验证使用离线本地待提交模式，没有访问外部 ITSM 系统。
- 外部 ITSM 成功和失败路径已通过通用 HTTP 投递器代码接入；真实系统字段映射、审批流回调和双向同步需在目标 ITSM 系统确认接口后做二期增强。

待继续：

- 当前 PRD 第一阶段多云管理闭环已完成；后续如接真实 Jira/ServiceNow、审批系统或自动双向同步，应作为二期需求单独规划。
- 评估旧任务通知是否复用 Webhook 投递队列和重试策略。

### 16.78 2026-06-20 V1.2 P1 项目级通知模板批量复制第一阶段

状态：已完成项目级通知模板批量复制第一阶段；项目设置的通知模板页可多选组织模板并批量复制到当前项目，复制结果会保留来源组织模板 ID。

已完成：

- 后端新增批量复制接口：
  - `POST /api/v1/notification-templates/copy-to-project`
- 批量复制请求体新增：
  - `sourceTemplateIds`
- 批量复制仅允许在项目上下文使用；组织上下文调用会返回参数错误。
- 批量复制事务内逐条处理组织模板：
  - 来源模板必须属于同一组织。
  - 来源模板必须是组织级活跃模板。
  - 重复传入的来源模板会跳过。
  - 当前项目已存在相同事件类型和通知类型的模板会跳过。
  - 新建项目模板会复制名称、事件类型、通知类型、标题、Markdown、文本和状态。
  - 新建项目模板会保存 `sourceTemplateId`。
- 批量复制响应返回：
  - `copied`
  - `skipped`
  - `items`
  - 每条结果包含来源模板 ID、项目模板 ID、事件类型、通知类型、状态和原因。
- 前端通知模板服务新增批量复制方法。
- 项目设置 `通知 -> 通知模板` 的组织模板表格新增多选能力：
  - 已覆盖模板的 checkbox 禁用。
  - 未选择模板时 `批量复制` 按钮禁用。
  - 选择组织模板后 `批量复制` 按钮启用。
  - 批量复制前弹出确认框，提示已覆盖模板会自动跳过。
  - 批量复制成功后展示新增数和跳过数。
  - 批量复制成功后刷新项目模板和组织模板列表。
- 单条 `复制` 操作改为复用批量复制接口，保持复制规则一致。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 临时目录执行 `gofmt`，覆盖本阶段修改的 Go 文件。
- `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；仅出现既有依赖废弃提示和 webpack 包体积 warning。
- 使用 Docker Compose 重启 `iac-portal`、`iac-web` 通过。
- API 创建 2 条临时组织级通知模板返回 `code=200`。
- API 首次调用批量复制返回：
  - `copied=2`
  - `skipped=0`
- API 第二次重复调用批量复制返回：
  - `copied=0`
  - `skipped=2`
- 查询项目模板确认 2 条项目模板的 `sourceTemplateId` 均正确回指对应组织模板。
- 验证后删除 API 临时项目模板和组织模板，活跃记录数为 `0`。
- 内置浏览器打开项目设置 `通知 -> 通知模板`：
  - 组织模板表格显示 checkbox。
  - 未选择时 `批量复制` 按钮禁用。
  - 勾选 2 条组织模板后 `批量复制` 按钮启用。
  - 点击 `批量复制` 后确认弹窗显示 `确定批量复制已选择的组织模板？`。
  - 点击确认后项目模板表格新增 2 条模板。
  - 新增项目模板的来源列显示对应组织模板名称。
  - 新增项目模板的差异列显示 `一致`。
  - 组织模板行显示 `已覆盖`，checkbox 禁用。
  - 成功提示显示 `新增2个，跳过0个`。
- 验证后删除浏览器临时项目模板和组织模板，活跃记录数为 `0`。

验证限制：

- 批量复制复制当前组织模板内容；复制动作的项目模板版本快照和来源组织模板快照已在 16.79 补齐。
- 回退组织模板仍通过删除项目模板回落到组织模板；历史项目模板版本回滚已在 16.79 补齐。
- 覆盖规则仍按事件类型和通知类型判断；如需覆盖已有项目模板，需要先编辑或删除项目模板。

待继续：

- 建设 ITSM 工单集成。
- 评估旧任务通知是否复用 Webhook 投递队列和重试策略。

### 16.79 2026-06-20 V1.2 P1 通知模板版本历史、来源快照和历史回滚第一阶段

状态：已完成通知模板版本历史、来源组织模板快照和历史版本回滚第一阶段；项目模板可查看版本链路，并可回滚到历史项目模板版本。

已完成：

- 新增通知模板版本模型：
  - `iac_notification_template_version`
- 版本表按模板记录：
  - 组织 ID。
  - 项目 ID。
  - 模板 ID。
  - 版本号。
  - 动作。
  - 模板字段快照。
  - 来源组织模板 ID。
  - 来源组织模板字段快照。
  - 操作人。
- 版本动作支持：
  - `create`
  - `copy`
  - `update`
  - `rollback`
- 模板创建成功后写入 `create` 版本。
- 组织模板批量复制到项目成功后写入 `copy` 版本，并保存来源组织模板快照。
- 模板编辑成功后写入 `update` 版本，并保存当前来源组织模板快照。
- 回滚历史版本成功后写入 `rollback` 版本。
- 新增模板版本列表 API：
  - `GET /api/v1/notification-templates/:id/versions`
- 新增历史版本回滚 API：
  - `POST /api/v1/notification-templates/:id/versions/:versionId/rollback`
- 版本列表和回滚均复用组织/项目作用域隔离：
  - 组织模板只能在组织上下文查看和回滚。
  - 项目模板只能在对应项目上下文查看和回滚。
- 回滚前检查同作用域下事件类型和通知类型是否会与其他模板冲突。
- 项目设置 `通知 -> 通知模板` 的项目模板行新增 `历史` 操作。
- 历史抽屉展示：
  - 版本号。
  - 动作。
  - 模板名称快照。
  - 来源模板快照。
  - 状态。
  - 操作人。
  - 版本时间。
  - 回滚操作。
- 历史版本回滚成功后刷新项目模板、组织模板和历史列表。
- 历史版本回滚成功后，历史抽屉标题同步为回滚后的模板名称。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 临时目录执行 `gofmt`，覆盖本阶段修改的 Go 文件。
- `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；仅出现既有依赖废弃提示和 webpack 包体积 warning。
- 使用 Docker Compose 重启 `iac-portal`、`iac-web` 通过。
- Portal 启动迁移后确认 `iac_notification_template_version` 表已创建。
- 确认 `iac_notification_template_version.source_name` 字段已创建。
- API 创建来源组织模板、批量复制为项目模板、编辑项目模板后，项目模板版本列表返回：
  - 回滚前版本总数为 `2`。
  - `v1` 动作为 `copy`。
  - `v2` 动作为 `update`。
  - `v1.sourceName` 等于来源组织模板名称。
- API 回滚到 `v1` 后：
  - 项目模板名称恢复为来源组织模板名称。
  - 版本总数变为 `3`。
  - `v3` 动作为 `rollback`。
- API 验证后删除临时模板并软删除临时版本记录，活跃模板和活跃版本数均为 `0`。
- 内置浏览器打开项目设置 `通知 -> 通知模板`：
  - 项目模板行显示 `历史` 操作。
  - 点击 `历史` 后打开历史抽屉。
  - 历史抽屉显示 `v2 更新` 和 `v1 复制`。
  - 历史抽屉显示来源快照名称。
  - 在历史抽屉点击 `v1` 的 `回滚` 并确认后，项目模板名称恢复为组织模板名称。
  - 回滚后历史抽屉显示 `v3 回滚`。
  - 回滚后页面不再显示旧项目模板名称。
- 浏览器验证后删除临时项目模板和组织模板，并软删除临时版本记录，活跃模板和活跃版本数均为 `0`。

验证限制：

- 当前版本快照保存模板文本字段和来源模板文本字段，尚未提供逐字段历史 diff 视图。
- 当前回滚为同步更新模板记录，尚未提供审批流或二次确认原因输入。
- 历史版本暂不随模板删除自动软删除；验证中通过数据库软删除临时版本记录清理测试数据。

待继续：

- 建设 ITSM 工单集成。
- 评估旧任务通知是否复用 Webhook 投递队列和重试策略。

### 16.80 2026-06-20 V1.2 P1 通知模板真实事件 Schema 动态变量字典第一阶段

状态：已完成通知模板真实事件 schema 动态变量字典第一阶段；通知模板变量字典和预览样例可从最近真实多云事件 payload 推导 `Payload.*` 变量。

已完成：

- 通知模板变量字典 API 继续复用：
  - `GET /api/v1/notification-templates/variables`
- 通知模板预览 API 继续复用：
  - `POST /api/v1/notification-templates/preview`
- 后端变量字典样例数据从静态样例扩展为真实事件优先：
  - 按当前组织和项目上下文查询 `iac_cloud_event`。
  - `eventType` 精确传入时按 `event_type` 过滤。
  - `cloud.*` 或未传事件类型时从最近多云事件中提取通用 payload 字段。
  - `xxx.*` 通配事件按来源或事件类型前缀过滤。
- 样例数据会使用最近真实事件覆盖：
  - `ViewUrl`
  - `Source`
  - `EventType`
  - `Level`
  - `Status`
  - `Provider`
  - `AccountId`
  - `Region`
  - `ResourceType`
  - `ResourceId`
  - `ResourceName`
  - `Title`
  - `Message`
  - `Payload`
- 新增动态变量组：
  - `eventPayload`
  - 页面展示名称为 `真实事件载荷变量`
- 动态 payload 变量命名规则：
  - 顶层字段渲染为 `{{.Payload.<key>}}`。
  - 二级嵌套字段渲染为 `{{.Payload.<parent>.<key>}}`。
  - 仅纳入 Go template 可安全访问的字段名。
  - 最近多条同类事件会合并字段集合，避免单条事件字段过少。
- 前端模板弹窗变量标签 tooltip 增加真实样例值展示：
  - `描述；示例：xxx`

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 临时目录执行 `gofmt`，覆盖 `backend/portal/apps/notification.go`。
- `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；仅出现既有依赖废弃提示和 webpack 包体积 warning。
- 使用 Docker Compose 重启 `iac-portal`、`iac-web` 通过。
- Docker Compose 服务状态确认：
  - `consul` healthy。
  - `mysql` healthy。
  - `iac-portal` healthy。
  - `iac-web` running。
  - `ct-runner` healthy。
- API 验证中插入一条临时云事件：
  - `eventType=cost.budget.threshold_exceeded`
  - `provider=aws`
  - `Payload.budgetName=codex-dynamic-budget`
  - `Payload.currentAmount=123.45`
  - `Payload.detail.owner=finops-team`
- 鉴权后调用变量字典 API 返回：
  - `eventPayload` 变量组。
  - `Payload.budgetName`
  - `Payload.currentAmount`
  - `Payload.detail.owner`
  - `Payload.detail.threshold`
  - `sampleData.Payload.budgetName=codex-dynamic-budget`
- 鉴权后调用预览 API 使用真实 payload 字段渲染：
  - 标题模板 `预算 {{.Payload.budgetName}} {{.Provider}}` 渲染为 `预算 codex-dynamic-budget aws`。
  - 文本模板 `owner={{.Payload.detail.owner}} amount={{.Payload.currentAmount}}` 渲染为 `owner=finops-team amount=123.45`。
  - Markdown 模板渲染出 `finops-team`。
- 验证后软删除临时云事件。
- 内置浏览器打开项目设置 `通知 -> 通知模板 -> 添加模板`：
  - 弹窗展示 `真实事件载荷变量` 分组。
  - 弹窗展示 `{{.Payload.*}}` 变量标签。
  - 变量 tooltip 展示从真实事件 payload 推导出的样例值。

验证限制：

- 第一阶段基于最近真实事件推导变量集合，不维护独立 schema registry。
- 对数组 payload 仅展示字段本身，不递归数组元素结构。
- 当前仍未提供前端自定义 sample payload 输入；预览优先使用真实事件样例，必要时可通过 API 的 `sampleData` 覆盖。

待继续：

- 建设 ITSM 工单集成。
- 评估旧任务通知是否复用 Webhook 投递队列和重试策略。

### 16.81 2026-06-20 V1.2 P1 磁盘扩容动作目录与 Dry-run 第一阶段

状态：已完成磁盘扩容动作目录与 dry-run 第一阶段；块存储资产详情可展示 `resize_volume` 动作，并在默认写保护下完成容量识别、参数提示、adapter 注册状态和执行保护提示。

已完成：

- 新增云资产动作常量：
  - `resize_volume`
- 新增动作目录配置：
  - 动作名称：磁盘扩容。
  - 支持资产类型：`block_volume`。
  - 支持云厂商：AWS、OCI、AliCloud。
  - 风险等级：高。
  - 执行方式：`provider_operation/provider`。
  - 需要审批。
- 新增 provider adapter 注册：
  - AWS：`aws_ebs_volume`
  - OCI：`oci_core_volume`
  - AliCloud：`alicloud_disk`
- 新增块存储 volume adapter 第一阶段：
  - 支持构建 provider request。
  - 支持从 CMDB 缓存识别当前容量。
  - 支持 dry-run 校验 `params.targetSizeGiB`。
  - live 只读和真实写执行器尚未开放时返回明确保护提示。
- 创建云资产动作表单新增通用参数：
  - `params`
- `resize_volume` 参数校验：
  - 必填 `params.targetSizeGiB`。
  - `targetSizeGiB` 必须为正整数。
  - 当可识别当前容量时，目标容量必须大于当前容量。
- 云操作重试链路会携带原始 `params`。
- 资产详情前端新增通用参数 JSON 输入框：
  - `resize_volume` 默认按当前容量加 10 GiB 生成 `targetSizeGiB`。
  - 无法识别当前容量时默认 `targetSizeGiB=100`。
  - 提交前校验 JSON 和目标容量。
- 操作任务列表新增 `resize_volume` 中文动作名。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 临时目录执行 `gofmt`，覆盖本阶段修改的 Go 文件。
- `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；仅出现既有依赖废弃提示和 webpack 包体积 warning。
- 使用 Docker Compose 重启 `iac-portal`、`iac-web` 通过。
- Docker Compose 服务状态确认：
  - `consul` healthy。
  - `mysql` healthy。
  - `iac-portal` healthy。
  - `iac-web` running。
  - `ct-runner` healthy。
- API 验证中插入临时 AWS EBS 卷资产：
  - `id=ci-codexvol620a`
  - `assetType=block_volume`
  - `nativeType=aws_ebs_volume`
  - `nativeId=vol-codex620a`
  - `attributes.sizeGiB=50`
- 鉴权后调用 `GET /api/v1/cloud/assets/:id/actions` 返回 `resize_volume`：
  - `name=磁盘扩容`
  - `providerAdapter=aws_ebs_volume`
  - `adapterStatus=registered`
  - `enabled=false`
  - `disabledReason` 提示 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED` 未开启。
- 鉴权后调用 `POST /api/v1/cloud/assets/:id/actions/resize_volume/dry-run` 返回：
  - `providerAdapter=aws_ebs_volume`
  - `executable=false`
  - `action_parameters` 提示当前识别容量约 `50 GiB`。
  - `adapter_registry` 通过并显示 `aws_ebs_volume`。
  - `execution_guard` 失败并提示写保护未开启。
  - `provider_volume_resize` 提示需要提交 `params.targetSizeGiB`。
- API 验证后删除临时资产。
- 内置浏览器打开云资产详情：
  - 临时块存储资产显示在云资产列表和详情中。
  - 资产详情 `操作` 页签显示 `磁盘扩容` 动作。
  - `磁盘扩容` 行显示 `aws_ebs_volume`、`已注册`、`需审批`。
  - `创建任务` 按钮在默认写保护下禁用。
  - 点击 `预检查` 后检查项显示：
    - `状态只读确认` 读取 `50GiB`。
    - `磁盘扩容校验` 提示 `params.targetSizeGiB`。
    - `适配器注册` 显示 `aws_ebs_volume`。
    - `执行保护` 提示 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED` 未开启。
    - `操作参数` 提示当前识别容量约 `50 GiB`。
- 浏览器验证后删除临时资产。

验证限制：

- 第一阶段只接入动作目录、参数校验、dry-run、前端参数输入和审计参数承载。
- 默认环境未开启真实云端写操作；未调用 AWS/OCI/AliCloud 块存储扩容 API。
- 块存储 live 只读查询和最终态轮询尚未接入，真实写执行器仍待 provider adapter 下一阶段实现。

待继续：

- 接入 AWS EBS、OCI Block Volume、AliCloud Disk 的 live 只读查询和扩容写执行器。
- 补充扩缩容、安全组等更多生命周期动作。
- 建设 ITSM 工单集成。
- 评估旧任务通知是否在下一阶段迁移到 Webhook 投递队列和重试策略。

### 16.82 2026-06-20 V1.2 P1 磁盘扩容参数化 Dry-run 第一阶段

状态：已完成磁盘扩容参数化 dry-run 第一阶段；资产详情可在点击 `磁盘扩容 -> 预检查` 时输入 `params.targetSizeGiB`，并把参数传入后端 provider transition 校验链路。

已完成：

- 新增 dry-run 请求表单：
  - `DryRunCloudAssetActionForm`
  - 支持 `params` 和 `tags` 请求体。
- dry-run URI 绑定改为直接声明 `Id`、`Action` 字段，避免嵌套表单导致 URI 参数未绑定。
- `DryRunCloudAssetAction` 会规范化 `params` 后传入 `cloudAssetActionDryRunWithParams`。
- provider preflight 校验支持接收 dry-run 参数：
  - `cloudAssetActionProviderPreflightChecks`
  - `cloudProviderOperationAdapterPreflightChecks`
- `resize_volume` 的 dry-run 参数校验增强：
  - 未传 `params.targetSizeGiB` 时保留 warn。
  - 目标容量小于等于当前容量时返回 fail。
  - 目标容量大于当前容量时返回 pass。
- 块存储 provider adapter 的 `provider_volume_resize` 检查可读取 dry-run 参数并输出 pass/fail。
- 前端资产详情 `操作` 页签增强：
  - `resize_volume` 点击 `预检查` 时弹出参数 JSON 弹窗。
  - 默认值按当前容量加 10 GiB 生成，例如当前 50 GiB 时默认 `targetSizeGiB=60`。
  - 提交前校验 JSON 和正整数。
  - 普通动作仍保持一键预检查。
- 云资产服务层 `dryRunAssetAction` 支持提交请求体。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 临时目录执行 `gofmt`，覆盖本阶段修改的 Go 文件。
- `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 构建 `iac-web` 通过；仅出现既有依赖废弃提示和 webpack 包体积 warning。
- 使用 Docker Compose 重启 `iac-portal`、`iac-web` 通过。
- 修正 dry-run URI 嵌套绑定问题后再次构建并重启 `iac-portal` 通过。
- Docker Compose 服务状态确认：
  - `consul` healthy。
  - `mysql` healthy。
  - `iac-portal` started and API `/api/v1/check` 返回 `success=true`。
  - `iac-web` running。
  - `ct-runner` healthy。
- API 验证中插入临时 AWS EBS 卷资产：
  - `id=ci-codexdry620`
  - `assetType=block_volume`
  - `nativeType=aws_ebs_volume`
  - `nativeId=vol-codexdry620`
  - `attributes.sizeGiB=50`
- 鉴权后调用 `POST /api/v1/cloud/assets/:id/actions/resize_volume/dry-run` 并提交 `{"params":{"targetSizeGiB":60}}` 返回：
  - `code=200`
  - `provider_volume_resize=pass`
  - `action_parameters=pass`
  - 消息包含 `当前容量 50 GiB，目标容量 60 GiB`
  - `execution_guard=fail`，继续提示写保护未开启。
- 鉴权后提交 `{"params":{"targetSizeGiB":40}}` 返回：
  - `code=200`
  - `provider_volume_resize=fail`
  - `action_parameters=fail`
  - 消息包含 `目标容量 40 GiB 必须大于当前容量 50 GiB`
- 鉴权后提交空请求体 `{}` 返回：
  - `code=200`
  - `provider_volume_resize=warn`
  - `action_parameters=warn`
  - 消息提示需要提交 `params.targetSizeGiB`。
- 内置浏览器打开云资产详情：
  - 资产详情 `操作` 页签显示 `磁盘扩容` 行。
  - 点击 `磁盘扩容` 的 `预检查` 后弹出参数 JSON 弹窗。
  - 弹窗默认参数为 `{"targetSizeGiB":60}`。
  - 点击弹窗 `预检查` 后弹窗关闭，预检查结果展示：
    - `状态只读确认` 通过，读取 `50GiB`。
    - `磁盘扩容校验` 通过，显示 `当前容量 50 GiB，目标容量 60 GiB`。
    - `适配器注册` 通过，显示 `aws_ebs_volume`。
    - `执行保护` 失败，显示 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED` 未开启。
    - `操作参数` 通过，显示参数校验通过。
- 浏览器/API 验证后删除临时资产，确认剩余记录数为 `0`。

验证限制：

- 本阶段只增强 dry-run 参数化校验，不开启真实云端写操作。
- 当前浏览器验证覆盖有效参数路径；无效目标容量已通过 API 验证。
- 块存储 live 只读查询、扩容写执行器和最终态轮询仍待 provider adapter 下一阶段实现。

待继续：

- 接入 AWS EBS、OCI Block Volume、AliCloud Disk 的 live 只读查询和扩容写执行器。
- 补充扩缩容、安全组等更多生命周期动作。
- 建设 ITSM 工单集成。
- 评估旧任务通知是否复用 Webhook 投递队列和重试策略。
### 16.83 2026-06-20 V1.2 P1 多云事件通知 Webhook 复用投递队列第一阶段

状态：已完成多云事件通知配置中的 Webhook 投递队列复用第一阶段；Webhook 类型通知会同步一条隐藏的多云 Webhook 镜像，并复用现有签名、投递记录、自动重试队列和死信能力。

已完成：

- 通知配置新增隐藏 Webhook 镜像同步：
  - 创建 Webhook 通知时同步创建 `iac_cloud_webhook` 镜像。
  - 更新 Webhook 通知时同步更新镜像名称、URL、事件类型和密钥。
  - 删除通知时软删除对应镜像。
  - 通知类型不再是 Webhook 或 URL 为空时自动删除镜像。
- 隐藏镜像标识：
  - `description` 使用 `[notification-webhook-mirror] notificationId=<id>` 前缀。
  - `id` 复用通知配置 ID，便于投递记录关联和重建。
- 普通 Webhook 管理视图过滤隐藏镜像：
  - `GET /api/v1/cloud/webhooks` 不展示通知镜像。
  - 普通 Webhook 的更新、删除、密钥轮换和测试接口只允许操作可见 Webhook。
- 多云事件通知发送链路增强：
  - `NotificationTypeWebhook` 不再直接同步调用原始 Webhook URL。
  - 投递前读取隐藏镜像，缺失时按通知事件配置补建。
  - 投递 payload 增加 `notification` 信息，包含通知 ID、类型、名称、标题和消息。
  - 初始投递复用 `deliverCloudWebhookWithOptions`，失败时进入现有自动重试队列。
- 多云 Webhook 正常事件分发排除隐藏通知镜像，避免同一事件在“事件中心 Webhook”和“通知 Webhook”两条链路重复投递。
- 自动重试、死信回放等内部路径继续使用基础 Webhook 查询，可按隐藏镜像 ID 消费队列。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 临时目录执行 `gofmt`，覆盖本阶段修改的 Go 文件。
- `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose 重启 `iac-portal` 通过。
- API `/api/v1/check` 返回：
  - `success=true`
  - `build=docker-compose`
- 创建临时 Webhook 通知：
  - `name=codex-webhook-queue-20260620`
  - `type=webhook`
  - `eventType=cost.budget.threshold_exceeded`
  - `url=http://127.0.0.1:1/codex-notification-webhook`
  - 返回 `code=200`
- 数据库验证隐藏镜像已同步：
  - `id=notif--d8rcfvnfa5vs73chuhr0`
  - `target_url=http://127.0.0.1:1/codex-notification-webhook`
  - `event_types=["cost.budget.threshold_exceeded"]`
  - `max_retries=3`
  - `retry_interval=60`
- 调用 `GET /api/v1/cloud/webhooks?pageSize=50`：
  - 返回 `code=200`
  - 列表只包含普通 Webhook `cwh-d8r5d5ucmfuc738bnang`
  - 不包含隐藏通知镜像 ID。
- 收紧管理接口后，调用隐藏镜像的普通 Webhook 测试接口：
  - `POST /api/v1/cloud/webhooks/notif--d8rcfvnfa5vs73chuhr0/test`
  - 返回 `40410013`
  - 消息为 `对象不存在或者无权限`
- 插入临时到期失败投递并调用正式重试接口：
  - 事件 ID：`evt-codexretry6201`
  - 初始失败投递 ID：`cwd-codexretry6201`
  - Webhook ID：`notif--d8rcfvnfa5vs73chuhr0`
  - 调用 `POST /api/v1/cloud/webhooks/retry-due` 返回 `code=200`
  - 队列项状态变为 `done`
  - 生成自动重试投递 `deliveryMode=auto`
  - 自动重试 attempt 为 `2`
  - 失败原因稳定为本地端口拒绝连接，说明没有依赖外网。
- 更新临时通知后验证镜像同步：
  - `target_url` 更新为 `http://127.0.0.1:2/codex-notification-webhook`
  - `event_types` 更新为 `["cost.*"]`
  - `secret_version` 从 `1` 更新为 `2`
  - `previous_secret` 保留旧密钥。
- 删除临时通知后验证：
  - 通知活跃记录数为 `0`
  - 通知事件记录数为 `0`
  - 隐藏镜像活跃记录数为 `0`
  - 本阶段专用测试事件、投递、队列和镜像物理清理后剩余记录数为 `0`。

验证限制：

- 本阶段覆盖的是多云事件通知链路中的 Webhook 类型通知。
- 旧任务通知服务 `services/notificationrc` 仍在服务包内直接发送 Webhook；要完全复用多云 Webhook 队列，需要后续将投递引擎下沉到无循环依赖的 service 包，或为任务通知建立独立队列适配层。
- 当前未新增前端页面；隐藏镜像由后端自动维护，普通 Webhook 列表保持不展示。

待继续：

- 接入 AWS EBS、OCI Block Volume、AliCloud Disk 的 live 只读查询和扩容写执行器。
- 补充扩缩容、安全组等更多生命周期动作。
- 建设 ITSM 工单集成。
- 下沉 Webhook 初始投递引擎，进一步统一旧任务通知和多云事件通知的签名、初始投递与重试记录。

### 16.84 2026-06-20 V1.2 P1 旧任务通知 Webhook 失败重试队列第一阶段

状态：已完成旧任务通知 Webhook 失败重试队列第一阶段；旧任务通知的 Webhook 初始发送仍保持原同步链路，发送失败后会写入统一 CloudWebhook delivery，并由现有重试 worker 接管后续重试。

已完成：

- 旧任务通知 `services/notificationrc` Webhook 发送入口增强：
  - `SendWebhookMessage` 增加标题参数，用于写入统一事件 payload。
  - 原同步 Webhook 发送行为保持不变。
  - 仅当同步发送失败时写入统一重试记录，成功发送不额外入队。
- 失败后自动维护隐藏 Webhook 镜像：
  - 历史 Webhook 通知配置缺少镜像时自动创建。
  - 镜像 ID 复用通知配置 ID。
  - 镜像 `description` 使用 `[notification-webhook-mirror] notificationId=<id>`。
  - 默认重试配置为 `maxRetries=3`、`retryInterval=60`。
- 失败后创建任务通知云事件：
  - `source=notification`
  - `eventType=task.failed` 等任务事件类型。
  - `resourceType=task`
  - `resourceId=<taskId>`
  - payload 包含通知信息和任务上下文。
- 失败后创建统一 Webhook delivery：
  - `status=failed`
  - `deliveryMode=initial`
  - `attempt=1`
  - `nextRetryAt` 按隐藏镜像 `retryInterval` 计算。
  - request payload 复用多云 Webhook 的 event/webhook 结构。
- 现有 `syncCloudWebhookDeliveryQueues` 和 retry worker 可直接发现旧任务通知失败投递，并生成 `deliveryMode=auto` 的后续重试。

验证：

- 使用 Docker Go 镜像经 `/private/tmp` 临时目录执行 `gofmt`，覆盖 `backend/portal/services/notificationrc/notificationservice.go`。
- `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose `up -d --force-recreate iac-portal` 重建 portal 容器通过。
- API `/api/v1/check` 返回：
  - `success=true`
  - `build=docker-compose`
- 使用 `/private/tmp` 临时源码副本运行一次性 Go 验证程序，直接调用旧任务通知入口：
  - 临时通知 ID：`notif-codextask620`
  - 事件类型：`task.failed`
  - 任务 ID：`run-codextask620`
  - Webhook URL：`http://127.0.0.1:1/codex-task-webhook`
- 一次性验证程序返回同步发送失败：
  - `connect: connection refused`
- 数据库验证隐藏镜像已创建：
  - `webhook=notif-codextask620`
  - `target=http://127.0.0.1:1/codex-task-webhook`
  - `retries=3`
  - `interval=60`
- 数据库验证初始失败投递已创建：
  - delivery ID：`cwd-d8rcqnmfksssgpd19cs0`
  - event ID：`evt-d8rcqnmfksssgpd19crg`
  - `status=failed`
  - `deliveryMode=initial`
  - `attempt=1`
  - `nextRetryAt` 已写入。
- retry worker 自动消费初始失败投递：
  - 队列项状态为 `done`
  - `result_delivery_id=cwd-d8rcqoav9fcc73ecktgg`
  - 生成自动重试投递 `deliveryMode=auto`
  - 自动重试 attempt 为 `2`
  - 失败原因稳定为本地端口拒绝连接。
- 数据库验证任务通知事件已写入：
  - `source=notification`
  - `event_type=task.failed`
  - `resource_id=run-codextask620`
  - `title=Codex task webhook retry`
- 调用普通 Webhook 列表验证隐藏镜像不可见：
  - `GET /api/v1/cloud/webhooks?pageSize=100`
  - 返回 `hiddenPresent=false`
- 验证后清理临时通知、通知事件、隐藏镜像、云事件、delivery、delivery queue 和 dead letter：
  - 剩余镜像数为 `0`
  - 剩余投递数为 `0`
  - 剩余事件数为 `0`
  - 剩余通知数为 `0`。

验证限制：

- 本阶段只把旧任务通知 Webhook 的失败重试接入统一队列；初始 HTTP 投递仍由 `notificationrc.Webhook` 同步执行。
- 旧任务通知初始发送暂未复用统一 CloudWebhook 签名头、签名上报 token 和完整 response header 记录。
- 完全统一初始发送需要把 CloudWebhook 投递引擎下沉到 service 层，避免 `apps -> notificationrc -> apps` 循环依赖。

待继续：

- 补充扩缩容、安全组等更多生命周期动作。
- 建设 ITSM 工单集成。
- 下沉 Webhook 投递引擎，统一旧任务通知的初始签名投递和多云事件 Webhook 投递。

### 16.85 2026-06-21 V1.2 P1 块存储 Provider Live 查询与扩容写执行器第一阶段

状态：已完成 AWS EBS、OCI Block Volume、AliCloud Disk 的块存储 live 只读查询和扩容写执行器第一阶段；`resize_volume` 已从动作目录、参数 dry-run 推进到 provider read/write/poll 代码路径。

已完成：

- 块存储 provider live 只读查询：
  - AWS EBS 使用 EC2 `DescribeVolumes` 读取 `status`、`size`、`volumeType`、`iops`、`availabilityZone`。
  - OCI Block Volume 使用 Core `GetVolume` 读取 `lifecycleState`、`sizeInGBs`、`vpusPerGB`、`availabilityDomain`。
  - 阿里云 Disk 使用 ECS `DescribeDisks` 读取 `Status`、`Size`、`Category`、`ZoneId`。
- 块存储容量归一化：
  - CMDB 缓存、provider 返回和 dry-run 状态统一解析为 `xxGiB`。
  - 支持 `size`、`sizeGiB`、`sizeInGBs`、`volumeSize` 等常见字段。
  - OCI `sizeInMBs`/`sizeMiB` 可按 1024 换算为 GiB。
- 扩容写执行器：
  - AWS EBS 接入 EC2 `ModifyVolume`，提交 `VolumeId` 和 `Size`。
  - OCI Block Volume 接入 Core `UpdateVolume`，提交 `sizeInGBs`。
  - 阿里云 Disk 接入 ECS `ResizeDisk`，提交 `DiskId` 和 `NewSize`。
- 写操作安全门禁：
  - 真实执行仍强制要求 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED=true`。
  - 真实执行仍强制要求 `CLOUD_OPERATION_PROVIDER_READ_MODE=live`。
  - 写前必须成功读取 provider live 状态。
  - 写前必须绑定 ready 云账号。
  - 当前容量未知时拒绝写入。
  - 目标容量等于当前容量时按幂等空操作处理。
  - 目标容量小于当前容量时拒绝执行，避免误触发缩容。
- 写后最终态确认：
  - 扩容提交后复用 provider poll 配置轮询 live 状态。
  - 当读取到当前容量大于等于目标容量时判定成功。
  - 轮询结果写入 `statePolling`，便于操作详情追踪每次读到的容量和状态。
- OCI 写请求签名增强：
  - GET 请求保持原有 `date/(request-target)/host` 签名。
  - POST/PUT 请求在设置 body hash 时额外签名 `x-content-sha256/content-type/content-length`。
  - 现有采集 GET 路径保持向后兼容。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 执行 `gofmt`，覆盖：
  - `backend/portal/apps/cloud_operation.go`
  - `backend/portal/apps/cloud_operation_provider_state.go`
  - `backend/portal/apps/cloud_operation_provider_execute.go`
  - `backend/portal/apps/cmdb_collect_oci.go`
- `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose `up -d --force-recreate iac-portal` 重建 portal 容器通过。
- API `/api/v1/check` 返回：
  - `success=true`
  - `build=docker-compose`
- 使用平台导入接口创建 3 条临时块存储资产：
  - AWS：`nativeId=vol-codex-resize-aws-20260621`，当前容量 `50 GiB`。
  - OCI：`nativeId=ocid1.volume.oc1..codexresize20260621`，当前容量 `80 GiB`。
  - 阿里云：`nativeId=d-codex-resize-aliyun-20260621`，当前容量 `120 GiB`。
- 动作目录验证：
  - AWS `resize_volume` 注册到 `providerAdapter=aws_ebs_volume`。
  - OCI `resize_volume` 注册到 `providerAdapter=oci_core_volume`。
  - 阿里云 `resize_volume` 注册到 `providerAdapter=alicloud_disk`。
  - 默认写保护下三者均显示 `writeEnabled=false`，符合安全门禁预期。
- dry-run 验证：
  - AWS 目标容量 `64 GiB`，`provider_volume_resize=pass`。
  - OCI 目标容量 `96 GiB`，`provider_volume_resize=pass`。
  - 阿里云目标容量 `140 GiB`，`provider_volume_resize=pass`。
  - 三者均可从 CMDB 缓存状态归一化为 `xxGiB` 并输出当前容量与目标容量。
- 验证后已清理临时块存储资产和对应资产变更记录。

验证限制：

- 当前本地环境未配置真实 AWS、OCI、阿里云生产凭证。
- 当前本地环境保持 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED` 未开启，真实云端扩容请求未发出。
- 本阶段验证覆盖编译、动作注册、缓存态 dry-run 和写保护门禁；provider live 查询、真实 resize 和写后轮询需要在配置真实云账号、开启 live read 和显式写保护开关的集成环境中复测。

待继续：

- 补充安全组规则写操作等更多生命周期动作。
- 建设 ITSM 工单集成。

### 16.87 2026-06-21 V1.2 P1 实例规格调整与块存储快照/备份生命周期动作第一阶段

状态：已完成实例规格调整和块存储快照/备份两个生命周期动作第一阶段；AWS、OCI、阿里云均已注册 provider adapter、dry-run 参数校验和写执行器分支，真实写操作默认继续受 provider 写保护开关和 live read 模式保护。

已完成：

- 新增云资产操作动作：
  - `resize_instance`：调整实例规格。
  - `create_snapshot`：创建块存储快照/备份。
- 实例规格调整：
  - AWS 通过 EC2 `ModifyInstanceAttribute` 调整 `InstanceType.Value`。
  - OCI 通过 `UpdateInstance` 调整 `shape`。
  - 阿里云通过 ECS `ModifyInstanceSpec` 调整 `InstanceType`。
  - 执行前要求实例处于停止状态。
  - 支持 `params.targetInstanceType` 参数校验。
  - provider live read 返回 `instanceType/shape`，写后轮询确认规格到达目标值。
- 快照/备份创建：
  - AWS 通过 EC2 `CreateSnapshot` 创建 EBS 快照，并支持 Name tag。
  - OCI 通过 `CreateVolumeBackup` 创建 Block Volume Backup，支持 `FULL/INCREMENTAL`。
  - 阿里云通过 ECS `CreateSnapshot` 创建磁盘快照。
  - 支持 `params.snapshotName`、`params.description`，OCI 支持 `params.backupType`。
  - 未传名称时生成 `cloudiac-<operation/resource>` 默认名称。
- 扩展动作目录：
  - `resize_instance` 支持 `compute_instance`。
  - `create_snapshot` 支持 `block_volume`。
  - 三云均返回明确的 `providerAdapter`。
- 扩展安全门禁：
  - 继续复用二次确认、项目/组织权限、云账号授权、资源标签授权、写保护开关、live read 模式和 provider 状态只读确认。
  - `resize_instance` 为高风险动作，需要审批。
  - `create_snapshot` 为中风险动作，项目负责人或操作员可申请。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 执行 `gofmt`，覆盖：
  - `backend/portal/models/cloud_operation.go`
  - `backend/portal/apps/cloud_operation.go`
  - `backend/portal/apps/cloud_operation_provider_state.go`
  - `backend/portal/apps/cloud_operation_provider_execute.go`
- `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose `up -d --force-recreate iac-portal` 重建 portal 容器通过。
- API `/api/v1/check` 返回：
  - `success=true`
  - `build=docker-compose`
- 使用平台导入接口创建 6 条临时资产：
  - AWS/OCI/阿里云各 1 条 `compute_instance`。
  - AWS/OCI/阿里云各 1 条 `block_volume`。
- 动作目录验证：
  - AWS `resize_instance` 注册到 `aws_ec2_instance`。
  - OCI `resize_instance` 注册到 `oci_compute_instance`。
  - 阿里云 `resize_instance` 注册到 `alicloud_ecs_instance`。
  - AWS `create_snapshot` 注册到 `aws_ebs_snapshot`。
  - OCI `create_snapshot` 注册到 `oci_block_volume_backup`。
  - 阿里云 `create_snapshot` 注册到 `alicloud_disk_snapshot`。
  - 默认写保护下均显示 `writeEnabled=false`，符合安全门禁预期。
- dry-run 验证：
  - 三云 `resize_instance` 的 `action_parameters=pass`。
  - 三云 `resize_instance` 的 `provider_state_transition=pass`。
  - 三云 `create_snapshot` 的 `action_parameters=pass`。
  - 三云 `create_snapshot` 的 `provider_snapshot_create=pass`。
- 验证后已清理临时资产和对应资产变更记录，剩余临时资产数为 `0`。

验证限制：

- 当前本地环境未配置真实 AWS、OCI、阿里云生产凭证。
- 当前本地环境保持 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED` 未开启，真实云端规格调整和快照/备份请求未发出。
- 本阶段验证覆盖编译、动作注册、缓存态 dry-run 和写保护门禁；provider live 查询、真实写操作和写后轮询需要在配置真实云账号、开启 live read 和显式写保护开关的集成环境中复测。

待继续：

- 补充安全组规则写操作等更多生命周期动作。
- 建设 ITSM 工单集成。

### 16.86 2026-06-21 V1.2 P1 Webhook 初始投递引擎下沉与旧任务通知统一投递第一阶段

状态：已完成 Webhook 初始投递引擎下沉第一阶段；多云事件 Webhook 与旧任务通知 Webhook 现在复用同一个 service 层投递器，旧任务通知初始投递也会写入统一 CloudWebhook delivery、签名头、response header 和重试队列。

已完成：

- 新增 `backend/portal/services/cloudwebhook` service 包：
  - 封装统一 Webhook payload 构造。
  - 封装 HMAC-SHA256 签名和签名版本头。
  - 封装签名回传 token 生成和 hash。
  - 封装 request/response headers 留痕。
  - 封装 `iac_cloud_webhook_delivery` 初始记录、状态更新、失败原因和 response body 记录。
  - 封装失败后 `iac_cloud_webhook_delivery_queue` 入队。
  - 封装 retry interval、指数退避、jitter 和最大重试窗口计算。
- `apps/cloud_webhook` 投递入口改为调用 service 层：
  - API 层继续保留 Webhook 管理、重试、死信、验签回传和失败事件记录。
  - `deliverCloudWebhookWithOptions` 只负责调用 service，并在最终失败时继续记录 dead letter 和 `webhook.delivery_failed` 事件。
  - `ensureCloudWebhookDeliveryQueue` 改为 service 包装，避免队列写入逻辑散落。
- 旧任务通知 `notificationrc.SendWebhookMessage` 改造：
  - 不再直接使用旧 `notificationrc.Webhook.Send` 同步裸发。
  - 发送前自动确保隐藏 Webhook 镜像存在。
  - 发送前创建 `source=notification` 的 `CloudEvent`。
  - 初始 HTTP 投递直接调用 `cloudwebhook.Deliver`。
  - 成功和失败初始投递均会记录 `iac_cloud_webhook_delivery`。
  - 失败初始投递自动写入 retry queue。
  - 旧任务通知自身的 `iac_notification_delivery` 仍按原入口记录成功/失败状态。
- 删除 `notificationrc` 中旧的手写失败入队函数，避免新旧两套队列写入逻辑并存。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 执行 `gofmt`，覆盖：
  - `backend/portal/apps/cloud_webhook.go`
  - `backend/portal/services/notificationrc/notificationservice.go`
  - `backend/portal/services/cloudwebhook/delivery.go`
- `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose `up -d --force-recreate iac-portal` 重建 portal 容器通过。
- 使用 `/private/tmp/cloudiac-task-webhook-unified` 临时源码副本运行一次性 Go 验证程序，直接调用旧任务通知 Webhook 入口：
  - 临时通知 ID：`notif-codexunified621`
  - 任务 ID：`run-codexunified621`
  - 事件类型：`task.failed`
  - Webhook URL：`http://127.0.0.1:1/codex-unified-task-webhook`
  - Secret：`codex-unified-secret`
- 一次性验证程序返回初始发送失败：
  - `connect: connection refused`
- 数据库验证统一初始投递已创建：
  - delivery ID：`cwd-d8rh94ui7eqcgt0lhlt0`
  - `status=failed`
  - `deliveryMode=initial`
  - `attempt=1`
  - `nextRetryAt` 已写入。
- 数据库验证签名信息已写入：
  - `signatureInfo.enabled=true`
  - `signatureInfo.algorithm=HMAC-SHA256`
  - `signatureInfo.reportTokenHeader=X-CloudIaC-Signature-Report-Token`
  - request headers 包含 `X-Cloudiac-Signature=sha256=<masked>`。
- 数据库验证 retry queue 已创建：
  - queue ID：`cwq-d8rh94ui7eqcgt0lhltg`
  - `status=queued`
  - `delivery_id=cwd-d8rh94ui7eqcgt0lhlt0`
  - `next_run_at` 与 delivery `next_retry_at` 一致。
- 验证后清理临时隐藏镜像、云事件、delivery、queue 和 dead letter：
  - 剩余 delivery 数为 `0`
  - 剩余 queue 数为 `0`
  - 剩余 webhook 数为 `0`

验证限制：

- 本阶段使用本地拒绝连接地址验证失败入队、签名和初始 delivery 留痕，没有依赖外网。
- 本阶段验证的是旧任务通知 Webhook 的失败初始投递路径；成功路径已复用同一 service 代码，但未额外启动临时 HTTP 接收器验证 2xx 成功记录。

待继续：

- 建设 ITSM 工单集成。

### 16.88 2026-06-21 V1.2 P1 安全组规则写操作生命周期动作第一阶段

状态：已完成安全组规则写操作第一阶段；新增 `update_security_rules` 生命周期动作，支持 AWS、OCI、阿里云三类 provider adapter 注册、参数校验、provider live read 和写执行器分支，真实写操作默认继续受 provider 写保护开关和 live read 模式保护。

已完成：

- 新增云资产操作动作：
  - `update_security_rules`：更新安全组规则。
- 参数模型：
  - `params.operation` 支持 `authorize/revoke`，兼容 `add/remove/delete` 等别名。
  - `params.direction` 支持 `ingress/egress`。
  - `params.protocol` 支持 `tcp/udp/icmp/all`。
  - `params.cidr` 支持新增或按 CIDR 撤销。
  - `params.fromPort/toPort`、`params.portRange` 支持 TCP/UDP 端口范围。
  - `params.description` 支持规则描述。
  - `params.ruleId` 支持 OCI NSG 按规则 ID 移除。
- provider adapter：
  - AWS 注册为 `aws_security_group`。
  - OCI 注册为 `oci_network_security_group_rules`。
  - 阿里云注册为 `alicloud_security_group`。
- provider live read：
  - AWS 使用 EC2 `DescribeSecurityGroups`。
  - OCI 支持 `GetNetworkSecurityGroup` 和 `GetSecurityList` 只读确认。
  - 阿里云使用 ECS `DescribeSecurityGroupAttribute`。
- provider 写执行器：
  - AWS 使用 `AuthorizeSecurityGroupIngress/Egress` 和 `RevokeSecurityGroupIngress/Egress`。
  - OCI NSG 使用 `addSecurityRules/removeSecurityRules` action；OCI Security List 因需要全量替换规则列表，当前真实写入明确阻断。
  - 阿里云使用 `AuthorizeSecurityGroup`、`AuthorizeSecurityGroupEgress`、`RevokeSecurityGroup`、`RevokeSecurityGroupEgress`。
- 安全门禁：
  - 继续复用二次确认、项目/组织权限、云账号授权、资源标签授权、写保护开关、live read 模式和 provider 状态只读确认。
  - `update_security_rules` 为高风险动作，需要审批。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 执行 `gofmt`，覆盖：
  - `backend/portal/models/cloud_operation.go`
  - `backend/portal/apps/cloud_operation.go`
  - `backend/portal/apps/cloud_operation_provider_state.go`
  - `backend/portal/apps/cloud_operation_provider_execute.go`
- `git diff --check` 通过。
- 使用 Docker Compose 构建 `iac-portal` 通过。
- 使用 Docker Compose `up -d --force-recreate iac-portal` 重建 portal 容器通过。
- API `/api/v1/check` 返回：
  - `success=true`
  - `build=docker-compose`
- 使用平台导入接口创建 3 条临时安全组资产：
  - AWS `network_security_group`。
  - OCI `network_security_group`。
  - 阿里云 `network_security_group`。
- 动作目录验证：
  - AWS `update_security_rules` 注册到 `aws_security_group`。
  - OCI `update_security_rules` 注册到 `oci_network_security_group_rules`。
  - 阿里云 `update_security_rules` 注册到 `alicloud_security_group`。
  - 默认写保护下均显示 `writeEnabled=false`，符合安全门禁预期。
- dry-run 验证：
  - 三云 `update_security_rules` 的 `action_parameters=pass`。
  - 三云 `update_security_rules` 的 `provider_security_rule_update=pass`。
- 验证后已清理临时安全组资产和对应资产变更记录，剩余临时资产数为 `0`。

验证限制：

- 当前本地环境未配置真实 AWS、OCI、阿里云生产凭证。
- 当前本地环境保持 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED` 未开启，真实云端安全组规则请求未发出。
- OCI Security List 写入需要全量替换规则列表，当前阶段不开放真实写入；OCI NSG 新增和按 `ruleId` 移除已接入执行器分支。
- 本阶段验证覆盖编译、动作注册、缓存态 dry-run 和写保护门禁；provider live 查询、真实写操作需要在配置真实云账号、开启 live read 和显式写保护开关的集成环境中复测。

待继续：

- ITSM 工单集成已在 16.89 完成；当前 PRD 第一阶段收口见 16.91。

### 16.91 2026-06-21 V1.2 P1 多云管理 PRD 第一阶段收口

状态：当前 PRD 第一阶段多云管理闭环已完成；16.89 已完成 ITSM 工单集成第一阶段，补齐连接器、工单、状态回写、事件沉淀和前端入口。

最终完成范围：

- 多云账号、云资产、成本、风险、事件、Webhook、通知投递、云操作生命周期动作、安全组规则写操作、ITSM 工单集成均已完成第一阶段闭环。
- ITSM 工单集成支持离线本地待提交工单和通用 HTTP 外部系统对接。
- 新增 `/org/:orgId/m-cloud-itsm` 前端页面，并接入多云管理菜单。

最终验证：

- `git diff --check` 通过。
- `iac-portal` Docker Compose 构建通过。
- `iac-web` Docker Compose 构建通过；仅有既有 webpack 包体积警告。
- Docker Compose 已重建 `iac-portal` 和 `iac-web`。
- `/api/v1/check` 返回 `success=true`、`build=docker-compose`。
- 浏览器验证 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-itsm` 可访问，页面非登录态，能看到 ITSM 标题、工单和连接器入口。

二期建议：

- 真实 Jira/ServiceNow 字段映射和回调验签。
- 审批系统与 ITSM 双向状态同步。
- 外部工单系统成功/失败路径在真实环境的集成测试。

### 16.92 2026-06-21 V1.2 P1/P2 AliCloud、Azure、GCP 云采集器第一阶段

状态：已完成 AliCloud、Azure、GCP 云采集器第一阶段，并完成腾讯云/华为云离线 inventory collector 第一阶段；AliCloud 从 placeholder 推进为可执行 collector，Azure/GCP 从无 collector 推进为基于官方 REST API 的轻量采集路径，腾讯云/华为云先以本地 JSON 清单满足离线环境采集闭环。

已完成：

- AliCloud collector：
  - 接入 `collectCmdbAlicloudAssets`，支持按区域、资产类型执行。
  - 覆盖 ECS、Disk、VPC、VSwitch、SecurityGroup、EIP、SLB、RDS、Redis、OSS、ACK。
  - 抽象 `alicloudRPCServiceAPI`，复用既有阿里云 RPC 签名能力，并让云操作 live read 继续通过统一 ECS RPC helper。
  - `supportedCmdbCloudAssetTypes("alicloud")` 已补齐 CMDB 标准类型。
- Azure collector：
  - 新增 `collectCmdbAzureAssets`，使用 Azure AD OAuth2 client credentials 获取 ARM token。
  - 使用订阅级 ARM Resources API 读取资源列表，并归一化到 CMDB 资产。
  - 覆盖 VM、VNet、Subnet、NSG、Public IP、Disk、Load Balancer、AKS、SQL/MySQL/PostgreSQL、Storage Account。
  - NSG 规则归一化为 `ingressSecurityRules` / `egressSecurityRules`，复用安全组规则视图。
  - 云账号中心新增 Azure 前端选项和凭证模板。
- GCP collector：
  - 新增 `collectCmdbGcpAssets`，支持 `GCP_ACCESS_TOKEN` 或 `GCP_SERVICE_ACCOUNT_JSON` 生成 OAuth token。
  - 通过 Compute、Container、SQL Admin、Storage JSON API 采集基础资产。
  - 覆盖 Compute Instance、VPC Network、Subnetwork、Firewall、Disk、Forwarding Rule、GKE、Cloud SQL、Bucket。
  - Firewall 规则归一化为安全规则字段，复用现有公网暴露识别。
  - 云账号中心新增 GCP 前端选项和凭证模板。
- 腾讯云/华为云离线 inventory collector：
  - 新增 `collectCmdbInventoryAssets`，支持 `tencentcloud`、`huawei`。
  - 支持直接 JSON 数组或 `{ "assets": [...] }` 两种清单格式。
  - 支持账号字段：
    - 腾讯云：`TENCENTCLOUD_INVENTORY_JSON`、`TENCENT_INVENTORY_JSON`、`CLOUD_INVENTORY_JSON`。
    - 华为云：`HUAWEI_INVENTORY_JSON`、`HUAWEICLOUD_INVENTORY_JSON`、`CLOUD_INVENTORY_JSON`。
  - 覆盖 CMDB 标准类型：计算、Kubernetes、VPC、Subnet、安全组、公网 IP、负载均衡、块存储、关系型数据库、Redis、对象存储。
  - 云账号中心新增腾讯云、华为云前端选项和离线 inventory 凭证模板。
- 同步任务和账号模型：
  - `collectCmdbCloudAssets` 已注册 `alicloud`、`azure`、`gcp`、`tencentcloud`、`huawei`。
  - `inferCloudProvider`、`inferCloudRegions`、`requiredCloudCredentialKeys`、`requiredCloudCredentialKeyGroups`、`credentialMapWithRegions`、`inferCloudAccountId` 已补齐 Azure/GCP/腾讯云/华为云。
  - 同步任务表单 `provider` 枚举已支持 `azure/gcp/tencentcloud/huawei`。
- ITSM 连接器编辑修复：
  - 前端编辑已有连接器时，空 `token/password` 不再提交给后端，避免覆盖已配置密钥。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 执行 `gofmt`，覆盖新增和修改的 Go 文件。
- `iac-portal` Docker Compose 构建通过。
- `iac-web` Docker Compose 构建通过；仅有既有 webpack 包体积 warning。
- Docker Compose 已重建 `iac-portal` 和 `iac-web`。
- `/api/v1/check` 返回：
  - `success=true`
  - `build=docker-compose`
- Azure 临时账号验证：
  - provider 返回 `azure`。
  - supportedTypes 返回 `compute_instance,kubernetes_cluster,network_vpc,network_subnet,network_security_group,public_ip,load_balancer,block_volume,relational_database,object_storage_bucket`。
  - `AZURE_CLIENT_SECRET` 未明文回显。
  - 使用假租户启动同步任务时，任务进入 `failed`，阶段日志完整记录 token 获取失败，未 panic。
  - 验证后已删除临时 Azure 云账号。
- GCP 临时账号验证：
  - provider 返回 `gcp`。
  - supportedTypes 返回 `compute_instance,kubernetes_cluster,network_vpc,network_subnet,network_security_group,load_balancer,block_volume,relational_database,object_storage_bucket`。
  - `GCP_ACCESS_TOKEN` 未明文回显。
  - 显式传 `provider=gcp` 可创建同步任务；使用假 token 时任务进入 `failed`，阶段日志完整记录鉴权失败，未 panic。
  - 验证后已删除临时 GCP 云账号。
- 腾讯云临时账号验证：
  - provider 返回 `tencentcloud`。
  - `TENCENTCLOUD_INVENTORY_JSON` 未明文回显。
  - 使用本地 JSON 清单同步 1 条 `compute_instance`，同步任务 `complete`，`collected=1`、`created=1`、`mode=offline_inventory`。
  - 验证后已删除临时腾讯云账号、临时资产、同步任务日志和事件记录。
- 华为云临时账号验证：
  - provider 返回 `huawei`。
  - `HUAWEI_INVENTORY_JSON` 未明文回显。
  - 使用本地 JSON 清单同步 1 条 `compute_instance`，同步任务 `complete`，`collected=1`、`created=1`、`mode=offline_inventory`。
  - 验证后已删除临时华为云账号、临时资产、同步任务日志和事件记录。

验证限制：

- 当前本地环境没有真实 Azure/GCP/AliCloud/TencentCloud/HuaweiCloud 生产凭证，真实资源采集需要在具备只读权限的账号中复测。
- Azure 第一阶段采用订阅级 ARM Resources API 做通用资源归一化，VM 网卡、子网、NSG、公网 IP 和磁盘引用已在 16.95 补充第一阶段展开；路由表、NAT、网关和更细 LB 后端关系仍待继续。
- GCP 第一阶段覆盖基础 API，分页和跨项目聚合可继续增强。
- 腾讯云/华为云离线 inventory collector 已保留为无公网或无云端凭证环境的回退路径；真实云 API collector 第一阶段见 16.94。

待继续：

- Azure/GCP 成本拉取已在 16.98 支持导出文件索引 URL 和增量游标，在 16.99 支持 Azure Blob/GCS/S3 对象存储原生列表 API 第一阶段，在 16.101 支持对象存储签名授权与分页第一阶段；真实云环境联调仍待继续。
- 补齐路由表、NAT、网关、LB 后端池、跨项目/跨订阅等更细网络拓扑关系。

### 16.93 2026-06-21 V1.2 P2 Azure/GCP 成本导入与计算实例操作第一阶段

状态：已完成 Azure/GCP 成本导入型数据源、成本异常规则复用、计算实例 provider live read 和低风险生命周期动作第一阶段；Azure/GCP 不再停留在只读资产采集，已进入成本与受控操作闭环。

本阶段新增：

- 成本导入 API：
  - 新增 `POST /api/v1/cloud/cost/import`，支持导入统一字段账单 JSON 数组或来自 Azure/GCP 导出的记录。
  - 新增成本来源 `azure_billing_export`、`gcp_billing_export` 和通用 `import`。
  - 导入逻辑会从 `resourceId`、`ResourceId`、`CostInBillingCurrency`、`service.description`、`project.id`、`BillingCurrencyCode`、`usage_start_time` 等字段推断资源 ID、服务、金额、币种、账期和账号。
  - 成本导入后会复用统一成本记录 upsert、CMDB 资产匹配、未匹配成本、高成本资源、缺少 Owner 和缺少成本中心等成本异常规则。
- 成本中心前端：
  - 新增“导入账单”入口，可粘贴 JSON 数组或 `{ records: [...] }`。
  - 支持选择 Azure/GCP、账期、币种、账号/订阅/项目 ID 和区域。
  - 成本记录筛选新增导入来源标签。
- Azure/GCP provider operation adapter：
  - 云资产动作目录中，Azure/GCP 计算实例支持 `start_instance`、`stop_instance`、`restart_instance`。
  - Azure 注册 `azure_virtual_machine` adapter，GCP 注册 `gcp_compute_instance` adapter。
  - 继续复用二次确认、项目/组织权限、云账号授权、资源标签授权、写保护开关、缓存/实时读取模式和操作审计。
- Azure/GCP provider live read：
  - Azure 使用 ARM VM `instanceView` 读取 `PowerState/*` 状态并归一化为 `running/stopped/starting/stopping`。
  - GCP 使用 Compute Engine `instances.get` 读取实例状态，支持从 selfLink 解析 `zone/instance`，并将 GCP `TERMINATED` 归一化为 `stopped`。
- 受保护写执行器：
  - Azure 写执行器支持 `start`、`powerOff`、`restart`。
  - GCP 写执行器支持 `start`、`stop`、`reset`。
  - 默认仍被 `CLOUD_OPERATION_PROVIDER_WRITE_ENABLED` 拦截，真实执行还要求 `CLOUD_OPERATION_PROVIDER_READ_MODE=live`。

验证：

- `docker compose build iac-portal` 成功。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功，`GET /api/v1/check` 返回 `build=docker-compose`、`version=v1.3.5`。
- `POST /api/v1/cloud/cost/import` 导入 1 条 Azure 和 1 条 GCP 验证成本，返回 `imported=2`、`providers=[azure,gcp]`、`sources=[azure_billing_export,gcp_billing_export]`。
- `GET /api/v1/cloud/cost/records?q=codex-validation&period=2026-06&currency=USD` 返回 2 条导入记录。
- `GET /api/v1/cloud/cost/insights?period=2026-06&currency=USD&q=codex` 返回未匹配成本和高成本资源建议，证明导入成本已进入异常规则。
- 创建临时 Azure/GCP 云账号和计算实例资产后，动作目录显示：
  - Azure：`start_instance/stop_instance/restart_instance` 使用 `azure_virtual_machine` adapter。
  - GCP：`start_instance/stop_instance/restart_instance` 使用 `gcp_compute_instance` adapter。
- Azure/GCP `start_instance` dry-run 均通过账号、凭证、区域、缓存态状态归一化和 adapter registry 检查，并在 `execution_guard` 被写保护开关拦截。
- 验证后已删除临时 Azure/GCP 云账号、临时资产、临时成本记录、成本建议和相关事件。

验证限制：

- 本地验证未配置真实 Azure/GCP 生产凭证，因此 live read 和真实写操作只完成编译、注册、缓存态 dry-run 与保护门禁验证。
- Azure/GCP 成本已在 16.95 补充手动拉取入口，在 16.96 补充同步任务、任务日志、失败重试和事件审计，在 16.97 补充同步计划、后台 worker 和前端配置入口，在 16.98 补充导出文件索引 URL 和增量游标，在 16.99 补充 Azure Blob/GCS/S3 对象存储原生列表 API 第一阶段，并在 16.100 补充计划级失败退避、通知事件和自动暂停第一阶段。

待继续：

- 将对象存储列表 API 的签名授权与分页第一阶段扩展到真实云环境联调、provider 原始错误映射和更多目录规则。
- 补齐路由表、NAT、网关、LB 后端池等更细网络拓扑关系。
- 在真实腾讯云/华为云账单环境补充端到端联调，并扩展 CSV 字段差异、Parquet/Excel、分区目录和 provider 错误码映射。

### 16.94 2026-06-21 V1.2 P2 腾讯云/华为云真实云 API collector 第一阶段

状态：已完成腾讯云和华为云真实云 API collector 第一阶段，并保留离线 inventory 作为回退路径；多云资产采集不再只覆盖 AWS/OCI/AliCloud/Azure/GCP。

本阶段新增：

- 腾讯云 collector：
  - 新增 `collectCmdbTencentAssets`，有 `TENCENTCLOUD_SECRET_ID` + `TENCENTCLOUD_SECRET_KEY` 时使用腾讯云 TC3-HMAC-SHA256 签名访问云 API。
  - 没有真实 API 凭证时继续使用 `TENCENTCLOUD_INVENTORY_JSON` / `TENCENT_INVENTORY_JSON` / `CLOUD_INVENTORY_JSON` 离线清单。
  - 覆盖 CVM、VPC、Subnet、安全组、EIP、CBS、CLB、TKE、CDB、Redis。
  - 云账号中心新增 `TENCENTCLOUD_SECRET_ID`、`TENCENTCLOUD_SECRET_KEY`、`TENCENTCLOUD_TOKEN` 凭证模板字段。
- 华为云 collector：
  - 新增 `collectCmdbHuaweiAssets`，有 `HUAWEI_AUTH_TOKEN` + `HUAWEI_PROJECT_ID` 时使用 `X-Auth-Token` 访问华为云 REST API。
  - 没有真实 API 凭证时继续使用 `HUAWEI_INVENTORY_JSON` / `HUAWEICLOUD_INVENTORY_JSON` / `CLOUD_INVENTORY_JSON` 离线清单。
  - 覆盖 ECS、VPC、Subnet、安全组、EIP、EVS、ELB、CCE、RDS、DCS Redis。
  - 云账号中心新增 `HUAWEI_PROJECT_ID`、`HUAWEI_AUTH_TOKEN`、`HUAWEI_ENDPOINT_SUFFIX` 凭证模板字段。
- 同步任务和账号校验：
  - `collectCmdbCloudAssets` 已将 `tencentcloud`、`huawei` 从纯 inventory 分支切换为“真实 API 优先，inventory 回退”。
  - `missingCloudCredentialKeys` 支持腾讯云、华为云真实 API 凭证与离线 inventory 二选一校验。

验证：

- `git diff --check` 通过。
- `docker compose build iac-portal` 成功。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功。
- `/api/v1/check` 返回 `success=true`、`build=docker-compose`、`version=v1.3.5`。
- web 首页返回 `HTTP/1.1 200 OK`。
- 腾讯云真实 API 假密钥路径：
  - 临时账号 provider 返回 `tencentcloud`，`validationStatus=valid`。
  - `TENCENTCLOUD_SECRET_KEY` 未明文回显。
  - 使用假 SecretId/SecretKey 采集 `compute_instance` 时，同步任务进入 `failed`，错误来自腾讯云 `AuthFailure.SecretIdNotFound`，平台无 panic。
- 腾讯云离线 inventory 回退路径：
  - `TENCENTCLOUD_INVENTORY_JSON` 未明文回显。
  - 使用本地 JSON 清单同步 1 条 `compute_instance`，同步任务 `complete`，`collected=1`、`created=1`、`mode=offline_inventory`。
- 华为云真实 API 假 Token 路径：
  - 临时账号 provider 返回 `huawei`，`validationStatus=valid`。
  - `HUAWEI_AUTH_TOKEN` 未明文回显。
  - 使用假 Token 采集 `compute_instance` 时，同步任务进入 `failed`，错误来自华为云 `APIGW.0301` 鉴权失败，平台无 panic。
- 华为云离线 inventory 回退路径：
  - `HUAWEI_INVENTORY_JSON` 未明文回显。
  - 使用本地 JSON 清单同步 1 条 `compute_instance`，同步任务 `complete`，`collected=1`、`created=1`、`mode=offline_inventory`。
- 验证后已删除临时腾讯云/华为云账号、临时资产、同步任务日志和事件记录，剩余临时账号和资产数均为 `0`。

验证限制：

- 当前本地环境没有真实腾讯云、华为云生产凭证，因此真实云 API 路径完成的是签名/鉴权请求、失败可观测、账号脱敏、任务状态和错误沉淀验证。
- 腾讯云 COS、华为云 OBS 桶列表鉴权和端点形态已在 16.95 以专项 collector 第一阶段补齐。
- 华为云第一阶段使用显式 `HUAWEI_AUTH_TOKEN`，后续可继续扩展 AK/SK 签名或 IAM 用户名密码换 token。

待继续：

- 将对象存储列表 API 的签名授权与分页第一阶段扩展到真实云环境联调、provider 原始错误映射和更多目录规则。
- 补齐路由表、NAT、网关、LB 后端池、跨项目/跨订阅等更细网络拓扑关系。
- 在真实腾讯云/华为云账单环境补充端到端联调，并扩展 CSV 字段差异、Parquet/Excel、分区目录和 provider 错误码映射。

### 16.95 2026-06-21 V1.2 P2 成本拉取、云侧关系推演与 COS/OBS 对象存储 collector 第一阶段

状态：已完成 Azure/GCP 成本拉取入口、云采集资产网络/安全/存储关系推演、腾讯云 COS 和华为云 OBS 对象存储桶 collector 第一阶段；PRD 中 16.92-16.94 标记的“成本拉取、Azure/GCP 拓扑精细关系、COS/OBS 桶采集”已从待开发推进为可验证闭环。

本阶段新增：

- Azure/GCP 成本拉取：
  - 新增 `POST /api/v1/cloud/cost/pull`，支持 Azure Cost Management API 拉取和 GCP Billing Export URL 拉取。
  - 成本中心新增“拉取账单”弹窗，支持选择 provider、source、云账号、导出 URL、账期、币种、账号/项目 ID 和区域。
  - `azure_cost_management` 纳入统一成本来源枚举，拉取后复用成本明细 upsert、资产匹配和成本异常规则。
  - GCP Billing Export 解析优先识别 `service.description`、`sku.description` 和 `project.id` 等嵌套字段，避免服务名被记录为 map 字符串。
- 云侧关系推演：
  - 新增 `rebuildCmdbCloudInferredRelations`，同步任务 upsert 云资产后删除并重建同账号 `cloud_inferred` 关系。
  - 支持从 `vpcId/network/vnetId/subnetIds/securityGroupIds/publicIpIds/diskIds/users` 等标准 attributes 推演 `contains` 和 `depends_on`。
  - Azure collector 增强 VM 网卡展开，补充 VM 到 Subnet/NSG/Public IP/Disk 的引用；Subnet 补充父 VNet；LB、AKS、数据库补充网络引用。
  - GCP collector 增强 Compute/GKE/SQL/LB/Disk 的 network、subnetwork、disk users 等扁平引用。
  - 同步任务 `stats.cloudInferredRelations` 记录本次重建的云侧关系数量。
- 腾讯云 COS collector：
  - 新增 COS XML API ListBuckets 采集，支持 `TENCENTCLOUD_COS_ENDPOINT` 覆盖 endpoint。
  - 云账号凭据模板新增 `TENCENTCLOUD_COS_ENDPOINT`。
  - COS bucket 归一化为 `object_storage_bucket / tencentcloud_cos_bucket`。
- 华为云 OBS collector：
  - 新增 OBS AK/SK ListBuckets 采集，支持 `HUAWEI_ACCESS_KEY`、`HUAWEI_SECRET_KEY` 和 `HUAWEI_OBS_ENDPOINT`。
  - 华为云 Token/Project 继续用于 ECS/VPC/RDS 等区域 REST API；OBS AK/SK 独立用于对象存储桶采集。
  - 云账号凭据模板新增 `HUAWEI_ACCESS_KEY`、`HUAWEI_SECRET_KEY`、`HUAWEI_OBS_ENDPOINT`。

验证：

- `git diff --check` 通过。
- `docker compose build iac-portal` 成功。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功。
- `/api/v1/check` 返回 `success=true`、`build=docker-compose`、`version=v1.3.5`，web 首页返回 `HTTP/1.1 200 OK`。
- Azure 成本拉取假凭据路径：
  - 创建临时 Azure 云账号后，凭据详情未明文回显 `codex-secret`。
  - 无 source URL 时进入 Azure token 请求路径，并以 Azure `unauthorized_client` 业务错误返回，平台无 panic。
- GCP Billing Export URL 拉取：
  - 使用本地导出 URL 拉取 1 条 GCP 成本，返回 `imported=1`、`providers=[gcp]`、`sources=[gcp_billing_export]`、`mode=export_url`。
  - 成本明细服务名为 `Compute Engine`，资源 ID 为 `//compute.googleapis.com/projects/codex-cost-project/zones/us-central1-a/instances/codex-gcp-url-vm`。
- 云侧关系推演：
  - 使用临时腾讯云 offline inventory 同步 VPC、Subnet、SecurityGroup、Disk、Instance 共 5 个资产。
  - 同步任务 `complete`，`collected=5`、`created=5`、`cloudInferredRelations=6`。
  - 实例详情关系展示 VPC/子网 `contains` 实例，以及实例 `depends_on` 安全组和磁盘。
- COS/OBS collector：
  - 使用本地 XML endpoint 验证腾讯云 COS ListBuckets，任务 `complete`，采集 `codex-cos-bucket`，类型为 `object_storage_bucket / tencentcloud_cos_bucket`。
  - 使用本地 XML endpoint 验证华为云 OBS ListBuckets，任务 `complete`，采集 `codex-obs-bucket`，类型为 `object_storage_bucket / huaweicloud_obs_bucket`。

验证限制：

- 本地未配置真实 Azure/GCP/腾讯云/华为云生产凭证；Azure Cost Management、腾讯 COS、华为 OBS 均完成协议路径、签名/请求构造、可控错误或本地 endpoint 正向解析验证，生产账号仍需用只读权限复测。
- 云侧关系推演第一阶段聚焦 VPC/VNet、Subnet、NSG/SecurityGroup、Public IP、Disk、LB、Kubernetes、Database/Redis 的常见引用；路由表、NAT、IGW、LB 后端池、跨项目/跨订阅聚合和图形化拓扑视图仍待继续。
- 成本拉取目前支持手动 API/页面入口；同步任务、任务日志、失败重试和事件审计已在 16.96 补齐第一阶段，同步计划、后台 worker 和前端配置入口已在 16.97 补齐第一阶段，导出文件索引 URL 和增量游标已在 16.98 补齐第一阶段，Azure Blob/GCS/S3 对象存储原生列表 API 已在 16.99 补齐第一阶段，计划级失败退避、通知事件和自动暂停已在 16.100 补齐第一阶段，对象存储签名授权与分页已在 16.101 补齐第一阶段；真实云环境联调仍待继续。

待继续：

- 对象存储列表 API 继续补齐真实云环境联调、provider SDK/REST 错误映射、ETag 和更复杂目录规则。
- 路由表、NAT、IGW、LB 后端池、跨项目/跨订阅等更细拓扑关系和可视化拓扑页面。
- 腾讯云/华为云成本账单真实环境联调、成本异常阈值配置和更细 FinOps 规则。

### 16.96 2026-06-21 V1.2 P2 成本拉取同步任务、日志与失败重试第一阶段

状态：已完成 Azure/GCP 成本拉取同步任务第一阶段；手动拉取不再只是一次性 API 调用，已经具备任务记录、任务日志、失败重试、成功/失败事件审计和前端可观测入口。

本阶段新增：

- 成本同步任务模型：
  - 新增 `iac_cloud_cost_sync_task`，记录 org、创建人、云账号、provider、账期、币种、来源、模式、状态、导入数量、匹配数量、尝试次数、错误、参数和结果。
  - 新增 `iac_cloud_cost_sync_task_log`，记录每次拉取执行阶段、状态、消息、错误、结果和时间。
- 成本同步任务 API：
  - `GET /api/v1/cloud/cost/sync-tasks`：查询任务列表，支持关键字、状态、provider、source、云账号、账号和账期过滤。
  - `POST /api/v1/cloud/cost/sync-tasks`：创建并同步执行一次成本拉取任务。
  - `GET /api/v1/cloud/cost/sync-tasks/:id`：查看任务详情和日志。
  - `POST /api/v1/cloud/cost/sync-tasks/:id/retry`：失败任务重试，复用原拉取参数并递增尝试次数。
- 事件审计增强：
  - 成本拉取成功写入 `cost.pull.completed`，失败写入 `cost.pull.failed`。
  - 事件 payload 带 `syncTaskId`、provider、source、mode、账期、币种、导入数量、匹配数量和错误摘要。
  - URL 类参数只展示脱敏预览，带 query 的签名 URL 会显示为 `?...`。
- 前端成本中心增强：
  - 成本中心新增“账单同步任务”面板，展示最近任务、状态、来源、账期、导入数量、尝试次数、时间、错误和重试入口。
  - 任务详情抽屉展示任务基础信息、任务日志和执行结果。
  - 原“拉取账单”按钮改为创建成本同步任务，保留 `/cloud/cost/pull` 作为兼容 API。

验证：

- `docker compose build iac-portal` 成功。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功。
- `/api/v1/check` 返回 `success=true`、`build=docker-compose`、`version=v1.3.5`。
- DB 自动迁移确认存在：
  - `iac_cloud_cost_sync_task`
  - `iac_cloud_cost_sync_task_log`
- 使用本地 GCP Billing Export JSON endpoint 创建成功任务：
  - 任务 `status=complete`、`imported=1`、`source=gcp_billing_export`、`mode=export_url`、日志 2 条。
  - 成本记录服务名为 `Compute Engine`，资源为 `codex-cost-sync-vm`。
  - 事件中心写入 `cost.pull.completed`，payload 包含 `syncTaskId`。
- 使用本地 404 endpoint 创建失败任务并重试：
  - 首次任务 `status=failed`、`attemptCount=1`、日志 2 条。
  - 重试后 `attemptCount=2`、日志 4 条。
  - 事件中心写入 `cost.pull.failed`，payload 包含 `syncTaskId`。
- 使用带 query 的 URL 验证脱敏：
  - `sourceUrlPreview=http://host.docker.internal:18081/missing-cost.json?...`。
  - 404 HTML 错误被压缩为一行短摘要，避免污染任务列表和事件载荷。
- 内置浏览器验证成本中心页面：
  - 页面展示“账单同步任务”“刷新任务”和“拉取账单”。
  - 切换账期到 `2026-07` 后，成功任务和失败任务均在任务表中渲染。
- 验证后已清理本轮临时成本记录、成本建议、事件、同步任务和任务日志，剩余临时数据计数均为 `0`；临时本地 HTTP endpoint 已停止并删除。

验证限制：

- 本阶段任务执行仍为创建后同步执行；后台调度 worker、同步计划和前端配置入口已在 16.97 补齐第一阶段。
- 失败任务手动重试复用任务参数；计划级最大重试次数、退避、通知事件和自动暂停已在 16.100 补齐第一阶段。

待继续：

- 对象存储列表 API 继续补齐真实云环境联调、provider SDK/REST 错误映射、ETag 和更复杂目录规则。
- 在真实腾讯云/华为云账单环境补充端到端联调，并扩展 CSV 字段差异、Parquet/Excel、分区目录和 provider 错误码映射。
- 扩展成本异常阈值配置、环比/同比突增规则和整改闭环。

### 16.97 2026-06-21 V1.2 P2 成本同步计划与后台 worker 第一阶段

状态：已完成 Azure/GCP 成本同步计划、后台到期扫描 worker、手动运行、到期扫描、前端计划配置入口和 URL 参数保留/脱敏第一阶段；成本拉取已从“人工触发任务”推进到“可配置周期同步”。

本阶段新增：

- 成本同步计划模型：
  - 新增 `iac_cloud_cost_sync_schedule`，记录 org、创建人、计划名称、provider、云账号、账号、区域、来源、账期、币种、状态、同步间隔、最近任务、最近状态、最近错误、最近同步时间、下次同步时间和参数。
  - 同步计划参数继续走脱敏返回，`sourceUrl` 响应只展示预览；编辑计划未填写新 URL 时保留原始参数，避免把脱敏 URL 预览覆盖为真实配置。
- 成本同步计划 API：
  - `GET /api/v1/cloud/cost/sync-schedules`：查询计划列表，支持关键字、状态、provider、source、云账号、账号和账期过滤。
  - `POST /api/v1/cloud/cost/sync-schedules`：创建同步计划。
  - `PUT /api/v1/cloud/cost/sync-schedules/:id`：更新同步计划。
  - `DELETE /api/v1/cloud/cost/sync-schedules/:id`：删除同步计划。
  - `POST /api/v1/cloud/cost/sync-schedules/:id/run`：立即运行指定计划并返回本次任务详情。
  - `POST /api/v1/cloud/cost/sync-schedules/run-due`：扫描并运行到期计划，支持 `force` 用于验证或人工强制扫描。
- 后台 worker：
  - Portal 启动时注册 `StartCloudCostSyncScheduleWorker`，按 `CLOUD_COST_SYNC_SCHEDULE_WORKER_INTERVAL` 扫描所有组织到期计划，默认 5 分钟，最小 30 秒，最大 24 小时。
  - 计划运行前使用 MySQL user-level lock 防止多 portal 实例重复触发；锁名改为 SHA1 短 key，规避 MySQL `GET_LOCK` 64 字符限制。
  - 计划运行后写回 `lastSyncTaskId`、`lastSyncStatus`、`lastError`、`lastSyncedAt` 和 `nextSyncAt`。
- 前端成本中心增强：
  - 成本中心新增“账单同步计划”面板，展示计划名称、状态、来源、账期、间隔、最近同步、下次同步和操作。
  - 支持新建/编辑计划、立即运行、删除计划和扫描到期计划。
  - 立即运行后自动打开本次账单同步任务详情抽屉，复用 16.96 的任务日志和结果展示。

验证：

- `docker compose build iac-portal` 成功。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功。
- `/api/v1/check` 返回 `success=true`、`build=docker-compose`、`version=v1.3.5`，web 首页返回 `HTTP/1.1 200 OK`。
- DB 自动迁移确认存在 `iac_cloud_cost_sync_schedule`。
- 使用本地 GCP Billing Export JSON endpoint 创建 `codex-cost-schedule`：
  - 到期 worker 自动触发计划，生成系统任务，任务 `complete`、`imported=1`。
  - `run-due force=true` 手动扫描返回 `totalCount=1`、`triggeredCount=1`、`completeCount=1`、`lockSkippedCount=0`。
  - 成本记录写入 `codex-cost-schedule-vm`，金额 `9.87 USD`，来源 `gcp_billing_export`，账号 `codex-cost-schedule-project`。
  - 计划回写 `lastSyncStatus=complete`、`lastSyncTaskId` 和向后推进的 `nextSyncAt`。
  - 事件中心写入 `cost.pull.completed`，payload 包含 `syncTaskId` 和脱敏后的 `sourceUrl`。
- 验证编辑计划未填写 URL：
  - 更新计划描述后，响应仍返回 `sourceUrlPreview=http://host.docker.internal:18082/gcp-cost.json?...`。
  - 立即运行计划成功，证明真实 URL 参数未被脱敏预览覆盖。
- 内置浏览器验证成本中心页面：
  - 页面展示“账单同步计划”“扫描到期”“新建计划”和“账单同步任务”。
  - 切换账期到 `2026-08` 后，计划表展示 `codex-cost-schedule`、`preserve source url`、`gcp / GCP Billing Export URL`、脱敏 URL、`60s` 间隔、最近同步状态“完成”和操作按钮。
  - “新建账单同步计划”弹窗正常展示计划名称、账单导出 URL 和同步间隔等字段。
- 验证后已清理本轮临时成本记录、成本建议、事件、同步任务、同步计划和任务日志，剩余临时数据计数均为 `0`；临时本地 HTTP endpoint 已停止并删除。

验证限制：

- 当前计划同步仍基于固定账期配置；生产可继续扩展为相对账期、账期滚动窗口和按天分区。
- 当前 URL 拉取已经支持显式导出文件 URL、导出文件索引 URL、增量游标、多文件批量导入，以及 Azure Blob/GCS/S3 对象存储原生列表 API、签名授权和分页第一阶段；真实云环境联调仍待继续。
- 当前计划已在 16.100 支持计划级最大重试次数、指数退避、失败通知事件和自动暂停第一阶段；生产级外部通知投递和更复杂调度窗口仍待继续。

待继续：

- 对象存储列表 API 继续补齐真实云环境联调、provider SDK/REST 错误映射、ETag 和更复杂目录规则。
- 在真实腾讯云/华为云账单环境补充端到端联调，并扩展 CSV 字段差异、Parquet/Excel、分区目录和 provider 错误码映射。
- 成本同步计划通知静默、通知窗口、负责人和路由已在 16.190 完成第一阶段，按错误类型路由和失败次数升级策略已在 16.191 完成第一阶段；继续增强真实通知通道端到端联调和更细的企业升级策略。

### 16.179 2026-06-22 V1.2 P0 OCI provider 原生错误码解析与同步失败分类第一阶段

状态：已完成 OCI collector 非 2xx API 响应的 provider 原生错误码保留、同步失败详情结构化和账号健康失败分类复用第一阶段。

已完成：

- OCI REST 调用失败时不再只返回 `HTTP status + body` 文本，改为保留 `provider=oci`、`status`、`code`、`requestId`、`retryAfter`、`service` 和 `path`。
- 支持解析 OCI 常见错误体 `{"code":"...","message":"..."}`，并在错误体不是 JSON 或字段缺失时回退原始 body。
- CMDB 同步任务 `failureDetails` 结构化写入 `provider`、`httpStatus`、`providerCode`、`requestId`、`retryAfter`、`providerService` 和 `providerPath`，便于前端、健康检查和审计定位。
- 同步失败分类优先读取 provider 原生元数据：
  - HTTP 429、`TooManyRequests`、`LimitExceeded` 等归类为 `rate_limit`，可重试。
  - HTTP 401、`NotAuthenticated`、`InvalidCredentials` 等归类为 `credential`，需更新凭证。
  - HTTP 403、`NotAuthorizedOrNotFound`、`Forbidden`、`Authorization*` 等归类为 `permission`，需检查权限策略。
  - HTTP 5xx 归类为 `network`，可重试。
- 云账号健康的同步失败影响分类改为复用 CMDB 同步分类函数，避免账号健康和同步任务对同一 provider 错误给出不同结论。

验证：

- 新增单元测试覆盖 OCI 原生错误 metadata 保留、`failureDetails` 结构化写入、限流/权限/凭证/5xx 分类和账号健康分类复用。
- 本阶段在清理部署服务和运行数据后进行代码级验证，不启动 compose 服务；验证命令记录见本次开发回归输出。

验证限制：

- 本阶段基于代码路径和模拟 OCI 错误样本验证，未连接真实 OCI 账号触发真实 `opc-request-id`、限流窗口和权限错误。
- provider 原生分页 token 和真实云端错误样本库仍待继续；按 region/assetType scope 的自动局部重试已在 16.188 完成第一阶段，API 子调用级局部补偿仍待 provider adapter 继续细化。

### 16.180 2026-06-22 V1.2 P0 OCI API 限流与临时错误短重试第一阶段

状态：已完成 OCI collector GET 调用层的有限短重试；采集遇到 429、5xx、408 或临时网络/读取错误时，会在当前请求上下文内短暂退避后重试，最终失败时保留尝试次数。

已完成：

- OCI REST GET 调用新增最多 3 次请求尝试，覆盖所有 OCI collector 复用的 `ociDoAPI` 调用路径。
- 重试条件：
  - HTTP 429：API 限流。
  - HTTP 408：请求超时。
  - HTTP 5xx：云 API 服务端临时异常。
  - 非上下文取消/超时类的 HTTP 客户端临时错误或响应体读取错误。
- 重试延迟：
  - 优先解析 OCI/API 返回的 `Retry-After` 秒数或 HTTP 时间。
  - 未返回 `Retry-After` 时使用短指数退避，默认 500ms、1s、2s，并设置最大 5s 上限。
  - 等待过程尊重采集上下文取消，避免 collector 超时后继续等待。
- 最终失败的 `failureDetails` 会写入 `providerAttempts`，便于任务详情判断是否已经过短重试。
- 401/403 等凭证和权限错误不进入 HTTP 短重试，继续交由 16.179 的失败分类给出不可重试建议。

验证：

- 新增单元测试覆盖 429/5xx/403/context cancel 的重试判定、`Retry-After` 秒数/HTTP 时间解析、指数退避和最大延迟上限。
- 新增单元测试覆盖最终错误中的 `attempts` 进入同步任务 `failureDetails.providerAttempts`。
- 本阶段在清理部署服务和运行数据后进行代码级验证，不启动 compose 服务。

验证限制：

- 本阶段为单请求内短重试；按 region/assetType scope 的失败范围补偿已在 16.188 完成第一阶段，API 子调用级局部补偿队列仍待继续。
- 未连接真实 OCI 账号验证生产限流窗口、真实 `Retry-After` 差异和跨 compartment API 错误样本。

### 16.181 2026-06-22 V1.2 P0 OCI API 子调用耗时与尝试次数指标第一阶段

状态：已完成 OCI collector 内部 API 子调用指标第一阶段；同步任务 stats 会记录每个 OCI API 调用的区域、服务、路径、耗时、尝试次数、HTTP 状态、错误分类和分页/compartment 范围信息。

已完成：

- OCI collector 创建按 region 隔离的 API metrics recorder，并将结果写入同步任务 `stats.apiMetrics`。
- 每次 `ociDoAPI` 完成后记录：
  - `provider=oci`
  - `region`
  - `service`
  - `path`
  - `durationMs`
  - `attempts`
  - `retryCount`
  - `httpStatus`
  - `requestId`
  - `retryAfter`
  - `compartmentId`
  - `pageTokenUsed`
  - `status`
  - `errorCategory`
  - `retryable`
  - `retryHint`
- 新增 `stats.apiMetricSummary`，汇总 API 调用总数、失败数、发生重试的调用数、最大耗时、最大尝试次数、最慢 endpoint 和 service 计数。
- 该指标覆盖所有复用 `ociDoAPI` 的 OCI 采集路径，包括列表分页请求和原始 JSON 请求。

验证：

- 新增单元测试覆盖 API metric recorder、失败指标字段、query scope 字段、重试计数和 summary 汇总。
- 本阶段在清理部署服务和运行数据后进行代码级验证，不启动 compose 服务。

验证限制：

- `apiMetrics` 第一阶段已经写入 stats JSON，并在 16.182 增加任务详情 API 调用页签、region/service/status 筛选、慢调用阈值过滤、任务详情慢调用告警和筛选结果 CSV 导出；按 API 维度趋势已在 16.183 完成第一阶段，事件中心慢调用告警已在 16.184 完成第一阶段。
- 未连接真实 OCI 账号验证生产 API 耗时分布、分页 token 长链路和跨 compartment 样本。

### 16.182 2026-06-22 V1.2 P0 云采集任务 API 调用耗时可视化第一阶段

状态：已完成任务详情 API 调用耗时可视化第一阶段；前端会把同步任务 `stats.apiMetrics` 展开为独立页签，并提供 region/service/status、慢调用阈值过滤、慢调用告警和筛选结果 CSV 导出，辅助定位 OCI 采集慢接口、重试接口和失败 API scope。

已完成：

- 云采集任务详情摘要新增 API 调用统计，展示总调用数、失败数、重试调用数、最大尝试次数、最大耗时和最慢 API。
- 云采集任务详情新增 `API调用` 页签，按行展示 region、service、API path、任务状态、HTTP 状态、耗时、尝试次数、重试次数、compartment scope、分页请求标记、错误分类、可重试标记和重试建议。
- `API调用` 页签支持按 region、service、status 多选过滤，并支持开启“只看慢调用”后按毫秒阈值过滤。
- `API调用` 页签基于当前慢调用阈值展示慢调用告警，汇总慢调用总数、失败慢调用数、重试慢调用数、最慢 API 和耗时；存在失败慢调用时提升为错误提示。
- `API调用` 页签支持导出当前筛选结果为 CSV，包含任务、账号、云厂商、region、service、API、状态、HTTP 状态、耗时、尝试/重试次数、compartment、分页、错误分类、重试建议、Request ID 和 Retry-After。
- 打开或关闭任务详情时会重置 API 调用过滤条件，避免跨任务串用筛选条件。
- 支持后端按 region 存储的 `stats.apiMetrics` 展平展示；当后续 provider 使用同一字段结构时，前端无需额外改造即可展示。
- 空数据时保留原始 `统计` 页签，可继续通过 stats JSON 排查历史任务或未上报 provider。

验证：

- 已通过前端构建验证任务详情新增页签语法和依赖引用。
- 本阶段在清理部署服务和运行数据后进行代码级验证，不启动 compose 服务。

验证限制：

- 事件中心慢调用告警已在 16.184 完成第一阶段，同步策略级慢调用阈值持久化已在 16.185 完成第一阶段，策略级慢 API 告警静默窗口已在 16.187 完成第一阶段，负责人分派和通知路由元数据已在 16.189 完成第一阶段，按错误类型/云服务动态路由和失败次数升级已在 16.192 完成第一阶段；按子周期独立阈值和真实通知通道联调仍待继续。
- 未连接真实 OCI 账号验证生产慢调用样本和分页链路展示。

### 16.183 2026-06-22 V1.2 P0 云采集任务 API 维度趋势第一阶段

状态：已完成云采集任务 API 维度趋势第一阶段；云采集任务 summary 会基于当前任务筛选条件和趋势日期范围，从同步任务 `stats.apiMetrics` 聚合 Top API 调用维度，前端在云采集统计区展示调用、失败、重试和耗时趋势。

已完成：

- `GET /api/v1/cmdb/sync-tasks` 与云资产包装接口的 summary 新增 `apiMetrics` 数组，按 provider、region、service、path 聚合。
- 每个 API 维度返回调用次数、失败次数、重试次数、失败率、平均耗时、最大耗时和日期趋势点。
- 日期趋势点与现有任务趋势范围保持一致，支持 7/14/30 天和自定义日期范围。
- 前端云采集统计区新增 `按 API 拆分（Top 10）` 表格，展示接口维度、调用/失败/重试、平均/最大耗时和最近有调用日期趋势。
- 聚合逻辑兼容后端按 region map 存储的 `stats.apiMetrics`，也兼容后续 provider 直接写入数组形式的 `apiMetrics`。

验证：

- 新增单元测试覆盖 API metrics region map 展平和趋势点补零/平均耗时计算。
- 已通过后端单元测试和前端构建验证。
- 本阶段在清理部署服务和运行数据后进行代码级验证，不启动 compose 服务。

验证限制：

- 第一阶段只返回 Top 10 API 维度，未实现按 API 搜索、后端分页和自定义排序。
- 未连接真实 OCI 账号验证生产 API 趋势、长分页链路和跨 compartment 样本。

### 16.184 2026-06-22 V1.2 P0 云采集慢 API 事件中心告警第一阶段

状态：已完成云采集慢 API 事件中心告警第一阶段；同步任务完成时会从 `stats.apiMetrics` 识别耗时不低于默认阈值的 API 调用，写入统一事件中心并复用 Webhook/通知策略分发链路。

已完成：

- 后端新增慢 API 摘要计算：
  - 默认阈值 `1000ms`。
  - 汇总慢调用数、失败慢调用数、重试慢调用数、最大耗时、最慢 API 和 Top 慢调用样本。
  - 保留 provider、region、service、path、status、durationMs、attempts、retryCount、requestId、错误分类、分页和 compartment 等排障字段。
- 云采集任务完成后自动写入事件中心：
  - 事件类型 `cloud.sync.task.slow_api_detected`。
  - 事件来源 `sync`，资源类型 `cmdb_sync_task`，资源 ID 为采集任务 ID。
  - 普通慢调用写入 warning，慢调用中存在失败 API 时提升为 error。
  - 同一任务同一事件类型按 `org_id/event_type/resource_type/resource_id` 去重。
- 事件 payload 提供：
  - `taskId`、`status`、`thresholdMs`、`slowCount`、`failedCount`、`retriedCount`、`maxDurationMs`。
  - `slowest`、`top`、`regions`、`assetTypes`、`syncPolicyId`、`syncPolicyScheduleKey`、`syncPolicyScheduleName`。
- 前端事件中心和组织通知设置新增中文事件类型：
  - `cloud.sync.task.slow_api_detected` -> `云采集慢 API 告警`。
- 通知模板变量中心新增慢 API 告警样例变量组，便于配置企业微信、钉钉、Slack、邮件和 Webhook 通知模板。

验证：

- 新增单元测试覆盖慢 API 摘要阈值过滤、失败/重试计数、最慢 API 选择、Top 排序和 requestId 保留。
- 本阶段在清理部署服务和运行数据后进行代码级验证，不启动 compose 服务。

验证限制：

- 同步策略级慢调用阈值持久化已在 16.185 完成第一阶段，策略级慢 API 告警静默窗口已在 16.187 完成第一阶段，负责人分派和通知路由元数据已在 16.189 完成第一阶段，按错误类型/云服务动态路由和失败次数升级已在 16.192 完成第一阶段；按子周期独立阈值和真实通知通道联调仍待继续。
- 未连接真实 OCI/AWS 账号验证生产慢调用样本、通知通道投递和 Webhook 验签。

### 16.185 2026-06-22 V1.2 P0 云采集慢 API 阈值持久化第一阶段

状态：已完成云采集慢 API 阈值持久化第一阶段；云账号同步策略可在 `params.slowApiThresholdMs` 中保存慢 API 告警阈值，策略触发的采集任务会把该阈值写入任务 `stats`，任务详情过滤、页面慢调用告警和事件中心告警使用同一个阈值。

已完成：

- 同步策略参数新增 `slowApiThresholdMs`：
  - 后端 `cloudSyncPolicyNormalizeParams` 会规范化并持久化正整数阈值。
  - 最大值限制为 `600000ms`，避免误配置导致异常大阈值。
  - 未配置时回退默认 `1000ms`。
- 同步策略触发采集任务时传递阈值：
  - 普通策略任务和按区域/资源类型独立子周期任务都会传入阈值。
  - 任务创建时写入 `stats.slowApiThresholdMs`。
  - 任务实际运行中 collector 返回 stats 后会重新保留该阈值，避免 provider 统计覆盖。
- 事件中心慢 API 告警改为使用任务 `stats.slowApiThresholdMs`：
  - `cloud.sync.task.slow_api_detected` 的 `payload.thresholdMs` 与任务详情页阈值一致。
  - 批量重跑、任务组和历史任务继续从任务 stats 回放阈值。
- 前端“云账号 - 同步策略”增强：
  - 创建/编辑同步策略表单新增“慢 API 告警阈值（毫秒）”。
  - 同步策略列表新增“慢 API”列展示当前阈值。
- 前端“云资产/资产 CMDB - 云采集任务详情”增强：
  - 打开任务详情时优先读取任务 `stats.slowApiThresholdMs` 作为 API 调用页签慢调用过滤默认值。
  - 重置 API 调用筛选时回到当前任务阈值，而不是固定回到 1000ms。

验证：

- 新增单元测试覆盖同步策略慢 API 阈值持久化、最大值裁剪、零值删除，以及任务 stats 中阈值参与慢调用摘要计算。
- 本阶段在清理部署服务和运行数据后进行代码级验证，不启动 compose 服务。

验证限制：

- 第一阶段为策略级统一阈值，策略级慢 API 告警静默窗口已在 16.187 完成，负责人分派和通知路由元数据已在 16.189 完成第一阶段，按错误类型/云服务动态路由和失败次数升级已在 16.192 完成第一阶段；尚未支持每个子周期独立阈值。
- 未连接真实 OCI/AWS 账号验证生产慢调用样本、通知通道投递和 Webhook 验签。

### 16.186 2026-06-22 V1.2 P1 EKS/OKE K8S 信息字段兼容增强

状态：已完成 EKS/OKE K8S 信息字段兼容增强；在已有 K8S 信息页签、Kubernetes 概览和快速入口基础上，继续解决“数据已采集但详情页看不到”的兼容问题。

已完成：

- 前端 K8S 详情读取增强：
  - `assetAttrValue` 支持大小写兼容、点路径读取、`rawData.response`、`rawData.properties` 和 `rawData.metadata` 来源。
  - K8S 集群识别兼容 `nodeGroups/nodegroups/node_groups`、`nodePools/nodepools/node_pools`、`kubernetesVersion/kubernetes_version`、`endpoints.kubernetes`、`endpointConfig` 和 `kubernetesNetwork`。
  - 节点组/节点池列表兼容 EKS `nodeGroups`、OKE `nodePools`、Azure `agentPoolProfiles`、导入数据中的 snake_case 字段和对象包装数组。
  - 工作负载层列表兼容 `statefulSets/daemonSets/replicaSets/cronJobs` 的大小写与 snake_case 变体。
- 前端 K8S 信息页签展示增强：
  - API Endpoint 兼容 `endpoints.kubernetes`、`publicEndpoint`、`privateEndpoint`、`fqdn` 等原生字段。
  - VPC/VNet/VCN、子网、安全组/NSG 和网络配置兼容 EKS `resourcesVpcConfig`、OKE `endpointConfig`、AKS/GKE 网络字段。
  - 节点表兼容 OKE `displayName/lifecycleState/nodeShape/subnetIds`、AKS `vmSize/provisioningState` 和 GKE `machineType`。
- OKE collector 增强：
  - OKE Cluster 采集写入 `attributes.endpoints`。
  - 优先把 `endpoints.kubernetes` 回填到资产 `address`，并回退 `publicEndpoint/privateEndpoint/endpoint`。
  - 写入 `attributes.version`，与 EKS/AKS/GKE 的版本展示字段保持一致。

验证：

- 新增单元测试覆盖 OKE endpoint 优先级和 public endpoint 回退。
- 使用 Docker Go 镜像执行 `gofmt`，覆盖 `backend/portal/apps/cmdb_collect_oci.go` 和 `backend/portal/apps/cmdb_collect_kubernetes_test.go`。
- 本阶段继续遵守部署清理约束：不启动 compose 服务，验证以单测、静态检查和镜像构建为主。

验证限制：

- 当前增强解决字段归一和展示兼容；真实 EKS/OKE 账号端到端采集、NodeGroup/NodePool 分页、权限错误码和 kubeconfig/Agent 实时工作负载采集仍待真实环境继续验证。

### 16.187 2026-06-22 V1.2 P0 云采集慢 API 告警静默窗口第一阶段

状态：已完成云采集慢 API 告警静默窗口第一阶段；同步策略可配置 `params.slowApiSilenceMinutes`，同一云账号、同步策略、子周期和最慢 API endpoint 在静默窗口内只写入一次 `cloud.sync.task.slow_api_detected` 事件，避免重复 Webhook/通知刷屏。

已完成：

- 同步策略参数新增 `slowApiSilenceMinutes`：
  - 后端 `cloudSyncPolicyNormalizeParams` 会规范化并持久化正整数分钟数。
  - `0` 或未配置表示不启用静默窗口。
  - 最大值限制为 `10080` 分钟，避免误配置导致长期静默。
- 同步策略触发采集任务时传递静默窗口：
  - 普通策略任务和按区域/资源类型独立子周期任务都会传入静默窗口。
  - 任务创建和运行 stats 会写入 `slowApiSilenceMinutes`。
  - collector 返回 stats 后会重新保留该配置，避免 provider 统计覆盖。
- 慢 API 事件写入增强：
  - 事件 payload 新增 `silenceMinutes` 和 `slowApiFingerprint`。
  - `slowApiFingerprint` 由 provider、account、syncPolicyId、syncPolicyScheduleKey 和最慢 API endpoint 组成。
  - 写事件前查询静默窗口内同 fingerprint 的历史事件；命中时跳过事件写入，并在同步任务日志写入 `slow_api_silenced`。
  - 不同 endpoint 的新慢调用不会被同一静默窗口误抑制。
- 前端“云账号 - 同步策略”增强：
  - 创建/编辑同步策略表单新增“慢 API 告警静默窗口（分钟）”。
  - 同步策略列表“慢 API”列同时展示阈值和静默窗口。

验证：

- 新增单元测试覆盖同步策略静默窗口持久化、最大值裁剪、零值删除、任务 stats 读取和 slow API fingerprint 生成。
- 本阶段继续遵守部署清理约束：不启动 compose 服务，验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段为策略级静默窗口；每个子周期单独配置静默窗口仍待继续；负责人分派和通知路由元数据已在 16.189 完成第一阶段，按错误类型/服务动态路由和真实通知通道投递验签仍待后续增强。

### 16.188 2026-06-22 V1.2 P0 云采集失败 scope 自动局部重试第一阶段

状态：已完成同步策略级失败 scope 自动局部重试第一阶段；策略显式开启后，失败采集任务会在可重试错误且识别到失败范围时自动创建一条 `auto_failed_scope` 重跑任务，只重跑失败的 region/assetType 范围。

已完成：

- 同步策略参数新增：
  - `autoRetryFailedScopes`：是否开启失败 scope 自动局部重试，默认关闭。
  - `autoRetryMaxScopes`：单次自动补偿最大 scope 数，默认 10，上限 50。
- 自动触发保护：
  - 仅同步策略触发的失败任务生效，手工任务和无策略任务不自动补偿。
  - 仅 `failureDetails.retryable=true` 或 `failureSummary.retryableTotal>0` 的错误生效，凭证/权限/配置类不可重试错误只写跳过日志。
  - 自动创建的重试任务带 `rerunMode=auto_failed_scope`，不会再次自动触发补偿，避免循环重试。
  - 同一源任务已存在自动补偿任务时跳过重复创建。
  - 当只有粗粒度 `scopeMetrics` 且失败范围覆盖整个原任务范围时，不自动隐藏创建全量重跑。
- scope 推导：
  - `failureDetails` 支持解析 provider 错误消息中的 `region`、`assetType`/`resourceType`，优先用于精准补偿。
  - 未提供精准失败详情时，回退读取 `stats.scopeMetrics` 中非 `complete` 的 region+assetType。
  - 新任务 stats 写入 `autoRetry`、`rerunFromTaskId`、`rerunGroupId`、`rerunMode` 和 `rerunParameterDiffs`，源任务日志写入 `auto_scope_rerun_created` 或跳过原因。
- 事件中心新增 `cloud.sync.task.auto_scope_rerun_started`，payload 包含源任务、重试任务、失败 scope、同步策略和子周期信息。
- 前端“云账号 - 同步策略”新增“失败 scope 自动局部重试”和“单次最大补偿 scope 数”，策略列表新增“失败补偿”列。

验证：

- 新增单元测试覆盖：
  - 策略参数 `autoRetryFailedScopes/autoRetryMaxScopes` 持久化、禁用删除和最大值裁剪。
  - provider 错误消息中的 `region`、`assetType` 元数据解析。
  - 基于 `failureDetails` 的精准失败 scope 推导。
  - 基于 `scopeMetrics` 的失败 scope 回退推导。
- 定向后端测试通过：
  - `go test -vet=off ./portal/apps -run 'Test(CmdbSyncTaskFailedRetryScopes|CloudSyncPolicyNormalizeSlowAPIThreshold|CmdbSyncFailureDetailsExtractsProviderMetadata|CmdbSyncTaskSlowAPI|OciAPI)' -count=1 -timeout=120s`

验证限制：

- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。
- 精准局部重试依赖 provider 错误详情或 `scopeMetrics` 提供 region/assetType；真实 provider 的更多错误样本、分页游标和 API 子调用级补偿仍待继续。
- 负责人分派和通知路由元数据已在 16.189 完成第一阶段，按错误类型/云服务动态路由和失败次数升级已在 16.192 完成第一阶段；外部通知通道投递验签和企业级值班升级仍待后续增强。

### 16.189 2026-06-22 V1.2 P0 云采集事件负责人分派与通知路由第一阶段

状态：已完成同步策略级通知负责人、通知路由和分派对象元数据第一阶段；云采集策略失败、慢 API 告警和失败 scope 自动局部重试事件会继承同一套路由元数据，并复用现有通知订阅机制做路由匹配。

已完成：

- 同步策略 `params` 新增并规范化：
  - `notificationOwner`：通知负责人或值班 owner。
  - `notificationRoutes`：通知路由 key，可填写如 `sre`、`cloud-platform`。
  - `notificationAssignees`：分派对象，可填写用户、团队或值班组标识。
- 云事件 payload 自动注入上述字段：
  - `cloud.sync.policy.triggered` / `cloud.sync.policy.completed` / `cloud.sync.policy.failed` / `cloud.sync.policy.auto_paused` 由 `cloudSyncPolicyEvent` 统一注入。
  - 慢 API 事件 `cloud.sync.task.slow_api_detected` 会按 `syncPolicyId` 读取策略并继承路由。
  - 自动局部重试事件 `cloud.sync.task.auto_scope_rerun_started` 继承源策略路由。
- 通知订阅匹配增强：
  - 保留现有精确事件、`cloud.*`、`sync.*`、前缀通配匹配。
  - 新增 `cloud.route.<route>`、`cloud.owner.<owner>`、`cloud.assignee.<assignee>` 候选事件类型，可直接在现有通知配置中订阅。
  - 通知 Markdown 默认模板新增“通知负责人 / 通知路由 / 分派对象”展示。
- 前端“云账号 - 同步策略”抽屉新增中文配置项：通知负责人、通知路由、分派对象。

验证：

- 新增单元测试覆盖同步策略通知路由参数标准化、事件 payload 注入和通知候选事件类型生成。

验证限制：

- 本阶段为事件元数据和通知订阅路由第一阶段；按错误类型/云服务动态路由和失败次数升级已在 16.192 完成第一阶段，云事件自动创建 ITSM 工单已在 16.193 完成第一阶段。组织架构/值班表模型、外部通知通道投递验签和外部 ITSM 双向同步仍待后续增强。

### 16.190 2026-06-22 V1.2 P2 成本同步计划通知静默与路由第一阶段

状态：已完成成本同步计划通知静默、通知窗口、负责人、通知路由和分派对象元数据第一阶段；成本计划失败事件可按计划参数控制通知频率，并可通过 `cost.route.*`、`cost.owner.*`、`cost.assignee.*` 订阅到不同处理组。

已完成：

- 成本同步计划 `params` 新增并规范化：
  - `notificationSilenceMinutes`：失败事件静默期，后端限制最大 7 天，`0` 表示不静默。
  - `notificationWindows`：通知窗口，支持 `HH:MM-HH:MM` 和跨天窗口，窗口外抑制普通失败事件。
  - `notificationOwner`：通知负责人或值班 owner。
  - `notificationRoutes`：通知路由 key，可填写如 `finops`、`cloud-platform`。
  - `notificationAssignees`：分派对象，可填写用户、团队或值班组标识。
- 成本同步计划失败事件增强：
  - 普通 `cost.sync.schedule.failed` 事件会按静默期和通知窗口做抑制。
  - `cost.sync.schedule.auto_paused` 自动暂停事件不被静默抑制，确保需要人工处理的停用事件一定落入事件中心。
  - 事件 payload 注入 `notificationOwner`、`notificationRoutes`、`notificationAssignees`、`notificationSilenceMinutes`、`notificationWindows` 和 `notificationSource`。
- 通知订阅匹配增强：
  - 在已有 `cost.*`、精确事件和通用 `cloud.route.*` 候选基础上，新增源类型维度候选。
  - 成本事件可通过 `cost.route.<route>`、`cost.owner.<owner>`、`cost.assignee.<assignee>` 精准订阅。
  - 通知 Markdown 默认模板继续展示“通知负责人 / 通知路由 / 分派对象”。
- 前端成本中心增强：
  - “账单同步计划”列表新增“通知”列，展示静默期、通知窗口、负责人、路由和分派对象摘要。
  - “新建/编辑账单同步计划”弹窗新增中文配置项：通知静默期、通知窗口、通知负责人、通知路由、分派对象。

验证：

- `gofmt` 覆盖 `cloud_event_notification.go`、`cloud_cost.go`、`cloud_cost_schedule_test.go` 和成本响应模型。
- 新增单元测试覆盖成本计划通知参数标准化、路由 payload 注入、跨天通知窗口判断和 `cost.route.*` 候选事件类型生成。

验证限制：

- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。
- 本阶段使用计划级静默和窗口配置；按失败错误类型追加路由和失败次数升级策略已在 16.191 完成第一阶段，更细的云服务、成本中心、业务线和值班表升级仍待后续增强。
- 云事件自动创建 ITSM 工单已在 16.193 完成第一阶段；外部通知通道投递验签、外部 ITSM 双向状态同步和企业真实机器人/SMTP 端到端联调仍待后续增强。

### 16.191 2026-06-22 V1.2 P2 成本同步计划按错误类型路由与升级策略第一阶段

状态：已完成成本同步计划失败事件按错误类型追加通知路由和失败次数升级策略第一阶段；成本计划失败不再只能走固定路由，可根据失败原因自动归类并追加处理组。

已完成：

- 成本同步计划 `params` 新增并规范化：
  - `notificationFailureRoutes`：错误类型到通知路由的映射，例如 `permission -> iam/security`、`rate_limit -> cloud-platform`。
  - `notificationEscalationAt`：失败次数升级阈值，后端限制最大 `100`，`0` 表示不按次数升级。
  - `notificationEscalationRoutes`：达到升级阈值或自动暂停时追加的升级路由。
- 成本计划失败原因归类：
  - `rate_limit`：429、限流、throttling、LimitExceeded。
  - `credential`：Token、AccessKey、签名、凭证过期或认证失败。
  - `permission`：AccessDenied、Forbidden、NotAuthorized、权限拒绝。
  - `not_found`：404、对象或账单文件不存在。
  - `network`：timeout、连接失败、临时网络错误、5xx。
  - `config`：非法参数、格式错误、unsupported、bad request。
  - `unknown`：未命中明确规则的其他失败。
- 成本同步计划失败事件 payload 增强：
  - 注入 `failureCategory`，便于模板、Webhook 和 ITSM 解析。
  - 命中 `notificationFailureRoutes` 时追加到最终 `notificationRoutes`。
  - 达到 `notificationEscalationAt` 或发生 `auto_paused` 时注入 `notificationEscalated`、`notificationEscalationReason`、`notificationEscalationRoutes`。
- 通知订阅匹配增强：
  - 新增 `cloud.failure.<category>` / `cost.failure.<category>` 候选事件类型。
  - 新增 `cloud.escalation.<reason>` / `cost.escalation.<reason>` 候选事件类型。
  - Markdown 默认通知展示“失败类型”和“通知升级”状态。
- 前端成本中心增强：
  - “账单同步计划”通知摘要展示错误路由配置数量和升级阈值。
  - “新建/编辑账单同步计划”弹窗新增“错误类型路由”“升级阈值（失败次数）”“升级路由”中文配置项。

验证：

- 新增单元测试覆盖错误类型路由参数标准化、失败原因分类、payload 路由合并、失败次数升级和 `cost.failure.*` / `cost.escalation.*` 通知候选事件类型。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段按文本错误原因归类，真实云 provider 原生错误码到成本计划失败分类的更细映射仍待结合真实账单拉取样本继续补强。
- 云事件自动创建 ITSM 工单已在 16.193 完成第一阶段；企业级值班表、逐级升级时间线、外部 ITSM 双向状态同步和真实通知通道端到端验签仍待后续增强。

### 16.192 2026-06-22 V1.2 P0 云采集事件按错误类型/云服务路由与升级策略第一阶段

状态：已完成云采集同步策略事件按失败类型、云服务和失败次数动态追加通知路由第一阶段；同步策略失败、自动暂停、慢 API 告警和失败 scope 自动局部重试事件不再只依赖固定路由。

已完成：

- 同步策略 `params` 新增并规范化：
  - `notificationFailureRoutes`：失败类型到通知路由的映射，例如 `permission -> iam/security`、`rate_limit -> cloud-platform`。
  - `notificationServiceRoutes`：云服务到通知路由的映射，例如 `ec2 -> compute-oncall`、`iaas -> oci-team`。
  - `notificationEscalationAt`：失败次数升级阈值，后端限制最大 `100`，`0` 表示不按次数升级。
  - `notificationEscalationRoutes`：达到升级阈值或自动暂停时追加的升级路由。
- 云采集事件 payload 增强：
  - 基于 `cmdbSyncClassifyFailure`、`failureDetails` 和 provider 原生错误元数据推导 `failureCategory`。
  - 基于 payload、慢 API `slowest/top`、`failureDetails` 和 provider 错误元数据推导 `cloudService`。
  - 命中失败类型路由或云服务路由时追加最终 `notificationRoutes`，并保留原有负责人、基础路由和分派对象。
  - 达到 `notificationEscalationAt` 或发生 `auto_paused` 时注入 `notificationEscalated`、`notificationEscalationReason`、`notificationEscalationRoutes`。
- 通知订阅匹配增强：
  - 新增 `cloud.failure.<category>` / `sync.failure.<category>` 候选事件类型。
  - 新增 `cloud.service.<service>` / `sync.service.<service>` 候选事件类型。
  - 新增 `cloud.escalation.<reason>` / `sync.escalation.<reason>` 候选事件类型。
  - Markdown 默认通知展示“失败类型”“云服务”和“通知升级”状态。
- 前端“云账号 - 同步策略”增强：
  - 同步策略列表新增“通知路由”摘要，展示负责人、基础路由、错误路由、服务路由和升级配置。
  - 同步策略抽屉新增“错误类型路由”“云服务路由”“升级阈值（失败次数）”“升级路由”中文配置项。

验证：

- 新增/更新单元测试覆盖同步策略通知参数标准化、失败类型路由、云服务路由、失败次数升级和通知候选事件类型。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段仍依赖已有错误文本、`failureDetails` 和 provider 元数据推导分类/服务；云事件自动创建 ITSM 工单已在 16.193 完成第一阶段，真实云 provider 更完整错误样本库、服务码映射、值班表、外部 ITSM 双向状态同步和真实通知通道端到端验签仍待后续增强。

### 16.193 2026-06-22 V1.2 P0 云事件自动创建 ITSM 工单第一阶段

状态：已完成云事件到 ITSM 工单自动分派第一阶段；事件中心不再只负责查询、Webhook 和通知投递，当事件 payload 显式启用 `itsmAutoTicket` 或指定 ITSM 连接器时，可自动生成本地或外部 ITSM 工单。

已完成：

- 事件中心派发链路增强：
  - `recordCloudEventWithDispatch` 在 Webhook 和通知派发后新增 ITSM 派发。
  - `source=itsm` 的事件不会再次触发 ITSM 派发，避免递归创建工单。
  - 仅当 payload 中存在 `itsmAutoTicket=true`、`itsmTicketAutoCreate=true` 或 `itsmConnectorId(s)` / `itsmTicketConnectorId(s)` 时才创建工单，避免普通事件无控制地产生工单。
- ITSM 事件工单创建：
  - 无连接器 ID 且启用自动工单时，复用 `CloudIaC 本地工单` 默认连接器。
  - 指定连接器 ID 时按连接器创建工单，连接器禁用或不存在时跳过并记录警告。
  - 同一源事件和同一连接器只创建一张工单，重复派发返回已有工单。
  - 本地连接器保留为 `pending` 工单；外部连接器继续复用现有 HTTP POST 提交能力。
  - 工单请求 payload 新增 `event`、`dispatch`、`requester`、`connector` 结构，包含事件类型、来源、级别、云账号、资源、失败分类、云服务、原始 payload 和连接器信息。
- 数据模型兼容：
  - `iac_cloud_itsm_ticket` 唯一约束从 `org_id + operation_id + connector_id` 调整为 `org_id + operation_id + cloud_event_id + connector_id`，兼容云操作工单和事件工单两种来源。
- 同步策略配置增强：
  - 同步策略 `params` 新增并规范化 `itsmAutoTicket`、`itsmConnectorIds`、`itsmPriority`。
  - 云采集失败、自动暂停、慢 API 和失败 scope 自动局部重试事件可从同步策略继承 ITSM 自动工单配置。
- 前端“云账号 - 同步策略”增强：
  - 同步策略抽屉新增“ITSM 自动工单”“ITSM 连接器 ID”“ITSM 优先级”配置项。
  - 同步策略列表“通知路由”摘要会展示 ITSM 自动工单配置数量或本地工单兜底状态。

验证：

- 新增单元测试覆盖事件 payload 到 ITSM 派发计划的解析、显式关闭优先级、连接器路由触发和事件工单请求 payload 生成。
- 更新同步策略通知路由测试，覆盖 ITSM 自动工单参数规范化和事件 payload 注入。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段完成平台内自动建单和通用 HTTP 提交路径，真实企业 ITSM 的字段映射、审批回调、SLA、双向状态同步和失败补偿仍待结合目标系统接口继续增强。

### 16.194 2026-06-22 V1.2 P0 GitOps/IaC PR review 门禁第一阶段

状态：已完成 GitOps/IaC 变更申请 PR review 和自动化流水线门禁第一阶段；自助申请不再只是普通 ITSM 工单，而是要求填写 PR/MR 地址，并把 Review 状态、Pipeline 状态和门禁结论写入云操作参数与工单载荷。

已完成：

- 后端自助申请增强：
  - `gitops_iac_change` 申请会规范化并校验 GitOps/IaC 门禁参数。
  - PR/MR 地址为必填，且仅接受 `http/https` URL，避免基础设施变更绕过 PR review 流程。
  - 支持识别 `gitOpsRepository`、`gitOpsBranch`、`gitOpsTargetBranch`、`gitOpsChangePath`、`gitOpsPullRequestUrl`、`gitOpsReviewStatus`、`gitOpsPipelineUrl`、`gitOpsPipelineStatus`。
  - Review 状态统一归一为 `pending/approved/changes_requested/rejected`。
  - Pipeline 状态统一归一为 `pending/running/passed/failed/canceled`。
  - 门禁结论写入 `params.gitOpsGate`，状态包括：
    - `passed`：Review 已通过、Pipeline 已通过且有流水线地址。
    - `waiting`：PR 已创建，但 Review 或 Pipeline 仍在等待。
    - `blocked`：Review 要求修改、拒绝，或 Pipeline 失败/取消。
  - `params.gateStatus` 写入顶层，便于后续列表筛选、统计、风险治理和通知路由复用。
- 工单与审计闭环：
  - GitOps/IaC 门禁信息随 `CloudOperation.Params` 进入 ITSM 工单请求 payload。
  - 本地工单、外部 ITSM 提交、操作任务结果和操作审计均能保留同一份门禁证据。
- 前端 ITSM 自助申请弹窗增强：
  - 选择“GitOps/IaC 变更申请”时展示 IaC 仓库、变更分支、目标分支、PR/MR 地址、Review 状态、Pipeline 地址、Pipeline 状态和 IaC 变更路径。
  - PR/MR 地址和 Review/Pipeline 状态在前端必填。
  - 用户无需手写 JSON；专用字段会合并进补充参数 JSON 后提交，原补充参数仍可承载团队扩展字段。

验证：

- 新增单元测试覆盖 GitOps/IaC 门禁参数规范化、PR/MR 地址必填、Review/Pipeline 状态归一化和 `passed/waiting/blocked` 判定。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段由申请人或上游系统提交 PR/MR 与 Pipeline 状态；外部 callback 回写已在 16.214 完成第一阶段，尚未主动调用 GitLab/GitHub API 拉取实时 Review/Pipeline 状态。
- 未新增 GitOps Repo 变更记录模型；后续可继续接入 GitLab/GitHub/Jenkins 主动拉取、GitOps repo diff 记录和更完整审批明细。

### 16.195 2026-06-22 V1.2 P0 风险/漂移到 ITSM 自助整改联动第一阶段

状态：已完成风险/漂移到 ITSM 自助整改工单联动第一阶段；风险合规页面不再只支持状态流转和例外处理，用户可以从风险列表或风险详情一键发起整改工单，平台会自动关联风险证据、资源、修复建议和处理状态。

已完成：

- ITSM 自助目录增强：
  - 新增 `risk_remediation` 自助申请类型，作为“风险整改申请”服务项。
  - `risk_remediation` 会写入 `riskRemediationRequired=true` 和默认 `targetState=risk_remediated_state`，便于后续自动化执行、SLA 统计和通知路由识别。
  - 保留既有 `drift_remediation`，用于专门的 IaC 漂移修复申请。
- 后端风险整改 API：
  - 新增 `POST /api/v1/cloud/risks/:id/remediation-ticket`。
  - 请求可选 `connectorId/title/description/priority/params/dryRun`；未指定连接器时复用 `CloudIaC 本地工单` 兜底。
  - 自动根据风险 finding 生成整改标题、说明和优先级。
  - 自动组装工单参数，包含 `riskId`、`riskSource`、`riskStatus`、`riskLevel`、`ruleKey`、`ruleName`、云厂商、账号、区域、资源类型、资源 ID、资产 ID、云账号 ID、项目/环境、修复建议和原始证据。
  - 漂移类风险会额外标记 `driftRemediation=true` 和 `targetState=iac_desired_state`，便于后续联动漂移自动修复。
  - 创建成功后将风险状态推进为 `in_progress`，并在风险 evidence 写入最近整改工单、操作任务、连接器、状态、发起人和时间。
  - 写入 `risk.remediation_ticket_created` 事件，事件 payload 记录风险状态变化、工单 ID、操作 ID 和连接器。
- 前端风险合规页面增强：
  - 风险列表操作列新增“整改”入口。
  - 风险详情抽屉新增“发起整改”按钮。
  - 调用成功后刷新风险列表，风险状态进入“处理中”。
- 事件展示增强：
  - 通知事件类型和事件中心页面新增 `risk.remediation_ticket_created` 中文展示。

验证：

- 新增单元测试覆盖风险整改参数生成、漂移风险自动标记、整改标题/说明生成和 `risk_remediation` 自助申请参数规范化。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 根据 ITSM 工单状态同步风险整改状态已在 16.196 完成第一阶段；外部 ITSM callback endpoint 和签名校验已在 16.197 完成第一阶段，外部状态周期拉取 worker 已在 16.198 完成第一阶段。
- 漂移风险已携带 `targetState=iac_desired_state`；工单解决后触发环境 drift 自动修复任务已在 16.199 完成第一阶段，风险/漂移详情页入口已在 16.201 完成第一阶段，任务审批策略配置已在 16.204 完成第一阶段，完整任务时间线已在 16.205 完成第一阶段，自动关联推荐已在 16.206 完成第一阶段。
- 前端当前使用默认本地工单兜底；如需在风险页直接选择外部连接器，可继续补充连接器选择弹窗。

### 16.196 2026-06-22 V1.2 P0 ITSM 状态同步风险整改第一阶段

状态：已完成 ITSM 工单状态同步风险整改第一阶段；风险整改不再停留在“创建工单后进入处理中”，当 ITSM 工单状态被平台更新为处理中、已解决、已关闭、失败或取消时，会反向同步风险状态、风险证据和风险事件。

已完成：

- ITSM 状态更新链路增强：
  - `UpdateCloudItsmTicketStatus` 在更新工单和记录 `itsm.ticket.updated` 事件后，识别关联云操作是否为 `risk_remediation`。
  - 支持通过云操作 action、`requestType`、`riskRemediation` 或 `riskRemediationRequired` 判断风险整改工单。
  - 支持从 `riskId`、`riskFindingId`、`cloudRiskId` 提取风险 ID。
- 工单状态到风险状态映射：
  - `submitted`、`in_progress` -> 风险 `in_progress`。
  - `resolved`、`closed` -> 风险 `resolved`，并写入 `resolvedAt`。
  - `failed`、`canceled` -> 风险重新打开为 `open`，便于继续处理。
  - `pending` 不触发风险状态变化，避免刚创建本地工单时误改风险。
- 风险 evidence 回写：
  - 写入最近同步时间、工单 ID、工单状态、风险状态、操作任务 ID、外部 ID、外部单号、外部 URL、连接器 ID、状态备注和外部状态 payload。
  - 对处理中/重新打开状态清空 `resolvedAt` 和过期例外时间，确保风险面板能反映真实待处理状态。
- 风险事件增强：
  - 新增 `risk.remediation_status_synced` 事件。
  - 事件 payload 记录上一风险状态、同步后状态、工单 ID、工单状态、操作任务、外部单号和备注。
  - 通知事件类型和事件中心页面新增中文展示“风险整改状态同步”。

验证：

- 新增单元测试覆盖 ITSM 工单状态到风险状态映射、风险整改操作识别、风险 ID 提取和 evidence 生成。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 外部 ITSM callback endpoint 和签名校验已在 16.197 完成第一阶段；周期状态拉取 worker 仍待继续。
- 风险状态同步仅覆盖 `risk_remediation` 工单；普通云操作工单和事件工单暂不反向更新风险。

### 16.197 2026-06-22 V1.2 P0 外部 ITSM 状态回调与签名校验第一阶段

状态：已完成外部 ITSM 状态回调与签名校验第一阶段；外部 Jira、ServiceNow 或通用 HTTP 工单系统不需要登录态即可把工单状态回写到 CloudIaC，平台通过连接器级共享密钥校验 HMAC-SHA256 签名，并复用 16.196 的风险整改状态同步逻辑。

已完成：

- 新增公开回调接口：
  - `POST /api/v1/cloud/itsm/callbacks/:connectorId/status`
  - 路由位于登录鉴权之前，适配外部 ITSM 系统主动回调。
  - 服务层通过 `connectorId` 反查连接器所属组织，并在验签通过后绑定 `OrgId` 执行状态更新。
- 签名校验：
  - 回调密钥从连接器 `metadata.callbackSecret` 读取，兼容 `statusCallbackSecret`、`incomingSecret`、`webhookSecret` 和 `signatureSecret`。
  - 签名头使用 `X-CloudIaC-ITSM-Signature`，兼容平台 Webhook 既有 `X-CloudIaC-Signature`。
  - 签名格式为 `sha256=<hmac_sha256>`，签名内容为原始 JSON 请求体；也兼容只传 hex 摘要。
  - 未配置回调密钥、缺少签名或签名不匹配时返回权限错误。
- 工单定位：
  - 支持通过 CloudIaC 本地 `ticketId` 定位。
  - 支持通过外部系统返回的 `externalId` 或 `externalKey` 定位。
  - 限定在同一连接器和同一组织内查询，避免跨租户或跨连接器误更新。
- 状态回写：
  - 回调 payload 复用 `UpdateCloudItsmTicketStatus`，继续写入 `itsm.ticket.updated` 事件、`lastSyncedAt`、关闭时间、外部单号和响应 payload。
  - 回调 payload 会追加 `callback.signatureVerified=true`、签名算法、签名版本、连接器 ID 和接收时间，便于审计。
  - 风险整改工单会继续触发 `risk.remediation_status_synced`，把外部 ITSM 的 `resolved/closed/failed/canceled/in_progress` 等状态同步回风险视图。
- 前端提示：
  - ITSM 连接器编辑弹窗展示状态回调 URL、签名头和 `callbackSecret` 配置说明。
  - 扩展配置 placeholder 补充 `callbackSecret` 示例，降低对接外部 ITSM 时的配置成本。

验证：

- 新增单元测试覆盖 ITSM callback 签名匹配、bare hex 签名兼容、无效签名拒绝、callback 转状态更新表单和 payload 审计信息保留。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 平台主动周期拉取外部 ITSM 状态的 worker 已在 16.198 完成第一阶段；失败补偿队列和 provider 专用字段映射仍待继续。
- 回调密钥复用连接器 metadata，尚未提供独立密钥轮换 UI 和双密钥灰度期。

### 16.198 2026-06-22 V1.2 P0 外部 ITSM 状态周期拉取第一阶段

状态：已完成外部 ITSM 状态周期拉取第一阶段；除 16.197 的外部主动 callback 外，平台也可以按连接器配置主动查询外部工单状态，并复用已有工单状态更新、事件记录和风险整改状态同步链路。

已完成：

- 后端新增手动触发 API：
  - `POST /api/v1/cloud/itsm/tickets/sync-due`
  - 支持 `connectorId`、`force`、`limit` 参数。
  - 只处理当前组织内启用且配置了状态拉取能力的 ITSM 连接器。
- 后端新增后台 worker：
  - 新增 `StartCloudItsmStatusSyncWorker`，随 portal 后台 worker 一起启动。
  - 使用组织级 MySQL lock `cloudiac:itsm_status_sync:<orgId>`，避免多实例重复拉取。
  - 默认每 5 分钟扫描一次，单连接器默认最多同步 50 个待处理工单，单次手动触发最多 200 个。
- 连接器 metadata 配置：
  - `statusSyncEnabled=true`：显式开启周期拉取。
  - `statusFetchUrl/statusQueryUrl/statusUrl/getTicketUrl`：配置完整状态查询 URL 模板。
  - `statusFetchPath/statusQueryPath/statusPath/getTicketPath`：配置基于 `baseUrl` 的状态查询路径模板。
  - 模板支持 `{ticketId}`、`{externalId}`、`{externalKey}`、`{key}`、`{number}`。
  - `statusSyncIntervalSeconds`：控制单工单最小拉取间隔，默认 300 秒，范围 60 到 86400 秒。
- Provider 默认路径：
  - Jira：显式开启 `statusSyncEnabled=true` 且有 `externalKey` 时，默认查询 `/rest/api/2/issue/{externalKey}`。
  - ServiceNow：显式开启 `statusSyncEnabled=true` 且有 `externalId` 时，默认查询 `/api/now/table/<ticketType>/{externalId}`。
  - 通用 HTTP：要求显式配置状态查询 URL 或路径，避免误请求页面型 `baseUrl`。
- 外部状态归一：
  - 支持 metadata `statusMap` 自定义外部状态到平台状态的映射。
  - 默认识别 `open/new/todo` -> `submitted`，`in progress/active/assigned/on hold` -> `in_progress`，`done/resolved/completed/fixed` -> `resolved`，`closed` -> `closed`，`canceled/rejected/aborted` -> `canceled`，`failed/error` -> `failed`。
  - 兼容 Jira `fields.status.name`、ServiceNow `result.state/result.incident_state` 和通用 `status/state/ticket.status/data.status`。
- 状态回写闭环：
  - 周期拉取结果复用 `UpdateCloudItsmTicketStatus`，继续写入 `itsm.ticket.updated`、`lastSyncedAt`、关闭时间、响应 payload 和风险整改同步事件。
  - 响应 payload 追加 `statusSync.source=poll`、同步时间和远端状态，便于审计。
- 前端增强：
  - ITSM 工单页新增“同步外部状态”按钮，可手动触发当前组织的外部状态拉取。
  - 连接器编辑弹窗补充 `statusSyncEnabled`、`statusFetchPath/statusFetchUrl` 配置提示和 placeholder 示例。

验证：

- 新增单元测试覆盖状态查询 URL 模板替换、Jira 默认状态查询路径、metadata `statusMap`、常见外部状态归一、嵌套 JSON 状态提取。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 本阶段使用通用 HTTP GET 拉取外部状态；Jira/ServiceNow 更完整字段映射、分页批量查询、失败补偿队列、重试退避和状态同步审计报表仍待继续。
- 未新增独立 ITSM 同步任务表，第一阶段结果通过工单 `responsePayload`、事件和风险 evidence 保留。

### 16.199 2026-06-22 V1.2 P0 漂移风险整改触发自动修复第一阶段

状态：已完成漂移风险整改触发自动修复第一阶段；风险/漂移 ITSM 工单被外部回调、周期拉取或平台手动更新为 `resolved/closed` 后，平台会识别漂移类风险并尝试复用环境现有漂移任务克隆逻辑创建 drift apply 自动修复任务。

已完成：

- 触发入口：
  - 复用 `UpdateCloudItsmTicketStatus` 和 16.196 的风险整改状态同步链路。
  - 仅当风险整改工单同步后的风险状态为 `resolved` 时触发自动修复检查，避免处理中或失败状态误创建任务。
  - 支持通过风险 `source=drift`、`ruleKey=terraform_drift_detected`、工单参数 `driftRemediation=true`、`requestType=drift_remediation` 或 `targetState=iac_desired_state` 识别漂移整改。
- 环境定位和安全校验：
  - envId 按风险 finding、云操作、工单参数依次兜底。
  - 要求环境属于当前组织、状态为 `active`、未锁定、已开启 `openCronDrift` 且开启 `autoRepairDrift`。
  - 要求环境存在 `lastTaskId`，并且当前没有 pending/running/approving 的 drift task。
- 自动修复任务创建：
  - 复用 `services.GetTaskById` 和 `services.CloneNewDriftTask`。
  - 因环境开启 `autoRepairDrift`，克隆任务会自动使用 `TaskTypeApply` 和 `TaskSourceDriftApply`。
  - 保留既有任务参数、模板、Runner、变量和审批策略，不新增旁路执行器。
- 风险证据和事件：
  - 风险 evidence 写入 `driftAutoRepairCheckedAt`、`driftAutoRepairSource`、`driftAutoRepairEnvId`、`driftAutoRepairTriggered`、`driftAutoRepairTaskId`、`driftAutoRepairSkippedReason` 或错误原因。
  - `risk.remediation_status_synced` 事件 payload 增加 `driftAutoRepair` 明细。
  - 新增 `risk.drift_auto_repair_triggered` 事件，事件中心和通知事件类型新增中文展示“漂移自动修复任务触发”。
- 失败处理：
  - 自动修复触发按 best-effort 执行；环境未满足条件、已有漂移任务、源任务缺失或克隆失败不会阻断 ITSM 状态回写。
  - 所有跳过和失败原因都会沉淀到风险 evidence，便于后续审计和人工处理。

验证：

- 新增单元测试覆盖漂移整改识别、envId 优先级、自动修复 evidence 和触发标记解析。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段复用环境级 `openCronDrift/autoRepairDrift` 策略，不提供工单级覆盖开关。
- 自动修复任务仍沿用现有 drift apply 执行与审批能力；任务执行结果回写风险已在 16.200 完成第一阶段，回滚策略已在 16.203 完成第一阶段，详情页自动关联推荐已在 16.206 完成第一阶段。

### 16.200 2026-06-22 V1.2 P0 漂移自动修复任务结果回写风险第一阶段

状态：已完成漂移自动修复任务结果回写风险第一阶段；16.199 创建的 drift apply 自动修复任务结束后，任务状态会回写到关联风险 finding 的 evidence 和事件中心，失败/取消/驳回会重新打开风险，避免 ITSM 工单已解决但 IaC 自动修复实际失败时被误判为闭环完成。

已完成：

- 任务结束 hook：
  - 在 `TaskManager.processTaskDone` 的 `ChangeTaskStatusWithStep` 之后追加 best-effort 回写。
  - 仅处理 `isDriftTask=true`、`type=apply`、`source=driftApply` 且任务已进入终态的任务。
  - 不改变普通部署、普通 drift plan、手工 apply、webhook apply 和自动部署任务的状态流。
- 风险定位：
  - 通过风险 evidence 中的 `driftAutoRepairTaskId` 反查 16.199 创建的自动修复任务。
  - 限定同一组织，避免跨租户任务 ID 误关联。
- 状态映射：
  - drift apply `complete` -> 风险保持/确认 `resolved`。
  - drift apply `failed/aborted/rejected` -> 风险重新打开为 `open`，并清空 `resolvedAt/suppressedUntil`。
  - 非终态不回写，避免运行中任务误改风险状态。
- 证据回写：
  - 写入 `driftAutoRepairTaskLastSyncedAt`、`driftAutoRepairTaskStatus`、`driftAutoRepairTaskMessage`、`driftAutoRepairTaskRiskStatus`、`driftAutoRepairTaskStartedAt`、`driftAutoRepairTaskEndedAt`、`driftAutoRepairTaskCompleted` 和 `driftAutoRepairTaskFailed`。
  - 保留 16.199 的 `driftAutoRepairTaskId`、触发时间、触发来源和 skip/error 信息。
- 事件中心：
  - 新增 `risk.drift_auto_repair_result_synced` 事件。
  - 事件 payload 记录上一风险状态、同步后状态、任务 ID、任务状态、任务类型、任务来源、任务消息和环境 ID。
  - 事件中心和通知事件类型新增中文展示“漂移自动修复结果同步”。

验证：

- 新增单元测试覆盖 drift auto repair 任务识别、任务状态到风险状态映射、任务结果 evidence 生成和失败标记。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段按任务 ID 回写最近一次自动修复结果，不处理一个风险并发触发多次修复任务时的历史结果列表。
- 风险详情页自动修复入口已在 16.201 完成第一阶段，失败后重试审批已在 16.202 完成第一阶段，回滚策略记录已在 16.203 完成第一阶段，审批策略配置已在 16.204 完成第一阶段，完整任务时间线已在 16.205 完成第一阶段，自动关联推荐已在 16.206 完成第一阶段。

### 16.201 2026-06-22 V1.2 P0 风险详情页漂移自动修复入口第一阶段

状态：已完成风险详情页漂移自动修复入口第一阶段；风险详情抽屉不再只展示原始 evidence JSON，而是把 16.199/16.200 写入的漂移自动修复触发、跳过、任务执行和结果同步信息结构化展示，并提供环境和任务详情跳转。

已完成：

- 风险详情展示：
  - 新增“漂移自动修复”详情区。
  - 当 evidence 含 `driftAutoRepair*` 字段时自动展示，不影响普通云配置、CMDB、策略风险。
  - 展示触发状态、触发检查时间、环境 ID、环境状态、漂移检测开关、自动修复开关、源任务、修复任务、任务状态、风险同步状态、任务开始/结束时间、结果同步时间和任务消息。
- 跳转入口：
  - 有 `projectId/envId` 时，环境 ID 可跳转到环境详情。
  - 有 `driftAutoRepairTaskId` 时，修复任务可跳转到环境任务详情。
  - 有 `driftAutoRepairSourceTaskId` 时，源任务可跳转到任务详情。
- 审计提示：
  - 触发失败或跳过时，将 `driftAutoRepairSkippedReason` 映射为中文原因。
  - `driftAutoRepairError` 以错误提示展示。
  - `driftAutoRepairTaskFailed=true` 时展示自动修复任务失败提示。
  - `driftAutoRepairTaskCompleted=true` 时展示自动修复任务完成提示。

验证：

- 本阶段为前端详情入口增强，不新增后端接口；数据来源为风险详情已有 evidence。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以静态检查和镜像构建为主。

验证限制：

- 第一阶段展示最近一次自动修复任务，不提供多次修复历史时间线。
- 失败后重试审批已在 16.202 完成第一阶段，回滚策略记录已在 16.203 完成第一阶段，自动关联推荐已在 16.206 完成第一阶段。

### 16.202 2026-06-22 V1.2 P0 漂移自动修复失败后重试审批第一阶段

状态：已完成漂移自动修复失败后重试审批第一阶段；当 16.199/16.200 触发的 drift apply 自动修复任务失败、取消或被驳回后，平台会在风险回写链路中创建一个新的 drift apply 重试任务，并强制 `AutoApprove=false`，让重试进入既有 Terraform apply 审批流程。

已完成：

- 后端任务克隆能力：
  - `CloneNewDriftTask` 保持原有行为，继续按环境 `AutoApproval` 决定自动审批。
  - 新增可指定自动审批策略的漂移任务克隆入口，用于失败后重试审批场景。
- 风险回写链路增强：
  - `SyncCloudRiskDriftAutoRepairTaskResult` 在自动修复任务失败、取消或驳回时检查是否需要创建重试审批任务。
  - 默认仅自动创建 1 次重试审批任务，最大支持 evidence 配置到 3 次，避免失败任务无限循环重试。
  - 创建重试前会校验环境存在、状态活跃、未锁定、已开启漂移检测、已开启自动修复且当前没有 pending/running/approving 的漂移任务。
  - 重试任务复用原 drift apply 任务的模板、变量、Runner、代码版本和环境配置，但强制关闭自动审批，使 apply/destroy 步骤进入既有审批流。
- 风险 evidence 增强：
  - 写入 `driftAutoRepairRetryApprovalRequired`、`driftAutoRepairRetryApprovalRequested`、`driftAutoRepairRetryTaskId`、`driftAutoRepairRetryFromTaskId`、`driftAutoRepairRetryAttempt`、`driftAutoRepairRetryMaxAttempts`、`driftAutoRepairRetrySkippedReason`。
  - 保留失败任务快照 `driftAutoRepairLastFailedTaskId`、`driftAutoRepairLastFailedTaskStatus`、`driftAutoRepairLastFailedTaskMessage`，同时把当前 `driftAutoRepairTaskId` 指向待审批重试任务，便于后续结果继续回写同一风险。
- 事件中心：
  - 新增 `risk.drift_auto_repair_retry_approval_requested` 事件，事件中心和通知事件类型新增中文展示“漂移自动修复重试审批”。
- 前端风险详情：
  - “漂移自动修复”详情区新增失败任务、重试任务、重试审批、重试次数和重试检查时间。
  - 已创建重试审批时展示“待审批”提示，并可跳转到重试任务详情。
  - 未创建重试时展示中文跳过原因，例如达到重试上限、已有重试任务、环境锁定或自动修复关闭。

验证：

- 新增单元测试覆盖失败/取消/驳回状态需要重试审批、重试次数默认值和上限、重试审批 evidence 关键字段。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段创建的是待审批 drift apply 重试任务，审批策略配置已在 16.204 完成第一阶段，审批通知/SLA 升级已在 16.207 完成第一阶段；团队/项目策略模板仍待后续增强。
- 本阶段只做失败后重试审批；失败后的回滚策略记录和前端提示已在 16.203 完成第一阶段。

### 16.203 2026-06-22 V1.2 P0 漂移自动修复回滚策略记录第一阶段

状态：已完成漂移自动修复回滚策略记录第一阶段；平台不再把 drift apply 失败只表现为普通任务失败，而是在风险 evidence、事件中心和风险详情中明确记录回滚评审策略、疑似部分变更、安全级别和下一步动作，避免自动修复失败后无人判断是否需要回退。

已完成：

- 后端回滚策略评估：
  - `cloudItsmDriftAutoRepairTaskResultEvidence` 在写入任务结果时同步生成 `driftAutoRepairRollback*` evidence。
  - 自动修复完成态写入 `driftAutoRepairRollbackRequired=false` 和 `no_platform_rollback_required`，说明已按 IaC 期望态完成，不需要平台自动回滚。
  - 自动修复失败、取消或驳回时写入 `driftAutoRepairRollbackRequired=true`、`gitops_iac_change_or_manual_review_required` 和人工评审下一步动作。
  - 当 Terraform 结果里存在 added/changed/destroyed 计数时，标记 `driftAutoRepairRollbackPartialApplySuspected=true`，并将安全级别提升为 `high`。
- 风险 evidence 增强：
  - 写入 `driftAutoRepairRollbackEvaluatedAt`、`driftAutoRepairRollbackStrategy`、`driftAutoRepairRollbackSafetyLevel`、`driftAutoRepairRollbackHint`、`driftAutoRepairRollbackNextAction`、`driftAutoRepairRollbackRequiresApproval`。
  - 写入 `driftAutoRepairRollbackPlanChanges` 和 `driftAutoRepairRollbackApplyChanges`，保留 plan/apply 的资源变更计数和成本摘要。
- 事件中心：
  - 自动修复失败且需要回滚评审时新增 `risk.drift_auto_repair_rollback_strategy_recorded` 事件。
  - 事件中心和通知事件类型新增中文展示“漂移自动修复回滚策略记录”。
- 前端风险详情：
  - “漂移自动修复”详情区新增回滚策略、回滚安全级别、是否需要回滚评审、是否疑似部分变更、回滚评估时间、是否需要审批和回滚下一步动作。
  - 需要回滚评审时展示告警，提示先核对云端资源、Terraform state 和 IaC 代码，再决定审批重试或通过 GitOps/IaC PR 回退。

验证：

- 新增单元测试覆盖完成态无需回滚、失败且疑似部分 apply 时需要高风险回滚评审、资源变更摘要和审批标记。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段是“回滚策略记录和提示”，不会自动执行反向 Terraform apply，也不会绕过 GitOps/IaC PR review。
- 真正的回滚动作仍建议通过 GitOps/IaC 变更申请、PR review、流水线和既有任务审批执行；按团队/项目定制审批策略已在 16.204 完成第一阶段，审批通知/SLA 升级已在 16.207 完成第一阶段。

### 16.204 2026-06-22 V1.2 P0 漂移自动修复审批策略配置第一阶段

状态：已完成漂移自动修复审批策略配置第一阶段；平台不再只用固定环境自动审批或失败重试强制审批，而是在环境扩展配置中支持漂移自动修复专用审批策略，并把策略判定结果写入风险 evidence 和风险详情页，便于审计“为什么自动审批/为什么需要审批”。

已完成：

- 后端审批策略解析：
  - 新增 `cloudItsmDriftAutoRepairApprovalPolicyForEnv`，从 `env.extraData.driftAutoRepairApprovalPolicy` 读取漂移自动修复审批策略。
  - 支持 `mode=inherit_env`、`require_approval`、`auto_approve`、`retry_require_approval`。
  - 支持 `autoApprove`、`retryAutoApprove`、`requireApproval`、`retryRequireApproval` 作为覆盖项，支持 `approverRoles` 记录审批角色。
  - 默认策略保持兼容：首次自动修复继承环境 `AutoApproval`；失败后重试默认仍需要审批，避免失败任务直接循环自动 apply。
  - 无效策略模式按 fail-closed 处理，强制进入审批，避免配置拼写错误导致绕过人工检查。
- 任务创建链路：
  - 初次漂移自动修复改为按策略调用 `CloneNewDriftTaskWithAutoApprove`，而不是只继承环境 `AutoApproval`。
  - 失败后重试改为按策略决定是否自动审批；默认行为仍是创建待审批重试任务。
  - 写入 `driftAutoRepairTaskAutoApprove`，保留任务实际自动审批状态。
- 风险 evidence 增强：
  - 写入 `driftAutoRepairApprovalPolicyContext`、`driftAutoRepairApprovalPolicySource`、`driftAutoRepairApprovalPolicyMode`。
  - 写入 `driftAutoRepairApprovalAutoApprove`、`driftAutoRepairApprovalRequired`、`driftAutoRepairApprovalReason`、`driftAutoRepairApprovalRoles`。
  - 失败重试 evidence 根据策略同步 `driftAutoRepairRetryApprovalRequired`、`driftAutoRepairRetryApprovalStatus` 和 `driftAutoRepairTaskRetryPendingApproval`。
- 前端风险详情：
  - “漂移自动修复”详情区新增审批策略、策略来源、自动审批、需要审批、审批角色和策略原因。
  - 中文展示默认环境审批、默认重试保护、环境扩展策略、强制审批、自动审批和失败重试需审批等策略含义。

验证：

- 新增单元测试覆盖默认策略、环境扩展强制审批、审批角色 evidence、失败重试自动审批覆盖、无效策略 fail-closed。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段复用 `env.extraData` 承载策略，不新增独立审批策略表，也不改变既有 Terraform apply 审批步骤。
- 策略只决定漂移自动修复任务 `AutoApprove`；审批通知/SLA 升级已在 16.207 完成第一阶段，团队/项目维度策略模板仍待后续补齐。

### 16.205 2026-06-22 V1.2 P0 漂移自动修复完整任务时间线第一阶段

状态：已完成漂移自动修复完整任务时间线第一阶段；风险详情不再只依赖散落的 `driftAutoRepair*` 字段判断处理过程，而是在风险 evidence 中生成统一的 `driftAutoRepairTimeline`，前端按阶段展示触发检查、源任务定位、审批策略判定、任务创建、任务执行、结果同步、失败重试和回滚评估。

已完成：

- 后端时间线生成：
  - 新增 `cloudItsmDriftAutoRepairTimeline` 和 `cloudItsmDriftAutoRepairAppendTimeline`，从现有 risk evidence 派生结构化时间线。
  - ITSM 工单解决触发自动修复时，自动写入触发检查、源任务定位、审批策略判定和任务创建节点。
  - 自动修复任务结果回写风险时，自动补齐任务结束、风险结果同步、失败重试检查/重试任务创建和回滚策略评估节点。
  - 时间线节点包含 `stage`、`title`、`status`、`time`、`taskId`、`reason`、`attempt`、`maxAttempts`、`safetyLevel` 等审计字段。
- 前端风险详情：
  - “漂移自动修复”详情区新增“处理时间线”。
  - 时间线按节点状态显示已触发、已跳过、自动审批、需要审批、执行中、完成、失败、待审批、需回滚评审等状态标签。
  - 节点附带任务 ID、风险状态、审批策略、跳过/审批原因、重试次数、回滚安全级别和任务消息。
- 兼容性：
  - 第一阶段不新增数据库表；时间线由 evidence 派生并持久化在风险 finding 中。
  - 旧 evidence 即使没有 `driftAutoRepairTimeline` 也仍可通过原有明细字段展示。

验证：

- 新增单元测试覆盖完成态自动修复时间线、失败后重试待审批时间线和需要回滚评审节点。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段为风险详情内的处理时间线，不提供跨风险/跨任务的全局时间线检索。
- 自动关联推荐已在 16.206 完成第一阶段，审批通知/SLA 升级已在 16.207 完成第一阶段，机器人处理标签已在 16.208 完成第一阶段，推荐采纳确认流已在 16.209 完成第一阶段，自助目录权限策略和 SLA 趋势已在 16.210 完成第一阶段。

### 16.206 2026-06-22 V1.2 P0 漂移自动修复自动关联推荐第一阶段

状态：已完成漂移自动修复自动关联推荐第一阶段；风险详情不再只展示任务和时间线，而是根据自动修复当前 evidence 派生下一步推荐动作，帮助运维/业务在跳过、待审批、失败、回滚评审和完成场景下直接定位环境、任务、审批或 GitOps/IaC PR 动作。

已完成：

- 后端推荐生成：
  - 新增 `driftAutoRepairRecommendations` evidence，由 `cloudItsmDriftAutoRepairRecommendations` 从当前 `driftAutoRepair*` evidence 自动派生。
  - 自动修复跳过时，按原因推荐开启漂移检测、开启自动修复、解除环境锁定、恢复环境活跃状态、先执行一次部署任务或检查已有漂移任务。
  - 自动修复任务创建后，推荐查看 drift apply 自动修复任务。
  - 审批策略要求人工审批时，推荐审批自动修复任务或失败后的重试任务。
  - 自动修复失败时，推荐查看失败任务日志和 Terraform 输出。
  - 回滚策略要求人工评审时，推荐回滚评审，并提示通过 GitOps/IaC PR 提交修复或回滚变更。
  - 自动修复完成且无需回滚时，推荐确认风险关闭并观察下一次漂移检测结果。
- 前端风险详情：
  - “漂移自动修复”详情区新增“关联推荐”。
  - 推荐项展示动作标题、优先级、目标链接和原因。
  - 环境类目标跳转环境详情，任务类目标跳转对应环境任务详情。
  - 支持严重/高/中/低优先级展示，便于优先处理待审批、失败和回滚评审。
- 兼容性：
  - 第一阶段不新增推荐表，不改变审批和任务执行流程；推荐内容由 evidence 派生并持久化。
  - 旧风险没有 `driftAutoRepairRecommendations` 时仍按时间线和明细字段展示。

验证：

- 新增单元测试覆盖自动修复跳过时的配置推荐、失败重试审批推荐、回滚评审推荐和 GitOps/IaC PR 推荐。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 推荐项展示和跳转已在 16.206 完成第一阶段；“一键确认采纳推荐”的闭环记录已在 16.209 完成第一阶段。
- 审批通知/SLA 升级已在 16.207 完成第一阶段，机器人处理标签已在 16.208 完成第一阶段，推荐采纳确认流和采纳率统计已在 16.209 完成第一阶段，自助目录权限策略和 SLA 趋势已在 16.210 完成第一阶段。

### 16.207 2026-06-22 V1.2 P0 漂移自动修复审批通知/SLA 升级第一阶段

状态：已完成漂移自动修复审批通知/SLA 升级第一阶段；自动修复任务进入人工审批、失败重试审批或回滚评审时，平台会在风险 evidence 中写入 SLA 状态，并通过风险事件复用现有事件中心、通知策略、Webhook 和 ITSM 自动派单链路进行通知或升级。

已完成：

- 后端 SLA 策略：
  - 新增 `driftAutoRepairSla*` evidence，记录 SLA 策略来源、审批截止分钟、回滚评审截止分钟、即将超时阈值、通知路由、负责人、指派人和是否超时自动创建 ITSM 工单。
  - 默认自动修复审批 SLA 为 4 小时，回滚评审 SLA 为 24 小时，即将超时阈值为 30 分钟。
  - 支持从环境 `extraData.driftAutoRepairSlaPolicy` / `driftAutoRepairSLA` / `driftSlaPolicy` 覆盖 SLA 分钟数、通知路由、负责人、指派人和超时自动派单开关。
- SLA 状态推导：
  - 自动修复任务需要人工审批时，生成 `approve_repair` SLA 动作。
  - 自动修复失败后创建重试审批任务时，生成 `approve_retry` SLA 动作。
  - 自动修复失败并需要回滚评审时，生成 `review_rollback` SLA 动作。
  - 计算 `driftAutoRepairSlaStartedAt`、`driftAutoRepairSlaDueAt`、`driftAutoRepairSlaStatus`、`driftAutoRepairSlaMinutesRemaining` 和 `driftAutoRepairSlaEscalationRequired`。
  - SLA 阶段写入 `driftAutoRepairTimeline`，与触发检查、审批策略、任务执行、重试和回滚评估一起展示。
- 通知和升级事件：
  - 新增 `risk.drift_auto_repair_approval_notification_requested`，用于审批/评审未超时但需要通知负责人处理的场景。
  - 新增 `risk.drift_auto_repair_sla_escalated`，用于 SLA 已超时并需要升级处理的场景。
  - 事件 payload 写入 `notificationRoutes`、`notificationOwner`、`notificationAssignees` 和 `notificationEscalationReason`，复用现有通知候选路由能力。
  - 环境策略开启 `autoTicket` 时，SLA 超时事件会带上 `itsmAutoTicket`、标题、描述和优先级，复用现有事件自动创建 ITSM 工单能力。
- 前端风险详情：
  - “漂移自动修复”详情区新增审批/评审 SLA 提示。
  - 展示 SLA 动作、状态、开始时间、截止时间、剩余分钟、是否升级、通知路由和负责人。
  - SLA 超时时展示错误提示；即将超时或进行中时展示通知提示。
- 事件中文展示：
  - 事件中心和通知事件类型新增“漂移自动修复审批通知”“漂移自动修复 SLA 升级”。

验证：

- 新增单元测试覆盖环境 SLA 策略覆盖、审批即将超时、回滚评审 SLA 超时升级、通知路由和超时自动派单 payload。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段在自动修复触发和任务结果回写时计算 SLA；尚未新增独立后台扫描器对长期未处理 SLA 做周期性二次升级。
- 审批通知复用平台事件/通知/Webhook/ITSM 派单链路；真实邮件、钉钉、企业微信、Slack、外部 ITSM 通道仍需按企业配置做端到端联调。
- 机器人处理标签已在 16.208 完成第一阶段，推荐采纳确认流已在 16.209 完成第一阶段，自助目录权限策略和 SLA 趋势统计已在 16.210 完成第一阶段，项目/申请类型维度目标趋势已在 16.211 完成第一阶段；组织架构团队维度仍待继续增强。

### 16.208 2026-06-22 V1.2 P0 ITSM 机器人处理标签第一阶段

状态：已完成 ITSM 机器人处理标签第一阶段；自助运维工单和云事件自动派单不再只依赖“提交痕迹”估算自动化处理，而是在工单请求载荷、响应和前端台账中显式记录平台机器人处理证据。

已完成：

- 后端工单载荷增强：
  - 自助运维申请在创建 `CloudOperation` 前写入 `robotProcessed`、`robotProcessor`、`robotProcessingTags`、`robotAutomationMode` 和 `robotTicketChannel`。
  - GitOps/IaC 变更、漂移修复、风险整改、权限申请会自动打上对应机器人处理标签。
  - 本地待提交工单标记 `local_ticket`，外部 ITSM 提交标记 `external_itsm`。
  - 云事件自动派单写入 `event_auto_ticket` 标签，并保留派单原因、失败分类和云服务信息。
- 响应和指标：
  - `CloudItsmTicketResp` 返回机器人处理状态、处理器、标签、自动化模式和处理通道。
  - ITSM 概览新增 `robotProcessedTicketTotal` 与 `robotProcessingRate`。
  - “工单自动化处理率”统计优先合并 `request_payload.robotProcessed=true`，兼容历史有提交痕迹的工单。
- 前端 ITSM 页面：
  - 工单列表新增“机器人处理”列。
  - 工单详情展示机器人处理状态、处理器、自动化模式、处理通道和处理标签。
  - 目标卡片补充机器人处理单数，便于跟踪公司要求的自动化处理率目标。

验证：

- 新增单元测试覆盖自助运维 GitOps/IaC 工单机器人标签、事件自动建单机器人标签和响应字段解析。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段提供平台内机器人处理标签和统计口径；项目/申请类型维度目标趋势已在 16.211 完成第一阶段，外部 ITSM 专用字段映射已在 16.212 完成第一阶段，失败补偿队列已在 16.213 完成第一阶段，组织架构团队维度仍待继续增强。
- 推荐采纳确认流和推荐采纳率统计已在 16.209 完成第一阶段，自助目录权限策略和组织级 SLA 趋势统计已在 16.210 完成第一阶段，项目/申请类型维度目标趋势已在 16.211 完成第一阶段，外部 ITSM 专用字段映射已在 16.212 完成第一阶段，失败补偿队列已在 16.213 完成第一阶段；组织架构团队维度仍待继续增强。

### 16.209 2026-06-22 V1.2 P0 漂移自动修复推荐采纳确认流第一阶段

状态：已完成漂移自动修复推荐采纳确认流第一阶段；风险详情中的自动修复关联推荐不再只是静态提示，用户可以确认已采纳某条推荐，平台会把采纳记录写回风险 evidence 并生成风险事件。

已完成：

- 后端推荐采纳 API：
  - 新增 `POST /api/v1/cloud/risks/:id/recommendations/adopt`。
  - 请求参数支持 `action`、`targetType`、`targetId` 和 `comment`。
  - 后端会校验推荐动作必须存在于当前风险 `driftAutoRepairRecommendations`，避免写入无效动作。
- Evidence 闭环：
  - 推荐项写入 `adopted`、`adoptedAt`、`adoptedBy` 和 `adoptionComment`。
  - 采纳历史追加到 `driftAutoRepairRecommendationAdoptions`。
  - 写入 `driftAutoRepairLastRecommendationAdoption`、`driftAutoRepairRecommendationTotal`、`driftAutoRepairRecommendationAdoptedTotal`、`driftAutoRepairRecommendationAdoptionRate` 和是否全部采纳。
  - 采纳动作生成 `risk.recommendation_adopted` 风险事件，便于事件中心、通知和审计追踪。
- 前端风险详情：
  - “关联推荐”每条推荐新增“确认采纳”按钮。
  - 已采纳推荐展示采纳状态和采纳时间。
  - 点击确认后刷新风险列表和详情，采纳记录即时可见。

验证：

- 新增单元测试覆盖推荐采纳 evidence 写入、采纳率计算、推荐项打标、采纳备注和未知推荐动作拒绝。
- 本阶段不启动 compose 服务，遵守已清理部署数据约束；验证以单测、静态检查和镜像构建为主。

验证限制：

- 第一阶段只记录“已采纳”确认，不自动执行环境配置、审批、GitOps PR 或风险关闭动作。
- 采纳统计先落在单个风险 evidence 中；跨团队/项目的推荐采纳趋势报表仍待后续增强。

### 16.210 2026-06-22 V1.2 P0 ITSM 自助目录权限策略/SLA 趋势第一阶段

状态：已完成 ITSM 自助目录权限策略和 SLA 趋势第一阶段；自助运维目录不再只是服务项列表，而是为每个目录项返回平台内置权限策略、适用范围和 SLA 目标，并在工单载荷与概览页中形成可审计的组织级趋势口径。

已完成：

- 后端目录策略增强：
  - `CloudItsmCatalogItemResp` 新增 `policyKey`、`policyName`、`policyDescription`、`requiredRoles`、`allowedScopes`、`slaMinutes` 和 `slaDescription`。
  - 内置自助申请提供专门策略：权限申请、GitOps/IaC 变更、风险整改、漂移修复分别绑定不同角色、范围和 SLA。
  - 云操作目录按生命周期、扩缩容、配置变更、备份和审批型操作自动补齐默认策略。
- 工单载荷增强：
  - 自助申请创建时把 `selfServicePolicy` 和 `sla` 快照写入 operation params。
  - ITSM 请求载荷顶层同步返回 `selfServicePolicy` 和 `sla`，便于外部 ITSM、审计和后续报表读取同一份策略证据。
- SLA 趋势统计：
  - ITSM overview 新增 `selfServicePolicyTotal`、`selfServicePolicyCovered`、`selfServicePolicyRate`。
  - 新增 `selfServiceSlaTicketTotal`、`selfServiceSlaMetTotal`、`selfServiceSlaBreached`、`selfServiceSlaMetRate` 和 `selfServiceSlaTarget`。
  - 返回近 7 天组织级 `slaTrend`，按工单创建日期统计 SLA 工单数、达成数、超时数和达成率。
  - 已解决/已关闭且未超时的工单计为达成；终态失败/取消或超时仍未完成的工单计为超时；仍在 SLA 窗口内的处理中工单计入 SLA 工单但不提前判定达成或超时。
- 前端 ITSM 页面：
  - 顶部新增“自助 SLA 达成”目标卡片，展示目标、达成数、SLA 工单数和超时数。
  - 自助目录新增“权限策略”“范围”“SLA”列。
  - 自助目录页签新增近 7 天自助 SLA 趋势表。

验证：

- 新增单元测试覆盖自助目录策略返回、策略/SLA 写入工单载荷和 SLA 达成/超时/处理中判定。
- 使用 Docker Go 镜像完成 `gofmt`。
- 容器化目标单元测试通过：`TestCloudItsmCatalogPolicyForSelfService`、`TestCloudItsmSelfServicePolicyPayload`、`TestCloudItsmTicketSlaResult`、既有 ITSM 机器人处理和事件建单测试。
- `git diff --check` 通过。
- `docker compose build iac-portal iac-web` 成功；前端仅存在既有 webpack bundle size warning。
- 验证后已执行 `docker compose down -v --remove-orphans`，部署数据目录保持清理状态。

验证限制：

- 第一阶段提供平台内置权限策略和组织级 7 日 SLA 趋势；策略尚未做成可配置数据表。
- 项目和申请类型维度趋势已在 16.211 完成第一阶段；真实组织架构团队、环境维度和更细筛选仍待继续增强。
- 外部 ITSM 专用字段映射和失败补偿队列仍需继续增强。

### 16.211 2026-06-22 V1.2 P0 ITSM 项目/申请类型目标趋势第一阶段

状态：已完成 ITSM 项目/申请类型目标趋势第一阶段；自助运维目标不再只看组织总览，而是能按项目和申请类型拆分近 30 天的自助占比、自动化处理率和 SLA 达成率，支撑业务团队自助化 80% 与工单自动化 60% 的落地跟踪。

已完成：

- 后端 overview 维度趋势：
  - `CloudItsmOverviewResp` 新增 `projectTrends` 和 `requestTypeTrends`。
  - 新增 `CloudItsmDimensionMetricResp`，返回维度类型、维度 ID、维度名称、窗口天数、工单总数、自助工单数、自动化工单数、机器人处理工单数、SLA 工单数、达成数、超时数和目标值。
  - 近 30 天工单按 `projectId` 归组，未关联项目归入“未关联项目”。
  - 近 30 天工单按申请类型归组，优先从请求载荷、operation action、params、robot requestType 提取并映射自助目录名称。
  - 自助占比按 `selfServiceTicketTotal / ticketTotal` 计算。
  - 自动化处理率复用 16.208 的机器人处理标签和历史提交痕迹兼容口径。
  - SLA 达成率复用 16.210 的 SLA 达成/超时判定。
- 前端 ITSM 页面：
  - 自助目录页签新增“项目目标趋势”和“申请类型趋势”两张表。
  - 每张表展示工单数、自助占比、自动化处理率和 SLA 达成率。
  - 指标标签按对应目标值显示达标/预警状态。

验证：

- 新增单元测试覆盖维度指标聚合中的自助占比、自动化处理率、机器人处理数和 SLA 达成/超时数。
- 使用 Docker Go 镜像完成 `gofmt`。
- 容器化目标单元测试通过：`TestCloudItsmDimensionMetricAddTicket` 以及既有 ITSM 自助目录、SLA、机器人处理和事件建单测试。

验证限制：

- 第一阶段使用项目作为团队/业务单元的落地点；真实组织架构团队、值班组或业务线模型接入后，可继续扩展团队维度。
- 第一阶段为 overview 内聚合指标；跨时间序列的项目日报/周报和导出仍待后续增强。
- 自助目录策略仍为内置规则，可配置策略表仍待继续增强；外部 ITSM 专用字段映射已在 16.212 完成第一阶段，失败补偿队列已在 16.213 完成第一阶段。

### 16.212 2026-06-22 V1.2 P0 外部 ITSM 专用字段映射第一阶段

状态：已完成外部 ITSM 专用字段映射第一阶段；平台在继续保留 CloudIaC 标准审计载荷的同时，可根据连接器 provider 生成 Jira、ServiceNow 或通用 ITSM 系统需要的提交字段，避免把平台内部 canonical payload 直接投递给外部系统。

已完成：

- 后端提交载荷增强：
  - 工单请求载荷新增 `externalPayload`、`externalPayloadProvider` 和 `externalPayloadMode`。
  - 本地工单和 dry-run 继续保留完整 CloudIaC canonical payload，便于审计和排障。
  - 外部提交时优先提交 `externalPayload`；未配置外部字段映射的通用连接器仍按原 canonical payload 提交，保持兼容。
- Jira 字段映射：
  - `provider=jira` 默认生成 Jira Create Issue 结构：`fields.project.key`、`fields.summary`、`fields.description`、`fields.issuetype.name`、`fields.priority.name` 和 `fields.labels`。
  - 支持 `priorityMapping` 和 `labels` 元数据覆盖，保留 `cloudiac` 和 `cloudiac_<priority>` 标签。
- ServiceNow 字段映射：
  - `provider=servicenow` 默认生成 Table API 结构：`short_description`、`description`、`urgency`、`impact`、`category`、`subcategory` 和 `u_cloudiac_*` 追踪字段。
  - 支持 `urgencyMapping`、`impactMapping`、`ticketTable/table`、`category` 和 `subcategory` 元数据覆盖。
  - 外部响应解析新增兼容 `result.sys_id`、`result.number`、`result.self/link`。
- 通用字段映射：
  - 元数据支持 `fieldDefaults/defaultFields`、`fieldMappings/fieldMapping`、`payloadTemplate/externalPayloadTemplate`。
  - `fieldMappings` 支持 `$.title`、`$.description`、`$.operation.id`、`$.event.id`、`$.operation.params.xxx` 等路径映射，也支持字面量值。
  - 支持点号目标路径，如 `fields.summary`、`details.operation_id`、`u_cloudiac_operation_id`。
- 前端连接器配置：
  - ITSM 连接器“扩展配置” placeholder 增加 `fieldDefaults` 和 `fieldMappings` 示例，降低 Jira、ServiceNow 和自定义 ITSM 对接配置成本。

验证：

- 新增单元测试覆盖 Jira 默认字段、优先级映射和标签生成。
- 新增单元测试覆盖通用连接器自定义字段默认值、路径映射和字面量映射。
- 新增 `httptest` 覆盖 ServiceNow 提交体确认为映射后的外部 payload，并验证 `result.sys_id/result.number` 解析为平台工单外部 ID/Key。

验证限制：

- 第一阶段提供 provider 默认字段和配置化字段映射；未接入真实 Jira/ServiceNow 租户做端到端联调。
- 字段映射先支持对象路径和常见默认值，不包含复杂模板表达式、数组下标、条件分支或 provider API schema 自动发现。
- 外部 ITSM 失败补偿队列、重试退避和提交失败审计报表仍待继续增强。

### 16.213 2026-06-22 V1.2 P0 外部 ITSM 失败补偿队列第一阶段

状态：已完成外部 ITSM 失败补偿队列第一阶段；外部 Jira、ServiceNow 或通用 HTTP 建单失败后，不再只停留在 failed 状态，而是记录提交 attempt、下一次重试时间、退避策略和死信状态，并支持后台 worker 与页面按钮触发到期补偿。

已完成：

- 后端提交失败留痕：
  - `submitCloudItsmTicket` 增加 attempt 参数和 `responsePayload.submitRetry`。
  - 失败提交记录 `attempt`、`maxAttempts`、`lastError`、`lastStatusCode`、`backoffSeconds`、`nextAttempt`、`nextRetryAt`、`deadLetter` 和 `reason`。
  - 成功提交也记录本次 attempt，便于审计确认补偿是否完成。
- 失败补偿策略：
  - 连接器 metadata 支持 `submitRetryEnabled`、`submitRetryMaxAttempts`、`submitRetryBackoffSeconds`、`submitRetryMaxBackoffSeconds`。
  - 默认启用，最大提交次数默认 3 次，上限 10 次。
  - 使用指数退避，默认 300 秒，默认最大退避 3600 秒。
  - 默认不重试已经带外部单号/外部 URL 的 failed 工单，避免重复创建外部工单；如确需覆盖可配置 `submitRetryAllowExternalIdentity=true`。
- 补偿执行入口：
  - 新增 `POST /api/v1/cloud/itsm/tickets/retry-failed`，支持 `connectorId`、`force` 和 `limit`。
  - 新增 `RetryDueCloudItsmTicketSubmissionsForAllOrgs` 和 `StartCloudItsmSubmitRetryWorker`，portal 启动后每 5 分钟处理一次到期失败提交。
  - 补偿成功生成 `itsm.ticket.retry_submitted` 事件；补偿失败生成 `itsm.ticket.retry_failed` 事件。
- 前端 ITSM 页面：
  - 工单页工具栏新增“重试失败提交”按钮。
  - 连接器扩展配置 placeholder 增加提交失败补偿配置示例。

验证：

- 新增单元测试覆盖补偿 metadata 写入、退避时间、到期判断、外部单号跳过、达到最大次数进入死信和 HTTP 502 失败后写入下一次重试时间。

验证限制：

- 第一阶段复用现有 `iac_cloud_itsm_ticket.response_payload.submitRetry` 作为轻量队列状态，尚未新增独立队列表；轻量队列报表和死信重放入口已在 16.215 完成第一阶段。
- 第一阶段支持后台 worker、手动触发、最大次数和指数退避；重试抖动、最大重试窗口和更细失败分类统计仍待继续。
- 未对真实 Jira/ServiceNow 租户做端到端失败补偿联调。

### 16.214 2026-06-22 V1.2 P0 GitOps/IaC 门禁外部回写第一阶段

状态：已完成 GitOps/IaC PR review 和 Pipeline 门禁外部回写第一阶段；自助申请创建后，GitLab/GitHub/Jenkins 等外部系统可通过签名 callback 把 Review、Pipeline 和运行信息回写到 CloudOperation、ITSM 工单载荷、事件和审计记录。

已完成：

- 新增免登录外部回调入口：
  - `POST /api/v1/cloud/itsm/callbacks/:connectorId/gitops-gate`。
  - 使用 `X-CloudIaC-ITSM-Signature` 或 `X-CloudIaC-Signature` 进行 HMAC-SHA256 验签。
  - 连接器 metadata 支持专用 `gitOpsGateCallbackSecret`，并兼容 `gitOpsCallbackSecret`、`vcsCallbackSecret`、`ciCallbackSecret`，未配置专用密钥时回退 `callbackSecret`。
- 回调定位能力：
  - 支持通过 `operationId`、`ticketId` 或 `pullRequestUrl` 定位 GitOps/IaC 自助申请。
  - 限定目标必须是 `self_service + gitops_iac_change`，避免外部回调误写普通工单或云操作任务。
- 门禁状态回写：
  - 回调字段支持 `repository`、`branch`、`targetBranch`、`changePath`、`pullRequestUrl`、`reviewStatus`、`pipelineUrl`、`pipelineStatus`、`commitSha`、`externalRunId`、`comment` 和原始 `payload`。
  - 复用 16.194 的 Review/Pipeline 状态归一化和 `passed/waiting/blocked` 判定。
  - 回写 `CloudOperation.Params.gitOpsGate`、`gateStatus`、`gitOpsLastCallback`、`gitOpsCallbackCount`、`CloudOperation.Result.gitOpsGate` 和 `gitOpsGateStatus`。
  - 同步刷新 `CloudItsmTicket.requestPayload.operation.params`、顶层 `gitOpsGate`、`gitOpsGateCallback` 和 `responsePayload.gitOpsGate`。
- 事件和审计：
  - 每次回写记录云操作审计。
  - 每次回写生成 `itsm.gitops_gate.updated` 事件。
  - `passed` 记为 info，`waiting` 记为 warning，`blocked` 记为 error，便于后续通知和治理统计。
- 前端 ITSM 连接器配置弹窗：
  - 展示 GitOps/IaC 门禁回写 URL。
  - 扩展配置示例新增 `gitOpsGateCallbackSecret`。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化本轮后端 Go 文件。
- 新增单元测试覆盖：
  - GitOps 专用回调密钥优先级和 `callbackSecret` 回退。
  - 回调合并 Review/Pipeline 状态后重新计算 `passed` 门禁。
  - 回调从既有 `gitOpsGate` 快照恢复 PR/Pipeline URL。
- `git diff --check` 通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-portal iac-web` 通过：
  - `iac-portal` Go 编译通过。
  - `iac-web` webpack 编译通过，仅存在既有 bundle size 警告。
- 使用 `CLOUDIAC_DEPLOY_ROOT=$PWD/.deploy/cloudiac`、`MYSQL_USER=cloudiac`、`MYSQL_PASSWORD=mysqlpass`、`MYSQL_DATABASE=cloudiac` 短暂启动 `iac-portal`：
  - `GET /api/v1/check` 返回 `success=true`，版本 `v1.3.5`。
  - `POST /api/v1/cloud/itsm/callbacks/citc-missing/gitops-gate` 命中新路由并返回应用层 `40410013`，确认不是路由缺失。
- 验证后执行 `docker compose down -v`，并删除 `.deploy/cloudiac` 临时运行目录。

验证限制：

- 第一阶段提供通用 webhook/callback 写入，不主动调用 GitLab/GitHub/Jenkins API 拉取 MR approval 或 Pipeline 结果。
- 第一阶段不新增 GitOps Repo 变更记录表，diff、commit、approval 明细先保存在回调 payload。
- 真实 GitLab/GitHub/Jenkins 租户 webhook 端到端联调仍待继续。
- 目标 Go 单测曾以 Docker Go 容器执行，但受首次依赖下载耗时影响未跑完；当前以 Docker compose 镜像构建、后端健康检查和路由探测作为本轮验证证据。

### 16.215 2026-06-22 V1.2 P0 ITSM 提交失败队列报表与死信重放第一阶段

状态：已完成 ITSM 提交失败补偿队列可观测和死信重放第一阶段；外部 ITSM 建单失败后，平台不再只有“批量重试”按钮，而是可以按 due/future/dead letter/skipped 查看队列、统计原因，并对单个死信或失败工单执行人工重放。

已完成：

- 后端新增提交失败补偿队列 API：
  - `GET /api/v1/cloud/itsm/tickets/retry-queue/summary`：返回失败总数、待重试、等待退避、死信、跳过、可重试数量、最早到期时间、下一次重试时间和原因分布。
  - `GET /api/v1/cloud/itsm/tickets/retry-queue`：分页返回失败提交队列，支持 `q`、`connectorId` 和 `queueStatus=due/future/dead_letter/skipped` 过滤。
  - `POST /api/v1/cloud/itsm/tickets/:id/replay-submit`：对单条失败提交执行人工重放，`force=true` 时允许从 `max_attempts_reached` 死信状态重新提交。
- 队列分类口径：
  - `due`：已到期且满足补偿条件。
  - `future`：仍在退避窗口内。
  - `dead_letter`：达到最大尝试次数或 `submitRetry.deadLetter=true`。
  - `skipped`：连接器缺失/停用、缺少外部 endpoint、缺少请求载荷、外部单号保护等不可自动重试场景。
- 后端执行链路：
  - 将单票重放和批量到期补偿复用同一个提交 helper，避免事件、状态和 response payload 口径分叉。
  - 人工重放会写入 `responsePayload.manualReplay`、`forcedReplay` 和 `retryEligibilityReason`。
  - 成功仍写入 `itsm.ticket.retry_submitted`，失败写入 `itsm.ticket.retry_failed`，继续复用事件中心和审计链路。
- 前端 ITSM 页面：
  - 新增“失败补偿队列”页签。
  - 展示待重试、等待退避、死信和跳过四个指标。
  - 支持按队列状态过滤、刷新、批量重试到期和单条重放。
  - 队列条目可打开原工单详情抽屉查看 request/response payload。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化本轮后端 Go 文件。
- 新增单元测试覆盖队列分类和死信强制重放 eligibility 口径。
- `docker compose -f backend/docker/docker-compose.yml build iac-portal` 通过，后端 Go 编译通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-web` 通过，前端 webpack 编译通过，仅存在既有 bundle size 警告。
- 使用 `CLOUDIAC_DEPLOY_ROOT=$PWD/.deploy/cloudiac`、`MYSQL_USER=cloudiac`、`MYSQL_PASSWORD=mysqlpass`、`MYSQL_DATABASE=cloudiac` 短暂启动 `iac-portal`：
  - `GET /api/v1/check` 返回 `success=true`，版本 `v1.3.5`。
  - 未登录访问 `GET /api/v1/cloud/itsm/tickets/retry-queue/summary` 返回 `40120000`，确认新路由已进入鉴权链路。
- 验证后执行 `docker compose down -v`，并删除 `.deploy/cloudiac` 临时运行目录。

验证限制：

- 第一阶段仍复用 `responsePayload.submitRetry` 作为轻量队列状态，未新增独立持久化队列表。
- 人工重放默认仍保留外部单号保护、endpoint 校验和请求载荷校验，避免重复创建已经带外部身份的工单。
- 真实 Jira/ServiceNow 租户的失败队列和死信重放端到端联调仍待继续。

### 16.151 2026-06-22 V1.2 P0 AWS collector 资产覆盖补齐第一阶段

状态：已完成 AWS P0 资产采集补齐第一阶段；AWS collector 在原 EC2、VPC、Subnet、SecurityGroup、EBS、EKS、RDS、ElastiCache 基础上，新增 Route Table、NAT Gateway、Internet Gateway、Elastic IP、Classic ELB、ALB/NLB 和 S3 Bucket 采集，并接入标准资产类型、中文展示和云侧关系推演。

已完成：

- 标准资产类型新增：
  - `network_nat_gateway`：NAT 网关。
  - `network_internet_gateway`：Internet 网关。
- AWS 支持资产类型更新：
  - AWS 账号 `supportedAssetTypes` 新增 NAT Gateway 和 Internet Gateway。
  - OCI 支持类型与 AWS 分离，避免误报 OCI 已支持新增 AWS 网络网关类型。
- AWS EC2 Query API 采集增强：
  - `DescribeRouteTables` 归一化为 `network_route_table / aws_route_table`，保留 VPC、Subnet association、Route、NAT Gateway、Internet Gateway 和 propagating VGW 引用。
  - `DescribeNatGateways` 归一化为 `network_nat_gateway / aws_nat_gateway`，保留 VPC、Subnet、EIP allocationId、公网 IP、网卡、失败原因和 connectivityType。
  - `DescribeInternetGateways` 归一化为 `network_internet_gateway / aws_internet_gateway`，保留 VPC attachment 和状态。
  - `DescribeAddresses` 归一化为 `public_ip / aws_eip`，保留 allocationId、associationId、Public IP、Private IP、Instance、ENI、domain 和 network border group。
- AWS Elastic Load Balancing 采集增强：
  - Classic ELB `DescribeLoadBalancers` 归一化为 `load_balancer / aws_elb_load_balancer`。
  - ALB/NLB/GWLB `DescribeLoadBalancers` 归一化为 `load_balancer / aws_lb`。
  - 保留 DNS、scheme、type、VPC、Subnet、安全组、可用区、地址、EIP allocationId 和创建时间。
- AWS S3 采集增强：
  - `ListBuckets` + `GetBucketLocation` 归一化为 `object_storage_bucket / aws_s3_bucket`。
  - 按 bucket location 过滤到当前同步 region，`us-east-1` 空 location 和历史 `EU` location 已做兼容映射。
  - Bucket Tagging、默认加密、版本控制、Public Access Block 和 Lifecycle 已在 16.174 完成第一阶段。
  - Bucket Policy、ACL、Object Lock、Replication、Logging、Notification、Website 和 Inventory 已在 16.175 完成第一阶段。
- 云侧关系推演增强：
  - Route Table 归属 VPC，并关联 Subnet、NAT Gateway、Internet Gateway。
  - NAT Gateway 归属 VPC/Subnet，并依赖 EIP。
  - Internet Gateway 归属 VPC。
  - 关系索引新增 `allocationId/publicIp/publicIpAddress` 等键，便于 EIP 与 NAT/LB 推演关联。
- 前端资产类型中文展示新增：
  - `network_nat_gateway` -> `NAT网关`。
  - `network_internet_gateway` -> `Internet网关`。

验证：

- 使用 Docker Go 镜像 `golang:1.26.4-alpine` 进行 Go 编译检查：
  - `go test ./portal/apps` 因既有 `portal/apps/cloud_cost.go:727` 的 vet 规则 `non-constant format string in call to fmt.Errorf` 失败；该问题与本轮 AWS collector 改动无关。
  - `go test -vet=off ./portal/apps` 通过，确认本轮 Go 类型和编译链路通过。
- `git diff --check` 通过。

验证限制：

- 本阶段完成协议解析、归一化和编译验证；尚未使用真实 AWS 账号做端到端采集。
- ELB 第一阶段采集负载均衡器主体和网络引用；Listener、Target Group、Target Health 已在 16.171 完成第一阶段，Rule 已在 16.172 完成第一阶段，认证类 Action 已在 16.173 完成第一阶段，跨账号/跨 VPC target 等运行期细节待继续。
- S3 第一阶段使用 `ListBuckets` 和 `GetBucketLocation`；Bucket Tagging、加密、版本、Public Access Block、Lifecycle 已在 16.174 完成第一阶段，Bucket Policy、ACL、Object Lock、Replication、Logging、Notification、Website 和 Inventory 已在 16.175 完成第一阶段，S3 风险规则映射已在 16.176 完成第一阶段，真实账号联调和更完整成本/合规映射待继续。

待继续：

- 使用真实 AWS 账号验证 Route Table、NAT Gateway、Internet Gateway、EIP、ELB/ALB/NLB、S3 的分页、权限错误和区域过滤。
- 补充真实目标健康端到端联调、跨账号/跨 VPC target 和更细关系推演。
- 将 AWS 新增资产接入更细成本、合规和风险规则映射。

### 16.152 2026-06-22 V1.2 P0 CMDB 返回来源环境资源详情入口第一阶段

状态：已完成 CMDB 资产详情返回来源环境资源详情入口第一阶段；当 CMDB 资产存在 `projectId`、`envId` 和 `iacResourceId` 时，资产详情抽屉会展示“返回环境资源详情”，跳转到环境资源页并自动打开对应资源详情抽屉。

已完成：

- CMDB 资产详情增强：
  - 新增来源资源入口，使用 `projectId/envId/iacResourceId` 生成环境资源深链。
  - 深链格式为 `/org/:orgId/project/:projectId/m-project-env/detail/:envId?tabKey=resource&resourceId=:iacResourceId`。
- 环境详情资源页增强：
  - 读取 URL 参数 `resourceId` 并写入详情页上下文。
  - 资源组件检测到 `resourceId` 时自动切换到表格模式，避免默认图形模式挡住资源抽屉。
  - 资源表格组件检测到 `resourceId` 后自动打开现有资源详情抽屉，复用 `getResourcesGraphDetail` 详情接口。

验证：

- 本阶段为前端深链和页面状态联动改造，未新增后端 API。
- `git diff --check` 通过。
- `docker compose -f docker-compose.yml build iac-web` 成功，前端生产 bundle 编译通过；仅存在既有 webpack bundle size warning。

验证限制：

- 由于上一轮已按要求删除部署服务和清理数据，本阶段只做镜像构建验证，未重新 `up -d` 启动服务做浏览器点击回归。
- 若来源资源已被删除或用户缺少项目权限，环境资源详情接口会按现有逻辑返回错误；后续可补专门的友好空态。

### 16.153 2026-06-22 V1.2 P0 云采集任务最近同步、范围和失败日志增强第一阶段

状态：已完成云采集任务最近同步、范围和失败日志增强第一阶段；任务列表、任务详情和多云总览最近任务不再只返回原始 `stats`，会额外返回稳定的范围摘要和失败摘要，前端可直接展示采集影响面、失败范围、失败分类和重试建议。

已完成：

- 后端同步任务响应增强：
  - `CmdbSyncTaskResp` 新增 `scopeSummary` 和 `failureSummary`。
  - `CmdbSyncTaskDetailResp` 和批量重跑任务组内任务复用增强后的任务响应。
  - `CmdbSyncTaskSummaryResp` 新增 `lastSuccessTask` 和 `lastFailureTask`，保留 `lastSuccessAt/lastFailureAt` 兼容字段。
  - 多云总览最近同步任务统一走增强后的任务响应。
- 范围摘要：
  - 从任务 `stats.regions/assetTypes/regionMetrics/assetTypeMetrics/scopeMetrics` 和任务原始 `regions/assetTypes` 兼容派生。
  - 返回区域数、资产类型数、scope 数、采集/新增/更新/跳过数量、collector/upsert/relation/总耗时。
  - 返回 region、assetType 和 region+assetType scope 明细，并标识非 `complete` 的失败 scope。
- 失败摘要：
  - 从 `stats.failureDetails/failureSummary` 派生失败总数、可重试失败数、分类、首条错误和重试建议。
  - 旧任务只有 `errorMessage` 时，会按现有文本分类逻辑补齐失败摘要，保持历史数据可读。
- 前端云采集页增强：
  - 任务列表新增“范围摘要”和“失败摘要”列。
  - 统计列展示采集耗时和总耗时。
  - 子周期任务历史摘要中的“最近成功/最近失败”可直接打开对应任务详情。
  - 任务详情新增采集范围、采集耗时、总耗时、失败摘要和重试建议。
  - 任务详情新增“范围明细”和“失败明细”页签，展示 region/assetType/scope 状态、采集数、耗时、错误分类和重试建议。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/models/resps/cmdb.go`、`backend/portal/apps/cmdb_sync.go`、`backend/portal/apps/cloud_overview.go`。
- `docker run --rm -v .../backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- `git diff --check` 通过。
- `docker compose build iac-portal iac-web` 通过；`DOCKER_REGISTRY` 未设置提示和前端 webpack bundle size warning 为既有构建提示。

验证限制：

- 由于当前部署服务已按要求删除并清理数据，本阶段先做编译和镜像构建验证，不重新启动平台做浏览器点击回归。
- OCI 云厂商原生错误码、短重试、collector 内部单 API 子调用耗时和失败 scope 自动局部重试已在后续阶段完成第一阶段；区域级权限矩阵、其他 provider 原生错误样本和 API 子调用级局部补偿仍待 provider adapter 继续增强。

### 16.154 2026-06-22 V1.2 P1 EKS/OKE K8S 信息可见性增强

状态：已完成 EKS/OKE K8S 信息可见性增强；在 16.150 已完成采集与“K8S信息”页签的基础上，继续放宽前端识别条件并把关键 K8S 摘要前置到资产详情基础信息区，避免老数据或未归一化数据看不到 K8S 信息。

已完成：

- Kubernetes 资产识别增强：
  - 不再只依赖 `assetType === kubernetes_cluster`。
  - 兼容 `nativeType` 包含 `eks_cluster`、`containerengine_cluster`、`kubernetes_cluster`、`k8s_cluster` 的历史或未归一化资产。
  - 兼容已带 `attributes.nodeGroups/nodePools/kubernetesVersion` 的资产。
- 资产详情基础信息增强：
  - Kubernetes 资产在基础信息区直接展示 K8S 版本、节点组/节点池摘要和 API Endpoint。
  - API Endpoint 支持从 `asset.address`、`attributes.endpoint/apiEndpoint/clusterEndpoint`、`attributes.endpoints/endpointConfig` 多来源兜底读取。
- 详情页签体验增强：
  - Kubernetes 资产详情默认打开“K8S信息”页签。
  - “K8S信息”页签继续展示 VPC/VCN、Endpoint 配置、子网、安全组/NSG、网络配置、节点组/节点池表格和采集错误。

验证：

- `git diff --check` 通过。
- `docker compose build iac-web` 通过；仅存在既有 webpack bundle size warning。

验证限制：

- 当前部署服务已清理，本阶段未重新启动服务做浏览器点击回归。
- 真实 EKS/OKE 账号的 NodeGroup/NodePool API 权限、分页和云厂商错误码映射仍需在真实环境继续补测。

### 16.155 2026-06-22 V1.2 P1 CMDB 应用依赖事件接入 Webhook 第一阶段

状态：已完成 CMDB 应用依赖变更事件接入第一阶段；CMDB 资产变更已存在 `cmdb.asset.created/updated` 事件，本阶段补齐应用依赖维护保存后的 `cmdb.application.relations_updated` 事件，使 CMDB Webhook/事件推送覆盖资产创建、资产更新、归属变更和应用依赖变更。

已完成：

- `PUT /api/v1/cmdb/applications/relations` 保存前会读取当前人工维护的上游/下游依赖快照。
- 保存后对比旧上游、旧下游与新上游、新下游；只有真实变化时才产生事件，避免重复保存制造事件噪声。
- 新增 `cmdb.application.relations_updated` 事件：
  - `source=cmdb`
  - `resourceType=cmdb_application`
  - `resourceName=<应用名称>`
  - `status=updated`
  - payload 记录应用名称、关系来源、关系类型、变更前上游/下游和变更后上游/下游。
- 事件写入后复用平台级事件中心和 Webhook 分发，不新增 CMDB 独立 Webhook 配置入口。
- CMDB PRD 已同步：
  - Webhook/事件推送状态更新为“已完成第一阶段”。
  - `/api/v1/cmdb/events/webhook` 调整为复用 `/api/v1/cloud/webhooks`。
  - 补充 `source=cmdb`、`eventType=cmdb.*` 的查询和订阅说明。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_event.go` 和 `backend/portal/apps/cmdb_application.go`。
- `docker run --rm -v backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- `git diff --check` 通过。
- `docker compose build iac-portal` 通过；服务保持停止状态，未执行 `docker compose up -d`。

验证限制：

- 当前部署服务已按要求删除并清理数据，本阶段未重新启动服务做 API 或浏览器回归。
- 下一阶段可在恢复服务后创建测试应用依赖关系，验证事件中心可按 `source=cmdb` 查询到 `cmdb.application.relations_updated`，并用临时 Webhook endpoint 做投递验签。

### 16.156 2026-06-22 V1.2 P1 CMDB 导入模板、预检和差异预览第一阶段

状态：已完成 CMDB/云资产导入模板、预检和差异预览第一阶段；资产导入从“上传后直接写入”优化为“下载模板 -> 上传 JSON -> dryRun 预检 -> 查看逐条差异 -> 确认导入”的两阶段流程。

已完成：

- 新增导入模板接口：
  - `GET /api/v1/cmdb/assets/import-template`
  - `GET /api/v1/cloud/assets/import-template`
  - 返回示例 JSON，包含 `overwriteOwnership` 和 `assets[]` 标准字段。
- 增强导入接口：
  - `POST /api/v1/cmdb/assets/import`
  - `POST /api/v1/cloud/assets/import`
  - 请求支持 `dryRun=true`，此时只做预检，不写入资产、不产生变更事件。
- 预检返回：
  - 汇总：total、created、updated、ownershipUpdated、skipped、errors。
  - 明细：行号、动作、资产 ID、provider、account、region、assetType、nativeId、现有资产名、资产字段 diff、归属字段 diff、异常原因。
  - 动作包括 `create`、`update`、`ownership_update`、`skip`、`error`。
- 前端资产/云资产页面增强：
  - 操作区新增“下载导入模板”。
  - 上传 JSON 后先调用 dryRun，弹出“导入预检”弹窗。
  - 弹窗展示新增、更新、归属更新、跳过和逐条异常/差异。
  - 用户点击“确认导入”后才调用真实导入；导入成功后刷新资产列表。
- CMDB PRD 已同步导入/导出状态、API 表和 P1 待开发清单。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化后端变更文件。
- `docker run --rm -v backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- `git diff --check` 通过。
- `docker compose build iac-portal iac-web` 通过；仅存在既有 `DOCKER_REGISTRY` 未设置、npm deprecated、webpack bundle size 和 swagger 路由重复提示。

验证限制：

- 当前部署服务已按要求删除并清理数据，本阶段未重新启动服务做 API 或浏览器点击回归。
- 下一阶段可在恢复服务后用模板下载、dryRun 预检、确认导入、事件中心变更记录和导入后列表刷新做完整闭环验证。

### 16.157 2026-06-22 V1.2 P1 CMDB 应用风险规则配置化第一阶段

状态：已完成 CMDB 应用风险规则配置化第一阶段；应用依赖风险从固定代码规则扩展为组织级可配置规则，默认值兼容原有近 7 天变更、上下游依赖和风险加分逻辑。

已完成：

- 新增模型 `iac_cmdb_risk_rule_config`：
  - 按组织保存一套应用风险规则。
  - 支持配置变更窗口天数。
  - 支持近期变更、变更资产、调用方、调用应用、高/严重合规风险、维护期/退役期和跨业务线权重。
  - 支持严重/高/中评分阈值。
  - 支持严重调用方、中风险调用方、中风险调用应用阈值。
  - 支持近期严重/高/中风险加分和依赖面加分。
- 新增 CMDB 风险规则 API：
  - `GET /api/v1/cmdb/risk-rules`：查询组织级应用风险规则，未配置时返回默认规则。
  - `PUT /api/v1/cmdb/risk-rules`：更新组织级应用风险规则，支持局部字段更新。
- 应用依赖风险计算增强：
  - `buildCmdbApplications` 加载组织级风险规则。
  - 近期变更窗口改为读取 `changeWindowDays`。
  - 风险评分纳入合规风险、生命周期和跨业务线因素。
  - 风险原因会按配置窗口输出“近 N 天”。
- 前端资产/云资产页面增强：
  - “应用依赖”页签新增“风险规则”按钮。
  - 新增“应用风险规则”弹窗，可编辑窗口、权重、阈值和加分。
  - 保存规则后刷新应用列表。
  - “近 N 天变更”列名和应用详情标签跟随配置窗口变化。
- CMDB PRD 已同步风险规则数据模型、API 表、状态总览、路线图和待开发清单。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化后端变更文件。
- `docker run --rm -v backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- `git diff --check` 通过。
- `docker compose build iac-portal iac-web` 通过；仅存在既有 `DOCKER_REGISTRY` 未设置、npm deprecated、webpack bundle size 和 swagger 路由重复提示。

验证限制：

- 当前部署服务已按要求删除并清理数据，本阶段未重新启动服务做 API 或浏览器点击回归。
- 下一阶段可在恢复服务后通过 `GET/PUT /api/v1/cmdb/risk-rules`、应用列表风险分数变化和“风险规则”弹窗做完整闭环验证。

### 16.158 2026-06-22 V1.2 P1 GKE/AKS Kubernetes 节点池信息第一阶段

状态：已完成 GKE/AKS Kubernetes 节点池信息第一阶段；在 EKS/OKE 已有 K8S 信息页签基础上，把 AKS `agentPoolProfiles` 和 GKE `nodePools` 归一化为同一套 `nodePools/nodePoolCount` 属性，并补齐版本、Endpoint、VPC/VNet、子网和网络配置展示字段。

已完成：

- Azure AKS collector 增强：
  - `azure_kubernetes_cluster` 写入 `endpoint/apiEndpoint`，并把 `fqdn/privateFQDN` 归一化到 `endpointConfig`。
  - `agentPoolProfiles` 转换为 `attributes.nodePools`，包含名称、状态、模式、K8S 版本、VM 规格、节点数、伸缩配置、可用区、子网、OS 信息、标签/污点和升级配置。
  - 从节点池和网络配置提取 `subnetIds`，并通过子网资源反查 `securityGroupIds`。
  - 从子网 ID 推导 `vnetId`，资产详情可展示 VNet 归属。
- GCP GKE collector 增强：
  - `gcp_container_cluster` 写入 `kubernetesVersion/version`、`endpoint/apiEndpoint`、`network/vpcId/networkId`、`subnetIds/subnetworkIds`。
  - GKE `nodePools` 转换为统一 `attributes.nodePools`，包含名称、状态、版本、机器规格、初始节点数、伸缩配置、可用区、实例组、节点配置、网络配置、管理配置和升级配置。
  - 集群网络信息统一写入 `kubernetesNetwork`，包含 networkConfig、ipAllocationPolicy、privateClusterConfig、releaseChannel、cluster/service CIDR。
- 前端云资产/CMDB 资产详情增强：
  - K8S 信息页签的 VPC 字段扩展为 `VPC/VNet/VCN`，兼容 `vpcId/vnetId/vcnId/networkId/network`。
  - 子网字段兼容 `subnetIds/subnetworkIds/subnetwork`。
  - 网络配置兼容 `kubernetesNetwork/networkProfile/options`。
  - 节点池规模列兼容 `nodeCount/initialNodeCount`。
- CMDB PRD 和多云 PRD 已同步：
  - GKE/AKS 节点池信息状态更新为“已完成第一阶段”。
  - K8S 工作负载层后续在 16.160 完成第一阶段资产类型、展示和关系推演。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cmdb_collect_azure.go` 和 `backend/portal/apps/cmdb_collect_gcp.go`。
- `docker run --rm -v /Volumes/scrt-sfx-923-2829osx_x64/workspace/cloudiac/backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- `git diff --check` 通过。
- `docker compose build iac-portal iac-web` 通过；仅存在既有 `DOCKER_REGISTRY` 未设置、npm deprecated 和 webpack bundle size warning。

验证限制：

- 当前部署服务已按要求删除并清理数据，本阶段先完成编译级验证，不重新启动服务做浏览器点击回归。
- 本阶段不接入 kubeconfig/Agent，不实时采集 Namespace、Node、Pod、Deployment、Service、Ingress 等工作负载对象；资产属性/导入数据展示和关系推演已在 16.160 补齐第一阶段。
- 真实 AKS/GKE 账号的 API 响应字段差异、权限不足、分页和错误码映射仍需在真实环境继续补测。

### 16.159 2026-06-22 V1.2 P1 CMDB/云资产编辑导出权限细分第一阶段

状态：已完成 CMDB/云资产编辑导出权限细分第一阶段；组织管理员和平台管理员可执行全组织资产导入、导出、治理、风险规则和采集同步，项目负责人、审批员和操作员可在项目范围内执行资产归属治理和项目资产导出，敏感治理字段收敛到项目负责人/审批员或组织级管理员。

已完成：

- 后端权限模型：
  - 新增 `GET /api/v1/cmdb/assets/permissions` 和 `GET /api/v1/cloud/assets/permissions`，返回当前用户组织角色、项目角色和导出、导入、归属编辑、批量治理、应用依赖维护、风险规则配置、IaC 同步等权限位。
  - 资产导出接口按筛选范围校验项目角色；未绑定项目资产仅组织管理员/平台管理员可导出。
  - 资产导入接口限定组织管理员/平台管理员，避免普通用户创建或覆盖组织资产。
  - 单资产归属编辑和批量治理按资产项目角色校验；合规风险、项目/环境绑定等敏感字段需要项目负责人/审批员或组织级管理员。
  - 应用依赖维护需要组织级管理员，或用户对该应用绑定资产所在项目具备负责人/审批员/操作员角色。
  - 风险规则配置限定组织管理员、平台管理员或合规管理员。
  - IaC 回填、云采集启动和无需审批的失败任务直接重跑限定组织管理员/平台管理员；失败任务仍可通过“提交审批后再启动”保留自助申请路径。
- 前端权限体验：
  - 云资产/CMDB 页面加载权限摘要，并对导入预检、导入模板、CSV/JSON 导出、批量治理、同步 IaC、启动云采集、风险规则、应用依赖保存和归属保存做按钮禁用与提示。
  - 导入预检、批量治理、风险规则和采集重跑弹窗增加 OK 按钮级禁用，避免弹窗内绕过主按钮禁用态。
  - 重跑子周期按 `canSyncIac` 禁用；失败任务批量重跑在无直接启动权限时默认勾选并锁定“提交审批后再启动”。
  - 批量重跑任务组审批按钮只对具备同步/审批权限的用户显示，后端仍保留组织/平台管理员审批校验。
- PRD 状态同步：
  - CMDB PRD 的安全章节和 P1 待开发清单已将“编辑/导出权限细分”标记为已完成第一阶段。
  - 多云 PRD 的导入导出和权限差距表已补充权限门禁状态。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化本轮新增/修改的 CMDB 权限、同步、handler、route 和响应结构文件。
- `docker run --rm -v /Volumes/scrt-sfx-923-2829osx_x64/workspace/cloudiac/backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- `git diff --check` 通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-portal iac-web` 通过；仅存在既有 `DOCKER_REGISTRY` 未设置、npm deprecated、webpack bundle size 和 swagger 重复路由提示。
- 当前部署服务已按要求删除并清理数据，本阶段不重新启动服务做浏览器点击回归。

验证限制：

- 尚未接入真实组织角色矩阵和多项目用户端到端浏览器回归；后续恢复服务后需分别用组织管理员、项目负责人、项目操作员、普通成员账号验证按钮禁用和接口拒绝。
- 响应层敏感字段脱敏已在 16.168 完成第一阶段；标签级授权、云账号级授权和资源类型级授权仍需继续开发。

### 16.160 2026-06-22 V1.2 P1 K8S 工作负载层信息第一阶段

状态：已完成 K8S 工作负载层信息第一阶段；在不引入 kubeconfig/Agent 的前提下，先支持 Kubernetes Namespace、Node、Workload、Pod、Service、Ingress 作为标准 CMDB 资产类型、导入/离线 inventory 属性展示和关系推演，让资产详情能够看到集群以下对象。

已完成：

- 后端资产标准化：
  - 新增 `kubernetes_namespace`、`kubernetes_node`、`kubernetes_workload`、`kubernetes_pod`、`kubernetes_service`、`kubernetes_ingress` 标准资产类型。
  - `NormalizeCmdbAssetType` 可识别 `kubernetes_*` 和 `k8s_*` 的 Namespace、Node、Pod、Service、Ingress、Deployment、StatefulSet、DaemonSet、ReplicaSet、Job、CronJob、Workload 原生类型。
- 后端关系推演：
  - 集群可根据 `namespaces/nodes/workloads/pods/services/ingresses` 或对应 ID 列表推演包含关系。
  - Namespace 可推演 Workload、Pod、Service、Ingress 的包含关系。
  - Workload 可推演 Pod 包含关系。
  - Pod 可根据 `ownerUid/ownerName/workloadId/workloadName` 推演所属 Workload，并根据 `nodeId/nodeName` 推演依赖节点。
  - Service 可根据目标 Workload/Pod 字段推演依赖关系，Ingress 可根据后端 Service 字段推演依赖关系。
  - 关系索引补充 `uid/name/clusterId/namespace/ownerUid/nodeName/serviceName` 等 Kubernetes 常用引用键。
- 前端资产详情：
  - 资产类型字典新增 K8S Namespace、Node、Workload、Pod、Service、Ingress。
  - K8S 信息页签不再只对集群展示，所有 `kubernetes_*` / `k8s_*` 资产都可展示。
  - 页签新增工作负载层摘要：Namespace、Node、Workload、Pod、Service、Ingress 计数。
  - 页签新增对象明细表，支持展示类型、命名空间、名称、状态、副本/可用副本、Service 类型/ClusterIP、PodIP、节点、Ingress Host、镜像和端口摘要。
  - 当资产属性带 `workloadCollectError` 或 `kubernetesWorkloadCollectError` 时展示工作负载层采集告警。
- PRD 状态同步：
  - CMDB PRD 已将“K8S 工作负载层信息”标记为已完成第一阶段。
  - 多云 PRD 的 Kubernetes 集群信息状态已补充工作负载层第一阶段范围。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/models/cmdb.go`、`backend/portal/apps/cmdb.go` 和 `backend/portal/apps/cmdb_relations_cloud.go`。
- `docker run --rm -v /Volumes/scrt-sfx-923-2829osx_x64/workspace/cloudiac/backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- `git diff --check` 通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-portal iac-web` 通过；仅存在既有 `DOCKER_REGISTRY` 未设置、npm deprecated、webpack bundle size 和 swagger 重复路由提示。
- 当前部署服务已按要求删除并清理数据，本阶段不重新启动服务做浏览器点击回归。

验证限制：

- 本阶段不直连 Kubernetes API Server，不读取 kubeconfig，不部署 Agent；真实 Namespace/Pod/Workload 采集需要后续接入受控采集器。
- 当前工作负载数据来自资产属性、导入 JSON 或离线 inventory；真实 EKS/OKE/AKS/GKE 工作负载权限、分页、资源版本和 watch 机制仍待后续阶段。

### 16.161 2026-06-22 V1.2 P2 CMDB 成本、合规和生命周期治理报表第一阶段

状态：已完成 CMDB/云资产成本、合规和生命周期治理报表第一阶段；多云资产页不再只展示覆盖率，还能按治理维度识别高成本资产、高风险资产、生命周期分布和归属缺口。

已完成：

- 后端只读报表 API：
  - 新增 `GET /api/v1/cmdb/assets/governance-report`。
  - 新增 `GET /api/v1/cloud/assets/governance-report`，复用 CMDB 报表聚合能力。
  - 报表执行前复用 IaC 资源回填和治理字段刷新，避免只读取旧归属字段。
  - 权限范围复用 CMDB 资产查询边界，组织管理员可看全量，项目角色按可见项目资产和未绑定项目资产查看。
- 报表指标：
  - 汇总资产总数、总成本、平均风险分、无负责人、未绑定应用、未设置业务线、未设置成本中心、未分类生命周期、未绑定项目、高风险和严重风险资产数量。
  - 按 provider、业务线、应用、负责人聚合成本。
  - 按生命周期和合规风险聚合资产数量与成本。
  - 输出高成本资产 Top 10 和高风险资产 Top 10，包含 provider、区域、资产类型、资源 ID、归属、生命周期、合规风险、成本、风险分和纳管状态。
- 前端云资产页：
  - `多云管理 -> 云资产` 顶部新增“治理报表”面板。
  - 面板展示总成本、平均风险分、高/严重风险、负责人缺口、应用绑定缺口和缺口进度条。
  - 新增生命周期分布、合规风险分布、高成本资产和高风险资产四张紧凑表格。
  - 高成本资产和高风险资产支持点击资产名进入资产详情，继续使用现有详情抽屉治理链路。
- PRD 状态同步：
  - CMDB PRD 已将“成本、合规、生命周期报表”从待开发标记为已完成第一阶段。
  - 多云 PRD 的 CMDB/云资产总览已补充“资产治理报表”能力。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化新增和修改的 Go 文件。
- `docker run --rm -v /Volumes/scrt-sfx-923-2829osx_x64/workspace/cloudiac/backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- `git diff --check` 通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-portal iac-web` 通过；仅存在既有 `DOCKER_REGISTRY` 未设置、npm deprecated、webpack bundle size 和 swagger 重复路由提示。
- 当前部署服务已按要求删除并清理数据，本阶段不重新启动服务做浏览器点击回归。

验证限制：

- 本阶段是组织/可见资产范围的聚合报表；按筛选条件动态联动、报表导出、趋势环比和按账期成本对账仍待后续增强。
- 高成本和风险 Top 列表来自 CMDB 资产当前成本与风险分字段；真实多账期账单、成本摊销和更细合规规则仍需继续接入成本中心与风险中心数据。

### 16.162 2026-06-22 V1.2 P2 CMDB 关系图性能优化和大规模压测支撑第一阶段

状态：已完成 CMDB/云资产关系图性能优化第一阶段；资产详情不再无上限聚合完整关系图，新增专用关系查询接口、返回数量保护、摘要指标和应用推演关系局部聚合，为后续真实大规模压测和图谱渐进加载打底。

已完成：

- 后端关系查询 API：
  - 新增 `GET /api/v1/cmdb/assets/:id/relations`。
  - 新增 `GET /api/v1/cloud/assets/:id/relations`，复用 CMDB 资产关系查询能力。
  - 查询参数支持 `limit` 和 `includeApplication`；默认返回 200 条，最大限制 1000 条，避免大规模关系一次性撑爆详情接口。
- 关系摘要：
  - 响应返回 `totalRelations`、`returnedRelations`、`directRelations`、`applicationInferredRelations`、`incomingRelations`、`outgoingRelations` 和 `truncated`。
  - 按关系来源、关系类型和方向输出 Top 摘要，用于前端在图谱截断时提示用户。
- 应用推演关系优化：
  - 资产详情关系不再为了单个资产调用完整 `buildCmdbApplications` 聚合全部应用图谱。
  - 改为围绕当前资产所属应用做局部上游/下游应用聚合，再按当前用户可见资产范围取必要候选资源。
  - 人工应用依赖和资产直接关系继续合并去重，保留 `application_inferred` 来源标识。
- 索引和数据模型：
  - `iac_cmdb_asset_relation` 增加 `org_id + source_asset_id`、`org_id + target_asset_id` 组合索引，支撑单资产入向/出向关系查询。
  - 资产详情响应补充 `relationSummary`，前端可在关系页签展示截断和摘要。
- 前端云资产/CMDB 资产详情：
  - “关系”页签增加关系摘要提示，展示已返回/总关系数、直接关系、应用推演关系、入向和出向数量。
  - 当关系被上限截断时提示用户当前为性能保护后的可视范围。
- PRD 状态同步：
  - CMDB PRD 已将“图谱性能优化和大规模资产压测”从待开发推进为“已完成第一阶段，真实压测待验证”。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化新增和修改的 Go 文件。
- `docker run --rm -v /Volumes/scrt-sfx-923-2829osx_x64/workspace/cloudiac/backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- `git diff --check` 通过。

验证限制：

- 当前部署服务已按要求删除并清理运行数据，本阶段不重新启动服务做浏览器点击回归。
- 本阶段完成查询保护、索引和局部聚合；真实百万级资产/关系数据压测、图谱渐进加载和前端虚拟化仍需在恢复测试环境后继续验证。

### 16.163 2026-06-22 V1.2 P1 EKS/OKE K8S 信息入口可发现性增强

状态：已完成 EKS/OKE K8S 信息入口可发现性增强；在已有“K8S信息”页签和工作负载层展示基础上，继续补齐云资产首页的 Kubernetes 概览与快速入口，避免用户需要先手工筛选 `kubernetes_cluster` 才能看到 EKS/OKE/AKS/GKE 相关信息。

已完成：

- Kubernetes 资产识别增强：
  - 前端详情识别不再只依赖标准资产类型和少量原生类型。
  - 兼容 `aws_eks`、`oci_containerengine`、`oke_cluster`、`aks_cluster`、`azure_kubernetes`、`gke_cluster`、`gcp_container`、`alicloud_cs_kubernetes`、`tencentcloud_kubernetes`、`huawei_cce` 等原生类型特征。
  - 兼容 `clusterEndpoint`、`apiEndpoint`、`endpointConfig`、`kubernetesNetwork` 等常见 Kubernetes 字段。
- 云资产首页可见性增强：
  - 在资产覆盖率与治理报表之间新增“Kubernetes 集群”概览区。
  - 展示集群数、工作负载对象数、K8S 资产总数、云采集数量和治理缺口。
  - 新增 EKS、OKE、AKS、GKE 快速筛选按钮，以及“查看K8S资产”入口。
  - 展示 K8S 类型覆盖表，可直接下钻到 Kubernetes 集群、Namespace、Node、Workload、Pod、Service、Ingress。
- PRD 状态同步：
  - CMDB PRD 已将 K8S 信息标记为“已完成第一阶段，可见性已增强”。

验证：

- `git diff --check` 通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-web` 通过；仅存在既有 npm deprecated、webpack bundle size 和 `DOCKER_REGISTRY` 未设置提示。

验证限制：

- 当前部署服务已按要求删除并清理运行数据，本阶段不重新启动服务做浏览器点击回归。
- 本阶段增强前端入口与识别兜底；真实 EKS/OKE 账号的采集结果仍需在恢复测试环境后继续做端到端回归。

### 16.164 2026-06-22 V1.2 P0 ITSM 自助申请连接器兜底体验增强

状态：已完成 ITSM 自助申请连接器兜底体验增强；在后端已有 `CloudIaC 本地工单` 默认连接器兜底的基础上，前端弹窗不再强制要求选择外部连接器，避免连接器列表为空或异步刷新未返回时阻塞自助申请。

已完成：

- 自助申请弹窗：
  - `ITSM 连接器` 改为可选，支持清空。
  - 占位提示调整为“可选，留空使用 CloudIaC 本地工单”。
  - 当连接器列表异步返回启用连接器后，自动回填第一个启用连接器；用户仍可清空并使用本地工单。
  - 新增弹窗提示：无启用外部连接器时使用 `CloudIaC 本地工单` 兜底；有外部连接器时，未选择仍使用本地工单。
- 前后端一致性：
  - 前端可留空 `connectorId`。
  - 后端 `CreateCloudItsmSelfServiceTicket` 继续在 `connectorId` 为空时调用 `ensureDefaultCloudItsmConfig`，自动创建/启用默认连接器。
- 代码整洁：
  - 清理 `cloud-itsm/index.jsx` 中本次触达区域的制表符缩进。

验证：

- `git diff --check -- frontend/app/containers/org/cloud-itsm/index.jsx` 通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-web` 通过；仅存在既有 npm deprecated、webpack bundle size 和 `DOCKER_REGISTRY` 未设置提示。
- `docker compose -f backend/docker/docker-compose.yml ps -a` 确认仍无运行容器。

验证限制：

- 当前部署服务已按要求删除并清理运行数据，本阶段不重新启动服务做浏览器点击回归。
- 后续恢复服务后，应在 `/m-cloud-itsm` 页面实际点击“发起自助申请”，验证无外部连接器、有外部连接器、清空连接器三种提交路径。

### 16.165 2026-06-22 V1.2 P2 CMDB 关系图筛选第一阶段

状态：已完成 CMDB/云资产详情关系图筛选第一阶段；在 16.162 已完成关系查询保护和摘要的基础上，前端支持对已返回关系集合按关键词、关系来源、关系类型和方向进行局部筛选，让大规模关系图在截断返回后仍可快速定位关键依赖。

已完成：

- 资产详情“关系”页签新增筛选工具条：
  - 支持按关联资产名称、资产 ID、资产类型、关系来源、关系类型和元数据关键词搜索。
  - 支持按关系来源多选过滤，例如 IaC 依赖、云推断、应用依赖推演、人工应用关系。
  - 支持按关系类型多选过滤，例如依赖、包含。
  - 支持按方向过滤：全部方向、上游、下游。
  - 支持重置筛选，并显示当前筛选后数量 / 已返回关系数量。
- 图谱和表格联动：
  - `RelationGraph` 使用筛选后的关系集合。
  - 关系明细表使用同一份筛选结果，避免图谱和表格结果不一致。
  - 切换资产详情时自动重置关系筛选。
- PRD 状态同步：
  - CMDB PRD 已将“关系搜索和按关系类型筛选”标记为已完成第一阶段。

验证：

- `git diff --check` 通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-web` 通过；仅存在既有 npm deprecated、webpack bundle size 和 `DOCKER_REGISTRY` 未设置提示。

验证限制：

- 当前部署服务已按要求删除并清理运行数据，本阶段不重新启动服务做浏览器点击回归。
- 本阶段是前端对已返回关系集合的局部筛选；后端按筛选条件分页查询、图谱渐进加载、前端虚拟化和真实百万级关系压测仍待后续阶段。

### 16.166 2026-06-22 V1.2 P2 CMDB 关系图服务端筛选第一阶段

状态：已完成 CMDB/云资产关系查询接口服务端筛选第一阶段；在 16.165 前端局部筛选基础上，关系页签会按当前筛选条件重新请求 `/relations` 接口，并使用后端返回的关系集合和摘要刷新图谱、表格和截断提示。

已完成：

- 后端关系查询参数：
  - `GET /api/v1/cmdb/assets/:id/relations` 和 `GET /api/v1/cloud/assets/:id/relations` 支持 `sources`、`relationTypes`、`direction`、`keyword`、`limit`、`includeApplication`。
  - `direction` 支持 `all`、`incoming`、`outgoing`，与前端“全部方向/上游/下游”保持一致。
- 直接关系服务端筛选：
  - 直接关系查询统一通过带资产 join 的查询构造器执行 count、来源分组、类型分组和列表返回。
  - 支持按关系来源、关系类型、方向、关联资产 ID、资产名称、nativeId、资产类型、provider、来源和关系类型关键词筛选。
  - 返回摘要中的直接关系总数、入向、出向、来源分布和类型分布与筛选条件保持同一口径。
- 应用推演关系筛选：
  - 应用推演关系按来源、关系类型、方向过滤聚合候选。
  - 有关键词时基于已生成候选关系的资产名称、资产 ID、资产类型和元数据做匹配，避免筛选后摘要明显误报。
- 前端关系页签联动：
  - 打开资产详情或调整关系筛选条件时，自动调用关系查询接口并传递筛选条件。
  - 图谱、关系表和顶部摘要优先使用服务端返回结果；异步响应通过 `assetId` 校验，避免串到其他资产详情。
  - 打开或关闭资产详情会清理上一资产的关系响应和筛选状态。
- 单元测试：
  - 补充关系筛选匹配测试，覆盖来源、关系类型、方向、关键词和应用聚合方向过滤。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，格式化 `backend/portal/models/forms/cmdb.go`、`backend/portal/apps/cmdb.go`、`backend/portal/apps/cmdb_relation_perf_test.go`。
- `docker run --rm -v .../backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。

验证限制：

- 当前部署服务已按要求删除并清理运行数据，本阶段不重新启动服务做浏览器点击回归。
- 本阶段尚未实现图谱按视口渐进展开和前端虚拟化；关系分页和加载更多已在 16.167 完成第一阶段。关键词筛选下的应用推演关系统计以已加载候选关系为准，后续大规模压测阶段再补精确全量统计。

### 16.167 2026-06-22 V1.2 P2 CMDB 关系图分页和渐进加载第一阶段

状态：已完成 CMDB/云资产关系查询分页和详情页渐进加载第一阶段；在 16.166 服务端筛选基础上，关系查询接口支持 cursor/offset 分页，前端关系页签默认按页加载并提供“加载更多关系”，避免大规模关系图一次性拉取最大数量。

已完成：

- 后端关系分页参数：
  - `GET /api/v1/cmdb/assets/:id/relations` 和 `GET /api/v1/cloud/assets/:id/relations` 新增 `cursor` 与 `offset`。
  - `cursor` 优先于 `offset`；当前第一阶段 cursor 为 offset-backed token，后续可替换为更稳定的排序游标而不改变前端调用字段。
  - `limit` 仍保留 1-1000 的保护范围，默认资产详情仍使用 200 条首屏关系保护。
- 关系摘要分页元信息：
  - `summary` 新增 `offset`、`nextOffset`、`cursor`、`nextCursor`、`hasMore`。
  - `returnedRelationCount` 表示本页返回数量；前端累计展示使用已加载关系数量。
  - `truncated` 与 `hasMore` 保持兼容，便于旧前端继续识别是否还有更多关系。
- 直接关系分页：
  - 直接关系列表查询在统一筛选 query 上追加 `Offset` 和 `Limit`。
  - 直接关系总数、入向/出向数量、来源分布和类型分布仍按完整筛选条件统计，不受当前页影响。
- 应用推演关系分页：
  - 直接关系不足一页时继续用应用推演关系补齐剩余页。
  - offset 超过直接关系总数后，会落到应用推演关系候选集合继续分页。
  - 无关键词时通过应用关系聚合数量快速跳过候选；有关键词时按匹配候选逐条计数和跳过。
- 前端渐进加载：
  - 关系页签首屏请求从 1000 条降为 200 条。
  - 顶部提示改为“已加载 x / y”，同时展示本次返回数量和每页上限。
  - 当后端返回 `hasMore=true` 时展示“加载更多关系”按钮，点击后用 `nextCursor` 获取下一页并合并到当前关系集合。
  - 调整筛选条件或切换资产时清空旧关系页，避免不同资产或不同筛选条件的分页结果串联。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，格式化 `backend/portal/models/forms/cmdb.go`、`backend/portal/models/resps/cmdb.go`、`backend/portal/apps/cmdb.go`、`backend/portal/apps/cmdb_relation_perf_test.go`。
- `docker run --rm -v .../backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。

验证限制：

- 当前部署服务已按要求删除并清理运行数据，本阶段不重新启动服务做浏览器点击回归。
- 本阶段 cursor 仍是 offset-backed token，尚未实现基于 `updated_at/id` 的稳定排序游标；图谱按视口展开和表格虚拟化仍待真实大规模数据压测后继续增强。

### 16.168 2026-06-22 V1.2 P1 CMDB/云资产响应层敏感字段脱敏第一阶段

状态：已完成 CMDB/云资产响应层敏感字段脱敏第一阶段；云账号凭证脱敏已在 16.109 完成，本阶段继续覆盖 CMDB 资产详情、资产列表、JSON/CSV 导出、关系元数据和安全规则 Raw 字段，避免采集属性或导入数据中的私钥、Token、Password、Secret、AccessKey 等敏感值通过资产接口或导出文件明文暴露。

已完成：

- 响应层脱敏：
  - 资产列表 `SearchCmdbAssets` 返回前会递归脱敏 `tags`、`attributes`、`rawData`。
  - 资产详情 `CmdbAssetDetail` 返回前会递归脱敏 `tags`、`attributes`、`rawData` 和内嵌关系元数据。
  - 独立关系查询 `/relations` 返回前会递归脱敏关系 `metadata`。
  - 安全规则接口中的 `Raw` 字段会返回脱敏副本，不影响规则展示字段和公网暴露判断。
- 导出脱敏：
  - JSON 导出使用脱敏后的资产集合，避免导出文件暴露 `attributes/rawData/tags` 中的敏感值。
  - CSV 导出的标签列使用同一套脱敏结果；CSV 行字段数量与表头保持一致。
- 敏感字段识别：
  - 字段名大小写不敏感，并忽略 `-`、`_`、`.` 和空格。
  - 覆盖 `accessKey`、`accountKey`、`apiSecret`、`authorization`、`authToken`、`clientSecret`、`credential`、`password`、`passphrase`、`privateKey`、`secretAccessKey`、`sessionToken`、`signature`、`token` 等常见字段。
  - 命中字段响应值统一替换为 `<masked>`，不改变数据库中真实数据，也不影响 provider adapter 使用解密后的凭证。
- 单元测试：
  - 覆盖嵌套 `map/list` 脱敏、大小写和分隔符差异、非敏感字段保留、资产关系元数据脱敏。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，格式化 `backend/portal/apps/cmdb.go`、`backend/portal/apps/cloud_asset_security.go`、`backend/portal/apps/cmdb_relation_perf_test.go`。
- `docker run --rm -v .../backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。

验证限制：

- 当前部署服务已按要求删除并清理运行数据，本阶段不重新启动服务做浏览器点击回归。
- 本阶段是保守字段名规则，后续接入新 provider 的特殊敏感字段时仍需继续扩展关键字或引入 provider schema。
- 标签级授权、云账号级授权、资源类型级授权和真实组织角色矩阵回归仍待后续增强。

### 16.169 2026-06-22 V1.2 P0 云账号权限验证结果持久化第一阶段

状态：已完成云账号权限验证结果持久化第一阶段；`GET /api/v1/cloud/accounts/:id/permissions` 继续兼容原有响应，同时可优先返回最近一次验证落库的权限快照，避免权限结果只存在于即时计算和页面提示中。

已完成：

- 数据模型：
  - 新增 `iac_cloud_account_permission` 模型并纳入自动迁移。
  - 每个云账号、每个权限检查项保存一条记录，唯一键为 `org_id + cloud_account_id + permission_key`。
  - 记录 provider、accountId、resource、action、status、message、source、checkedAt 和 evidence，为后续 `provider_api` 主动权限校验预留审计字段。
- 验证落库：
  - `POST /api/v1/cloud/accounts/:id/validate` 在更新账号验证状态后，会把账号状态、凭证完整性、区域范围、资产采集和操作授权策略检查结果写入权限快照。
  - 账号删除时同步清理权限快照，避免孤儿权限记录。
- 查询兼容：
  - `GET /api/v1/cloud/accounts/:id/permissions` 优先使用不早于账号最近验证时间的快照。
  - 快照缺失或过期时仍按当前账号配置实时计算，并标记来源为 `computed`，旧账号首次访问不会出现空结果。
- 前端体验：
  - 云账号“区域与权限”抽屉新增权限检查时间展示。
  - 权限表新增“来源”和“检查时间”列，区分本地预检、云端验证和实时计算。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，格式化云账号模型、响应、应用逻辑、服务和测试文件。
- `docker run --rm -v .../backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-portal` 通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-web` 通过。

验证限制：

- 本阶段仍是本地预检快照，不直接调用 AWS/OCI/AliCloud 等云端 API 做主动权限验证。
- 当前部署服务已按要求删除并清理运行数据，本阶段不重新启动服务做浏览器点击回归。
- 云账号 `regions` 字段仍保留用于兼容旧逻辑；独立区域表已在 16.170 完成第一阶段。

### 16.170 2026-06-22 V1.2 P0 云账号区域独立表第一阶段

状态：已完成云账号区域独立表第一阶段；保留云账号 `regions` JSON 字段兼容旧逻辑，同时新增 `iac_cloud_account_region` 作为区域明细表，支撑默认区域、启用状态、同步开关、区域状态、资源类型范围和最近同步时间展示。

已完成：

- 数据模型：
  - 新增 `iac_cloud_account_region` 模型并纳入自动迁移。
  - 唯一键为 `org_id + cloud_account_id + region`。
  - 字段覆盖 provider、accountId、region、enabled、isDefault、syncEnabled、source、status、message、resourceTypes、lastSyncAt 和 metadata。
- 生命周期同步：
  - 创建云账号后写入区域快照。
  - 更新 provider、regions、credentials、accountId、tenantId 或 status 后刷新区域快照。
  - `POST /api/v1/cloud/accounts/:id/validate` 完成账号验证后刷新区域快照。
  - 删除云账号时同步清理区域快照。
  - CMDB 云采集成功更新账号 `lastSyncAt` 时，同步更新启用区域的 `lastSyncAt/status/message`。
- API 兼容：
  - `GET /api/v1/cloud/accounts/:id/regions` 优先读取区域快照。
  - 没有快照时仍按账号配置或凭证推断区域，旧账号不会出现空结果。
  - `PUT /api/v1/cloud/accounts/:id/regions` 继续更新账号 `regions` 字段，同时刷新区域表；被移除的旧区域保留为 disabled，不再参与同步。
- 前端体验：
  - 云账号“区域与权限”抽屉的区域表新增启用、同步和状态列。
  - 区域来源继续区分手动配置和自动推断。

验证：

- 使用 Docker Go 镜像执行 `gofmt`，格式化云账号模型、响应、应用逻辑、同步逻辑、服务和测试文件。
- `docker run --rm -v .../backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-portal` 通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-web` 通过。

验证限制：

- 本阶段仍以本地账号配置和已有同步任务结果维护区域状态，不直接调用云厂商 API 探测区域可用性。
- 当前部署服务已按要求删除并清理运行数据，本阶段不重新启动服务做浏览器点击回归。

### 16.171 2026-06-22 V1.2 P0 AWS LB Listener/Target Group/Target Health 采集第一阶段

状态：已完成 AWS ELBv2 Listener、Target Group 和 Target Health 采集第一阶段；ALB/NLB/GWLB 负载均衡器资产不再只保留主体、VPC、Subnet 和安全组信息，还会保存监听端口、协议、证书、转发目标组、目标实例/IP 和健康状态摘要，并接入云侧关系推演。

已完成：

- AWS ELBv2 Query API 采集增强：
  - `DescribeListeners` 按 LoadBalancerArn 分页采集 listener，归一化 listenerArn、port、protocol、sslPolicy、certificate、defaultActions 和转发目标组。
  - `DescribeTargetGroups` 按 LoadBalancerArn 分页采集 target group，归一化 targetGroupArn、protocol、port、targetType、vpcId、healthCheck 和 matcher。
  - `DescribeTargetHealth` 按 TargetGroupArn 采集目标健康，归一化 targetId、port、availabilityZone、state、reason 和 description。
- 资产属性增强：
  - `listeners`、`listenerCount`、`listenerPorts`、`listenerProtocols`、`listenerTargetGroupArns`。
  - `targetGroups`、`targetGroupCount`、`targetGroupArns`、`targetIds`、`targetInstanceIds`、`targetIpAddresses`。
  - `healthyTargetCount`、`unhealthyTargetCount`。
- 失败隔离：
  - Listener 或 Target Group 细节采集失败时，不阻断负载均衡器主体资产入库。
  - 细节失败会写入 `listenerCollectError`、`targetGroupCollectError` 或 `targetHealthCollectError`，方便任务详情排障。
- 云侧关系推演增强：
  - Load Balancer 继续通过 VPC/Subnet 归属网络资源。
  - Load Balancer 通过 `securityGroupIds/securityGroupId/networkSecurityGroupId` 推演依赖安全组，`inferredBy=lb_security`。
  - Load Balancer 通过 `targetInstanceIds/targetIds/targetIpAddresses` 推演依赖计算实例，`inferredBy=lb_targets`。

验证：

- 使用 Docker Go 镜像执行 `gofmt` 格式化 AWS collector、云侧关系推演和新增测试文件。
- `docker run --rm -v .../backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- 新增单元测试覆盖 listener 证书和目标组提取、target health 目标引用提取，以及 LB 到安全组/计算实例的推演关系。

验证限制：

- 本阶段未连接真实 AWS 账号做端到端采集验证。
- Listener Rule `DescribeRules` 已在 16.172 完成第一阶段；认证类 Action 已在 16.173 完成第一阶段，跨账号/跨 VPC target 解析仍待后续增强。
- LB 细节暂未接入成本、合规和风险规则映射。

### 16.172 2026-06-22 V1.2 P0 AWS LB Listener Rule 采集第一阶段

状态：已完成 AWS ELBv2 Listener Rule 采集第一阶段；ALB/NLB/GWLB 负载均衡器资产会在 Listener 维度补充 Rule 条件、动作和转发目标组，并在 LB 维度汇总 Rule 数量与 Rule 引用的 Target Group ARN，支撑后续按域名、路径、Header、Query 和来源 IP 分析流量入口。

已完成：

- AWS ELBv2 Query API 采集增强：
  - `DescribeRules` 按 ListenerArn 分页采集 rule。
  - Rule 采集失败时不阻断 Listener、Target Group、Target Health 和 LB 主体资产入库。
  - Rule 失败明细写入 `listenerRuleCollectErrors`，包含 listenerArn、port、protocol 和 error。
- Rule 属性归一化：
  - `listenerRules`：LB 级扁平 Rule 列表。
  - `listenerRuleCount`：LB 级 Rule 数量。
  - `ruleTargetGroupArns`：Rule action 引用的 Target Group ARN 汇总。
  - Listener 级 `rules`、`ruleCount`、`ruleTargetGroupArns`。
- Rule 条件归一化：
  - 支持 `host-header`、`path-pattern`、`http-header`、`query-string`、`http-request-method`、`source-ip`。
  - 条件字段统一保留为 `conditions[].field`、`values`、`hostHeaderValues`、`pathPatternValues`、`httpHeaderName`、`httpHeaderValues`、`queryStringValues`、`httpRequestMethods`、`sourceIpValues`。
- Rule action 归一化：
  - 复用 Listener action 结构，支持 `forward`、`redirect`、`fixed-response`。
  - `forward` 同时保留单 TargetGroupArn 和 ForwardConfig 中的加权 TargetGroups。

验证：

- 使用 Docker Go 镜像执行 `gofmt` 格式化 AWS collector 和新增测试。
- `docker run --rm -v .../backend:/workspace -w /workspace golang:1.26.4-alpine go test -vet=off ./portal/apps` 通过。
- 新增单元测试使用 `DescribeRulesResponse` XML 样例验证 Rule 响应标签解析、条件归一化、目标组 ARN 提取和 Listener 内嵌 Rule 汇总。

验证限制：

- 本阶段未连接真实 AWS 账号做端到端采集验证。
- `authenticate-oidc`、`authenticate-cognito`、`jwt-validation` 认证类 Action 已在 16.173 完成第一阶段。
- 跨账号、跨 VPC、Lambda target 和 IP target 的更细目标归属仍需结合真实 AWS 返回继续校准。

### 16.173 2026-06-22 V1.2 P0 AWS LB 认证类 Action 采集第一阶段

状态：已完成 AWS ELBv2 认证类 Action 采集第一阶段；Listener default action 和 Rule action 不再只支持转发、重定向和固定响应，也能展示 HTTPS Listener 的 OIDC、Cognito 和 JWT validation 配置，用于识别入口认证方式、认证失败策略、Session 配置和 JWT claim 校验要求。

已完成：

- AWS ELBv2 Action 结构增强：
  - `AuthenticateOidcConfig`：归一化 issuer、authorizationEndpoint、tokenEndpoint、userInfoEndpoint、clientId、sessionCookieName、scope、sessionTimeout、authenticationRequestExtraParams、onUnauthenticatedRequest、useExistingClientSecret。
  - `AuthenticateCognitoConfig`：归一化 userPoolArn、userPoolClientId、userPoolDomain、sessionCookieName、scope、sessionTimeout、authenticationRequestExtraParams、onUnauthenticatedRequest。
  - `JwtValidationConfig`：归一化 issuer、jwksEndpoint 和 additionalClaims。
- 敏感字段保护：
  - OIDC `clientSecret` 不写入明文，统一写为 `<masked>`。
  - 额外保留 `clientSecretConfigured`，便于判断规则是否配置了 secret。
  - 资产响应层的通用敏感字段脱敏仍会继续兜底处理。
- 适用范围：
  - Listener default action 和 Listener Rule action 都复用同一套 action 归一化逻辑。
  - Rule 级认证 action 会自然落入 `listenerRules[].actions[]`。

验证：

- 使用 Docker Go 镜像执行 `gofmt` 格式化 AWS collector 和 LB 测试文件。
- `docker compose -f backend/docker/docker-compose.yml build iac-portal` 通过，后端编译链路通过。
- 使用临时 Dockerfile + BuildKit Go 缓存执行定向测试：
  - `go test -vet=off ./portal/apps -run TestAwsV2ListenerActionsExtractsAuthenticationConfigs -count=1 -timeout=120s` 通过。

验证限制：

- 本阶段未连接真实 AWS 账号做端到端采集验证。
- 跨账号、跨 VPC、Lambda target 和 IP target 的更细目标归属仍需结合真实 AWS 返回继续校准。
- LB 认证配置暂未接入成本、合规和风险规则映射。

### 16.174 2026-06-22 V1.2 P0 AWS S3 Bucket 配置采集第一阶段

状态：已完成 AWS S3 Bucket 配置采集第一阶段；S3 bucket 资产不再只保存名称、创建时间和 region，还会补充标签、默认加密、版本控制、Public Access Block 和 Lifecycle 规则，为后续成本分摊、合规风险和对象存储治理打底。

已完成：

- S3 只读配置采集：
  - `GetBucketTagging`：写入 `bucketTags`、`bucketTagList`、`bucketTagCount`，并同步到资产 `Tags`。
  - `GetBucketEncryption`：写入 `encryption` 和 `encryptionEnabled`，包含 SSE 算法、KMS Key 和 BucketKeyEnabled。
  - `GetBucketVersioning`：写入 `versioning`、`versioningStatus` 和 `mfaDelete`。
  - `GetPublicAccessBlock`：写入 `publicAccessBlock` 和 `publicAccessBlockEnabled`。
  - `GetBucketLifecycleConfiguration`：写入 `lifecycleRules` 和 `lifecycleRuleCount`，覆盖 Prefix/Tag/And filter、Transition、NoncurrentVersionTransition、Expiration、NoncurrentVersionExpiration 和 AbortIncompleteMultipartUpload。
- 失败隔离：
  - 子配置采集失败不阻断 bucket 主体资产入库。
  - `NoSuchTagSet`、`ServerSideEncryptionConfigurationNotFoundError`、`NoSuchPublicAccessBlockConfiguration`、`NoSuchLifecycleConfiguration` 按“未配置”处理，不写失败噪音。
  - 其他权限、签名、区域或网络错误写入对应 `*CollectError` 字段，方便任务详情排障。
- Endpoint 与签名：
  - Bucket 子资源按 bucket region 使用 S3 regional host。
  - `us-east-1` 继续兼容 `s3.amazonaws.com`。

验证：

- 使用 Docker Go 镜像执行 `gofmt` 格式化 AWS collector 和 S3 测试文件。
- 使用临时 Dockerfile + BuildKit Go 缓存执行定向测试：
  - `go test -vet=off ./portal/apps -run 'TestAwsS3.*' -count=1 -timeout=120s` 通过。
- 单元测试覆盖 Tagging、Encryption、Versioning、PublicAccessBlock、Lifecycle XML 解析和未配置错误识别。

验证限制：

- 本阶段未连接真实 AWS 账号做端到端采集验证。
- Bucket Policy、ACL、Object Lock、Replication、Logging、Notification、Website、Inventory 等更多 S3 配置已在 16.175 完成第一阶段。
- S3 配置暂未接入成本、合规和风险规则映射。

### 16.175 2026-06-22 V1.2 P0 AWS S3 Bucket 深层治理配置采集第一阶段

状态：已完成 AWS S3 Bucket 深层治理配置采集第一阶段；在 16.174 基础配置采集之上，继续补齐 Bucket Policy、ACL、Object Lock、Replication、Logging、Notification、Website 和 Inventory，让对象存储资产可以支撑公开访问、保留策略、跨桶复制、审计日志、事件触发和清单治理。

已完成：

- S3 深层只读配置采集：
  - `GetBucketPolicy`：写入 `bucketPolicy`，保留 JSON document、版本、ID、Statement 数量，并标识 `publicAllow`。
  - `GetBucketAcl`：写入 `bucketAcl`、`bucketAclGrantCount` 和 `bucketAclPublic`，识别 `AllUsers` / `AuthenticatedUsers` 公开授权。
  - `GetObjectLockConfiguration`：写入 `objectLock` 和 `objectLockEnabled`，保留默认保留模式、天数和年数。
  - `GetBucketReplication`：写入 `replication`、`replicationRuleCount` 和 `replicationEnabledRuleCount`，保留目标桶、目标账号、KMS Key、Replication Time、Metrics 和 SourceSelectionCriteria。
  - `GetBucketLogging`：写入 `logging` 和 `loggingEnabled`，保留日志目标桶、前缀和授权。
  - `GetBucketNotificationConfiguration`：写入 `notifications`、`notificationRuleCount` 和 `eventBridgeEnabled`，覆盖 SNS Topic、SQS Queue、Lambda/CloudFunction 与 EventBridge。
  - `GetBucketWebsite`：写入 `website` 和 `websiteEnabled`，保留 Index/Error Document、全量重定向和 RoutingRules。
  - `ListBucketInventoryConfigurations`：写入 `inventoryConfigurations`、`inventoryConfigurationCount` 和 `inventoryEnabledCount`，支持 `NextContinuationToken` 分页。
- 失败隔离：
  - `NoSuchBucketPolicy`、`ObjectLockConfigurationNotFoundError`、`ReplicationConfigurationNotFoundError`、`NoSuchBucketWebsite`/`NoSuchWebsiteConfiguration` 按“未配置”处理。
  - 权限、签名、区域或网络错误写入对应 `*CollectError` 字段，不阻断 bucket 主体资产。
- 单元测试：
  - 覆盖 Policy JSON、ACL XML、Object Lock、Replication、Logging、Notification、Website 和 Inventory XML 解析。

验证：

- 使用 Docker Go 镜像执行 `gofmt` 格式化 AWS collector 和 S3 测试文件。
- 使用临时 Dockerfile + BuildKit Go 缓存执行定向测试：
  - `go test -vet=off ./portal/apps -run 'TestAwsS3.*' -count=1 -timeout=120s` 通过。

验证限制：

- 本阶段未连接真实 AWS 账号做端到端采集验证。
- S3 深层配置风险规则映射已在 16.176 完成第一阶段；成本分摊和更完整合规规则映射仍待继续。

### 16.176 2026-06-22 V1.2 P0 AWS S3 Bucket 风险规则映射第一阶段

状态：已完成 AWS S3 Bucket 风险规则映射第一阶段；16.174/16.175 采集到的对象存储安全配置已经接入云风险发现派生，风险列表和风险合规页面可直接识别 S3 公开访问与基础合规缺口。

已完成：

- 风险派生规则：
  - `s3_bucket_public_policy`：Bucket Policy 存在公开 Allow 时生成严重风险。
  - `s3_bucket_public_acl`：Bucket ACL 对 `AllUsers` / `AuthenticatedUsers` 公开授权时生成严重风险。
  - `s3_public_access_block_disabled`：Public Access Block 四项未完全开启时生成高风险。
  - `s3_default_encryption_disabled`：默认加密未开启时生成高风险。
  - `s3_versioning_disabled`：版本控制采集成功但未启用时生成中风险。
  - `s3_access_logging_disabled`：访问日志采集成功但未启用时生成中风险。
  - `s3_object_lock_disabled`：Object Lock 采集成功但未启用时生成中风险。
- 风险摘要：
  - `publicExposure` 统计扩展到 S3 公开 Policy 和公开 ACL。
- 适配边界：
  - 仅对 AWS S3 Bucket 或 `aws_s3_bucket` 原生类型生效，避免误伤 GCS/OSS/OBS/COS 等对象存储。
  - 版本控制风险要求已存在版本控制采集证据，避免未采集字段导致误报。

验证：

- 使用 Docker Go 镜像执行 `gofmt` 格式化风险逻辑和测试文件。
- 使用临时 Dockerfile + BuildKit Go 缓存执行定向测试：
  - `go test -vet=off ./portal/apps -run 'Test(CloudRiskS3|AwsS3).*' -count=1 -timeout=120s` 通过。

验证限制：

- 本阶段为基于资产属性的风险派生单测验证，未连接真实 AWS 账号做风险端到端回归。
- S3 成本分摊、数据分类、合规框架映射和组织级风险阈值配置仍待后续增强。

### 16.177 2026-06-22 V1.2 P0 OCI 网络路由与网关采集第一阶段

状态：已完成 OCI 网络路由与网关采集第一阶段；OCI collector 在既有 Compute、VCN/Subnet、Security List/NSG、Public IP、Block Volume、LB、Bucket、OKE、DB、Redis 基础上，补齐 Route Table、NAT Gateway、Internet Gateway、Service Gateway 和 DRG 标准资产，并接入云侧关系推演。

已完成：

- 标准资产类型：
  - 新增 `network_service_gateway`：OCI Service Gateway。
  - 新增 `network_drg`：OCI Dynamic Routing Gateway。
  - `NormalizeCmdbAssetType` 识别 `service_gateway/servicegateway` 与 `drg/dynamic_routing_gateway/oci_core_drg`。
- OCI collector：
  - `collectOciRouteTables` 调用 Core `/routeTables`，写入 `vcnId`、`routeRules`、`destinationCidrBlocks`、`destinationServiceIds`、`gatewayIds`，并从 `networkEntityId` 推导 `natGatewayIds`、`internetGatewayIds`、`serviceGatewayIds`、`drgIds`。
  - `collectOciNatGateways` 调用 Core `/natGateways`，归一化为 `network_nat_gateway / oci_core_nat_gateway`，保留 `vcnId` 和 `blockTraffic`。
  - `collectOciInternetGateways` 调用 Core `/internetGateways`，归一化为 `network_internet_gateway / oci_core_internet_gateway`，保留 `vcnId` 和 `isEnabled`。
  - `collectOciServiceGateways` 调用 Core `/serviceGateways`，归一化为 `network_service_gateway / oci_core_service_gateway`，保留 `vcnId`、`services`、`serviceIds` 和 `blockTraffic`。
  - `collectOciDrgs` 调用 Core `/drgs`，归一化为 `network_drg / oci_core_drg`，保留默认 DRG route table 和导出 route distribution 引用。
  - OCI 云账号 `supportedAssetTypes` 新增 NAT Gateway、Internet Gateway、Service Gateway 和 DRG；已有 Route Table 选择项现在有真实采集实现。
- 关系推演：
  - Route Table 归属 VCN，并依赖 Subnet、NAT Gateway、Internet Gateway、Service Gateway 和 DRG。
  - NAT Gateway、Internet Gateway、Service Gateway 可按 `vcnId` 归属 VCN。
  - Subnet、Security List/NSG、Compute、LB、Kubernetes、DB/Redis 的关系推演补充 `vcnId`、`securityListIds`、`nsgIds` 等 OCI 常见字段。
- 前端展示：
  - 云资产/CMDB 资源类型中文名新增 `network_service_gateway` 与 `network_drg`。

验证：

- 使用 Docker Go 镜像执行 `gofmt` 格式化 OCI collector、关系推演、模型和测试文件。
- 新增单元测试覆盖：
  - `TestOciRouteRuleRefsExtractsGatewayRefs`：验证 OCI Route Rule 可解析 NAT/IGW/Service Gateway/DRG 引用和目标地址。
  - `TestInferCloudRelationsAddsOciNetworkGatewayRefs`：验证 VCN、Subnet、Route Table、NAT/IGW/Service Gateway/DRG 的推演关系。
- 使用带 `cloudiac-go-mod` / `cloudiac-go-build` 缓存卷的 Docker Go 镜像执行定向测试：
  - `go test -vet=off ./portal/apps -run 'Test(OciRouteRuleRefs|InferCloudRelationsAddsOciNetworkGatewayRefs)$' -count=1 -timeout=120s` 通过。
  - 测试仅运行一次性 Go 容器，未启动平台服务。

验证限制：

- 本阶段为代码路径和单元测试用例补齐，未连接真实 OCI 账号验证分页、权限错误、限流、跨 compartment 可见性和 provider 原始错误码。
- DRG attachment、DRG route table、route distribution 等更深层网络资源仍可在后续阶段拆成独立资产或增强关系。
- compartment/identity 归属映射已在 16.178 完成第一阶段；成本/合规风险映射仍待继续。

### 16.178 2026-06-22 V1.2 P0 OCI compartment/identity 归属映射第一阶段

状态：已完成 OCI compartment/identity 归属映射第一阶段；OCI 云采集不再只在资产上保存 `compartmentId`，而是把 tenancy、user、credential fingerprint、compartment 名称、父级和路径写入资产属性，并在同步任务 stats 中返回 compartment 明细，便于后续做组织归属、权限漂移和成本/合规分析。

已完成：

- Compartment 元数据：
  - 新增 `ociCompartmentRef`，统一保存 `id/name/parentId/path/lifecycleState/description`。
  - `ociCompartments` 支持从 `OCI_COMPARTMENT_OCIDS`、`OCI_COMPARTMENT_OCID` 或 `OCI_TENANCY_OCID` 构造采集范围。
  - 开启 `OCI_INCLUDE_SUBCOMPARTMENTS` 或 `OCI_COMPARTMENT_IN_SUBTREE` 时，继续调用 Identity `/compartments` 获取可访问子 compartment，并按父子关系构建路径。
  - 未开启子树采集或 Identity API 失败时，仍保留配置的 compartment ID 作为降级采集范围。
- 资产归属字段：
  - 每个 OCI 采集资产写入 `tenancyId`、`tenancyName`、`userId`、`credentialFingerprint`。
  - 每个 OCI 采集资产写入 `compartmentId`、`compartmentName`、`compartmentPath`、`compartmentParentId`、`compartmentLifecycleState`。
  - `rawData` 同步保存关键 tenancy/compartment 归属字段，方便排查原始来源。
- 同步任务统计：
  - `stats.compartments` 保留按 region 返回的 compartment ID 列表。
  - 新增 `stats.compartmentDetails`，按 region 返回 compartment 名称、路径、父级和状态。

验证：

- 使用 Docker Go 镜像执行 `gofmt` 格式化 OCI collector 和 OCI 网络测试文件。
- 使用带 `cloudiac-go-mod` / `cloudiac-go-build` 缓存卷的 Docker Go 镜像执行定向测试：
  - `go test -vet=off ./portal/apps -run 'Test(OciRouteRuleRefs|InferCloudRelationsAddsOciNetworkGatewayRefs|BuildOciCompartmentRefs|AnnotateOciCompartmentAsset)$' -count=1 -timeout=120s` 通过。
  - 测试仅运行一次性 Go 容器，未启动平台服务。

验证限制：

- 本阶段验证的是归属字段构建和资产注入逻辑，未连接真实 OCI 账号验证跨 tenancy/跨 compartment 权限边界。
- Identity 权限漂移、用户/组/动态组/策略解析和 provider 原生错误码仍待真实环境继续补齐。

### 16.110 2026-06-21 V1.2 P1 云资产同步策略第一阶段

状态：已完成云资产同步策略模型、API、后台 worker、任务绑定、资产 `syncPolicyId` 写入和云账号页面前端入口第一阶段。

已完成：

- 新增 `iac_cloud_sync_policy` 模型，支持按组织保存策略名称、统一云账号、provider、账号 ID、区域、资源类型、启停状态、同步间隔、失败最大重试次数、失败退避秒数、失败通知事件、自动暂停、最近任务和下次同步时间。
- `iac_cmdb_sync_task` 新增 `syncPolicyId`，同步任务列表支持按策略 ID 查询，任务详情返回策略 ID。
- `StartCmdbSyncTask` 支持携带 `syncPolicyId`，并校验策略、统一云账号和 provider 一致，避免手动任务绕过策略范围。
- 云采集任务执行时会把策略 ID 写入本轮采集资产的 `syncPolicyId`，让 16.8 中预留的治理字段具备真实来源。
- 采集任务完成后回写同步策略：
  - 成功时更新 `lastSyncTaskId`、`lastSyncStatus=complete`、`lastSyncedAt`、`nextSyncAt` 并清空失败状态。
  - 失败时递增 `failureCount`，记录 `lastFailureAt`、`lastFailureReason`，按退避秒数计算下一次重试。
  - 超过最大重试次数且开启自动暂停时，策略状态写为 `disable`。
- 新增同步策略 API：
  - `GET /api/v1/cloud/sync-policies`
  - `POST /api/v1/cloud/sync-policies`
  - `GET /api/v1/cloud/sync-policies/:id`
  - `PUT /api/v1/cloud/sync-policies/:id`
  - `DELETE /api/v1/cloud/sync-policies/:id`
  - `POST /api/v1/cloud/sync-policies/:id/run`
  - `POST /api/v1/cloud/sync-policies/run-due`
- Portal 启动时新增 `StartCloudSyncPolicyWorker`，后台扫描到期启用策略并创建云采集任务；使用 MySQL 分布式锁避免多实例重复触发。
- 事件中心接入：
  - 策略触发写入 `cloud.sync.policy.triggered`。
  - 策略成功写入 `cloud.sync.policy.completed`。
  - 开启失败通知时写入 `cloud.sync.policy.failed` 或 `cloud.sync.policy.auto_paused`。
- 云账号页面新增“同步策略”列表和“创建/编辑同步策略”抽屉，支持搜索、按云厂商/状态筛选、创建、编辑、启停、删除、手动运行和运行到期策略。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化本阶段新增/修改的 Go 文件。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`。
- 数据库迁移验证：
  - `iac_cloud_sync_policy` 表已创建。
  - `iac_cloud_sync_policy.retry_backoff_seconds` 字段存在。
  - `iac_cmdb_sync_task.sync_policy_id` 字段存在。
- API 端到端验证：
  - 创建临时腾讯云离线 inventory 云账号，账号 `ready=true`、`validationStatus=valid`。
  - 创建同步策略，返回 `status=enable`、`provider=tencentcloud`、`nextSyncAt` 非空。
  - 调用 `POST /api/v1/cloud/sync-policies/:id/run` 创建云采集任务，任务返回 `status=running` 且 `syncPolicyId` 等于策略 ID。
  - 轮询任务详情后任务 `complete`，`stats.mode=offline_inventory`、`collected=1`、`created=1`、`syncPolicyId` 写入统计。
  - 策略详情回写 `lastSyncTaskId`、`lastSyncStatus=complete`、`failureCount=0`、`lastError=""`。
  - 数据库查询临时资产，`iac_cmdb_asset.sync_policy_id` 等于本轮策略 ID。
  - 验证后已清理本轮临时云账号、同步策略、同步任务、任务日志、云事件、资产和资产变更记录，剩余临时数据计数均为 `0`。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size warning。
- `docker compose up -d iac-web` 成功。
- 内置浏览器验证云账号页面：
  - `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-account` 正常加载。
  - 页面展示“同步策略”“创建同步策略”“运行到期策略”。
  - 控制台无 error。
  - 打开“创建同步策略”抽屉后可见“策略名称”“云账号”“资源类型范围”“失败最大重试次数”“超过重试次数后自动暂停”等字段，关闭抽屉后页面状态正常。

验证限制：

- 本阶段使用腾讯云离线 inventory JSON 验证策略闭环，未依赖真实云 API；真实 AWS/OCI/AliCloud/Azure/GCP/腾讯云/华为云账号仍需按 provider 做端到端采集联调。
- 后台 worker 扫描周期已在 16.116 完成配置化，默认仍为 5 分钟。
- 增量采集仍以 provider collector 当前能力为准，尚未做 provider 原生分页游标和区域级失败重试；按资源类型/区域独立子周期已在 16.118 完成第一阶段。
- 失败通知先进入平台 CloudEvent；外部通知/Webhook/ITSM 分发可复用事件订阅继续扩展。

待继续：

- 在真实多云只读账号上验证同步策略的分页、限流、区域级错误映射和增量采集。
- 在云账号页面继续补充策略与账号健康状态的联动提示，例如“最近同步超过策略间隔”按策略阈值判断。
- 继续在真实多云只读账号上验证子周期模式下的分页、限流、区域级错误映射和增量采集。

### 16.116 2026-06-21 V1.2 P1 云资产同步策略 worker 周期配置化第一阶段

状态：已完成云资产同步策略后台 worker 周期配置化和启动可观测日志第一阶段。

已完成：

- 新增环境变量 `CLOUDIAC_CLOUD_SYNC_POLICY_WORKER_INTERVAL_SECONDS`：
  - 默认值 300 秒。
  - 最小值 30 秒。
  - 最大值 86400 秒。
  - 非法值或空值自动回退到默认 5 分钟。
- `StartCloudSyncPolicyWorker` 使用配置化周期，不再固定返回 5 分钟。
- 同步策略 worker 启动时输出 `cloud sync policy worker started`，并通过日志字段展示实际 `interval`、`serviceId` 和 `worker=cloudSyncPolicy`。
- `backend/configs/dotenv.sample` 增加同步策略 worker 周期配置说明。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_sync_policy.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- 临时在部署 `.env` 中设置 `CLOUDIAC_CLOUD_SYNC_POLICY_WORKER_INTERVAL_SECONDS=60` 后重启 Portal：
  - `iac-portal` 状态为 `healthy`。
  - `GET /api/v1/check` 返回 `success=true`。
  - 日志输出 `cloud sync policy worker started`，且 `interval=1m0s`、`worker=cloudSyncPolicy`。
- 恢复部署 `.env` 后重启 Portal：
  - `iac-portal` 状态为 `healthy`。
  - `GET /api/v1/check` 返回 `success=true`。
  - 日志输出 `cloud sync policy worker started`，且 `interval=5m0s`、`worker=cloudSyncPolicy`。

验证限制：

- 当前配置是全局 worker 扫描周期；暂停窗口、并发控制和采集配额保护已在 16.117 完成第一阶段，按资源类型/区域的差异化周期仍待继续。
- 真实多云账号下的分页、限流、区域级错误映射和增量采集仍待 provider 端到端联调。

### 16.117 2026-06-21 V1.2 P1 云资产同步策略暂停窗口、并发控制和采集配额保护第一阶段

状态：已完成云资产同步策略自动调度保护第一阶段，支持策略暂停窗口、每日运行上限、单轮触发上限、组织内运行中采集任务并发上限，以及云账号页面保护状态展示。

已完成：

- 同步策略 `params` 增加保护参数并保持旧数据兼容：
  - `pauseWindows`：暂停窗口列表，格式为 `HH:MM-HH:MM`，支持跨天窗口；`00:00-00:00` 表示全天暂停。
  - `maxRunsPerDay`：策略每日运行上限，`0` 或空值表示不限制。
- 创建/更新同步策略时会规范化保护参数：
  - 非法暂停窗口会被忽略。
  - 合法暂停窗口会统一存储为 `HH:MM-HH:MM`。
  - 非正数每日上限会被移除，避免旧策略被误限流。
- `POST /api/v1/cloud/sync-policies/run-due` 和后台 worker 的自动调度新增保护判断：
  - 命中暂停窗口时跳过策略并返回原因。
  - 今日运行次数达到 `maxRunsPerDay` 时跳过策略并返回原因。
  - 环境变量 `CLOUDIAC_CLOUD_SYNC_POLICY_MAX_TRIGGERED_PER_RUN` 控制单轮最多触发策略数，默认 `0` 不限制。
  - 环境变量 `CLOUDIAC_CLOUD_SYNC_POLICY_MAX_CONCURRENT_RUNNING` 控制组织内运行中采集任务上限，默认 `0` 不限制。
- `CloudSyncPolicyRunResp` 增加：
  - `protectionSkippedCount`
  - `protectionSkippedReasons`
- 同步策略列表和详情响应增加 `protection`：
  - `pausedNow`
  - `pauseReason`
  - `pauseWindows`
  - `maxRunsPerDay`
  - `runsToday`
  - `maxTriggeredPerRun`
  - `maxConcurrentRunning`
  - `runningCount`
- 云账号页面同步策略列表新增“保护”列，展示暂停状态、暂停窗口、每日配额、单轮上限和并发上限。
- “创建/编辑同步策略”抽屉新增：
  - “暂停窗口”
  - “每日运行上限”
- 启停策略时保留现有 `params`，避免通过快捷启停丢失保护配置。
- 修复同步策略自动调度锁名过长问题：
  - 原锁名 `cloud_sync_policy:<orgId>:<policyId>` 可能超过 MySQL `GET_LOCK` 64 字节限制，导致自动调度无法获取锁。
  - 新锁名缩短为 `csp:<orgId>:<policyId>`，保持同组织同策略粒度且满足 MySQL 长度限制。
- `backend/configs/dotenv.sample` 增加同步策略保护配置示例：
  - `CLOUDIAC_CLOUD_SYNC_POLICY_MAX_TRIGGERED_PER_RUN=0`
  - `CLOUDIAC_CLOUD_SYNC_POLICY_MAX_CONCURRENT_RUNNING=0`

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_sync_policy.go` 和 `backend/portal/models/resps/cloud_sync_policy.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size warning。
- `docker compose up -d iac-portal iac-web` 成功，`GET /api/v1/check` 返回 `success=true`。
- API 端到端验证暂停窗口与每日配额：
  - 创建临时腾讯云离线 inventory 云账号。
  - 创建暂停窗口策略，`params.pauseWindows=["00:00-00:00"]`。
  - 创建每日配额策略，`params.maxRunsPerDay=1`。
  - 手动运行每日配额策略生成今日运行记录。
  - 调用 `POST /api/v1/cloud/sync-policies/run-due` 返回 `protectionSkippedCount=2`。
  - `protectionSkippedReasons` 同时包含“暂停窗口”和“今日同步次数”。
  - 策略详情返回 `protection.pausedNow=true`、`pauseWindows=["00:00-00:00"]`、`maxRunsPerDay=1`、`runsToday>=1`。
- API 端到端验证单轮触发上限：
  - 临时设置部署 `.env`：`CLOUDIAC_CLOUD_SYNC_POLICY_MAX_TRIGGERED_PER_RUN=1`、`CLOUDIAC_CLOUD_SYNC_POLICY_MAX_CONCURRENT_RUNNING=0`，重启 Portal。
  - 创建两条到期启用策略并调用 `run-due`。
  - 返回 `triggeredCount=1`、`protectionSkippedCount=1`，跳过原因包含“本轮触发数 1 已达到全局上限 1”。
  - 策略详情返回 `protection.maxTriggeredPerRun=1`。
- API 端到端验证并发上限：
  - 临时设置部署 `.env`：`CLOUDIAC_CLOUD_SYNC_POLICY_MAX_TRIGGERED_PER_RUN=0`、`CLOUDIAC_CLOUD_SYNC_POLICY_MAX_CONCURRENT_RUNNING=1`，重启 Portal。
  - 插入一条临时 `running` 采集任务后调用 `run-due`。
  - 返回 `triggeredCount=0`、`protectionSkippedCount=1`，跳过原因包含“运行中采集任务 1 已达到全局并发上限 1”。
  - 策略详情返回 `protection.maxConcurrentRunning=1`、`runningCount=1`。
- 验证过程中创建的临时云账号、同步策略、同步任务、任务日志和云事件均已清理，临时 `.env` 已恢复。
- 内置浏览器验证云账号页面：
  - `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-account` 正常加载。
  - 同步策略表头展示“保护”列。
  - 打开“创建同步策略”抽屉后可见“暂停窗口”和“每日运行上限”字段，关闭抽屉后页面状态正常。

验证限制：

- 保护逻辑当前作用于自动调度入口 `run-due` 和后台 worker；单条策略手动运行保留管理员强制执行语义，不受暂停窗口和配额保护限制。
- 并发上限当前按组织内 `pending/running` 采集任务计数，平台级跨组织统一并发池可在调度器分片阶段继续扩展。
- 按资源类型/区域的独立同步周期已在 16.118 完成第一阶段；本阶段保护参数先挂载在策略级 `params`。
- 真实多云账号下的 provider API 限流、分页游标、区域级错误映射和增量采集仍待端到端联调。

### 16.118 2026-06-21 V1.2 P1 云资产同步策略按区域/资源类型独立周期第一阶段

状态：已完成同步策略按区域/资源类型独立子周期第一阶段；单个策略可配置多个子计划，后台到期调度按子计划独立触发采集任务，并把任务状态回写到对应子计划。

已完成：

- 同步策略 `params` 扩展：
  - `scheduleOverrides`：保存子计划配置，包含 `key`、`name`、`regions`、`assetTypes`、`syncInterval`。
  - `scheduleState`：保存每个子计划的运行态，包含 `nextSyncAt`、`lastSyncedAt`、`lastSyncTaskId`、`lastSyncStatus`、`lastError` 和失败计数。
  - 创建/更新策略时会规范化子计划配置，并清理不再存在子计划的运行态。
- 自动调度增强：
  - 没有子计划的旧策略保持原有按策略整体同步行为。
  - 配置子计划后，`run-due` 和后台 worker 会展开到期子计划，只为到期范围创建采集任务。
  - 子计划采集任务会按自身 `regions`、`assetTypes` 覆盖策略默认范围。
  - 策略 `nextSyncAt` 按所有子计划中最早的下一次同步时间写回，避免未到期子计划反复被扫描。
- 任务状态回写：
  - 采集任务 `stats` 和日志增加 `syncPolicyScheduleKey`、`syncPolicyScheduleName`。
  - 子计划任务完成后只更新对应 `scheduleState`，并同步更新策略的最近任务、最近状态和整体最早 `nextSyncAt`。
  - 子计划失败沿用策略级最大重试次数、失败退避、通知事件和自动暂停规则。
- 前端云账号页面增强：
  - “创建/编辑同步策略”抽屉新增“独立周期”配置行，可填写名称、区域、资源类型和间隔秒数。
  - 同步策略列表新增“子周期”列，展示子计划名称、到期状态、最近执行状态、区域、资源类型、间隔和下次同步时间。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_sync_policy.go`、`backend/portal/apps/cmdb_sync.go`、`backend/portal/models/forms/cmdb.go`、`backend/portal/models/resps/cloud_sync_policy.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功，`/api/v1/check` 返回 `success=true`，Web 首页 HTTP 200。
- API 验证独立子周期：
  - 创建临时腾讯云离线 inventory 云账号 `codex-schedule-*`，包含广州 CVM、上海 COS 和上海 CVM 三个资源。
  - 创建同步策略，配置两个子计划：`guangzhou-compute` 仅同步 `ap-guangzhou + compute_instance`，`shanghai-cos` 仅同步 `ap-shanghai + object_storage_bucket`。
  - 第一次调用 `run-due` 返回 `triggeredCount=2`，生成两个采集任务，两个任务分别只携带各自区域和资源类型。
  - 等待任务完成后，策略详情两个子计划 `lastSyncStatus=complete`，并分别写入对应 `lastSyncTaskId` 和 `nextSyncAt`。
  - 手动把 `guangzhou-compute` 的 `nextSyncAt` 回拨到过去后再次调用 `run-due`，返回 `triggeredCount=1`，只生成广州计算子计划任务。
  - 第二次任务完成后，`guangzhou-compute.lastSyncTaskId` 指向新任务，`shanghai-cos` 子计划未被重复触发。
- 验证后已清理本轮临时云账号、同步策略、同步任务、任务日志、云事件、资产和资产变更记录，剩余临时数据计数均为 `0`。

验证限制：

- 本阶段使用腾讯云离线 inventory JSON 验证子周期闭环，未依赖真实云 API；真实 AWS/OCI/AliCloud/Azure/GCP/腾讯云/华为云账号仍需做端到端采集联调。
- 子计划失败退避复用策略级失败计数和自动暂停规则；后续可继续扩展为每个子计划独立失败预算和错误分类。
- 子计划配置保存在策略 `params` 中，避免新增迁移表；如后续需要大量子计划、分片 worker 或审计化变更历史，可再拆出独立子计划表。

### 16.119 2026-06-21 V1.2 P1 云账号健康详情接入同步策略子周期第一阶段

状态：已完成云账号健康详情按同步策略子周期推导健康状态第一阶段；账号健康检查不再只按账号或策略整体最近同步时间判断，可识别具体区域/资源类型子周期的过期或失败，并在云账号列表展示子周期健康摘要。

已完成：

- 健康判断增强：
  - 云账号健康检查会读取账号下启用同步策略的 `scheduleOverrides` 和 `scheduleState`。
  - 配置子周期时，按每个子周期的 `syncInterval`、`lastSyncedAt`、`lastSyncStatus` 和 `nextSyncAt` 生成健康窗口。
  - 告警优先级按失败、未同步、过期、正常排序，最终 `healthMessage` 会指向具体策略和子周期名称。
  - 未配置子周期的旧策略继续按策略整体 `syncInterval` 和账号最近同步时间判断，保持兼容。
- 健康详情响应扩展：
  - `CloudAccountHealthDetailResp` 新增 `schedules`。
  - 每个子周期返回策略 ID/名称、子周期 key/名称、区域、资源类型、同步间隔、健康窗口、最近任务、最近状态、最近错误、最近同步时间、是否过期和是否到期。
- 前端云账号页面增强：
  - “健康依据”列新增“子周期：正常数/总数 正常”摘要。
  - 子周期摘要 Tooltip 展示每个子周期名称、健康状态、区域和资源类型。
- 同步策略状态回写稳定性增强：
  - 子周期触发时写入 `scheduleState` 前会读取最新策略参数，避免多个子周期顺序触发时后一个覆盖前一个状态。
  - 对同一任务已完成/失败的子周期状态增加防回退保护，避免异步采集任务过快完成后又被触发态覆盖为 `running`。
  - 触发态写入后会检查任务是否已进入终态，如已完成则立即复用完成回写逻辑刷新策略状态。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_account.go`、`backend/portal/apps/cloud_sync_policy.go`、`backend/portal/models/resps/cloud_account.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功，`/api/v1/check` 返回 `success=true`，Web 首页 HTTP 200。
- API 验证同步策略子周期健康：
  - 创建临时腾讯云离线 inventory 云账号 `codex-health-schedule-*`。
  - 创建两个子周期：`guangzhou-compute` 为 `ap-guangzhou + compute_instance + 60 秒`，`shanghai-cos` 为 `ap-shanghai + object_storage_bucket + 3600 秒`。
  - 预置两个子周期到期后调用 `run-due`，返回 `triggeredCount=2`，两个子周期最终均回写 `lastSyncStatus=complete`。
  - 首次健康检查返回 `healthStatus=healthy`，`healthDetail.schedules` 含两个子周期，且 `stale=false`。
  - 将 `guangzhou-compute` 的 `lastSyncedAt` 和 `nextSyncAt` 回拨到 2 小时前后再次健康检查，返回 `healthStatus=warning`，`healthMessage` 点名 `guangzhou-compute`，该子周期 `stale=true`，`shanghai-cos.stale=false`。
- 内置浏览器验证云账号页面：
  - 云账号列表可见“健康依据”列展示 `子周期：1/2 正常`。
  - 同步策略列表可见两个子周期、到期状态、区域、资源类型、间隔和下次同步时间。
- 验证后已清理本轮临时云账号、同步策略、同步任务、任务日志、云事件、资产和资产变更记录，剩余临时数据计数均为 `0`。

验证限制：

- 本阶段仍基于本地同步策略和任务状态判断健康；同步任务失败分类已在 16.123 接入账号健康推断，真实云 provider 主动凭证过期、API 限流、区域级权限漂移探测仍待 adapter 增强。
- 子周期失败预算仍复用策略级失败计数；如需区分不同区域/资源类型的失败预算、静默期和通知策略，后续可继续拆分子周期状态模型。
- 最近成功/失败任务摘要的账号级“同步摘要”列仍保持汇总展示；健康依据中的子周期最近任务证据已在 16.120 完成第一阶段。

### 16.120 2026-06-21 V1.2 P1 云账号健康子周期最近任务摘要第一阶段

状态：已完成云账号健康详情中同步策略子周期最近任务摘要第一阶段；每个子周期可返回最近成功任务和最近失败任务，前端健康依据 Tooltip 可展示子周期级成功/失败时间。

已完成：

- 健康详情响应扩展：
  - `CloudAccountHealthScheduleResp` 新增 `lastSuccessTask` 和 `lastFailureTask`。
  - 复用 `CloudAccountSyncTaskBriefResp`，返回任务 ID、同步策略 ID、状态、错误信息、统计、开始/结束/创建时间。
- 子周期任务查询：
  - 按组织、统一云账号、同步策略 ID、任务状态过滤 CMDB 同步任务。
  - 使用任务 `stats.syncPolicyScheduleKey` 精确匹配子周期，避免账号级最近任务误归属到其他区域或资源类型。
- 前端云账号页面增强：
  - “健康依据”列的子周期 Tooltip 增加最近成功和最近失败时间。
  - 账号级“同步摘要”列继续保持成功/失败汇总，避免列表主视图过宽。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_account.go`、`backend/portal/models/resps/cloud_account.go`。
- `docker compose build iac-portal iac-web` 成功；前端仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功，`iac-portal` 健康检查为 `healthy`。
- API 验证子周期最近任务：
  - 创建临时腾讯云离线 inventory 云账号 `codex-schedule-task-*`。
  - 创建两个子周期：`guangzhou-compute` 和 `shanghai-cos`。
  - 预置两个子周期到期后调用 `run-due`，返回 `triggeredCount=2`，生成两个采集任务。
  - 调用云账号健康检查后，`healthDetail.schedules` 两个子周期均返回 `lastSuccessTask.status=complete`。
  - 两个 `lastSuccessTask.stats.syncPolicyScheduleKey` 分别与对应 `scheduleKey` 一致，确认不是账号级任务误匹配。
- 验证后已清理本轮临时云账号、同步策略、同步任务、任务日志、云事件、资产和资产变更记录，剩余临时数据计数均为 `0`。

验证限制：

- 本阶段先把子周期最近任务作为健康依据 Tooltip 展示；按子周期跳转到任务详情的交互已在 16.127 完成第一阶段。
- 子周期失败任务需要真实失败采集或失败任务样本；本轮自动化验证覆盖成功任务精确匹配，失败任务路径复用同一查询条件和响应字段。
- 账号级“同步摘要”列仍按账号汇总，后续可增加展开行或详情抽屉展示子周期任务历史。

### 16.121 2026-06-21 V1.2 P1 云账号健康检查 worker 分片与并发控制第一阶段

状态：已完成后端实现、Docker Compose 构建启动和 API 验证。

已完成：

- 批量云账号健康检查支持账号级分片参数：
  - `shardIndex`：当前分片索引，从 `0` 开始。
  - `shardTotal`：总分片数，最大 `128`。
  - 分片规则复用 `md5(id)` 前缀取模，和 Webhook 队列分片方式保持一致。
- 批量云账号健康检查支持并发度参数：
  - `concurrency`：本次最多并发检查账号数，最大 `32`。
  - 手动 API 未传入时保持串行，避免改变旧调用行为。
- 后台 `cloudAccountHealth` worker 增加环境变量：
  - `CLOUDIAC_CLOUD_ACCOUNT_HEALTH_WORKER_SHARDS`：后台 worker 分片数，默认 `1`，最大 `128`。
  - `CLOUDIAC_CLOUD_ACCOUNT_HEALTH_WORKER_CONCURRENCY`：后台 worker 并发账号检查数，默认 `4`，最大 `32`。
- 后台 worker 分片模式下使用 per-shard MySQL advisory lock：
  - 未启用分片时继续使用 `cloudiac:cloud_account_health:all` 全局锁，兼容 16.115 的多实例保护。
  - 启用分片后使用 `cloudiac:cloud_account_health:all:shard:{total}:{index}`，避免多个 Portal 实例重复处理同一分片。
- 健康检查汇总响应新增调度可观测字段：
  - `shardIndex`、`shardTotal`、`concurrency`。
  - `locked`、`lockSkipped`、`lockSkippedCount`。
  - `shards[]`：分片级 total、checked、healthy、warning、unhealthy、lock 状态。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化相关 Go 文件。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`iac-portal` 状态为 `healthy`。
- `GET /api/v1/check` 返回 `success=true`。
- worker 启动日志确认新调度字段：
  - `worker=cloudAccountHealth`。
  - `shardTotal=1`。
  - `concurrency=4`。
  - `lockSkipped=0`。
- API 验证：
  - 创建 4 个临时腾讯云离线 inventory 云账号 `codex-health-shard-*`。
  - 使用数据库表达式 `mod(cast(conv(substr(md5(id),1,8),16,10) as unsigned),2)` 计算临时账号分片，两个账号落在 shard `0`，两个账号落在 shard `1`。
  - 调用 `POST /cloud/accounts/health-check`，参数 `shardIndex=0&shardTotal=2&concurrency=2`，响应 `shardIndex=0`、`shardTotal=2`、`concurrency=2`，只包含 shard `0` 的临时账号，且 `shards[0]` 的 total/checked 与顶层汇总一致。
  - 调用 `POST /cloud/accounts/health-check`，参数 `shardIndex=1&shardTotal=2&concurrency=2`，响应 `shardIndex=1`、`shardTotal=2`、`concurrency=2`，只包含 shard `1` 的临时账号，且 `shards[0]` 的 total/checked 与顶层汇总一致。
  - 使用非法参数 `shardIndex=2&shardTotal=2` 验证返回 HTTP `400`。
- 验证后已清理本轮临时云账号和健康事件，剩余临时数据计数均为 `0`。

验证限制：

- 第一阶段先提供 worker 内部分片和 per-shard 锁保护；多个 Portal 实例同时启动时仍会竞争所有分片锁，后续可进一步按实例注册表做稳定 shard 分配。
- 真实云 provider 权限漂移、凭证过期和 API 限流识别仍待 adapter 增强。

### 16.122 2026-06-21 V1.2 P1 云同步任务区域/资源类型统计与失败分类第一阶段

状态：已完成 CMDB 云同步任务统计增强、Docker Compose 构建启动和 API 验证。

已完成：

- CMDB 云同步任务 `stats` 增加阶段耗时：
  - `collectorDurationMs`：云厂商 collector 执行耗时。
  - `upsertDurationMs`：资产入库/更新耗时。
  - `relationDurationMs`：云资产关系推演耗时。
  - `durationMs`：任务总耗时。
- CMDB 云同步任务 `stats` 增加区域和资源类型统计：
  - `regionMetrics[]`：按 region 返回 collected 和 status。
  - `assetTypeMetrics[]`：按 assetType 返回 collected 和 status。
  - `scopeMetrics[]`：按 region + assetType 返回 collected 和 status；当任务本身为单 region + 单 assetType 时返回该 scope 的 `durationMs`。
- CMDB 云同步任务 `stats` 增加失败分类和重试提示：
  - `failureDetails[]`：包含原始错误、分类、是否建议重试和重试提示。
  - `failureSummary`：返回失败总数和可重试失败总数。
  - 第一阶段内置 `rate_limit`、`network`、`permission`、`credential`、`configuration`、`unknown` 分类。
- 云资产中心包装路由复用同一详情响应：
  - `GET /api/v1/cloud/sync-tasks`。
  - `GET /api/v1/cloud/sync-tasks/:id`。
  - `POST /api/v1/cloud/sync-tasks`。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cmdb_sync.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`iac-portal` 状态为 `healthy`。
- 成功任务 API 验证：
  - 创建临时腾讯云离线 inventory 云账号 `codex-sync-metrics-*-valid`。
  - 调用 `POST /cloud/sync-tasks`，限定 `regions=[ap-guangzhou]` 和 `assetTypes=[compute_instance]`。
  - 任务最终状态为 `complete`。
  - `GET /cloud/sync-tasks/:id` 返回 `collectorStatus=complete`。
  - `collectorDurationMs`、`upsertDurationMs`、`relationDurationMs`、`durationMs` 均为数字。
  - `regionMetrics` 中 `ap-guangzhou.collected=1` 且 `status=complete`。
  - `assetTypeMetrics` 中 `compute_instance.collected=1` 且 `status=complete`。
  - `scopeMetrics` 中 `ap-guangzhou + compute_instance` 返回 `collected=1`、`status=complete` 和数字型 `durationMs`。
  - `failureSummary.total=0`，任务日志数量不小于 `4`。
- 失败任务 API 验证：
  - 创建临时腾讯云离线 inventory 云账号 `codex-sync-metrics-*-invalid`，inventory JSON 故意传入非法内容。
  - 调用 `POST /cloud/sync-tasks` 后任务最终状态为 `failed`。
  - `GET /cloud/sync-tasks/:id` 返回 `collectorStatus=failed`。
  - `failureSummary.total>=1`。
  - `failureDetails[0]` 返回错误分类、原始错误、`retryable` 和 `retryHint`。
  - `regionMetrics` 中 `ap-guangzhou.collected=0` 且 `status=partial_failed`。
- 验证后已清理本轮临时云账号、同步任务、任务日志、云资产、资产变更、资产关系和事件，剩余临时数据计数均为 `0`。

验证限制：

- 当前 scope 耗时在单 region + 单 assetType 任务中可精确映射；多 region + 多 assetType 任务第一阶段先返回 scope 数量统计，OCI provider 内部单 API 子调用耗时已在 16.181 完成第一阶段，其他 provider 可继续按同一结构上报。
- 失败分类基于错误文本做第一阶段归类；云厂商原生错误码、region/assetType 自动局部重试已完成第一阶段，分页游标、API 子调用级局部补偿仍待真实 provider 联调后继续增强。

### 16.123 2026-06-21 V1.2 P1 云账号健康接入同步失败分类第一阶段

状态：已完成云账号健康检查接入 CMDB 同步任务失败分类第一阶段；账号健康不再只看本地凭证、区域和最近同步时间，也会读取最新失败同步任务的 `failureDetails`，将凭证、权限、配置、限流和网络类异常反映到账号健康状态。

已完成：

- 云账号健康判断新增同步失败影响：
  - 健康检查会读取账号最近失败同步任务的 `stats.failureDetails`。
  - 如存在启用同步策略或子周期健康窗口，则按账号、同步策略或子周期范围读取对应的最新失败/成功任务。
  - 仅当最新失败任务晚于最新成功任务时，失败分类才会影响健康状态；后续成功同步会自动清除失败影响。
  - 本地凭证缺失、账号禁用、无支持资源类型等基础校验仍优先，不会被同步任务分类覆盖。
- 失败分类到健康状态映射：
  - `credential`、`permission`、`configuration`：账号健康写为 `unhealthy`。
  - `rate_limit`、`network`、`unknown`：账号健康写为 `warning`。
  - 缺少结构化分类时，会根据错误文本做第一阶段兜底归类。
- 健康详情响应扩展：
  - `healthDetail.failureImpact` 返回 `taskId`、`status`、`category`、`message`、`retryable`、`retryHint` 和 `occurredAt`。
  - `healthMessage` 会带出分类中文标签，例如“权限异常”“API 限流”和建议处理动作。

验证：

- `docker compose up -d iac-portal` 成功，`iac-portal` 健康检查为 `healthy`。
- API 验证权限失败影响：
  - 创建临时腾讯云离线库存账号 `codex-health-impact-20260621`，本地凭证校验返回 `validationStatus=valid`。
  - 插入最新失败 CMDB 同步任务，`failureDetails[0].category=permission`、`retryable=false`。
  - 调用 `POST /api/v1/cloud/accounts/:id/health-check`，返回 `healthStatus=unhealthy`。
  - 响应 `healthDetail.failureImpact.category=permission`、`failureImpact.status=unhealthy`，`healthMessage` 包含“权限异常”。
- API 验证后续成功清除失败影响：
  - 插入时间更新的 `complete` 同步任务后再次健康检查，`healthDetail.failureImpact=null`。
  - 账号不再因旧权限失败保持 `unhealthy`，回到基础同步健康判断。
- API 验证限流失败影响：
  - 插入时间更新的失败任务，`failureDetails[0].category=rate_limit`、`retryable=true`。
  - 再次健康检查返回 `healthStatus=warning`，`failureImpact.status=warning`，`healthMessage` 包含“API 限流”。
- 验证后已清理本轮临时云账号、同步任务、任务日志和事件，剩余临时数据计数均为 `0`。

验证限制：

- 本阶段消费同步任务已采集到的失败分类，用于账号健康推断；不会额外主动请求云厂商 API。
- 失败分类仍以 16.122 的文本归类为主，云厂商原生错误码和 region/assetType 自动局部重试已完成第一阶段，区域级权限矩阵和 API 子调用级局部补偿仍待 provider adapter 继续增强。
- 云账号页面健康依据列已在 16.124 结构化展示 `failureImpact` 字段；本阶段后端仍只负责提供同步失败影响，不主动探测云厂商 API。

### 16.124 2026-06-21 V1.2 P1 云账号健康依据展示同步失败影响第一阶段

状态：已完成云账号页面健康依据展示同步失败影响第一阶段；后端 `healthDetail.failureImpact` 不再只停留在 API 响应里，列表健康依据列会直接展示失败分类，并通过 Tooltip 提供任务、时间、错误、重试建议等排障信息。

已完成：

- 云账号页面健康依据列新增失败影响展示：
  - 当 `healthDetail.failureImpact` 存在时，展示“失败影响：权限异常 / API 限流 / 凭证异常 / 配置异常 / 网络异常 / 未分类异常”。
  - `unhealthy` 级失败影响使用异常色，`warning` 级失败影响使用提醒色。
  - 保留原有同步策略、健康窗口、启用策略数和子周期摘要展示。
- 失败影响 Tooltip 展示结构化排障信息：
  - 同步任务 ID。
  - 失败分类。
  - 发生时间。
  - 错误信息。
  - 是否建议直接重试。
  - 重试建议。

验证：

- `docker compose build iac-web` 成功，前端 webpack 编译通过，仅存在既有 bundle size warning。
- `docker compose up -d iac-web` 成功。
- 创建临时腾讯云离线库存账号 `codex-health-ui-20260621` 并插入 `permission` 失败同步任务。
- 调用账号健康检查后，API 返回 `healthStatus=unhealthy`、`healthDetail.failureImpact.category=permission`。
- 应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-account`：
  - 页面展示临时账号 `codex-health-ui-20260621`。
  - 健康依据列展示“失败影响：权限异常”。
  - 页面未跳转到登录页。

验证限制：

- 本阶段展示账号级命中的失败影响；子周期级失败影响已能通过后端按窗口计算，但前端仍以账号列表当前命中窗口展示为主。
- 从失败影响一键跳转同步任务详情已在 16.125 完成第一阶段。

### 16.125 2026-06-21 V1.2 P1 云账号健康失败影响跳转同步任务详情第一阶段

状态：已完成云账号健康失败影响跳转同步任务详情第一阶段；用户可从云账号页面的“失败影响”直接进入对应云采集任务详情，减少从账号异常到任务日志之间的排障跳转成本。

已完成：

- 云账号页面失败影响文案改为内部链接：
  - 链接格式为 `/org/:orgId/m-cloud-assets?syncTaskId=:taskId`。
  - 文案继续展示“失败影响：权限异常 / API 限流 / ...”。
  - Tooltip 仍保留任务 ID、分类、发生时间、错误信息和重试建议。
- 云资产页面支持 `syncTaskId` 查询参数：
  - 进入页面时自动切换到“云采集”页签。
  - 自动打开“云采集任务详情”抽屉。
  - 复用现有同步任务详情接口和阶段日志/统计展示。

验证：

- `docker compose build iac-web` 成功，前端 webpack 编译通过，仅存在既有 bundle size warning。
- `docker compose up -d iac-web` 成功。
- 创建临时腾讯云离线库存账号 `codex-health-link-20260621` 并插入失败任务 `cst-healthlink-fail-01`。
- 调用账号健康检查后，API 返回 `healthStatus=unhealthy`、`healthDetail.failureImpact.taskId=cst-healthlink-fail-01`。
- 应用内浏览器验证：
  - 云账号页展示 `codex-health-link-20260621`。
  - “失败影响：权限异常”渲染为链接，href 为 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-assets?syncTaskId=cst-healthlink-fail-01`。
  - 打开目标 URL 后，云资产页展示“云采集任务详情”抽屉。
  - 抽屉内展示账号 `codex-health-link-20260621` 和错误 `cam policy denied`。

验证限制：

- 本阶段先支持跳转到云资产页的任务详情抽屉；浏览器返回后的列表筛选状态保持云资产页自身默认行为。
- 云采集任务详情抽屉的任务 ID 显式展示和复制按钮已在 16.126 完成第一阶段。

### 16.126 2026-06-21 V1.2 P1 云采集任务详情任务 ID 复制第一阶段

状态：已完成云采集任务详情任务 ID 显式展示和复制入口第一阶段；从云账号健康失败影响跳转到同步任务详情后，用户可直接复制任务 ID 继续排障、工单沟通或日志检索。

已完成：

- 云资产页面“云采集任务详情”抽屉新增“任务 ID”字段。
- 任务 ID 使用等宽样式展示，便于快速辨识。
- 任务 ID 旁新增“复制”按钮，复用平台已有 `utils/copy` 复制工具和成功/失败通知。
- 保留原有账号、来源、云厂商、状态、区域、资产类型、开始/结束时间、错误日志、阶段日志和统计信息。

验证：

- `docker compose build iac-web` 成功，前端 webpack 编译通过，仅存在既有 bundle size warning。
- `docker compose up -d iac-web` 成功。
- 插入临时云采集失败任务 `cst-healthcopy-fail-01`，账号名 `codex-health-copy-20260621`。
- 应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-assets?syncTaskId=cst-healthcopy-fail-01`：
  - 页面自动展示“云采集任务详情”抽屉。
  - 抽屉内展示“任务 ID”字段。
  - 抽屉内展示 `cst-healthcopy-fail-01`。
  - 抽屉内展示“复制”按钮。
  - 抽屉内仍展示账号 `codex-health-copy-20260621` 和错误 `copy button validation error`。

验证限制：

- 本阶段验证任务 ID 和复制入口渲染；浏览器自动化未点击复制按钮读取系统剪贴板，复制逻辑复用平台已有 `utils/copy`。

### 16.127 2026-06-21 V1.2 P1 云账号健康子周期任务跳转详情第一阶段

状态：已完成云账号健康依据中子周期最近成功/失败任务跳转详情第一阶段；用户可从云账号列表的子周期健康弹层直接进入云采集任务详情抽屉。

已完成：

- 云账号列表“健康依据”列的子周期健康摘要从纯 Tooltip 改为可点击 Popover。
- Popover 展示子周期名称、状态、区域、资源类型、最近成功任务和最近失败任务。
- 最近成功/失败任务时间在存在任务 ID 时渲染为链接，目标为 `/org/:orgId/m-cloud-assets?syncTaskId=:taskId`。
- 复用 16.125 的云资产页 `syncTaskId` 自动打开任务详情抽屉能力，保持任务详情入口一致。

验证：

- `docker compose build iac-web` 成功，前端 webpack 编译通过，仅存在既有 bundle size warning。
- `docker compose up -d iac-web` 成功。
- 创建临时腾讯云离线 inventory 云账号 `codex-schedule-link-20260621` 和同步策略 `csp-schedulelink-01`。
- 构造两个子周期：`广州计算` / `ap-guangzhou` / `compute_instance`，`上海对象存储` / `ap-shanghai` / `object_storage_bucket`。
- 插入并对齐最近成功/失败任务，失败任务 ID 为 `cst-schedulelink-fail`。
- 应用内浏览器验证云账号页展示“子周期：1/2 正常”，点击子周期健康摘要后弹出子周期任务 Popover。
- Popover 内至少展示 3 个 `syncTaskId` 任务链接，其中失败任务链接为 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-assets?syncTaskId=cst-schedulelink-fail`。
- 点击失败任务链接后，云资产页自动打开“云采集任务详情”抽屉，并展示任务 ID `cst-schedulelink-fail` 与错误日志 `schedule failure validation error`。

验证限制：

- 本阶段提供列表健康依据 Popover 中最近一条成功/失败任务入口；更完整的子周期任务历史筛选入口已在 16.128 完成第一阶段。
- 浏览器验证中成功任务可能被后台同步流程更新为更近任务 ID；已验证成功时间同样渲染为 `syncTaskId` 链接，失败任务使用固定临时任务 ID 做端到端点击验证。

### 16.128 2026-06-21 V1.2 P1 云账号健康子周期任务历史第一阶段

状态：已完成云账号健康子周期任务历史筛选第一阶段；用户可从云账号健康依据 Popover 进入云资产页，并查看指定同步策略子周期的任务历史列表。

已完成：

- 云采集任务查询表单新增 `syncPolicyScheduleKey` 过滤条件。
- 云采集任务列表后端支持按 `sync_policy_id` 与 `stats.syncPolicyScheduleKey` 精确过滤任务历史。
- 云账号健康依据 Popover 中每个子周期新增“任务历史”入口。
- “任务历史”链接只携带稳定的 `syncPolicyId` 与 `syncPolicyScheduleKey`，避免中文子周期名称参与 URL 编码。
- 云资产页识别 `syncPolicyId` 与 `syncPolicyScheduleKey` URL 参数，自动切换到“云采集”页签并加载任务历史。
- 云资产页云采集任务列表新增“子周期”列，展示任务 `stats.syncPolicyScheduleName` 和 `stats.syncPolicyScheduleKey`。
- 云资产页展示“任务历史筛选”标签，并提供“清除”入口恢复默认任务列表。
- 云采集任务刷新、运行中轮询和分页切换会保留当前任务历史筛选条件。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 方式执行 `gofmt`，格式化 `backend/portal/models/forms/cmdb.go` 和 `backend/portal/apps/cmdb_sync.go`。
- `docker compose build iac-portal iac-web` 成功，后端 Go 构建通过，前端 webpack 编译通过，仅存在既有 bundle size warning。
- `docker compose up -d iac-portal iac-web` 成功。
- 插入临时云账号 `codex-schedule-history-20260621`、同步策略 `csp-schedulehist-01` 和两个子周期：`history-compute`、`history-cos`。
- 插入三条临时云采集任务，其中 `history-cos` 子周期有 `cst-schedulehist-cos-fail` 与 `cst-schedulehist-cos-success` 两条历史任务。
- API 验证 `/api/v1/cloud/sync-tasks?syncPolicyId=csp-schedulehist-01&syncPolicyScheduleKey=history-cos` 返回 `total=2`，任务 ID 仅包含 `cst-schedulehist-cos-fail` 与 `cst-schedulehist-cos-success`。
- 应用内浏览器验证云账号页子周期 Popover 出现两个“任务历史”链接，href 分别指向 `history-compute` 与 `history-cos`。
- 点击 `history-cos` 任务历史链接后，云资产页自动进入“云采集”页签，展示“任务历史筛选”、策略标签 `csp-schedulehist-01` 和子周期标签 `history-cos`。
- 云采集任务列表显示 `共2条`，未展示 `history-compute` 任务。
- 点击失败行“详情”后，任务详情抽屉展示任务 ID `cst-schedulehist-cos-fail`、错误日志 `schedule history validation error` 和统计中的 `syncPolicyScheduleKey=history-cos`。

验证限制：

- 本阶段提供按同步策略子周期的任务历史筛选入口；子周期一键重跑入口已在 16.129 完成第一阶段，子周期级趋势统计和失败率图表已在 16.130 完成第一阶段。

### 16.129 2026-06-21 V1.2 P1 云账号健康子周期任务历史一键重跑第一阶段

状态：已完成云账号健康子周期任务历史一键重跑第一阶段；用户在指定同步策略子周期的任务历史列表中，可以基于最近任务上下文重跑当前子周期。

已完成：

- 云资产页“云采集”页签在存在 `syncPolicyId` 与 `syncPolicyScheduleKey` 历史筛选时展示“重跑子周期”入口。
- 重跑入口复用当前子周期历史列表中的最近任务作为上下文，自动带入：
  - `accountSource`
  - `accountId`
  - `provider`
  - `regions`
  - `assetTypes`
  - `syncPolicyId`
  - `syncPolicyScheduleKey`
- 当前子周期没有历史任务时禁用重跑入口，避免缺少账号和 scope 上下文时误创建任务。
- 重跑后继续复用云采集任务创建 API；列表刷新、运行中轮询和分页仍保留当前子周期历史筛选条件。
- 重跑创建的新任务会继续写入 `stats.syncPolicyScheduleKey`，并被当前历史筛选列表自动纳入。

验证：

- `docker compose build iac-web` 成功，前端 webpack 编译通过，仅存在既有 bundle size warning。
- `docker compose up -d iac-web` 成功。
- 插入临时腾讯云离线 inventory 云账号 `codex-schedule-rerun-20260621`，同步策略 `csp-schedulererun-01`，子周期 `rerun-cos`。
- 插入历史失败任务 `cst-schedulererun-seed`，任务 `stats.syncPolicyScheduleKey=rerun-cos`。
- 应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-assets?syncPolicyId=csp-schedulererun-01&syncPolicyScheduleKey=rerun-cos`：
  - 页面自动进入“云采集”页签。
  - 展示“任务历史筛选”、策略标签 `csp-schedulererun-01`、子周期标签 `rerun-cos`。
  - 展示“重跑子周期”按钮。
  - 初始列表展示历史失败任务和错误 `seed failure before rerun`。
- 点击“重跑子周期”后，页面提示“采集任务已启动，后台运行中”，当前筛选列表保留并变为 `共2条`。
- 数据库验证新任务 `cst-d8rpjdbpdg6c73djil30`：
  - `status=complete`
  - `sync_policy_id=csp-schedulererun-01`
  - `stats.syncPolicyScheduleKey=rerun-cos`
  - `stats.collected=1`
- 数据库验证本轮离线 inventory 采集到对象存储资产 `cos-rerun-bucket`，并写入 `sync_policy_id=csp-schedulererun-01` 与 `cloud_account_id=cla-schedulererun-01`。

验证限制：

- 本阶段提供“当前子周期重跑”入口；多选历史失败任务批量重跑已在 16.136 完成第一阶段，重跑原因填写与详情留痕已在 16.138 完成第一阶段，重跑前参数编辑已在 16.143 完成第一阶段，审批接入已在 16.146 完成第一阶段。

### 16.130 2026-06-21 V1.2 P1 云账号健康子周期任务历史统计与失败率趋势第一阶段

状态：已完成云账号健康子周期任务历史统计与失败率趋势第一阶段；用户从云账号健康依据进入指定子周期任务历史后，可以直接查看当前子周期的任务总数、成功率、失败率、最近成功/失败时间和近 7 日趋势。

已完成：

- 云采集任务列表接口在原有 `total/pageSize/list` 分页结构上新增 `summary`，保持现有列表消费兼容。
- `summary` 与任务列表复用同一套过滤条件，支持按：
  - `provider`
  - `accountSource`
  - `accountId`
  - `syncPolicyId`
  - `stats.syncPolicyScheduleKey`
  - `status`
  精确统计当前任务历史结果。
- 后端统计字段包括：
  - `totalCount`
  - `completeCount`
  - `failedCount`
  - `runningCount`
  - `pendingCount`
  - `successRate`
  - `failureRate`
  - `lastSuccessAt`
  - `lastFailureAt`
  - `trend`
- 近 7 日趋势按 `created_at` 日期聚合，返回每日总任务数、完成数与失败数；日期使用 `YYYY-MM-DD` 稳定字符串，避免数据库 DATE 类型扫描后格式不一致。
- 云资产页“云采集”页签在任务历史筛选状态下新增统计区：
  - 展示“总任务 / 完成 / 失败 / 运行中 / 成功率 / 失败率 / 最近成功 / 最近失败”。
  - 展示“近7日趋势”小柱状图，使用绿色表示完成任务、红色表示失败任务。
  - 普通未筛选的云采集列表保持原展示，不额外占用页面空间。

验证：

- `docker compose build iac-portal iac-web` 成功，后端 Go 编译和前端 webpack 编译通过；前端仅存在既有 bundle size warning。
- 修正趋势聚合日期格式后，`docker compose build iac-portal` 成功。
- `docker compose up -d iac-portal iac-web` 成功，`/api/v1/check` 返回 `success=true`。
- 插入临时 AWS 云账号 `codex-schedule-stats-20260621`、同步策略 `csp-schedstats01` 和子周期 `stats-cos`。
- 插入 4 条临时任务历史：
  - 2026-06-21：1 条完成、1 条失败。
  - 2026-06-20：1 条完成。
  - 2026-06-19：1 条失败。
- API 验证 `/api/v1/cloud/sync-tasks?syncPolicyId=csp-schedstats01&syncPolicyScheduleKey=stats-cos` 返回：
  - `total=4`
  - `summary.totalCount=4`
  - `summary.completeCount=2`
  - `summary.failedCount=2`
  - `summary.successRate=50`
  - `summary.failureRate=50`
  - `trend[2026-06-19].failedCount=1`
  - `trend[2026-06-20].completeCount=1`
  - `trend[2026-06-21].completeCount=1`
  - `trend[2026-06-21].failedCount=1`
- 应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-assets?syncPolicyId=csp-schedstats01&syncPolicyScheduleKey=stats-cos`：
  - 页面展示“任务历史筛选”、策略标签 `csp-schedstats01` 和子周期标签 `stats-cos`。
  - 统计区展示“总任务 4 / 完成 2 / 失败 2 / 运行中 0 / 成功率 50.0% / 失败率 50.0%”。
  - 页面展示“近7日趋势”和“完成 / 失败 / 总数”。
  - 云采集任务表展示 `共4条`，且 4 条任务均为 `stats-cos` 子周期。

验证限制：

- 本阶段提供当前子周期的近 7 日简洁趋势和失败率；趋势时间范围切换已在 16.131 完成第一阶段，按区域/资产类型拆分已在 16.132 完成第一阶段，失败阈值告警已在 16.133 完成第一阶段，趋势数据导出已在 16.134 完成第一阶段。

### 16.131 2026-06-21 V1.2 P1 云账号健康子周期任务趋势时间范围切换第一阶段

状态：已完成云账号健康子周期任务趋势时间范围切换第一阶段；用户在子周期任务历史统计区可以在 7 天、14 天和 30 天之间切换趋势范围。

已完成：

- 云采集任务列表接口新增 `trendDays` 查询参数：
  - 默认返回 7 天趋势。
  - 支持 `14` 和 `30` 两个显式范围。
  - 非法值自动回落到 7 天，避免任意范围拖大查询。
- 任务历史统计响应新增 `summary.trendDays`，前端可按后端实际采用范围展示标题。
- 近 N 日趋势日期桶按 `trendDays` 动态生成；`summary.totalCount`、成功率和失败率仍统计当前过滤条件下的完整任务历史，趋势图只改变时间范围。
- 云资产页“云采集”页签的任务历史统计区新增范围下拉：
  - 默认显示“7天”。
  - URL 携带 `trendDays=14` 或 `trendDays=30` 时自动使用对应范围。
  - 切换下拉后立即刷新当前子周期任务历史统计。
  - 运行中任务轮询会保留当前趋势范围。

验证：

- 使用 Docker Go 镜像格式化：
  - `backend/portal/apps/cmdb_sync.go`
  - `backend/portal/models/forms/cmdb.go`
  - `backend/portal/models/resps/cmdb.go`
- `docker compose build iac-portal iac-web` 成功，后端 Go 编译和前端 webpack 编译通过；前端仅存在既有 bundle size warning。
- `docker compose up -d iac-portal iac-web` 成功。
- 插入临时 AWS 云账号 `codex-schedule-stats-20260621`、同步策略 `csp-schedstats01` 和子周期 `stats-cos`。
- 插入 5 条临时任务历史，其中 4 条位于 2026-06-19 至 2026-06-21，1 条旧任务位于 2026-06-03。
- API 验证：
  - `trendDays=14` 返回 `summary.trendDays=14`、`trend.length=14`、趋势合计 `4`，不包含 2026-06-03 旧任务。
  - `trendDays=30` 返回 `summary.trendDays=30`、`trend.length=30`、趋势合计 `5`，包含 2026-06-03 旧任务。
  - `trendDays=99` 自动回落到 `summary.trendDays=7`。
- 应用内浏览器验证：
  - 打开带 `trendDays=14` 的云资产任务历史 URL 后，页面展示“近14日趋势”、`14天` 下拉值、`总任务 5` 和任务表 `共5条`。
  - 打开趋势范围下拉后可见 `7天 / 14天 / 30天` 选项。
  - 通过下拉切换到 `30天` 后，页面展示“近30日趋势”、`30天` 下拉值，并出现 2026-06-03 旧任务对应的 `06-03` 趋势日期。

验证限制：

- 本阶段提供固定 7/14/30 天范围；按区域/资产类型拆分已在 16.132 完成第一阶段，失败阈值告警已在 16.133 完成第一阶段，趋势数据导出已在 16.134 完成第一阶段，自定义日期区间已在 16.135 完成第一阶段。

### 16.132 2026-06-21 V1.2 P1 云账号健康子周期任务历史区域/资产类型拆分第一阶段

状态：已完成云账号健康子周期任务历史区域/资产类型拆分第一阶段；用户在指定同步策略子周期任务历史统计区可以查看当前子周期按区域和资产类型拆分的任务数量、采集数量、失败数量和失败率。

已完成：

- 云采集任务列表接口的 `summary` 新增：
  - `regions`
  - `assetTypes`
- 每个拆分项返回：
  - `key`
  - `name`
  - `taskCount`
  - `completeCount`
  - `failedCount`
  - `runningCount`
  - `pendingCount`
  - `collected`
  - `failureRate`
- 拆分统计与任务列表复用同一套过滤条件，支持按 `syncPolicyId` 和 `stats.syncPolicyScheduleKey` 聚合指定子周期历史。
- 后端优先使用任务 `stats.regionMetrics` 与 `stats.assetTypeMetrics` 中的 `collected` 数据统计采集数量；老任务缺少 metrics 时回退到任务级 `regions` 与 `asset_types`，缺失标签统一归入 `unknown`。
- 拆分列表按任务数、失败数和 key 稳定排序，便于优先查看覆盖最多且失败更多的范围。
- 云资产页“云采集”页签的任务历史统计区新增：
  - “按区域拆分”表。
  - “按资产类型拆分”表。
  - 两张表均展示“任务数 / 采集数 / 失败 / 失败率”。
  - 资产类型沿用页面中文映射，例如 `compute_instance` 展示为“计算实例”、`object_storage_bucket` 展示为“对象存储”。

验证：

- 使用 Docker Go 镜像格式化：
  - `backend/portal/apps/cmdb_sync.go`
  - `backend/portal/models/resps/cmdb.go`
- `docker compose build iac-portal iac-web` 成功，后端 Go 编译和前端 webpack 编译通过；前端仅存在既有 bundle size warning。
- `docker compose up -d iac-portal iac-web` 成功，`iac-portal` 健康检查通过，`iac-web` 启动成功。
- 插入临时 AWS 云账号 `codex-schedule-breakdown-20260621`、同步策略 `csp-schedbreakdown01` 和子周期 `breakdown-cos`。
- 插入 3 条临时任务历史：
  - `ap-southeast-1 / object_storage_bucket`：完成，采集 3 个资源。
  - `ap-southeast-1 / compute_instance`：失败，采集 1 个资源。
  - `us-east-1 / compute_instance`：完成，采集 2 个资源。
- API 验证 `/api/v1/cloud/sync-tasks?syncPolicyId=csp-schedbreakdown01&syncPolicyScheduleKey=breakdown-cos&trendDays=14` 返回：
  - `summary.totalCount=3`
  - `summary.completeCount=2`
  - `summary.failedCount=1`
  - `summary.successRate=66.7`
  - `summary.failureRate=33.3`
  - `summary.regions[ap-southeast-1].taskCount=2`
  - `summary.regions[ap-southeast-1].collected=4`
  - `summary.regions[ap-southeast-1].failedCount=1`
  - `summary.regions[ap-southeast-1].failureRate=50`
  - `summary.regions[us-east-1].taskCount=1`
  - `summary.regions[us-east-1].collected=2`
  - `summary.assetTypes[compute_instance].taskCount=2`
  - `summary.assetTypes[compute_instance].collected=3`
  - `summary.assetTypes[compute_instance].failedCount=1`
  - `summary.assetTypes[compute_instance].failureRate=50`
  - `summary.assetTypes[object_storage_bucket].taskCount=1`
  - `summary.assetTypes[object_storage_bucket].collected=3`
- 应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-assets?syncPolicyId=csp-schedbreakdown01&syncPolicyScheduleKey=breakdown-cos&trendDays=14`：
  - 页面展示“按区域拆分”和“按资产类型拆分”。
  - “按区域拆分”表展示 `ap-southeast-1 / 2 / 4 / 1 / 50.0%` 与 `us-east-1 / 1 / 2 / 0 / 0.0%`。
  - “按资产类型拆分”表展示 `计算实例 / 2 / 3 / 1 / 50.0%` 与 `对象存储 / 1 / 3 / 0 / 0.0%`。

验证限制：

- 本阶段提供当前子周期任务历史的区域和资产类型拆分；失败阈值告警已在 16.133 完成第一阶段，趋势数据导出已在 16.134 完成第一阶段，自定义日期区间已在 16.135 完成第一阶段。

### 16.133 2026-06-21 V1.2 P1 云账号健康子周期任务失败阈值告警第一阶段

状态：已完成云账号健康子周期任务失败阈值告警第一阶段；用户在指定同步策略子周期任务历史统计区可以按失败率阈值查看是否触发页面告警。

已完成：

- 云采集任务列表接口新增 `failureThreshold` 查询参数：
  - 默认阈值为 `50%`。
  - 支持 `1-100` 范围内的自定义阈值。
  - 非法值、空值和大于 `100` 的值自动回落到 `50%`。
- 云采集任务列表接口的 `summary` 新增：
  - `failureThreshold`
  - `failureThresholdExceeded`
  - `failureAlertLevel`
  - `failureAlertMessage`
- 阈值判断与任务历史统计复用同一套过滤条件，按当前 `syncPolicyId`、`syncPolicyScheduleKey`、`status` 等筛选结果计算失败率。
- 当当前子周期失败数大于 0 且失败率达到阈值时：
  - `failureThresholdExceeded=true`
  - `failureAlertLevel=warning`
  - 失败率达到 `80%` 及以上时升级为 `error`
  - `failureAlertMessage` 返回“当前子周期失败率 ... 已达到 ... 告警阈值（失败 X / 总数 Y）”
- 当未达到阈值或暂无任务历史时，后端仍返回清晰的说明文案，便于 API 使用方展示或排障。
- 云资产页“云采集”页签的任务历史统计区新增告警阈值选择：
  - 支持 `30% / 50% / 80%` 三档。
  - URL 携带 `failureThreshold=30/50/80` 时自动使用对应阈值。
  - 切换阈值后立即刷新当前子周期任务历史统计。
  - 运行中任务轮询保留当前阈值。
- 当前失败率达到阈值时，前端在统计区展示告警条；未达到阈值时不展示告警条，避免普通历史查看页面被干扰。

验证：

- 使用 Docker Go 镜像通过 stdin 方式校验 `gofmt` 输出一致：
  - `backend/portal/apps/cmdb_sync.go`
  - `backend/portal/models/forms/cmdb.go`
  - `backend/portal/models/resps/cmdb.go`
- `git diff --check` 通过。
- `docker compose build iac-portal iac-web` 成功，后端 Go 编译和前端 webpack 编译通过；前端仅存在既有 bundle size warning。
- `docker compose up -d iac-portal iac-web` 成功，`iac-portal` 健康检查通过，`iac-web` 启动成功。
- 插入临时 AWS 云账号 `codex-schedule-alert-20260621`、同步策略 `csp-schedulealert01` 和子周期 `alert-cos`。
- 插入 4 条临时任务历史，其中 2 条完成、2 条失败，形成 `50%` 失败率。
- API 验证：
  - `failureThreshold=30` 返回 `failureThreshold=30`、`failureThresholdExceeded=true`、`failureAlertLevel=warning`，告警文案包含“已达到 30.0% 告警阈值”。
  - `failureThreshold=80` 返回 `failureThreshold=80`、`failureThresholdExceeded=false`、`failureAlertLevel=info`，告警文案包含“未达到 80.0% 告警阈值”。
  - `failureThreshold=120` 自动回落到 `failureThreshold=50`，并因当前失败率为 `50%` 返回 `failureThresholdExceeded=true`。
- 应用内浏览器验证：
  - 打开带 `failureThreshold=30` 的云资产任务历史 URL 后，页面展示告警条：“当前子周期失败率 50.0%，已达到 30.0% 告警阈值（失败 2 / 总数 4）”。
  - 页面阈值下拉展示 `30%`，趋势范围下拉展示 `14天`。
  - 打开带 `failureThreshold=80` 的云资产任务历史 URL 后，页面不展示告警条，阈值下拉展示 `80%`。

验证限制：

- 本阶段提供查询结果内的阈值判断和页面告警；事件中心记录已在 16.184 完成第一阶段，每个同步策略独立保存阈值和告警静默窗口已在 16.185/16.187 完成第一阶段，负责人分派和通知路由元数据已在 16.189 完成第一阶段，按错误类型/云服务动态路由和失败次数升级已在 16.192 完成第一阶段；通知渠道真实投递和企业级值班升级仍待后续增强。

### 16.134 2026-06-21 V1.2 P1 云账号健康子周期任务趋势数据导出第一阶段

状态：已完成云账号健康子周期任务趋势数据导出第一阶段；用户在指定同步策略子周期任务历史统计区可以导出当前趋势范围内的 CSV 数据。

已完成：

- 云资产页“云采集”页签的任务历史趋势区新增“导出趋势”按钮。
- 导出使用当前页面 `summary.trend` 数据生成 CSV，不额外发起后端写入或异步任务。
- CSV 使用 UTF-8 BOM，兼容常见表格软件直接打开中文列名。
- CSV 字段包括：
  - 日期
  - 完成任务
  - 失败任务
  - 总任务
  - 当日成功率
  - 当日失败率
  - 统计范围
  - 失败阈值
  - 策略 ID
  - 子周期
- 导出文件名包含当前策略 ID、子周期 key 和趋势范围，例如 `cloud-sync-trend-<policy>-<schedule>-7d.csv`。
- 导出内容与当前筛选条件、趋势范围和失败阈值一致：
  - `syncPolicyId`
  - `syncPolicyScheduleKey`
  - `trendDays`
  - `failureThreshold`
- 当当前 summary 没有趋势数据时，前端提示“暂无可导出的趋势数据”，避免生成空文件。

验证：

- `git diff --check` 通过。
- `docker compose build iac-web` 成功，前端 webpack 编译通过；仅存在既有 bundle size warning。
- `docker compose up -d iac-web` 成功，前端容器重建并启动。
- 插入临时 AWS 云账号 `codex-schedule-export-20260621`、同步策略 `csp-scheduleexport01` 和子周期 `export-cos`。
- 插入 4 条临时任务历史：
  - 2026-06-19：1 条完成。
  - 2026-06-20：1 条失败。
  - 2026-06-21：1 条完成、1 条失败。
- API 验证 `/api/v1/cloud/sync-tasks?syncPolicyId=csp-scheduleexport01&syncPolicyScheduleKey=export-cos&trendDays=7&failureThreshold=50` 返回：
  - `summary.totalCount=4`
  - `summary.completeCount=2`
  - `summary.failedCount=2`
  - `summary.failureRate=50`
  - `trend[2026-06-19].completeCount=1`
  - `trend[2026-06-20].failedCount=1`
  - `trend[2026-06-21].completeCount=1`
  - `trend[2026-06-21].failedCount=1`
- 应用内浏览器验证：
  - 打开带 `trendDays=7&failureThreshold=50` 的云资产任务历史 URL 后，页面展示“导出趋势”按钮。
  - 页面展示“近7日趋势”、阈值下拉 `50%` 和趋势范围下拉 `7天`。
  - 趋势区展示 `06-19 1 / 0 / 1`、`06-20 0 / 1 / 1`、`06-21 1 / 1 / 2`。
  - 点击“导出趋势”后页面没有前端错误或通知报错。

验证限制：

- 当前内置浏览器工具未捕获 Blob URL 触发的下载事件，因此本轮验证覆盖按钮渲染、导出源数据、点击无错误和前端生产构建；下载文件内容依据同一 `summary.trend` 数据生成逻辑完成代码审查确认。
- 本阶段导出当前趋势范围内的 CSV；自定义日期区间已在 16.135 完成第一阶段，后端异步导出任务、导出审计和更多文件格式仍待后续增强。

### 16.135 2026-06-21 V1.2 P1 云账号健康子周期任务趋势自定义日期区间第一阶段

状态：已完成云账号健康子周期任务趋势自定义日期区间第一阶段；用户在指定同步策略子周期任务历史统计区可以按自定义起止日期查看趋势数据。

已完成：

- 云采集任务列表接口新增趋势日期范围参数：
  - `trendStartDate`
  - `trendEndDate`
- 日期格式使用 `YYYY-MM-DD`。
- 当 `trendStartDate/trendEndDate` 均合法、结束日期不早于开始日期且范围不超过 90 天时，后端优先使用自定义日期区间。
- 自定义日期区间只影响 `summary.trend` 的日期桶，不改变 `summary.totalCount`、成功率、失败率、区域拆分、资产类型拆分等完整历史统计口径。
- 非法日期、反向日期或超过 90 天的区间自动回落到原有固定范围逻辑，避免拖大查询。
- `summary` 新增：
  - `trendStartDate`
  - `trendEndDate`
  - `trendCustomRange`
- 云资产页“云采集”页签趋势工具栏新增日期范围控件：
  - URL 携带 `trendStartDate=YYYY-MM-DD&trendEndDate=YYYY-MM-DD` 时自动填充日期控件。
  - 选择日期范围后立即刷新当前子周期任务历史统计。
  - 切换 `7天 / 14天 / 30天` 固定范围时自动清除自定义日期区间。
- 趋势标题在固定范围下继续显示“近 N 日趋势”；在自定义范围下显示“趋势：开始日期 至 结束日期”。
- 趋势 CSV 导出会跟随当前自定义日期范围，文件名和 CSV `统计范围` 字段同步展示日期区间。

验证：

- 使用 Docker Go 镜像通过 stdin 方式校验 `gofmt` 输出一致：
  - `backend/portal/apps/cmdb_sync.go`
  - `backend/portal/models/forms/cmdb.go`
  - `backend/portal/models/resps/cmdb.go`
- `git diff --check` 通过。
- `docker compose build iac-portal iac-web` 成功，后端 Go 编译和前端 webpack 编译通过；前端仅存在既有 bundle size warning。
- `docker compose up -d iac-portal iac-web` 成功，`iac-portal` 健康检查通过，`iac-web` 启动成功。
- 插入临时 AWS 云账号 `codex-schedule-custom-20260621`、同步策略 `csp-schedulecustom01` 和子周期 `custom-cos`。
- 插入 4 条临时任务历史：
  - 2026-06-18：1 条完成。
  - 2026-06-19：1 条完成。
  - 2026-06-20：1 条失败。
  - 2026-06-21：1 条完成。
- API 验证合法自定义区间：
  - 请求 `trendStartDate=2026-06-19&trendEndDate=2026-06-20`。
  - 返回 `summary.trendDays=2`。
  - 返回 `summary.trendStartDate=2026-06-19`。
  - 返回 `summary.trendEndDate=2026-06-20`。
  - 返回 `summary.trendCustomRange=true`。
  - 趋势只包含 2026-06-19 和 2026-06-20 两个日期桶。
  - 2026-06-19 为 `completeCount=1`。
  - 2026-06-20 为 `failedCount=1`。
- API 验证非法区间回落：
  - 请求 `trendDays=14&trendStartDate=2026-06-20&trendEndDate=2026-06-19`。
  - 返回 `summary.trendDays=14`。
  - 返回 `summary.trendCustomRange=false`。
  - 返回 `trend.length=14`。
- 应用内浏览器验证：
  - 打开带 `trendStartDate=2026-06-19&trendEndDate=2026-06-20` 的云资产任务历史 URL。
  - 页面标题展示“趋势：2026-06-19 至 2026-06-20”。
  - 日期控件值为 `2026-06-19` 和 `2026-06-20`。
  - 趋势区只展示 `06-19 1 / 0 / 1` 和 `06-20 0 / 1 / 1`。

验证限制：

- 本阶段支持最长 90 天的自定义日期趋势；更长周期归档查询、后端异步趋势报表和保存个人常用日期范围仍待后续增强。

### 16.136 2026-06-21 V1.2 P1 云账号健康子周期失败任务批量重跑第一阶段

状态：已完成云账号健康子周期失败任务批量重跑第一阶段；用户可在指定同步策略子周期任务历史中选择多个失败任务并批量发起重跑。

已完成：

- 云资产页“云采集”页签任务列表支持按行选择失败任务：
  - 仅 `failed` 状态任务允许勾选。
  - `complete/running/pending/canceled` 等非失败任务禁用选择，避免误触发重复采集。
- 任务历史工具栏新增“重跑失败任务”入口：
  - 未选择失败任务时按钮置灰。
  - 选择失败任务后按钮展示数量，例如“重跑失败任务(2)”。
  - 批量触发完成后自动清空选择并刷新任务历史。
- 批量重跑复用现有 `POST /api/v1/cloud/sync-tasks` 创建任务接口，逐条提交被选中失败任务的采集参数。
- 每个重跑任务保留原失败任务的关键范围：
  - 账号来源。
  - 云账号 ID。
  - Provider。
  - 区域。
  - 资产类型。
  - 同步策略 ID。
  - 子周期 key。
  - 子周期名称。
- 当前阶段在前端完成批量编排，不新增后端批量接口，保持与单任务重跑、当前子周期重跑的执行语义一致。

验证：

- `git diff --check` 通过。
- `docker compose build iac-web` 成功，前端 webpack 编译通过；仅存在既有 npm/webpack warning 和 bundle size warning。
- `docker compose up -d iac-web` 成功，前端容器重建并启动。
- 插入临时腾讯云离线 inventory 云账号 `codex-schedule-batch-rerun-20260621`、同步策略 `csp-schedulebatch01` 和子周期 `batch-cos`。
- 插入 3 条临时任务历史：
  - 2 条 `failed` 任务。
  - 1 条 `complete` 任务。
- 应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-assets?syncPolicyId=csp-schedulebatch01&syncPolicyScheduleKey=batch-cos`：
  - 页面展示“重跑子周期”和“重跑失败任务”按钮。
  - 成功任务行选择框为禁用状态。
  - 两条失败任务可勾选。
  - 勾选两条失败任务后按钮展示“重跑失败任务(2)”。
- 点击“重跑失败任务(2)”后：
  - 任务选择被清空。
  - 按钮恢复为“重跑失败任务”。
  - 当前任务历史从 `共3条` 刷新为 `共5条`。
- 数据库验证新增 2 条重跑任务：
  - `status=complete`
  - `sync_policy_id=csp-schedulebatch01`
  - `stats.syncPolicyScheduleKey=batch-cos`
  - `stats.collected=1`

验证限制：

- 本阶段先提供失败任务多选和批量触发能力；重跑原因填写与详情留痕已在 16.138 完成第一阶段，服务端批量重跑接口与任务组留痕已在 16.139 完成第一阶段，批量创建事务保护已在 16.140 完成第一阶段，重跑前参数编辑已在 16.143 完成第一阶段；审批接入已在 16.146 完成第一阶段。

### 16.137 2026-06-21 V1.2 P1 云操作任务详情深链第一阶段

状态：已完成云操作任务详情深链第一阶段；用户可通过 `operationId` URL 参数直达操作任务详情抽屉，方便从资产详情和审计记录追踪操作结果。

已完成：

- “多云管理 - 操作任务”页面支持读取 `operationId` 查询参数：
  - 打开 `/org/:orgId/m-cloud-operations?operationId=<operationId>` 时自动打开操作任务详情抽屉。
  - 详情仍复用既有任务详情 API、步骤和审计展示。
- 操作任务列表点击任务名称时同步写入 `operationId` 查询参数：
  - 当前页面可刷新、复制或分享后继续定位同一任务。
  - 切换到重试任务详情时同步更新为新的重试任务 ID。
- 关闭操作任务详情抽屉时自动清理 `operationId` 查询参数，避免关闭后刷新又重新打开旧详情。
- 云资产详情“最近操作”链接从只跳转操作任务列表增强为直达对应操作任务详情。

验证：

- `git diff --check` 通过。
- `docker compose build iac-web` 成功，前端 webpack 编译通过；仅存在既有 npm/webpack warning 和 bundle size warning。
- `docker compose up -d iac-web` 成功，前端容器重建并启动。
- 使用现有操作任务 `cop-d8r4fovnbvoc73dhp550` 验证深链：
  - 打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-operations?operationId=cop-d8r4fovnbvoc73dhp550`。
  - 登录后自动回到深链 URL。
  - 页面自动打开“操作任务详情”抽屉。
  - 抽屉内容包含目标任务 ID 和动作“更新标签”。
  - 浏览器控制台无 error 日志。
- 点击抽屉关闭按钮后，页面回到 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-operations`，URL 查询参数被清空。
- 从操作任务列表点击“更新标签 - codex-auth-instance”后，URL 自动写入 `?operationId=cop-d8r4fovnbvoc73dhp550` 并打开详情抽屉。

验证限制：

- 本阶段提供前端深链和资产详情跳转增强；跨系统通知中的操作任务链接、外部 ITSM 回链和权限不足时的友好错误页仍待后续增强。

### 16.138 2026-06-21 V1.2 P1 云采集任务重跑原因填写与留痕第一阶段

状态：已完成云采集任务重跑原因填写与留痕第一阶段；用户重跑当前子周期或批量重跑失败任务时必须填写原因，原因会写入新采集任务并在详情中展示。

已完成：

- 云采集任务创建表单新增 `reason` 字段，最大 255 字符。
- 后端创建云采集任务时将原因写入初始 `stats.reason`。
- 后台采集任务运行过程中会在最终 `stats` 中保留 `reason`，避免采集器重建统计对象后丢失原因。
- 任务创建阶段日志和后台运行阶段日志附带 `reason` 字段，便于排查重跑来源。
- 云资产页“云采集”页签增强重跑入口：
  - 点击“重跑子周期”先弹出“重跑当前子周期”原因输入框。
  - 点击“重跑失败任务(N)”先弹出“重跑失败任务”原因输入框。
  - 未填写原因时前端阻止提交并提示“请输入重跑原因”。
  - 提交后按原重跑逻辑创建采集任务，并把同一原因传给每个新任务。
- 云采集任务详情抽屉新增“重跑原因”字段，从 `stats.reason` 直接展示。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 方式执行 `gofmt`，覆盖：
  - `backend/portal/apps/cmdb_sync.go`
  - `backend/portal/models/forms/cmdb.go`
- `git diff --check` 通过。
- `docker compose build iac-portal iac-web` 成功，后端 Go 编译和前端 webpack 编译通过；前端仅存在既有 npm/webpack warning 和 bundle size warning。
- `docker compose up -d iac-portal iac-web` 成功，`iac-portal` 健康检查通过，`iac-web` 启动成功。
- 插入临时腾讯云离线 inventory 云账号 `codex-rerun-reason-20260621`、同步策略 `csp-reason162138` 和子周期 `reason-cos`。
- API 验证：
  - 调用 `POST /api/v1/cloud/sync-tasks`，提交 `reason=codex reason validation 20260621`。
  - 返回创建任务 `cst-d8rr0tohiqic73e5re90`，初始 `stats.reason` 已存在。
  - 任务完成后数据库确认 `status=complete`、`stats.reason=codex reason validation 20260621`、`stats.syncPolicyScheduleKey=reason-cos`、`stats.collected=1`。
- 应用内浏览器验证：
  - 打开带同步策略和子周期参数的云资产页后，页面展示“重跑子周期”和“重跑失败任务”按钮。
  - 点击“重跑子周期”后展示“重跑当前子周期”弹窗。
  - 弹窗包含原因输入框，placeholder 为“请输入重跑原因”，确认按钮为“启动重跑”。
  - 输入 `browser reason validation 20260621` 后提交，页面提示“采集任务已启动，后台运行中”，弹窗关闭。
  - 数据库确认浏览器提交的新任务 `cst-d8rr15ghiqic73e5reeg` 最终 `status=complete`、`stats.reason=browser reason validation 20260621`、`stats.collected=1`。
  - 打开 `syncTaskId=cst-d8rr15ghiqic73e5reeg` 的云采集任务详情，抽屉展示“重跑原因”和该原因值，浏览器控制台无 error 日志。

验证限制：

- 本阶段完成重跑原因填写和任务级留痕；服务端批量重跑接口与任务组留痕已在 16.139 完成第一阶段，批量创建事务保护已在 16.140 完成第一阶段，通知事件联动已在 16.141 完成第一阶段，重跑前参数编辑已在 16.143 完成第一阶段；审批接入已在 16.146 完成第一阶段。

### 16.139 2026-06-21 V1.2 P1 云采集失败任务服务端批量重跑接口与任务组留痕第一阶段

状态：已完成云采集失败任务服务端批量重跑接口与任务组留痕第一阶段；前端批量重跑失败任务不再逐条编排单任务创建，而是调用服务端批量接口统一校验、创建和返回任务组信息。

已完成：

- 新增服务端批量重跑接口：
  - `POST /api/v1/cloud/sync-tasks/rerun-failed`
  - `POST /api/v1/cmdb/sync-tasks/rerun-failed`
- 请求参数：
  - `taskIds`：失败采集任务 ID 列表，最多 50 个，服务端会去重。
  - `reason`：批量重跑原因，必填，最大 255 字符。
- 服务端批量预校验：
  - 所有任务必须属于当前组织。
  - 所有任务必须存在。
  - 所有任务状态必须为 `failed`。
  - 预校验发现错误时不创建新任务，返回 `errors` 说明具体任务和原因。
- 服务端统一生成 `csr...` 批量任务组 ID，并返回：
  - `groupId`
  - `total`
  - `created`
  - `items`
  - `errors`
- 每条新重跑任务沿用源失败任务的账号、云厂商、区域、资产类型、同步策略和子周期信息。
- 新重跑任务 stats 保留批量来源字段：
  - `reason`
  - `rerunGroupId`
  - `rerunFromTaskId`
  - `rerunMode=batch_failed`
- 源失败任务写入批量重跑日志：
  - 成功创建新任务时写入 `rerun_created`。
  - 创建失败时写入 `rerun_failed`。
  - 日志 data 包含 `rerunGroupId`、`rerunTaskId`、`reason` 或错误信息。
- 云资产页“重跑失败任务(N)”改为调用服务端批量接口：
  - 弹窗仍要求填写重跑原因。
  - 成功后展示已启动数量和任务组 ID。
  - 服务端返回部分错误时前端展示 warning 和错误摘要。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 方式执行 `gofmt`，覆盖：
  - `backend/portal/apps/cmdb_sync.go`
  - `backend/portal/models/forms/cmdb.go`
  - `backend/portal/models/resps/cmdb.go`
  - `backend/portal/web/api/v1/handlers/cloud_asset.go`
  - `backend/portal/web/api/v1/handlers/cmdb.go`
- `git diff --check` 通过。
- `docker compose build iac-portal iac-web` 成功，后端 Go 编译和前端 webpack 编译通过；前端仅存在既有 npm/webpack warning 和 bundle size warning。
- `docker compose up -d iac-portal iac-web` 成功，`iac-portal` 健康检查通过，`iac-web` 启动成功。
- `/api/v1/check` 返回 `success=true`。
- 插入临时腾讯云离线 inventory 云账号 `codex-batch-api-20260621`、同步策略 `csp-batchapi162139` 和子周期 `batchapi-cos`。
- 插入 3 条临时任务历史：
  - 2 条 `failed` 任务。
  - 1 条 `complete` 任务。
- API 成功路径验证：
  - 调用 `POST /api/v1/cloud/sync-tasks/rerun-failed`，提交 2 条失败任务和 `reason=codex batch api rerun 20260621`。
  - 返回 `groupId=csr-d8rr8rdmsabc739f60jg`、`total=2`、`created=2`、`errors=[]`。
  - 数据库确认 2 条新任务最终 `status=complete`。
  - 数据库确认新任务 `stats.reason=codex batch api rerun 20260621`。
  - 数据库确认新任务 `stats.rerunGroupId=csr-d8rr8rdmsabc739f60jg`。
  - 数据库确认新任务 `stats.rerunFromTaskId` 分别指向两条源失败任务。
  - 数据库确认新任务 `stats.rerunMode=batch_failed`、`stats.syncPolicyScheduleKey=batchapi-cos`。
  - 数据库确认源失败任务日志写入 `rerun_created`，并包含同一任务组 ID、新任务 ID 和重跑原因。
- API 异常路径验证：
  - 调用批量接口提交 1 条 `complete` 任务。
  - 返回 `created=0`，`errors` 包含“当前状态为 complete，只有失败任务可以批量重跑”。
  - 数据库确认任务总数调用前后不变。
- 应用内浏览器验证：
  - 打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-assets?syncPolicyId=csp-batchapi162139&syncPolicyScheduleKey=batchapi-cos&syncPolicyScheduleName=Batch%20API%20COS`。
  - 页面展示“重跑子周期”和“重跑失败任务”按钮。
  - 成功任务行选择框为禁用状态。
  - 两条失败任务可勾选。
  - 勾选两条失败任务后按钮展示“重跑失败任务(2)”且可点击。
  - 点击后展示“重跑失败任务（2）”弹窗，确认按钮为“启动重跑”。
  - 输入 `browser batch api rerun 20260621` 后提交，弹窗关闭，页面提示包含“已启动”和“任务组”。
  - 数据库确认浏览器提交生成 2 条新任务，最终 `status=complete`，且写入同一个 `rerunGroupId=csr-d8rra7tmsabc739f6130`、`reason=browser batch api rerun 20260621`、`rerunMode=batch_failed`。
  - 浏览器控制台无 error 日志。

验证限制：

- 本阶段完成服务端批量接口、批量预校验、任务组 ID 和源/新任务留痕；批量创建事务保护已在 16.140 完成第一阶段。
- 重跑前参数编辑已在 16.143 完成第一阶段，审批接入已在 16.146 完成第一阶段；通知事件联动已在 16.141 完成第一阶段，批量任务组详情页已在 16.142 完成第一阶段。

### 16.140 2026-06-21 V1.2 P1 云采集失败任务批量重跑事务保护第一阶段

状态：已完成云采集失败任务批量重跑事务保护第一阶段；批量重跑不再出现“部分源任务已创建新任务、部分源任务准备失败”的半批状态。

已完成：

- 批量重跑服务端实现调整为两阶段执行：
  - 准备阶段：对每条源失败任务构造新采集任务、后台启动参数和任务组元数据。
  - 写入阶段：使用数据库事务统一写入新采集任务、新任务创建日志、新任务任务组日志和源任务 `rerun_created` 日志。
  - 启动阶段：事务提交成功后再启动所有后台采集 goroutine。
- 准备阶段任意一条源任务失败时：
  - 整个批次返回 `created=0`。
  - 不创建任何新采集任务。
  - `groupId` 置空，避免调用方误认为任务组已生效。
  - 对失败源任务写入 `rerun_failed` 日志。
- 事务阶段任意写入失败时：
  - 事务整体回滚。
  - 不启动后台采集 goroutine。
  - 返回批量事务失败错误，并对源任务补写 `rerun_failed` 日志。
- 单任务重跑路径保持行为一致：
  - 仍复用同一套任务准备和日志写入 helper。
  - 创建成功后立即启动后台采集任务。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 方式执行 `gofmt`，覆盖 `backend/portal/apps/cmdb_sync.go`。
- `git diff --check` 通过。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`。
- 插入临时腾讯云离线 inventory 云账号 `codex-batch-txn-20260621`、同步策略 `csp-batchtxn162140` 和子周期 `batchtxn-cos`。
- 插入 3 条临时源任务：
  - 2 条有效 `failed` 任务，引用账号 `cla-batchtxn162140`。
  - 1 条 `failed` 任务，故意引用不存在账号 `cla-batchtxn-missing`。
- 准备阶段失败整批不创建验证：
  - 调用 `POST /api/v1/cloud/sync-tasks/rerun-failed`，提交 1 条有效失败任务和 1 条无效账号失败任务。
  - 返回 `groupId=""`、`created=0`。
  - 返回错误 `cloud account cloud_account/cla-batchtxn-missing not found`。
  - 数据库确认同步策略 `csp-batchtxn162140` 下任务总数调用前后均为 `3`，没有半批创建新任务。
- 事务成功路径验证：
  - 调用批量接口提交 2 条有效失败任务和 `reason=codex batch transaction success 20260621`。
  - 返回 `groupId=csr-d8rrdvhld38s73867r80`、`created=2`、`errors=[]`。
  - 数据库确认 2 条新任务最终 `status=complete`。
  - 数据库确认新任务保留 `reason`、`rerunGroupId`、`rerunFromTaskId`、`rerunMode=batch_failed` 和 `syncPolicyScheduleKey=batchtxn-cos`。
  - 数据库确认源失败任务写入 `rerun_created` 日志，并包含同一任务组 ID、新任务 ID 和重跑原因。

验证限制：

- 本阶段完成批量创建阶段的事务保护；后台采集执行仍按每条任务独立运行，采集过程失败不会回滚已经提交的任务创建记录。
- 重跑前参数编辑已在 16.143 完成第一阶段，审批接入已在 16.146 完成第一阶段；通知事件联动已在 16.141 完成第一阶段，批量任务组详情页已在 16.142 完成第一阶段。

### 16.141 2026-06-21 V1.2 P1 云采集失败任务批量重跑通知事件联动第一阶段

状态：已完成云采集失败任务批量重跑通知事件联动第一阶段；批量重跑任务组创建成功后会写入事件中心，并复用现有 CloudEvent Webhook/通知分发链路。

已完成：

- 批量重跑服务端成功创建任务组后写入 CloudEvent：
  - `source=sync`
  - `eventType=cloud.sync.task.batch_rerun_started`
  - `level=info`
  - `status=running`
  - `resourceType=cmdb_sync_task_group`
  - `resourceId=<rerunGroupId>`
- 事件 title 固定为“云采集失败任务批量重跑已启动”。
- 事件 message 包含已创建重跑任务数量。
- 事件 payload 包含：
  - `rerunGroupId`
  - `reason`
  - `mode`
  - `created`
  - `sourceTaskIds`
  - `taskIds`
- 事件只在批量创建事务提交成功后写入；准备阶段失败或事务失败不会产生“已启动”事件。
- 事件写入走 `recordCloudEventBestEffort`，自动复用现有 Webhook 和通知配置中的 `cloud.*` / 精确事件类型订阅。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 方式执行 `gofmt`，覆盖 `backend/portal/apps/cmdb_sync.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`。
- 插入临时腾讯云离线 inventory 云账号 `codex-batch-event-20260621`、同步策略 `csp-batchevent162141` 和子周期 `batchevent-cos`。
- 插入 2 条临时 `failed` 源任务。
- 调用 `POST /api/v1/cloud/sync-tasks/rerun-failed`，提交两条源任务和 `reason=codex batch event validation 20260621`。
- 返回 `groupId=csr-d8rrfpcne0fs73fo9qn0`、`created=2`、`errors=[]`。
- 数据库确认 2 条新任务最终 `status=complete`，并写入同一 `rerunGroupId=csr-d8rrfpcne0fs73fo9qn0`。
- 数据库确认 `iac_cloud_event` 写入 1 条事件：
  - `event_type=cloud.sync.task.batch_rerun_started`
  - `source=sync`
  - `resource_type=cmdb_sync_task_group`
  - `resource_id=csr-d8rrfpcne0fs73fo9qn0`
  - `payload.reason=codex batch event validation 20260621`
  - `payload.created=2`
  - `payload.taskIds` 包含两条新任务 ID。

验证限制：

- 本阶段完成批量重跑启动事件；批量任务组详情页已在 16.142 完成第一阶段，批量任务组完成/失败聚合事件已在 16.144 完成第一阶段，组级通知模板专项变量已在 16.145 完成第一阶段。

### 16.142 2026-06-21 V1.2 P1 云采集失败任务批量重跑任务组详情页第一阶段

状态：已完成云采集失败任务批量重跑任务组详情页第一阶段；批量重跑返回的 `rerunGroupId` 可以在前端打开聚合详情，服务端按任务组汇总源任务、新任务和状态计数。

已完成：

- 新增批量重跑任务组详情 API：
  - `GET /api/v1/cloud/sync-task-rerun-groups/:groupId`
  - `GET /api/v1/cmdb/sync-task-rerun-groups/:groupId`
- 服务端按 `iac_cmdb_sync_task.stats.rerunGroupId` 聚合任务组：
  - 返回 `groupId`、`reason`、`mode`、任务总数和 pending/running/complete/failed 计数。
  - 返回组内新任务 ID 列表、源失败任务 ID 列表和每条新任务的 `sourceTaskId/sourceTaskStatus`。
  - 返回任务组创建时间、最早开始时间和最近结束时间。
  - 任务组不存在或不属于当前组织时返回 404。
- 前端批量重跑成功提示中的任务组 ID 改为可点击入口。
- 云采集任务详情新增重跑链路字段：
  - 重跑任务组，可点击打开任务组抽屉。
  - 源任务，可点击切换查看源失败任务详情。
  - 重跑模式，显示中文模式名。
- 新增“批量重跑任务组”抽屉：
  - 展示任务组 ID、模式、总数、完成/失败/运行中/等待中计数、创建时间、最近结束时间和重跑原因。
  - 展示组内新任务、源失败任务、源状态、新状态、账号、云厂商、子周期和任务时间。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 方式执行 `gofmt`，覆盖本阶段修改的后端 Go 文件。
- `git diff --check` 通过。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose build iac-web` 成功，前端 webpack 编译通过，仅存在既有 bundle size warning。
- `docker compose up -d iac-portal iac-web` 成功：
  - `iac-portal` 状态为 healthy。
  - `iac-web` 状态为 running。
  - `/api/v1/check` 返回 `success=true`。
- 插入临时腾讯云离线 inventory 云账号 `codex-batch-group-20260621`、同步策略 `csp-batchgroup162142` 和子周期 `batchgroup-cos`。
- 插入 2 条临时 `failed` 源任务，并调用 `POST /api/v1/cloud/sync-tasks/rerun-failed`：
  - 返回 `groupId=csr-d8rrn742e0hs73b3395g`。
  - 返回 `created=2`、`errors=[]`。
  - 数据库确认组内新任务最终 `complete=2`。
- 调用 `GET /api/v1/cloud/sync-task-rerun-groups/csr-d8rrn742e0hs73b3395g`：
  - 返回 `total=2`。
  - 返回 `completeCount=2`、`failedCount=0`。
  - 返回 `reason=codex batch group detail validation 20260621`。
  - 返回 2 条任务明细，并包含源失败任务 `cst-batchgroup162142-a`、`cst-batchgroup162142-b`。
- 内置浏览器验证：
  - 打开 `/m-cloud-assets?syncTaskId=cst-d8rrn742e0hs73b33960`。
  - 页面打开“云采集任务详情”，可见 `rerunGroupId=csr-d8rrn742e0hs73b3395g`。
  - 点击任务组 ID 后打开“批量重跑任务组”抽屉。
  - 抽屉可见任务总数、完成数、重跑原因和两条源任务 ID。
  - 当前 `m-cloud-assets` 页面控制台无 error 日志。
- 验证后清理临时云账号、同步策略、同步任务、同步日志、事件和临时资产，剩余计数均为 `0`。

验证限制：

- 本阶段复用任务 `stats.rerunGroupId` 做轻量聚合，尚未新增独立任务组持久化表；重跑前参数编辑已在 16.143 完成第一阶段，任务组完成/失败聚合事件已在 16.144 完成第一阶段，组级通知模板变量已在 16.145 完成第一阶段，审批接入已在 16.146 完成第一阶段。

### 16.143 2026-06-21 V1.2 P1 云采集失败任务批量重跑前参数编辑第一阶段

状态：已完成云采集失败任务批量重跑前参数编辑第一阶段；批量重跑失败任务时可以在提交前覆盖区域和资产类型，留空时继续沿用每条源失败任务的原始参数。

已完成：

- 批量重跑服务端表单新增可选参数：
  - `regions`
  - `assetTypes`
- `POST /api/v1/cloud/sync-tasks/rerun-failed` 和 `POST /api/v1/cmdb/sync-tasks/rerun-failed` 支持上述覆盖参数。
- 批量重跑创建新采集任务时：
  - `regions` 非空时使用覆盖区域。
  - `assetTypes` 非空时使用覆盖资产类型。
  - 覆盖参数为空时继续使用每条源失败任务的 `regions/assetTypes`。
  - 仍保留源任务的云账号、同步策略、子周期、provider、重跑原因和 `rerunGroupId/rerunFromTaskId/rerunMode` 留痕。
- 前端“重跑失败任务”弹窗新增：
  - “覆盖区域”多选/输入控件，候选来自当前选中失败任务区域。
  - “覆盖资产类型”多选/输入控件，候选来自当前选中失败任务资产类型。
- 成功提示、任务组详情和任务详情继续复用 16.139-16.142 的批量接口、任务组和留痕链路。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 方式执行 `gofmt`，覆盖 `backend/portal/apps/cmdb_sync.go` 和 `backend/portal/models/forms/cmdb.go`。
- `git diff --check` 通过。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose build iac-web` 成功，前端 webpack 编译通过，仅存在既有 bundle size warning。
- `docker compose up -d iac-portal iac-web` 成功，`iac-portal` 状态为 healthy，`iac-web` 状态为 running。
- API 覆盖参数验证：
  - 插入临时腾讯云离线 inventory 云账号 `codex-batch-param-20260621`、同步策略 `csp-batchparam162143` 和 2 条失败源任务。
  - 源失败任务原始参数为 `regions=["ap-guangzhou"]`、`assetTypes=["object_storage_bucket"]`。
  - 调用 `POST /api/v1/cloud/sync-tasks/rerun-failed`，提交 `regions=["ap-shanghai"]`、`assetTypes=["compute_instance"]`。
  - 返回 `groupId=csr-d8rrr93drn6s73ds3ql0`、`created=2`、`errors=[]`。
  - 数据库确认两条新任务最终 `complete=2`。
  - 数据库确认两条新任务写入 `regions=["ap-shanghai"]`、`assetTypes=["compute_instance"]`，并保留 `syncPolicyScheduleKey=batchparam-scope`。
  - 验证后清理临时云账号、同步策略、同步任务、同步日志和事件，剩余计数均为 `0`。
- 内置浏览器验证：
  - 插入临时 UI 验证失败任务并打开 `/m-cloud-assets?syncPolicyId=csp-batchparamui162143&syncPolicyScheduleKey=batchparam-ui`。
  - 勾选采集任务表两条失败任务后，按钮显示“重跑失败任务(2)”。
  - 打开“重跑失败任务（2）”弹窗后可见“覆盖区域”“覆盖资产类型”和“请输入重跑原因”。
  - 当前 `m-cloud-assets` 页面控制台无 error 日志。
  - 验证后清理临时 UI 云账号、同步策略和同步任务，剩余计数均为 `0`。

验证限制：

- 本阶段提供区域/资产类型覆盖，不提供云账号、provider、同步策略或子周期改写。
- 覆盖参数已随批量重跑审批在 16.146 接入第一阶段；参数变更差异审计已在 16.147 完成第一阶段。

### 16.144 2026-06-21 V1.2 P1 云采集失败任务批量重跑任务组完成事件第一阶段

状态：已完成云采集失败任务批量重跑任务组完成/失败聚合事件第一阶段；同一 `rerunGroupId` 下所有新任务进入终态后会写入 1 条 `cloud.sync.task.batch_rerun_finished` 事件，用于事件中心、Webhook 和通知策略订阅。

已完成：

- 批量重跑任务完成后触发组级终态检查：
  - 每个重跑任务在后台收尾时按 `stats.rerunGroupId` 查询同组任务。
  - 只要同组仍存在 `pending/running` 任务，就不写组级完成事件。
  - 同组全部进入终态后写入聚合事件。
- 新增组级 finished CloudEvent：
  - `source=sync`
  - `eventType=cloud.sync.task.batch_rerun_finished`
  - `resourceType=cmdb_sync_task_group`
  - `resourceId=<rerunGroupId>`
  - 全部成功时 `status=complete`、`level=info`。
  - 存在失败时 `status=failed`、`level=warning`。
- 事件 payload 包含：
  - `rerunGroupId`
  - `reason`
  - `mode`
  - `total`
  - `pendingCount`
  - `runningCount`
  - `completeCount`
  - `failedCount`
  - `createdAt`
  - `startedAt`
  - `endedAt`
  - `sourceTaskIds`
  - `taskIds`
- 幂等与并发保护：
  - 写事件前先检查同组织、同 `eventType/resourceType/resourceId` 是否已存在。
  - 使用 MySQL `GET_LOCK/RELEASE_LOCK` 对同一任务组加短锁，避免多个后台 goroutine 同时收尾时重复写事件。
  - 锁名使用 `md5(orgId:groupId)` 生成短 key，避免 MySQL 用户锁 64 字符限制。
- 启动事件 payload 补充 `mode=batch_failed`，与任务组详情和 finished 事件保持一致。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 方式执行 `gofmt`，覆盖 `backend/portal/apps/cmdb_sync.go`。
- `git diff --check -- backend/portal/apps/cmdb_sync.go` 通过。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`。
- 首轮验证发现 MySQL 用户锁名超过 64 字符会导致 `GET_LOCK` 失败；修复为 md5 短锁名后重新构建并验证通过。
- 成功组 API 验证：
  - 插入临时腾讯云离线 inventory 云账号 `cla-batchfinish162144`、同步策略 `csp-batchfinish162144` 和 2 条失败源任务。
  - 调用 `POST /api/v1/cloud/sync-tasks/rerun-failed`，提交 `reason=codex batch finish event validation 20260621 success`。
  - 返回 `groupId=csr-d8rs1vgn16fc73cku8jg`、`created=2`、`errors=[]`。
  - 数据库确认组内新任务 `complete=2`、`failed=0`、`running/pending=0`。
  - 数据库确认写入 1 条 `cloud.sync.task.batch_rerun_finished`：`status=complete`、`level=info`、`payload.total=2`、`payload.completeCount=2`、`payload.failedCount=0`。
- 失败组 API 验证：
  - 插入临时缺凭证腾讯云账号 `cla-batchfinishfail162144`、同步策略 `csp-batchfinishfail162144` 和 2 条失败源任务。
  - 调用 `POST /api/v1/cloud/sync-tasks/rerun-failed`，提交 `reason=codex batch finish event validation 20260621 failed`。
  - 返回 `groupId=csr-d8rs29gn16fc73cku8vg`、`created=2`、`errors=[]`。
  - 数据库确认组内新任务 `complete=0`、`failed=2`、`running/pending=0`。
  - 数据库确认写入 1 条 `cloud.sync.task.batch_rerun_finished`：`status=failed`、`level=warning`、`payload.total=2`、`payload.completeCount=0`、`payload.failedCount=2`。

验证限制：

- 本阶段没有新增独立任务组持久化表，仍复用 `stats.rerunGroupId` 聚合。
- 组级通知模板专项变量已在 16.145 完成第一阶段；审批接入已在 16.146 完成第一阶段。

### 16.145 2026-06-21 V1.2 P1 云采集失败任务批量重跑通知模板变量第一阶段

状态：已完成云采集失败任务批量重跑任务组通知模板变量第一阶段；管理员配置 `cloud.sync.task.batch_rerun_started` / `cloud.sync.task.batch_rerun_finished` / `cloud.sync.task.batch_rerun_approval_requested` 通知模板时，可以直接选择任务组级 payload 变量并预览渲染结果。

已完成：

- 通知模板变量字典为批量重跑任务组事件新增固定变量组“批量重跑任务组变量”。
- 支持以下任务组变量：
  - `Payload.rerunGroupId`
  - `Payload.reason`
  - `Payload.mode`
  - `Payload.created`
  - `Payload.total`
  - `Payload.completeCount`
  - `Payload.failedCount`
  - `Payload.approvingCount`
  - `Payload.rejectedCount`
  - `Payload.runningCount`
  - `Payload.pendingCount`
  - `Payload.requiresApproval`
  - `Payload.createdAt`
  - `Payload.startedAt`
  - `Payload.endedAt`
  - `Payload.sourceTaskIds`
  - `Payload.taskIds`
- 通知模板样例数据针对批量重跑事件补充任务组 payload：
  - 启动事件使用 `status=running`，标题为“云采集失败任务批量重跑已启动”。
  - 完成事件使用 `status=complete`，标题为“云采集失败任务批量重跑已完成”。
  - 审批请求事件使用 `status=approving`，标题为“云采集失败任务批量重跑等待审批”。
  - 无真实历史事件时也能提供可预览的任务组样例。
  - 存在真实历史事件时优先使用真实事件字段，并补齐缺失的任务组样例字段。
- 前端事件类型中文枚举补充：
  - `cloud.sync.task.batch_rerun_started`：云采集批量重跑启动
  - `cloud.sync.task.batch_rerun_finished`：云采集批量重跑完成
  - `cloud.sync.task.batch_rerun_approval_requested`：云采集批量重跑待审批
- 事件中心列表的事件类型映射同步补充上述中文名称。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 方式执行 `gofmt`，覆盖 `backend/portal/apps/notification.go`。
- `git diff --check -- backend/portal/apps/notification.go frontend/app/constants/types.js frontend/app/containers/org/cloud-event/index.jsx` 通过。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose build iac-web` 成功，前端 webpack 编译通过，仅存在既有 bundle size warning。
- `docker compose up -d iac-portal iac-web` 成功，`iac-portal` 状态为 healthy，`iac-web` 状态为 running。
- `/api/v1/check` 返回 `success=true`。
- 鉴权后调用 `GET /api/v1/notification-templates/variables?eventType=cloud.sync.task.batch_rerun_finished&type=webhook`：
  - 返回变量组“多云事件变量”和“批量重跑任务组变量”。
  - “批量重跑任务组变量”包含 `Payload.rerunGroupId`、`Payload.total`、`Payload.completeCount`、`Payload.failedCount`、`Payload.taskIds` 等任务组字段。
  - `sampleData.Payload` 返回 `rerunGroupId=csr-sample-rerun-group`、`total=2`、`completeCount=2`、`failedCount=0`。
- 鉴权后调用 `POST /api/v1/notification-templates/preview`：
  - 模板 `重跑组 {{ .Payload.rerunGroupId }}` 渲染为 `重跑组 csr-sample-rerun-group`。
  - 模板 `总数 {{ .Payload.total }} 成功 {{ .Payload.completeCount }} 失败 {{ .Payload.failedCount }}` 渲染为 `总数 2 成功 2 失败 0`。
- 内置浏览器验证：
  - 打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-org-setting`。
  - 切换到“通知” -> “通知模板”，打开“添加模板”抽屉。
  - 事件类型选择“云采集批量重跑完成”后，变量区显示“批量重跑任务组变量”。
  - 可见 `{{.Payload.rerunGroupId}}`、`{{.Payload.completeCount}}`、`{{.Payload.failedCount}}`、`{{.Payload.taskIds}}` 等标签。
  - 当前设置页未产生新的 console error；控制台仅存在此前成本页面的历史接口错误日志。

验证限制：

- 本阶段完成模板变量字典、样例和预览，不自动创建内置通知模板实例。
- 批量重跑审批接入已在 16.146 完成第一阶段；参数差异审计已在 16.147 完成第一阶段，任务组独立持久化仍待后续增强。

### 16.146 2026-06-21 V1.2 P1 云采集失败任务批量重跑审批接入第一阶段

状态：已完成云采集失败任务批量重跑审批接入第一阶段；批量重跑可选择先提交审批，审批通过后启动采集，审批驳回后进入最终驳回态并写入组级事件。

已完成：

- 批量重跑表单新增 `requiresApproval`：
  - `false` 时保持原有直接创建并启动重跑任务。
  - `true` 时新任务进入 `approving`，不会启动后台采集。
- 云采集任务新增状态：
  - `approving`：待审批。
  - `rejected`：已驳回。
- 新增审批 API：
  - `POST /api/v1/cloud/sync-task-rerun-groups/:groupId/approve`
  - `POST /api/v1/cmdb/sync-task-rerun-groups/:groupId/approve`
  - 请求参数 `action=approved|rejected`、`comment`。
- 审批权限第一阶段限定平台管理员或组织管理员。
- 审批请求会写入 `cloud.sync.task.batch_rerun_approval_requested` 事件：
  - `status=approving`
  - `level=info`
  - `resourceType=cmdb_sync_task_group`
  - payload 包含 `requiresApproval=true`、任务组 ID、原因、源任务和新任务列表。
- 审批通过：
  - 更新组内任务 `approval.status=approved`。
  - 将任务从 `approving` 改为 `pending`。
  - 写入 `cloud.sync.task.batch_rerun_started` 事件。
  - 启动后台采集。
  - 采集完成后保留 `stats.approval`，并继续触发 16.144 的组级完成事件。
- 审批驳回：
  - 更新组内任务 `status=rejected`。
  - 写入错误信息“批量重跑审批驳回”。
  - 写入 `cloud.sync.task.batch_rerun_finished`，事件 `status=rejected`、`level=warning`，payload 包含 `rejectedCount`。
- 批量重跑任务组详情扩展：
  - 返回 `approval`。
  - 返回 `approvingCount`、`rejectedCount`。
  - 任务详情展示审批状态和审批备注。
- 前端“重跑失败任务”弹窗新增“提交审批后再启动”开关：
  - 开启后确认按钮显示“提交审批”。
  - 成功提示区分“已提交审批”和“已启动”。
- 前端批量重跑任务组抽屉新增审批操作：
  - 待审批任务组展示“审批通过”和“驳回”按钮。
  - 审批后刷新任务组详情和任务列表。
- 批量重跑启动器调整为批次内顺序执行：
  - 避免同一账号、同一采集范围的多条失败任务审批通过后并发 upsert 同一个 CMDB 资产，触发唯一键冲突。
  - 普通单任务采集仍保持原有异步启动方式。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 方式执行 `gofmt`，覆盖 `backend/portal/apps/cmdb_sync.go`。
- `git diff --check` 通过。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`。
- API 审批通过路径验证：
  - 插入临时腾讯云离线 inventory 云账号和 2 条同账号失败源任务。
  - 调用 `POST /api/v1/cloud/sync-tasks/rerun-failed`，提交 `requiresApproval=true`。
  - 返回 `created=2`，新任务状态为 `approving`。
  - 调用审批通过 API 后，任务先进入 `pending`，再由批次顺序启动器执行。
  - 最终任务组返回 `completeCount=2`、`failedCount=0`、`approval.status=approved`。
  - 数据库确认两条新任务均为 `complete`，且 `stats.approval.status=approved`。
  - 事件中心写入 `cloud.sync.task.batch_rerun_approval_requested`、`cloud.sync.task.batch_rerun_started` 和 `cloud.sync.task.batch_rerun_finished`，终态事件 `status=complete`、`level=info`。
- API 审批驳回路径验证：
  - 插入临时腾讯云离线 inventory 云账号和 1 条失败源任务。
  - 创建审批任务组后调用审批驳回 API。
  - 任务组返回 `rejectedCount=1`、`approval.status=rejected`。
  - 数据库确认新任务 `status=rejected`、`stats.approval.status=rejected`。
  - 事件中心写入 `cloud.sync.task.batch_rerun_finished`，终态事件 `status=rejected`、`level=warning`、`payload.rejectedCount=1`。
- 通知模板变量验证：
  - `cloud.sync.task.batch_rerun_approval_requested` 返回“批量重跑任务组变量”。
  - 变量包含 `Payload.requiresApproval`、`Payload.approvingCount`、`Payload.rejectedCount`。

验证限制：

- 审批权限第一阶段只区分平台管理员/组织管理员，尚未细化到同步策略负责人、云账号授权策略或项目角色。
- 本阶段仍复用 `stats.rerunGroupId` 聚合任务组，尚未新增独立任务组表、审批单表或审批历史时间线。
- 覆盖参数已随审批流进入待审批和审批后执行链路；参数差异审计已在 16.147 完成第一阶段。

### 16.147 2026-06-21 V1.2 P1 云采集失败任务批量重跑参数差异审计第一阶段

状态：已完成云采集失败任务批量重跑参数差异审计第一阶段；批量重跑覆盖区域或资产类型时，新任务和源任务日志都会记录覆盖前后差异，并在任务详情展示。

已完成：

- 批量重跑准备阶段新增参数差异计算：
  - 对比源失败任务 `regions/assetTypes` 与本次实际执行 `regions/assetTypes`。
  - 记录是否显式覆盖、是否发生变化、原值和目标值。
- 新采集任务 `stats.rerunParameterDiffs` 写入：
  - `changed`
  - `regions.from`
  - `regions.to`
  - `regions.overridden`
  - `regions.changed`
  - `assetTypes.from`
  - `assetTypes.to`
  - `assetTypes.overridden`
  - `assetTypes.changed`
- 后台采集完成后保留 `stats.rerunParameterDiffs`：
  - 采集器返回的新 `stats` 不会覆盖掉重跑差异审计字段。
  - 成功、失败和审批通过后的执行路径均复用同一保留逻辑。
- 源失败任务 `rerun_created` 日志新增 `data.parameterDiffs`：
  - 从源任务侧也能追溯“为什么这次重跑使用了不同区域/资产类型”。
- 前端“云采集任务详情”新增“参数差异”：
  - 展示区域和资产类型的原值、目标值。
  - 使用“已覆盖/沿用原值”和“已变化/未变化”标签表达审计结果。
  - 资产类型按现有中文映射展示，例如 `object_storage_bucket` 展示为“对象存储”。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 方式执行 `gofmt`，覆盖 `backend/portal/apps/cmdb_sync.go`。
- `git diff --check` 通过。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose build iac-web` 成功，前端 webpack 编译通过，仅存在既有 bundle size warning。
- `docker compose up -d iac-portal iac-web` 成功，`/api/v1/check` 返回 `success=true`。
- API 验证：
  - 插入临时腾讯云离线 inventory 云账号和 1 条失败源任务。
  - 源失败任务原始参数为 `regions=["ap-guangzhou"]`、`assetTypes=["object_storage_bucket"]`。
  - 调用 `POST /api/v1/cloud/sync-tasks/rerun-failed`，提交 `regions=["ap-shanghai"]`、`assetTypes=["compute_instance"]`。
  - 新任务最终 `status=complete`。
  - 数据库确认新任务 `stats.rerunParameterDiffs.changed=true`。
  - 数据库确认 `regions.from=["ap-guangzhou"]`、`regions.to=["ap-shanghai"]`、`assetTypes.from=["object_storage_bucket"]`、`assetTypes.to=["compute_instance"]`。
  - 数据库确认源任务 `rerun_created` 日志包含同样的 `data.parameterDiffs`。
- 内置浏览器验证：
  - 打开新任务详情 URL。
  - “云采集任务详情”显示“参数差异”。
  - 可见“区域 已变化 已覆盖 ap-guangzhou -> ap-shanghai”。
  - 可见“资产类型 已变化 已覆盖 对象存储 -> 计算实例”。

验证限制：

- 本阶段完成任务级差异审计，尚未新增独立任务组表或参数差异查询 API。
- 差异字段只覆盖批量重跑支持覆盖的区域和资产类型；云账号、provider、同步策略和子周期仍不允许在批量重跑中改写。

### 16.148 2026-06-21 V1.2 P1 多云总览 IaC 治理视图优化第一阶段

状态：已完成多云总览 IaC 治理视图优化第一阶段；总览页不再只展示粗粒度 IaC 纳管数，而是同时呈现直接 IaC 资源、云采集已关联 IaC、项目/环境绑定和云上未纳管资产。

已完成：

- 后端总览指标扩展：
  - `iacDirectAssets`：直接来自 `iac_resource` 的 CMDB 资产。
  - `iacLinkedAssets`：云采集或其他来源资产中已关联 `iac_resource_id` 的资产。
  - `iacManagedAssets`：直接 IaC 资产和已关联 IaC 资产的合并覆盖数。
  - `cloudLinkedAssets`：云采集资产中已绑定项目、环境或 IaC 资源的资产。
  - `governanceRate`：已治理资产占比，按非云上未纳管资产计算。
  - `cloudOnlyRate`：云上未纳管资产占比。
- Provider 覆盖表扩展 `iacManagedAssetCount` 和 `cloudLinkedAssetCount`，用于按云厂商观察 IaC 覆盖和治理关联情况。
- 总览待办新增 `iacCoverage`：当 IaC 覆盖率低于 80% 时提示“提升 IaC 覆盖率”，并下钻到云资产治理页面。
- 前端总览页优化：
  - CMDB 资产卡片展示“云采集 X，IaC覆盖 Y”。
  - 新增“IaC 覆盖”指标卡。
  - 进度区新增 IaC 覆盖率、治理关联率、云上未纳管率。
  - 新增“IaC 治理视图”区块，展示“直接来自 IaC”“云采集已关联 IaC”“云采集已绑定项目/环境”“云上未纳管”四类资产。
  - Provider 表新增“IaC覆盖”列。
  - 补充腾讯云、华为云 provider 中文名，以及 `approving`、`rejected` 任务状态中文文案。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_overview.go` 和 `backend/portal/models/resps/cloud_overview.go`。
- `git diff --check -- backend/portal/apps/cloud_overview.go backend/portal/models/resps/cloud_overview.go frontend/app/containers/org/cloud-overview/index.jsx frontend/app/containers/org/cloud-overview/styles.less` 通过。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功，`/api/v1/check` 返回 `success=true`。
- 鉴权调用 `GET /api/v1/cloud/overview` 返回新增指标：
  - `iacDirectAssets`、`iacLinkedAssets`、`cloudLinkedAssets`、`governanceRate`、`cloudOnlyRate`。
  - Provider 行返回 `iacManagedAssetCount` 和 `cloudLinkedAssetCount`。
  - 当前验证组织在 2 个云采集资产、0 个 IaC 关联资产场景下返回 `coverageRate=0`、`cloudOnlyRate=100`，并生成 `iacCoverage` 待办。
- 内置浏览器验证 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-overview`：
  - 页面展示“IaC 治理视图”“IaC 覆盖”“治理关联率”“云上未纳管率”和 Provider 表“IaC覆盖”列。
  - 页面展示“提升 IaC 覆盖率”待办。
  - 二次刷新后当前总览页无新增 console error。

验证限制：

- 本阶段先完成组织级只读治理视图，尚未提供独立的 IaC 覆盖修复向导、漂移时间线或自动关联推荐确认流。
- 覆盖率依赖 CMDB 资产中的 `source`、`iac_resource_id`、`project_id`、`env_id` 回填完整度；历史数据仍需通过 IaC 回填、云采集关联和批量治理逐步修正。
- 云采集资产与 IaC 资源的更强匹配规则仍可继续扩展 provider/account/region/nativeId/address 多字段匹配和人工确认闭环。

待继续：

- 在云资产详情中补充 IaC 覆盖来源解释、关联修复入口和推荐匹配确认。
- 将 IaC 覆盖率接入风险中心和通知策略，对长期未纳管或高风险未纳管资产产生治理事件。
- 按 provider、账号、区域和资源类型继续扩展 IaC 覆盖趋势与治理进度。

### 16.149 2026-06-21 V1.2 P0 插队需求：ITSM 自助运维与 GitOps/IaC 目标治理第一阶段

状态：已完成公司插队需求第一阶段；ITSM 页面从“工单台账”升级为“自助运维治理台”，可量化展示 80% 自助覆盖、60% 工单自动化处理率、100% IaC 代码化覆盖和环境漂移治理状态，并支持自助提交权限申请、GitOps/IaC 变更申请和漂移修复申请。

背景目标：

- GitOps/IaC 全面落地：
  - 基础设施 100% 代码化，所有变更通过 PR review 和自动化流水线执行。
  - dev/staging/prod 通过 IaC 做配置漂移检测和自动修复。
- 自助化平台建设：
  - 业务团队可自助完成 80% 以上常规运维操作，包括扩缩容、配置变更、权限申请。
  - 运维工单自动化处理率目标大于 60%。

已完成：

- 后端新增 ITSM 目标治理概览 API：
  - `GET /api/v1/cloud/itsm/overview`
  - 返回工单总数、自动化处理工单数、工单自动化率和 60% 目标值。
  - 返回自助目录总数、已上架目录数、自助覆盖率和 80% 目标值。
  - 返回 CMDB 资产数、IaC 覆盖资产数、IaC 代码化覆盖率、云上未纳管资产数和 100% 目标值。
  - 返回环境总数、开启漂移检测环境数、开启自动修复环境数，以及 GitOps/IaC 门禁状态和漂移治理状态。
- 后端新增自助运维申请 API：
  - `POST /api/v1/cloud/itsm/self-service-tickets`
  - 支持 `permission_request`、`gitops_iac_change`、`drift_remediation` 三类申请。
  - 提交时先创建 `CloudOperationTypeSelfService` 自助操作任务，再复用 ITSM 连接器创建本地或外部工单。
  - 自助操作任务会记录 `selfService`、`requestType`、`automationMode`、`catalogKey` 和补充参数，形成审计链路。
- 后端新增内置本地 ITSM 连接器兜底：
  - 当组织没有启用的外部 ITSM 连接器时，自动创建或启用 `CloudIaC 本地工单`。
  - 自助申请未传 `connectorId` 时自动使用本地工单连接器，避免首次使用平台时连接器下拉为空导致无法提交。
  - 本地连接器不配置外部 `baseUrl`，提交后保留为平台内待提交工单，同时写入事件和操作审计链路。
- 云操作动作目录进入 ITSM 自助目录：
  - 生命周期：启动实例、停止实例、重启实例。
  - 扩缩容：调整实例规格、磁盘扩容。
  - 配置变更：更新标签、更新安全组规则。
  - 备份：创建快照/备份。
  - 高危删除资源保留在目录但标记未开放。
- 前端 ITSM 页面增强：
  - 顶部新增“自助运维覆盖”“工单自动化处理”“IaC 代码化覆盖”目标卡片。
  - 新增 GitOps/IaC 变更门禁和环境一致性保障提示。
  - 工单页新增“发起自助申请”按钮。
  - 新增“自助目录”页签，展示服务项、类别、处理方式、来源、风险和上架状态。
  - 自助申请弹窗支持选择申请类型、可选 ITSM 连接器、优先级、项目 ID、环境 ID、申请说明和补充参数 JSON。
  - 自助申请弹窗打开时会刷新连接器并默认选中启用连接器；若请求尚未返回、没有启用外部连接器或用户清空连接器，后端仍可使用本地工单兜底。

验证：

- 使用 Docker Compose 构建 `iac-portal` 成功，后端 Go 编译通过。
- 使用 Docker Compose 构建 `iac-web` 成功，前端生产构建通过，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功，`iac-portal` healthy，`iac-web` 正常启动，`/api/v1/check` 返回 `success=true`。
- 鉴权调用 `GET /api/v1/cloud/itsm/overview`：
  - 返回 `selfServiceCatalogTotal=13`、`selfServiceAvailableTotal=12`、`selfServiceCoverageRate=92.3076923076923`。
  - 返回三类自助申请：`permission_request`、`gitops_iac_change`、`drift_remediation`。
  - 当前验证组织资产数为 2、IaC 覆盖资产为 0，`gitOpsGuardStatus=warning`，符合现有测试数据。
- 调用 `GET /api/v1/cloud/itsm/configs` 后可自动生成 `CloudIaC 本地工单`，连接器下拉不再为空。
- 不传 `connectorId` 调用 `POST /api/v1/cloud/itsm/self-service-tickets`：
  - 使用 `requestType=gitops_iac_change` 成功生成自助操作任务和本地 ITSM 工单。
  - 响应中 `connectorName=CloudIaC 本地工单`、`operationType=self_service`、工单 `status=pending`。
  - 验证后仅清理测试工单、操作任务、审计和事件数据，保留内置本地连接器作为平台兜底配置。
- 内置浏览器验证 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-itsm`：
  - 页面展示“自助运维覆盖”“工单自动化处理”“IaC 代码化覆盖”“GitOps/IaC 变更门禁”“环境一致性保障”“自助目录”“发起自助申请”。
  - 点击“发起自助申请”后弹窗展示“申请类型”“ITSM 连接器”“申请说明”“补充参数 JSON”，取消关闭未提交变更。
  - 二次刷新当前 ITSM 页无新增 console error。

验证限制：

- 本阶段先把公司目标落成平台可度量、可申请、可审计的闭环；GitLab/GitHub PR review 与外部流水线的真实状态回写仍需后续对接 VCS/CI API。
- “IaC 代码化覆盖率”以 CMDB 资产 `source=iac_resource` 或 `iac_resource_id` 关联为准；历史资产需要继续通过 IaC 回填和资产治理补齐。
- “工单自动化处理率”已在 16.208 优先合并 `robotProcessed=true` 的机器人处理标签，并兼容本平台工单提交/处理中/已解决/关闭且有提交痕迹的历史口径；外部 ITSM 主动状态回调已在 16.197 完成第一阶段，外部状态周期拉取已在 16.198 完成第一阶段，漂移自动修复审批通知/SLA 升级已在 16.207 完成第一阶段，外部 ITSM 失败补偿队列已在 16.213 完成第一阶段。

待继续：

- GitOps/IaC 申请侧 PR review 和自动化流水线门禁已在 16.194 完成第一阶段，外部 callback 回写已在 16.214 完成第一阶段；继续对接 GitLab/GitHub/Jenkins 主动拉取、GitOps Repo diff 记录和更完整审批明细。
- 风险/漂移到 ITSM 自助整改工单联动已在 16.195 完成第一阶段，ITSM 状态同步风险整改已在 16.196 完成第一阶段，外部 ITSM 主动回调和签名校验已在 16.197 完成第一阶段，外部状态周期拉取已在 16.198 完成第一阶段，漂移风险整改触发自动修复已在 16.199 完成第一阶段，漂移自动修复任务结果回写风险已在 16.200 完成第一阶段，风险详情页漂移自动修复入口已在 16.201 完成第一阶段，漂移自动修复失败后重试审批已在 16.202 完成第一阶段，漂移自动修复回滚策略记录已在 16.203 完成第一阶段，漂移自动修复审批策略配置已在 16.204 完成第一阶段，漂移自动修复完整任务时间线已在 16.205 完成第一阶段，漂移自动修复自动关联推荐已在 16.206 完成第一阶段，漂移自动修复审批通知/SLA 升级已在 16.207 完成第一阶段，机器人处理标签已在 16.208 完成第一阶段，推荐采纳确认流已在 16.209 完成第一阶段，自助目录权限策略和组织级 SLA 趋势已在 16.210 完成第一阶段，项目/申请类型维度目标趋势已在 16.211 完成第一阶段，外部 ITSM 专用字段映射已在 16.212 完成第一阶段，外部 ITSM 失败补偿队列已在 16.213 完成第一阶段。
- 继续增强自助目录策略可配置能力、真实组织架构团队维度目标趋势、外部 ITSM 独立队列报表和死信重放页面。

### 16.150 2026-06-21 V1.2 P1 EKS/OKE Kubernetes 集群信息展示第一阶段

状态：已完成 EKS/OKE 相关 K8S 信息第一阶段；Kubernetes 集群不再只作为普通 CMDB 资产展示，而是在云资产详情中提供专门的 K8S 信息页签。

已完成：

- AWS EKS collector 增强：
  - 保留已有 EKS Cluster 采集。
  - 新增 EKS NodeGroup 列表与详情采集，写入 `attributes.nodeGroups`。
  - 记录节点组名称、状态、K8S 版本、Release Version、容量类型、实例规格、子网、AMI 类型、节点角色、伸缩配置、标签和污点。
  - 节点组权限不足或 API 失败时不阻断集群采集，写入 `attributes.nodeGroupCollectError`。
- OCI OKE collector 增强：
  - 保留已有 OKE Cluster 采集。
  - 新增 OKE NodePool 列表采集，写入 `attributes.nodePools`。
  - 记录节点池名称、状态、K8S 版本、节点规格、节点配置、镜像来源、Subnet/AD 信息和创建更新时间。
  - 节点池权限不足或 API 失败时不阻断集群采集，写入 `attributes.nodePoolCollectError`。
- 云资产详情页增强：
  - 当资产类型为 `kubernetes_cluster` 时新增“K8S信息”页签。
  - 展示集群名称、K8S 版本、平台版本、状态、API Endpoint、VPC/VCN、Endpoint 配置、子网、安全组/NSG 和网络配置。
  - 展示 EKS NodeGroup / OKE NodePool 表格，包括类型、名称、状态、版本、规格、规模和子网。
  - 节点组/节点池采集失败时在页签内展示告警，不再让用户只能展开原始 JSON 排查。

验证：

- 使用 Docker Compose 构建 `iac-portal` 成功，后端 Go 编译通过。
- 使用 Docker Compose 构建 `iac-web` 成功，前端生产构建通过，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功，`iac-portal` healthy，`/api/v1/check` 返回 `success=true`。

待继续：

- 在真实 EKS/OKE 账号上补充端到端采集验证，确认 NodeGroup/NodePool API 权限、分页和错误映射。
- 在真实 GKE/AKS 账号上补充端到端采集验证，确认 NodePool 字段差异、权限、分页和错误映射。
- 后续如进入多集群管理，需要继续扩展 Namespace、Node、Pod、Workload、Service/Ingress、事件和 kubeconfig/Agent 接入。

### 16.111 2026-06-21 V1.2 P1 云账号健康检查接入同步策略阈值

状态：已完成云账号健康检查按启用同步策略判断最近同步是否过期；无启用策略时继续保留 24 小时兜底。

已完成：

- 云账号健康检查从固定 24 小时同步阈值扩展为策略感知阈值：
  - 查询当前账号下启用状态的 `iac_cloud_sync_policy`。
  - 以最短 `syncInterval` 作为账号同步健康基线。
  - 额外增加容忍窗口：`syncInterval / 5`，最少 5 分钟，最多 1 小时。
  - 超过阈值时健康状态写为 `warning`，健康消息包含策略名称、健康阈值和最近同步时间。
- 没有启用同步策略时，仍使用原有 24 小时阈值，避免老账号健康检查行为突变。
- 云账号列表“健康”列新增 Tooltip，悬停可查看 `healthMessage`，同步策略超时原因不再只停留在 API 响应里。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_account.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`。
- API 验证同步策略阈值：
  - 创建临时腾讯云离线 inventory 云账号。
  - 创建启用同步策略，`syncInterval=60` 秒。
  - 将账号 `last_sync_at` 设置为 10 分钟前。
  - 调用 `POST /api/v1/cloud/accounts/:id/health-check` 返回 `healthStatus=warning`。
  - `healthMessage` 返回“账号最近同步时间超过同步策略 ... 的健康阈值 6 分钟...”，确认不再使用固定 24 小时提示。
  - 验证后已清理本轮临时云账号、同步策略和健康检查事件，剩余临时数据计数均为 `0`。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size warning。
- `docker compose up -d iac-web` 成功。

验证限制：

- 当前仍是 Portal 本地健康判断；同步失败分类已在 16.123 接入账号健康推断，主动调用真实云厂商 API 检查凭证过期、API 限流或细粒度权限漂移仍待 adapter 增强。
- 策略健康窗口在本阶段按账号启用策略中的最短同步间隔计算；资源类型、区域和子周期最近同步状态细分已在 16.119 完成第一阶段。

待继续：

- 后台周期健康检查已在 16.112 完成第一阶段。
- 账号健康详情中的命中同步策略、健康窗口、最近成功任务和失败任务摘要已在 16.113 完成第一阶段。
- 继续接入真实云 provider 权限漂移、凭证过期和 API 限流检测。

### 16.112 2026-06-21 V1.2 P1 云账号健康检查后台调度第一阶段

状态：已完成云账号健康检查后台 worker 第一阶段，Portal 启动后会周期性刷新所有组织云账号健康状态。

已完成：

- 新增 `StartCloudAccountHealthWorker(serviceId)`：
  - Portal 启动时自动拉起后台协程。
  - 启动后立即执行一次全量检查，之后每 30 分钟执行一次。
  - 复用现有云账号健康检查逻辑、同步策略阈值判断和事件写入路径。
- 新增 `CheckCloudAccountsHealthForAllOrgs()`：
  - 从 `iac_cloud_account` 扫描所有云账号。
  - 按账号 `orgId` 构造系统上下文，逐个调用既有健康检查逻辑。
  - 汇总 total、checked、healthy、warning、unhealthy 统计并写入 worker 日志。
- Portal 启动流程接入账号健康检查 worker，与预算、成本计划、Webhook 队列等后台任务一起运行。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_account.go` 和 `backend/cmds/portal/main.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`iac-portal` 状态为 `healthy`。
- `GET /api/v1/check` 返回 `success=true`。
- 查看 `iac-portal` 近 3 分钟日志，确认后台 worker 已执行：
  - `worker=cloudAccountHealth`。
  - `interval=30m0s`。
  - `check cloud account health result: total=1 checked=1 healthy=0 warning=1 unhealthy=0`。

验证限制：

- 周期配置化已在 16.114 完成第一阶段。
- 组织级分布式锁已在 16.115 完成第一阶段；账号级分片和并发度控制已在 16.121 完成后端实现。
- 后台重复健康事件降噪已在 16.114 完成第一阶段。
- 真实云 API 权限漂移、凭证过期和 API 限流识别仍待 provider adapter 增强。

### 16.113 2026-06-21 V1.2 P1 云账号健康详情可观测性第一阶段

状态：已完成云账号健康详情结构化响应和云账号页面健康依据/同步摘要展示第一阶段。

已完成：

- 云账号列表、详情和健康检查响应新增 `healthDetail`：
  - `healthWindowSeconds`、`healthWindowText`：当前账号健康判断窗口。
  - `enabledPolicyCount`：当前账号启用同步策略数量。
  - `policy`：命中的同步策略，按启用策略中最短 `syncInterval` 选取，返回策略 ID、名称、同步间隔、健康窗口、最近同步任务和最近同步状态。
  - `lastSuccessTask`：最近一次成功 CMDB 同步任务摘要。
  - `lastFailureTask`：最近一次失败 CMDB 同步任务摘要。
- 最近同步任务摘要从 `iac_cmdb_sync_task` 按统一云账号、provider 和任务状态读取，返回任务 ID、同步策略 ID、状态、错误信息、统计、开始/结束/创建时间。
- 云账号页面新增“健康依据”和“同步摘要”列：
  - “健康依据”展示命中的同步策略、健康窗口和启用策略数。
  - “同步摘要”展示最近成功/失败同步时间，失败任务使用醒目颜色提示。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_account.go` 和 `backend/portal/models/resps/cloud_account.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`iac-portal` 状态为 `healthy`，`GET /api/v1/check` 返回 `success=true`。
- API 端到端验证：
  - 创建临时腾讯云离线 inventory 云账号。
  - 创建启用同步策略，`syncInterval=60` 秒。
  - 写入最近成功和最近失败 CMDB 同步任务。
  - 调用 `POST /api/v1/cloud/accounts/:id/health-check` 和 `GET /api/v1/cloud/accounts?q=...`。
  - 返回 `healthDetail.policy.name` 为本轮临时策略名。
  - 返回 `healthDetail.healthWindowSeconds=360`。
  - 返回 `lastSuccessTask.id` 和 `lastFailureTask.id` 分别匹配本轮写入的成功/失败任务。
  - 验证后已清理本轮临时云账号、同步策略和同步任务，剩余临时数据计数均为 `0`。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size warning。
- `docker compose up -d iac-web` 成功，首页返回 `HTTP/1.1 200 OK`。
- 应用内浏览器打开 `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-account`：
  - 页面展示“健康依据”列。
  - 页面展示“同步摘要”列。
  - 页面展示“同步策略”和“健康检查”入口。
  - 浏览器控制台无 error 级日志。

验证限制：

- 最近成功/失败任务的列表级“同步摘要”当前按账号级别汇总；资源类型、区域和同步策略子周期最近任务证据已在 16.120 接入健康依据 Tooltip。
- 健康详情仍基于本地同步任务和本地凭证校验；同步失败分类影响已在 16.123 接入，真实云 provider 主动权限漂移、凭证过期和 API 限流检测仍待 adapter 增强。

### 16.114 2026-06-21 V1.2 P1 云账号健康检查后台调度配置化与事件降噪第一阶段

状态：已完成云账号健康检查后台 worker 周期配置化和后台重复健康事件降噪第一阶段。

已完成：

- 新增环境变量：
  - `CLOUDIAC_CLOUD_ACCOUNT_HEALTH_WORKER_INTERVAL_SECONDS`：控制云账号健康检查后台 worker 周期，默认 1800 秒，最小 60 秒，最大 86400 秒。
  - `CLOUDIAC_CLOUD_ACCOUNT_HEALTH_EVENT_QUIET_HOURS`：控制后台健康事件静默窗口，默认 24 小时，最小 1 小时，最大 720 小时。
- `StartCloudAccountHealthWorker` 从配置化函数读取周期，日志继续输出实际 `interval`，便于运行期排障。
- 手动健康检查保持原行为，仍会强制写入 `cloud_account.health_checked` 事件。
- 后台 worker 健康检查增加事件降噪：
  - 健康状态或健康消息变化时写事件。
  - 上次检查为空时写事件。
  - 状态和消息未变化且未超过静默窗口时不重复写事件。
  - 超过静默窗口后再次写事件，保留长期巡检可见性。
- `backend/configs/dotenv.sample` 新增上述环境变量说明。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_account.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`iac-portal` 状态为 `healthy`。
- `GET /api/v1/check` 返回 `success=true`。
- 查看 `iac-portal` 日志确认后台 worker 已执行：
  - `worker=cloudAccountHealth`。
  - `interval=30m0s`。
  - `check cloud account health result: total=1 checked=1 healthy=0 warning=1 unhealthy=0`。
- 数据库验证后台重复事件降噪：
  - 重启 Portal 前 `cloud_account.health_checked` 事件数为 `2`。
  - 后台 worker 首次执行后事件数仍为 `2`。
  - 说明健康状态和消息未变化时，后台周期检查没有重复写入健康事件。

验证限制：

- 多实例锁保护已在 16.115 完成第一阶段；账号级分片和并发度控制已在 16.121 完成后端实现。
- 事件静默窗口为全局环境变量，尚未支持组织级或账号级差异化配置。
- 真实云 API 权限漂移、凭证过期和 API 限流识别仍待 provider adapter 增强。

### 16.115 2026-06-21 V1.2 P1 云账号健康检查 worker 多实例锁保护第一阶段

状态：已完成云账号健康检查后台 worker 多实例 MySQL 锁保护第一阶段，避免多 Portal 实例同时执行全量账号健康扫描。

已完成：

- 新增 `CheckCloudAccountsHealthForAllOrgsWithLock()`：
  - 复用现有 MySQL advisory lock 工具 `cloudWebhookAcquireMysqlLock`。
  - 使用全局锁名 `cloudiac:cloud_account_health:all`。
  - 获取锁成功后执行全量账号健康检查。
  - 获取锁失败时返回跳过，不写健康事件、不更新账号健康检查时间。
- `StartCloudAccountHealthWorker` 接入锁保护：
  - 锁被其它实例持有时输出 `check cloud account health skipped: lock is held by another portal`。
  - 锁获取成功时继续输出健康检查汇总。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_account.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- 手动通过 MySQL 持有 `cloudiac:cloud_account_health:all` 锁后重启 Portal：
  - `iac-portal` 状态为 `healthy`。
  - `GET /api/v1/check` 返回 `success=true`。
  - 日志输出 `check cloud account health skipped: lock is held by another portal`。
- 释放手动锁后重启 Portal：
  - `iac-portal` 状态为 `healthy`。
  - `GET /api/v1/check` 返回 `success=true`。
  - 日志输出 `check cloud account health result: total=1 checked=1 healthy=0 warning=1 unhealthy=0`。

验证限制：

- 账号级分片、并发度控制和 per-shard 锁已在 16.121 完成第一阶段；当前仍未按组织单独配置健康检查调度周期。
- 锁状态已进入健康检查汇总响应和 worker 日志；尚未提供独立 worker 指标页面。

### 16.104 2026-06-21 V1.2 P1/P2 AWS/OCI 成本导出 JSON URL 第一阶段

状态：已完成 AWS CUR、AWS Cost Explorer 与 OCI Usage/Cost 成本导出 JSON URL 接入第一阶段；成本中心从 Azure/GCP/腾讯云/华为云继续扩展到 AWS/OCI，可通过统一同步任务和同步计划导入标准成本明细。

已完成：

- 后端成本来源扩展：
  - 新增 `aws_cur_export`、`aws_cost_explorer_export` 和 `oci_usage_cost_export` 成本来源常量。
  - 成本导入、拉取、同步任务和同步计划表单枚举均允许 `provider=aws/oci`。
  - `cloudCostImportSource` 与 `cloudCostPullSource` 可按 provider 推导 AWS CUR 或 OCI Usage/Cost 默认账单来源。
  - AWS 支持从云账号凭证读取 `AWS_CUR_EXPORT_URL`、`AWS_COST_EXPLORER_EXPORT_URL`、`AWS_BILLING_EXPORT_URL`、`AWS_COST_EXPORT_URL` 以及对应 index URL。
  - OCI 支持从云账号凭证读取 `OCI_USAGE_COST_EXPORT_URL`、`OCI_BILLING_EXPORT_URL`、`OCI_COST_EXPORT_URL`、`ORACLE_USAGE_COST_EXPORT_URL` 以及对应 index URL。
  - AWS 对象存储账单来源默认使用 `s3`，继续复用已完成的对象存储列表、签名授权和分页链路。
- 成本字段归一化扩展：
  - AWS CUR 兼容 `lineItem/UsageAccountId`、`bill/PayerAccountId`、`lineItem/ResourceId`、`lineItem/ProductCode`、`product/ProductName`、`product/region`、`product/instanceType`、`lineItem/UnblendedCost`、`lineItem/CurrencyCode` 和 `resourceTags/user:*` 等常见字段。
  - AWS Cost Explorer 兼容 `Metrics.UnblendedCost.Amount`、`Metrics.BlendedCost.Amount`、`Metrics.AmortizedCost.Amount`、`Metrics.NetUnblendedCost.Amount` 等嵌套金额字段。
  - OCI Usage/Cost 兼容 `tenant_id`、`compartment_id`、`resource_ocid`、`displayName`、`service`、`resource_type`、`region`、`computed_amount`、`time_usage_started` 等常见字段。
  - 继续复用统一成本明细指纹、资产匹配、成本建议和事件审计链路。
- 前端成本中心增强：
  - 成本来源展示新增 “AWS CUR 导出”、“AWS Cost Explorer 导出”、“OCI Usage/Cost 导出”。
  - “新建账单同步计划”“拉取账单”“导入账单” provider 下拉新增 AWS 和 OCI。
  - 拉取型来源下拉新增 “AWS CUR 导出 URL”、“AWS Cost Explorer 导出 URL”、“OCI Usage/Cost 导出 URL”。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 执行 `gofmt`，覆盖：
  - `backend/portal/apps/cloud_cost.go`
  - `backend/portal/models/forms/cloud_cost.go`
  - `backend/portal/models/cloud_cost.go`
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`、`build=docker-compose`、`version=v1.3.5`。
- 启动本地账单导出 mock endpoint `http://host.docker.internal:18089`：
  - `/aws-cur.json` 返回 1 条 AWS CUR 风格账单，资源 ID 为 `i-codex-aws-cost-20260621`，金额 `31.23 USD`。
  - `/aws-ce.json` 返回 1 条 AWS Cost Explorer 风格账单，资源 ID 为 `i-codex-aws-ce-20260621`，金额 `17.89 USD`。
  - `/oci-cost.json` 返回 1 条 OCI Usage/Cost 风格账单，资源 ID 为 `ocid1.instance.oc1.ap-singapore-1.codexoci20260621`，金额 `44.56 USD`。
- AWS CUR API 验证：
  - 创建成本同步任务，`provider=aws`、`source=aws_cur_export`、`sourceUrl=http://host.docker.internal:18089/aws-cur.json`。
  - 任务 `complete`，成本明细写入 `provider=aws`、`source=aws_cur_export`、`accountId=123456789012`、`region=us-east-1`。
  - 成本记录字段映射确认：`service=Amazon Elastic Compute Cloud`、`resourceType=t3.micro`、`resourceId=i-codex-aws-cost-20260621`、`amount=31.23`、`currency=USD`、`period=2026-06`、`costCenter=codex-cost-center-aws`。
- AWS Cost Explorer API 验证：
  - 创建成本同步任务，`provider=aws`、`source=aws_cost_explorer_export`、`sourceUrl=http://host.docker.internal:18089/aws-ce.json`。
  - 任务 `complete`，成本明细写入 `provider=aws`、`source=aws_cost_explorer_export`、`accountId=123456789012`、`region=us-west-2`。
  - 成本记录字段映射确认：`service=AmazonEC2`、`resourceType=CostExplorerGroupedResource`、`resourceId=i-codex-aws-ce-20260621`、`amount=17.89`、`currency=USD`、`period=2026-06`、`costCenter=codex-cost-center-aws-ce`。
- OCI API 验证：
  - 创建成本同步任务，`provider=oci`、`source=oci_usage_cost_export`、`sourceUrl=http://host.docker.internal:18089/oci-cost.json`。
  - 任务 `complete`，成本明细写入 `provider=oci`、`source=oci_usage_cost_export`、`accountId=ocid1.tenancy.oc1..codex`、`region=ap-singapore-1`。
  - 成本记录字段映射确认：`service=Compute`、`resourceType=Compute Instance`、`resourceId=ocid1.instance.oc1.ap-singapore-1.codexoci20260621`、`amount=44.56`、`currency=USD`、`period=2026-06`、`costCenter=codex-cost-center-oci`。
- 验证后已清理本轮临时成本记录、成本建议、事件、同步任务和任务日志，剩余临时数据计数均为 `0`；临时 mock 服务已停止。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-web` 成功。
- 内置浏览器验证成本中心页面：
  - `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-costs` 正常加载，无控制台错误。
  - “新建账单同步计划”弹窗 provider 下拉包含 AWS、OCI、Azure、GCP、腾讯云、华为云。
  - “新建账单同步计划”来源下拉包含 “AWS CUR 导出 URL”、“AWS Cost Explorer 导出 URL”、“OCI Usage/Cost 导出 URL”。
  - “拉取账单”弹窗 provider 和来源下拉包含上述 AWS/OCI 选项。
  - “导入账单”弹窗 provider 和来源下拉包含 “AWS CUR 导出”、“AWS Cost Explorer 导出”、“OCI Usage/Cost 导出”。

验证限制：

- 本阶段验证的是 JSON URL 导出格式和统一字段归一化；CSV/TSV 与 gzip/zip 压缩账单已在 16.105 补齐第一阶段。
- 真实 AWS CUR、AWS Cost Explorer 和 OCI Usage/Cost 导出目录、字段差异、Excel、Parquet、按日/按小时分区目录和 provider 原生错误码映射仍需在生产只读账号或脱敏账单样本上复测。
- AWS S3 原生对象存储列表和签名授权可复用 16.99/16.101 第一阶段能力；OCI Object Storage 原生列表和签名授权已在 16.106 补齐第一阶段。

待继续：

- 在真实 AWS/OCI 账单环境补充端到端联调，并扩展 Excel、Parquet、分区目录和 provider 错误码映射。
- 扩展汇率、摊销、Savings Plans/Reserved Instances 分摊、环比/同比突增规则和整改闭环。

### 16.105 2026-06-21 V1.2 P2 成本 CSV/TSV 与压缩账单解析第一阶段

状态：已完成成本拉取 CSV/TSV 与 gzip/zip 压缩账单解析第一阶段；单文件 URL、导出索引 URL 和对象存储文件下载现在都可复用同一套 JSON/CSV/TSV/gzip/zip 自动识别解析逻辑。

已完成：

- 成本文件解析增强：
  - `cloudCostPullFromURL` 与 `cloudCostPullFromObjectURL` 从只解析 JSON 扩展为统一调用 `cloudCostPullItemsFromExportBody`。
  - 支持按文件名和内容魔数自动识别 gzip 与 zip。
  - gzip 解压后继续按原始文件名后缀解析 JSON、CSV 或 TSV。
  - zip 支持遍历包内 JSON/CSV/TSV 文件并合并为同一个同步任务的成本明细，包内文件名写入 `payload.sourceArchiveFile`。
  - CSV/TSV 使用首行 header 映射为账单 payload，支持 AWS CUR、OCI Usage/Cost、腾讯云/华为云等已接入字段归一化规则。
  - 默认 `Accept` 头放宽为 `application/json,text/csv,text/tab-separated-values,application/gzip,application/zip,*/*`，适配内网对象服务或离线镜像服务。
- 字段归一化修正：
  - 拉取型成本 item 生成时优先从 payload 读取币种，再回退到表单默认币种。
  - 统一币种识别支持 `lineItem/CurrencyCode`、`pricing/currency`、`BillingCurrencyCode`、`currencyCode` 和 Cost Explorer `Metrics.*.Unit`。
  - 最终成本记录归一化同样复用 payload 优先的币种识别逻辑，避免 provider 导出币种被表单默认值覆盖。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 执行 `gofmt`，覆盖 `backend/portal/apps/cloud_cost.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功。
- 启动本地账单导出 mock endpoint `http://host.docker.internal:18090`：
  - `/aws-cur.csv` 返回 1 条 AWS CUR CSV 账单，资源 ID 为 `i-codex-aws-csv-20260621`，金额 `41.23 USD`。
  - `/oci-cost.csv.gz` 返回 1 条 gzip 压缩 OCI CSV 账单，资源 ID 为 `ocid1.instance.oc1.ap-singapore-1.codexocicsv20260621`，金额 `52.34 USD`。
  - `/cost-bundle.zip` 返回 zip 包，包内 `billing/huawei-cost.csv` 为 1 条华为云 CSV 账单，资源 ID 为 `codex-huawei-csv-vm-20260621`，金额 `63.45 CNY`。
- API 验证：
  - AWS CSV 同步任务 `complete`，成本记录写入 `provider=aws`、`source=aws_cur_export`、`period=2026-07`、`currency=USD`。
  - OCI gzip CSV 同步任务 `complete`，成本记录写入 `provider=oci`、`source=oci_usage_cost_export`、`period=2026-07`、`currency=USD`。
  - 华为云 zip CSV 同步任务 `complete`，成本记录写入 `provider=huawei`、`source=huawei_billing_export`、`period=2026-07`、`currency=CNY`。
  - 三条记录均确认金额、币种、账号、区域、服务、资源类型、资源 ID、成本中心和账期映射正确。
- 验证后已清理本轮临时成本记录、成本建议、事件、同步任务和任务日志，剩余临时数据计数均为 `0`；临时 mock 服务已停止。

验证限制：

- 本阶段支持文本 CSV/TSV 及 gzip/zip 容器，不解析 Excel、Parquet、ORC 等二进制/列式格式。
- zip 只解析包内 JSON/CSV/TSV/gzip 文件，跳过隐藏文件和未知后缀文件。
- 单文件和 zip 内单文件解压后仍限制 32MB，生产超大账单建议继续通过索引 URL、对象存储前缀和分区游标分批导入。

待继续：

- 在真实 AWS/OCI/腾讯云/华为云账单样本上补充 CSV 字段差异映射和 provider 原始错误码。
- 扩展 Parquet/Excel 或通过离线转换工具把列式/表格账单预处理为 CSV/JSON。
- 继续补充分区日期、ETag、对象大小、重复文件检测和失败文件重试策略。

### 16.106 2026-06-21 V1.2 P2 OCI Object Storage 成本对象列表与签名授权第一阶段

状态：已完成 OCI Object Storage 原生成本对象列表与签名下载第一阶段；OCI Usage/Cost 现在可以像 S3/GCS/Azure Blob 一样通过对象存储前缀分页发现账单文件，并复用 JSON/CSV/TSV/gzip/zip 解析管线导入统一成本明细。

已完成：

- 后端对象存储枚举扩展：
  - `PullCloudCostRecordForm` 与 `CreateCloudCostSyncScheduleForm` 的 `sourceObjectProvider` 新增 `oci_object_storage`。
  - OCI 成本拉取默认对象存储类型改为 `oci_object_storage`，并兼容 `oci`、`oci_object`、`oracle_object_storage` 别名。
  - 对象存储配置支持从云账号凭证读取 OCI 区域，未显式填写 endpoint 时按区域推导 `https://objectstorage.<region>.oraclecloud.com`。
- OCI Object Storage 列表与下载：
  - 新增 `cloudCostOCIObjectStorageListFiles`，调用 `/n/{namespace}/b/{bucket}/o` 列表 API。
  - 支持 `prefix`、`limit=1000`、`nextStartWith` 分页，最多 100 页，生成 `sourceFileKey` 和增量 `nextCursor`。
  - namespace 支持从 `OCI_OBJECT_STORAGE_NAMESPACE`、`OCI_NAMESPACE`、`OCI_USAGE_COST_OBJECT_NAMESPACE`、`CLOUD_COST_OBJECT_NAMESPACE` 等凭证读取；缺失时复用既有 OCI namespace API 查询。
  - 对象下载 URL 使用 `/n/{namespace}/b/{bucket}/o/{objectName}`，对象名按 path escape 保留包含 `/` 的账单路径。
  - 列表与下载请求复用已有 `signOCIRequest`，覆盖 `Date`、`Host`、`(request-target)` 签名头。
- 前端成本中心增强：
  - 对象存储类型下拉新增 “OCI Object Storage”，可用于“新建账单同步计划”和“拉取账单”。
  - 继续复用现有 endpoint、bucket、prefix、base URL、云账号 ID、区域等输入项。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 执行 `gofmt`，覆盖：
  - `backend/portal/apps/cloud_cost.go`
  - `backend/portal/models/forms/cloud_cost.go`
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`iac-portal` 进入 `healthy`。
- 启动本地 OCI Object Storage mock endpoint `http://host.docker.internal:18091`：
  - 列表接口 `/n/codexnamespace/b/codex-bucket/o?limit=1000&prefix=cost%2F` 返回 1 个对象 `cost/oci-object-cost-20260621.csv`。
  - 下载接口 `/n/codexnamespace/b/codex-bucket/o/cost%2Foci-object-cost-20260621.csv` 返回 1 条 OCI CSV 账单。
  - mock 服务检查 `Authorization: Signature version="1"`、`algorithm="rsa-sha256"`、`headers="date (request-target) host"`、`Date` 和 `Host`，未签名请求会返回 `401`。
- API 验证：
  - 临时创建 OCI 云账号 `codex-oci-object-cost-20260621`，凭证包含 `OCI_TENANCY_OCID`、`OCI_USER_OCID`、`OCI_FINGERPRINT`、`OCI_PRIVATE_KEY`、`OCI_REGION` 和 `OCI_OBJECT_STORAGE_NAMESPACE`，本地凭证校验结果为 `valid`。
  - 直接拉取 `/cloud/cost/pull` 成功，返回 `imported=1`、`mode=export_object_list`、`sourceObjectProvider=oci_object_storage`、`nextCursor=2026-07-15T00:01:00Z`。
  - 成本记录写入 `provider=oci`、`source=oci_usage_cost_export`、`resourceId=ocid1.instance.oc1.ap-singapore-1.codexobject20260621`、`amount=71.89`、`currency=USD`、`period=2026-07`、`region=ap-singapore-1`。
  - 创建成本同步任务 `ccs-d8rlm90tf1ms73csiqug` 成功，任务状态 `complete`，`mode=export_object_list`，`imported=1`，结果写入 `sourceObjectProvider=oci_object_storage` 和 `nextCursor=2026-07-15T00:01:00Z`。
  - mock 日志确认列表和下载请求均命中已签名路径：
    - `GET /n/codexnamespace/b/codex-bucket/o?limit=1000&prefix=cost%2F`
    - `GET /n/codexnamespace/b/codex-bucket/o/cost%2Foci-object-cost-20260621.csv`
- 验证后已清理本轮临时成本记录、成本建议、事件、同步任务、任务日志和临时云账号，剩余临时数据计数均为 `0`。

验证限制：

- 本阶段验证 OCI Object Storage 的列表、签名下载和 CSV 导入链路，不覆盖真实 OCI Usage/Cost 导出桶的目录命名差异。
- 仍需在真实只读 OCI 账号或脱敏账单样本上补充按日/按小时分区目录、失败文件重试和 provider 原生错误码映射；对象大小、ETag 和重复文件跳过已在 16.107 补齐第一阶段。
- Excel、Parquet、ORC 等二进制/列式账单仍需通过离线转换工具预处理为 CSV/JSON，或后续扩展专用解析器。

待继续：

- 用真实 OCI Usage/Cost 导出桶样本补充分区目录识别、重复文件检测和字段差异映射。
- 将对象存储列表结果中的对象大小、ETag、时间字段纳入失败重试队列。
- 扩展汇率、摊销、Reserved Capacity/Commitment 折扣分摊、环比/同比突增规则和整改闭环。

### 16.107 2026-06-21 V1.2 P2 成本对象文件元数据审计与重复文件检测第一阶段

状态：已完成成本导出文件元数据审计与重复文件检测第一阶段；导出索引 URL 和对象存储列表现在会把对象大小、ETag、更新时间写入拉取响应、同步任务结果、任务日志和成本记录 payload，并在同一批文件内跳过重复对象。

已完成：

- 成本文件结构扩展：
  - `cloudCostExportFile` 新增 `Size`、`ETag`、`LastModified`。
  - `CloudCostPullResp` 新增 `files` 与 `skippedFiles`，用于 API 返回和同步任务结果落库。
  - 成本记录 payload 新增 `sourceFileSize`、`sourceFileETag`、`sourceFileLastModified`。
- Provider 元数据映射：
  - S3 ListObjectsV2 读取 `ETag`、`Size`、`LastModified`。
  - GCS objects list 读取 `etag`、`size`、`updated`。
  - Azure Blob list 读取 `Etag`、`Content-Length`、`Last-Modified`。
  - OCI Object Storage list 读取 `etag`、`size`、`timeModified/timeCreated`。
  - 导出索引 JSON 支持 `size/contentLength/content_length`、`eTag/etag/ETag`、`lastModified/updatedAt/timeModified/timeCreated`。
- 重复文件检测：
  - 同一批导出文件按 `key|ETag`、`key|cursor` 或 `key|lastModified` 生成文件身份。
  - 同一批里重复的文件不再重复下载和导入，写入 `skippedFiles`，`reason=duplicate`。
  - 同步任务 `result.files` 记录实际导入文件，`result.skippedFiles` 记录跳过文件；任务完成日志复用同一结果。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 执行 `gofmt`，覆盖：
  - `backend/portal/apps/cloud_cost.go`
  - `backend/portal/models/resps/cloud_cost.go`
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`iac-portal` 进入 `healthy`。
- 启动本地成本索引 mock endpoint `http://host.docker.internal:18092`：
  - `/index.json` 返回两个相同文件条目，均指向 `/cost-a.csv`，`key=billing/cost-a.csv`，`eTag=etag-codex-cost-a`，`size=321`，`lastModified=2026-07-20T00:00:00Z`。
  - `/cost-a.csv` 返回 1 条 AWS CUR CSV 账单，资源 ID 为 `i-codex-metadata-20260621`，金额 `88.88 USD`。
- 直接拉取 API 验证：
  - `/cloud/cost/pull` 返回 `imported=1`、`mode=export_index`、`fileCount=1`。
  - `files[0]` 包含 `eTag=etag-codex-cost-a`、`size=321`、`lastModified=2026-07-20T00:00:00Z`。
  - `skippedFiles[0]` 包含相同文件元数据，`reason=duplicate`。
  - 成本记录 payload 确认写入 `sourceFileETag=etag-codex-cost-a`、`sourceFileSize=321`、`sourceFileLastModified=2026-07-20T00:00:00Z`。
- 同步任务 API 验证：
  - 创建同步任务 `ccs-d8rlr965s74c739elbh0` 成功，任务状态 `complete`，`mode=export_index`，`imported=1`。
  - 任务 `result.files` 长度为 `1`，`result.skippedFiles` 长度为 `1`，`result.files[0].eTag=etag-codex-cost-a`。
  - 任务完成日志中的 `result` 同步包含 `files` 与 `skippedFiles`。
- 验证后已清理本轮临时成本记录、成本建议、事件、同步任务和任务日志，剩余临时数据计数均为 `0`；临时 mock 服务已停止。

验证限制：

- 本阶段只做同一批文件列表内的重复检测，不跨同步任务持久化去重。
- `skippedFiles` 当前记录跳过原因和文件元数据，不进入独立失败重试队列表。
- 未实现真实对象存储 provider 的 ETag 弱一致性差异处理，需要结合真实 AWS/GCS/Azure/OCI 样本继续校准。

待继续：

- 增加跨任务文件指纹表或轻量游标表，支持持久化去重、失败文件重试和人工重跑。
- 在真实对象存储账单桶上校准 ETag、size、lastModified 的 provider 差异。
- 在前端同步任务详情中增加失败文件重试入口。

### 16.108 2026-06-21 V1.2 P2 成本同步任务文件审计前端展示第一阶段

状态：已完成成本同步任务文件审计前端展示第一阶段；任务详情抽屉现在会展示导入文件与跳过文件列表，用户不需要展开原始 JSON 就能看到对象 key、游标、记录数、大小、ETag、更新时间和跳过原因。

已完成：

- 前端展示增强：
  - `账单同步任务详情` 抽屉新增“导入文件”表格，读取 `syncTaskDetail.result.files`。
  - 新增“跳过文件”表格，读取 `syncTaskDetail.result.skippedFiles`。
  - 导入文件表格展示：文件、游标、记录、大小、ETag、更新时间。
  - 跳过文件表格展示：文件、游标、原因、大小、ETag、更新时间。
  - `duplicate` 跳过原因在 UI 显示为“重复文件”。
  - 文件大小按 B/KB/MB/GB 自动格式化，长 key 和 ETag 使用现有行内省略样式。
- 后端结果继续保留：
  - 详情抽屉仍保留“执行结果” JSON，方便排查完整原始 result。
  - 任务日志、任务基础信息和 source preview 不变。

验证：

- `docker compose build iac-web` 成功，前端 bundle 编译通过，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-web` 成功。
- 启动本地成本索引 mock endpoint `http://host.docker.internal:18093`：
  - `/index.json` 返回两个相同文件条目，均指向 `/cost-ui.csv`，`key=billing/cost-ui.csv`，`eTag=etag-codex-ui-cost`，`size=321`，`lastModified=2026-07-21T00:00:00Z`。
  - `/cost-ui.csv` 返回 1 条 AWS CUR CSV 账单，资源 ID 为 `i-codex-ui-files-20260621`，金额 `99.91 USD`。
- API 创建 2026-06 临时同步任务 `ccs-d8rluse5s74c739elbrg` 成功，任务状态 `complete`，`result.files` 长度为 `1`，`result.skippedFiles` 长度为 `1`。
- 内置浏览器验证成本中心页面：
  - `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-costs` 正常加载，无控制台错误。
  - 任务表展示临时任务 `ccs-d8rluse5s74c739elbrg`。
  - 打开任务详情抽屉后，能看到“导入文件”和“跳过文件”两张表。
  - “导入文件”表展示 `billing/cost-ui.csv`、`321 B`、`etag-codex-ui-cost`、更新时间。
  - “跳过文件”表展示相同文件和“重复文件”原因。
- 验证后已清理本轮临时成本记录、成本建议、事件、同步任务和任务日志，剩余临时数据计数均为 `0`；临时 mock 服务已停止。

验证限制：

- 本阶段只展示后端已返回的文件审计结果，不提供文件级重试操作。
- 长 URL 仍主要在执行结果 JSON 或 source preview 中查看，文件列表聚焦 key、游标和对象元数据。

待继续：

- 增加失败文件状态、失败原因和文件级重试入口。
- 支持按文件 key、ETag、跳过原因过滤或展开完整 URL。

### 16.109 2026-06-21 V1.2 P0 云账号敏感凭证响应强制脱敏

状态：已完成云账号敏感凭证响应强制脱敏；即使调用方把敏感字段误标为 `isSecret=false`，创建、详情等响应也不会回显私钥、Token、Password、Secret、AccessKey 等敏感值。

已完成：

- 后端响应脱敏增强：
  - `maskedCloudCredentials` 不再只依赖 `isSecret` 标记。
  - 新增敏感字段名识别，匹配 `ACCESS_KEY`、`ACCOUNT_KEY`、`CLIENT_SECRET`、`PASSWORD`、`PASSPHRASE`、`PRIVATE_KEY`、`SECRET`、`TOKEN`。
  - 命中敏感字段时响应值统一置空，并把 `isSecret` 归一为 `true`。
  - 该逻辑只作用于 API 响应脱敏，不改变数据库中的真实凭证，也不影响 provider adapter 使用解密后的凭证。

验证：

- 使用 Docker Go 镜像通过 stdin/stdout 执行 `gofmt`，覆盖 `backend/portal/apps/cloud_account.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`iac-portal` 进入 `healthy`。
- API 创建临时 OCI 云账号 `codex-sensitive-mask-20260621`：
  - 请求中故意把 `OCI_PRIVATE_KEY`、`AWS_ACCESS_KEY_ID`、`CUSTOM_TOKEN` 标记为 `isSecret=false`。
  - 创建响应中三者均返回 `value=""` 且 `isSecret=true`。
  - 详情响应中三者同样返回 `value=""` 且 `isSecret=true`。
  - 非敏感字段 `OCI_TENANCY_OCID`、`OCI_USER_OCID`、`OCI_FINGERPRINT`、`OCI_REGION` 仍按原值返回。
- 验证后已删除临时云账号和相关事件，剩余临时数据计数均为 `0`。

验证限制：

- 本阶段只做响应层防回显，不引入外部 KMS 或 Vault 引用式凭证存储。
- 字段名识别是保守规则，后续若接入新 provider 的特殊敏感字段，需要补充关键字或 provider schema。

待继续：

- 接入凭证引用式存储或外部密钥管理系统。
- 前端编辑云账号时基于 `isSecret=true` 提示“留空保留原值”，并避免把脱敏空值误当作用户清空。

### 16.102 2026-06-21 V1.2 P2 腾讯云/华为云成本账单导出第一阶段

状态：已完成腾讯云/华为云成本账单导出 URL 导入第一阶段；成本中心不再只覆盖 Azure/GCP 拉取型数据源，现已可通过统一同步任务从 TencentCloud/Huawei Billing Export JSON URL 导入标准成本明细。

已完成：

- 后端成本来源扩展：
  - 新增 `tencentcloud_billing_export` 与 `huawei_billing_export` 成本来源常量。
  - 成本导入、拉取、同步任务和同步计划表单枚举均允许 `provider=tencentcloud/huawei`。
  - `cloudCostImportSource` 与 `cloudCostPullSource` 可按 provider 推导默认账单来源。
  - 腾讯云支持从云账号凭证读取 `TENCENTCLOUD_BILLING_EXPORT_URL`、`TENCENT_BILLING_EXPORT_URL`、`TENCENTCLOUD_COST_EXPORT_URL` 以及对应 index URL。
  - 华为云支持从云账号凭证读取 `HUAWEI_BILLING_EXPORT_URL`、`HUAWEICLOUD_BILLING_EXPORT_URL`、`HUAWEI_COST_EXPORT_URL` 以及对应 index URL。
  - 腾讯云/华为云对象存储账单来源默认复用 `s3` 兼容对象存储路径，便于后续接入 COS/OBS S3 兼容 endpoint。
- 成本字段归一化扩展：
  - 腾讯云账单字段兼容 `OwnerUin`、`PayerUin`、`ResourceId`、`ResourceName`、`BusinessCodeName`、`ProductCodeName`、`ResourceTypeName`、`RegionName`、`ProjectName`、`RealTotalCost`、`TotalCost`、`CashPayAmount`、`VoucherPayAmount`、`BillMonth` 等常见字段。
  - 华为云账单字段兼容 `customer_id`、`resource_id`、`resource_name`、`cloud_service_type_name`、`cloud_service_type`、`resource_type`、`region_id`、`enterprise_project_name`、`official_amount`、`cash_amount`、`credit_amount`、`coupon_amount`、`effective_time` 等常见字段。
  - 继续复用统一成本明细指纹、资产匹配、成本建议和事件审计链路。
- 前端成本中心增强：
  - 成本来源展示新增“腾讯云账单导出”和“华为云账单导出”。
  - “拉取账单”“账单同步任务”“账单同步计划”相关 provider 下拉新增“腾讯云”“华为云”。
  - 拉取型来源下拉新增“腾讯云账单导出 URL”和“华为云账单导出 URL”。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_cost.go`、`backend/portal/models/forms/cloud_cost.go`、`backend/portal/models/cloud_cost.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`、`build=docker-compose`、`version=v1.3.5`。
- 启动本地账单导出 mock endpoint `http://host.docker.internal:18087`：
  - `/tencent-cost.json` 返回 1 条腾讯云风格账单，资源 ID 为 `codex-tencent-cost-vm-20260621`，金额 `12.34 CNY`。
  - `/huawei-cost.json` 返回 1 条华为云风格账单，资源 ID 为 `codex-huawei-cost-vm-20260621`，金额 `23.45 CNY`。
- 腾讯云 API 验证：
  - 创建成本同步任务，`provider=tencentcloud`、`source=tencentcloud_billing_export`、`sourceUrl=http://host.docker.internal:18087/tencent-cost.json`。
  - 任务 `complete`，成本明细写入 `provider=tencentcloud`、`source=tencentcloud_billing_export`、`accountId=codex-tencent-account`、`region=ap-guangzhou`。
  - 成本记录字段映射确认：`service=云服务器 CVM`、`resourceType=云服务器实例`、`resourceId=codex-tencent-cost-vm-20260621`、`resourceName=codex-tencent-vm`、`amount=12.34`、`currency=CNY`、`period=2026-06`、`costCenter=codex-cost-center-tencent`。
- 华为云 API 验证：
  - 创建成本同步任务，`provider=huawei`、`source=huawei_billing_export`、`sourceUrl=http://host.docker.internal:18087/huawei-cost.json`。
  - 任务 `complete`，成本明细写入 `provider=huawei`、`source=huawei_billing_export`、`accountId=codex-huawei-account`、`region=cn-north-4`。
  - 成本记录字段映射确认：`service=弹性云服务器 ECS`、`resourceType=huawei.ecs.instance`、`resourceId=codex-huawei-cost-vm-20260621`、`resourceName=codex-huawei-vm`、`amount=23.45`、`currency=CNY`、`period=2026-06`、`costCenter=codex-cost-center-huawei`。
- 验证后已清理本轮临时成本记录、成本建议、事件、同步任务和任务日志，剩余临时数据计数均为 `0`；临时 mock 服务已停止。

验证限制：

- 本阶段验证的是 JSON URL 导出格式和统一字段归一化；真实腾讯云账单、华为云账单的下载授权、导出目录和字段差异仍需在生产只读账号或脱敏账单样本上复测。
- Excel、Parquet、按日/按小时分区目录和 provider 原生分页/错误码映射仍待继续。
- COS/OBS 账单对象存储可先复用对象存储列表 API 和 `s3` 兼容路径，后续需要补充真实 endpoint 签名兼容性验证。

待继续：

- 在真实腾讯云/华为云账单环境补充端到端联调，并扩展 CSV 字段差异、Parquet/Excel、分区目录和 provider 错误码映射。
- 成本同步计划通知静默、通知窗口、负责人和路由已在 16.190 完成第一阶段，按错误类型路由和失败次数升级策略已在 16.191 完成第一阶段；继续增强真实通知通道端到端联调和更细的企业升级策略。
- 扩展成本异常阈值配置、环比/同比突增规则和整改闭环。

### 16.103 2026-06-21 V1.2 P2 成本计划失败外部 Webhook 通知第一阶段

状态：已完成成本计划失败事件对外部 Webhook/通知策略的可配置投递第一阶段；成本同步计划失败和自动暂停事件不仅落入平台事件中心，也可通过 `cost.*`、`cost.sync.schedule.failed` 或 `cost.sync.schedule.auto_paused` 订阅推送到外部系统。

已完成：

- Webhook 事件匹配增强：
  - `stringSliceMatches` 新增 `.*` 前缀通配语义的实际匹配能力，支持 `cost.*`、`risk.*`、`operation.*`、`cloud.*` 等事件类型配置。
  - `cost.*` 可匹配 `cost.pull.failed`、`cost.sync.schedule.failed`、`cost.sync.schedule.auto_paused` 等成本事件。
  - `operation.*` 兼容 `cloud_operation.*` 历史事件命名，避免前端已有通配选项配置后无法命中。
- 前端事件类型增强：
  - 事件中心 Webhook 配置新增 `cloud.*`、`account.*`、`sync.*`、`operation.*`、`risk.*`、`cost.*`、`cmdb.*`、`notification.*`、`webhook.*`、`itsm.*` 通配选项。
  - 事件中心和组织/项目通知策略新增成本事件选项：`cost.pull.completed`、`cost.pull.failed`、`cost.sync.schedule.failed`、`cost.sync.schedule.auto_paused`。
  - 通知模板事件类型选择同步补齐上述成本事件，便于为成本计划失败配置专用通知内容。

验证：

- 使用 Docker stdin/stdout 方式对 `backend/portal/apps/cloud_webhook.go` 执行 `gofmt`；Docker Desktop 对当前 `/Volumes/...` 目录挂载存在路径冲突，因此未使用挂载式格式化。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`、`build=docker-compose`、`version=v1.3.5`。
- 启动本地 Webhook 接收器 `http://host.docker.internal:18088/webhook`。
- 创建 Webhook：
  - 名称 `codex-cost-webhook-wildcard-20260621-0251`。
  - `eventTypes=["cost.*"]`。
  - `sources=["cost"]`。
  - 目标地址 `http://host.docker.internal:18088/webhook`。
- 创建成本同步计划 `codex-cost-webhook-failure-20260621-0251`：
  - `provider=gcp`、`source=gcp_billing_export`、`sourceUrl=http://host.docker.internal:18088/missing-cost.json`。
  - `notifyOnFailure=true`、`maxRetryAttempts=1`、`retryBackoffSeconds=60`。
- 手动运行计划：
  - 拉取 URL 返回 404，任务失败并写入 `cost.pull.failed`。
  - 计划记录 `failureCount=1`、`nextRetryAt`，并写入 `cost.sync.schedule.failed`。
  - Webhook 投递记录显示 `cost.pull.failed` 和 `cost.sync.schedule.failed` 均 `status=success`、`response_code=200`、`delivery_mode=initial`。
  - Webhook 配置回写 `last_status=success`、`last_status_code=200`、`last_message=ok`。
  - `request_payload` 为 `{event, webhook}` 包装结构，`event.eventType=cost.sync.schedule.failed`，`event.source=cost`，`event.payload.scheduleId`、`failureCount`、`nextRetryAt`、`taskId` 和失败原因均可供外部系统解析。
- 验证后已清理本轮临时 Webhook、投递记录、队列、死信、事件、成本同步计划、同步任务、任务日志、成本记录和成本建议，剩余临时数据计数均为 `0`；临时 Webhook mock 服务已停止。

验证限制：

- 本阶段验证的是平台 Webhook 投递链路；邮件、钉钉、企业微信、Slack 等通知类型复用同一事件分发逻辑和通知模板，仍需按企业实际机器人/SMTP 配置做端到端联调。
- 成本计划通知已支持事件类型订阅、模板配置、计划级静默期、通知窗口、负责人分派、按错误类型路由和失败次数升级第一阶段；真实通知通道联调和更细企业升级策略仍待继续。

待继续：

- 继续增强成本计划真实通知通道端到端联调和更细的企业升级策略。
- 扩展成本异常阈值配置、环比/同比突增规则和整改闭环。
- 在真实腾讯云/华为云账单环境补充端到端联调，并扩展 CSV 字段差异、Parquet/Excel、分区目录和 provider 错误码映射。

### 16.101 2026-06-21 V1.2 P2 成本对象存储签名授权与分页第一阶段

状态：已完成成本对象存储列表的签名授权与分页第一阶段；Azure Blob、GCS、S3 原生列表不再只依赖公开 URL、预授权 endpoint 或手工 `sourceHeaders`，有云账号凭证时可自动补齐授权，并支持 provider 原生分页 token。

已完成：

- 对象存储请求授权：
  - S3：复用现有 AWS SigV4 签名函数，对列表请求和对象文件下载请求自动写入 `Authorization`、`X-Amz-Date`、`X-Amz-Content-Sha256`，支持 `AWS_SESSION_TOKEN`。
  - GCS：复用现有 GCP access token 获取逻辑，支持 `GCP_ACCESS_TOKEN` 或 `GCP_SERVICE_ACCOUNT_JSON`，对列表请求和对象文件下载请求自动写入 `Authorization: Bearer ...`。
  - Azure Blob：支持 `AZURE_STORAGE_SAS_TOKEN` / `AZURE_BLOB_SAS_TOKEN` / `AZURE_SAS_TOKEN` 等 SAS token 自动追加到列表和文件 URL；同时支持 `AZURE_STORAGE_ACCOUNT(_NAME)` + `AZURE_STORAGE_ACCOUNT_KEY` / `AZURE_BLOB_ACCOUNT_KEY` 生成 SharedKey 授权。
- 对象存储配置增强：
  - `cloudCostObjectStorageConfig` 带入云账号、区域和对象存储配置，S3 默认区域为 `us-east-1`。
  - 兼容从云账号凭证读取 `S3_OBJECT_*`、`AWS_S3_OBJECT_*`、`CLOUD_COST_OBJECT_*`、`AZURE_BLOB_*`、`GCS_*` 等对象存储配置 key。
  - 仍保留无凭证时的公开/预授权 endpoint 行为，兼容离线镜像服务、内网代理和现有 mock。
- provider 原生分页：
  - S3 支持 `IsTruncated` + `NextContinuationToken`，下一页请求写入 `continuation-token`。
  - GCS 支持 `nextPageToken`，下一页请求写入 `pageToken`。
  - Azure Blob 支持 `NextMarker`，下一页请求写入 `marker`。
  - 每类列表最多读取 100 页，防止异常 token 造成无限循环。
- 对象文件下载授权：
  - `export_object_list` 模式下，多文件导入使用对象存储专用下载器。
  - 列表返回的文件 URL 或平台拼接出的对象文件 URL，在下载 JSON 成本文件时会复用对应 S3/GCS/Azure Blob 授权。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_cost.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`。
- 启动本地强鉴权 mock 对象存储服务 `http://host.docker.internal:18086`：
  - S3 mock 要求 `Authorization` 以 `AWS4-HMAC-SHA256` 开头，并要求 `X-Amz-Date`、`X-Amz-Content-Sha256` 存在。
  - GCS mock 要求 `Authorization: Bearer gcs-token`。
  - Azure Blob mock 要求 URL query 自动携带 `sv=mock&sig=abc`。
  - 三类 mock 均返回两页列表和两个成本 JSON 文件，列表和文件下载都拒绝未授权请求。
- S3 API 验证：
  - 创建临时云账号 `codex-auth-s3-object-20260621`，写入 `AWS_ACCESS_KEY_ID`、`AWS_SECRET_ACCESS_KEY`、`AWS_REGION`。
  - 创建成本同步任务，`sourceObjectProvider=s3`、`sourceObjectEndpoint=http://host.docker.internal:18086/s3auth`。
  - 任务 `complete`，`mode=export_object_list`，`imported=2`，`fileCount=2`，`sourceObjectProvider=s3`。
  - mock 日志确认请求了第一页、`continuation-token=page2` 第二页，以及两个对象文件。
- GCS API 验证：
  - 创建临时云账号 `codex-auth-gcs-object-20260621`，写入 `GCP_ACCESS_TOKEN=gcs-token`。
  - 创建成本同步任务，`sourceObjectProvider=gcs`、`sourceObjectEndpoint=http://host.docker.internal:18086/gcsauth`。
  - 任务 `complete`，`mode=export_object_list`，`imported=2`，`fileCount=2`，`sourceObjectProvider=gcs`。
  - mock 日志确认请求了第一页、`pageToken=page2` 第二页，以及两个对象文件。
- Azure Blob API 验证：
  - 创建临时云账号 `codex-auth-azure-object-20260621`，写入 `AZURE_STORAGE_SAS_TOKEN=sv=mock&sig=abc`。
  - 创建成本同步任务，`sourceObjectProvider=azure_blob`、`sourceObjectEndpoint=http://host.docker.internal:18086/azureauth`。
  - 任务 `complete`，`mode=export_object_list`，`imported=2`，`fileCount=2`，`sourceObjectProvider=azure_blob`。
  - mock 日志确认列表第一页、`marker=page2` 第二页和两个对象文件 URL 均自动带 SAS query。
- 验证后已清理本轮临时云账号、成本记录、成本建议、事件、同步任务和任务日志，剩余临时数据计数均为 `0`；临时 mock 服务已停止。

验证限制：

- 本阶段使用本地 mock 验证签名头、Bearer、SAS、分页和下载授权闭环；真实 AWS S3、Google Cloud Storage、Azure Blob 账号的端到端联调仍待在具备真实账单 bucket/container 的环境执行。
- Azure SharedKey 已实现签名逻辑，但本轮自动化验证使用 SAS token；SharedKey 需要在真实或 Azurite 环境补充一次端到端验签。
- 分页当前最多读取 100 页，并在读取后再按 `cursor/maxFiles` 过滤导入；超大 bucket 后续需要结合 provider 分区、前缀规划和更细游标减少扫描量。

待继续：

- 在真实 AWS/GCP/Azure 账单对象存储中做端到端联调，并补 provider SDK/REST 错误码映射。
- 在真实腾讯云/华为云账单环境补充端到端联调，并扩展 CSV 字段差异、Parquet/Excel、分区目录和 provider 错误码映射。
- 扩展 ETag、分区日期、Excel/Parquet 等非文本成本导出格式。

### 16.98 2026-06-21 V1.2 P2 成本导出文件索引 URL 与增量游标第一阶段

状态：已完成成本拉取导出文件索引 URL、增量游标、单次最大文件数、多文件批量导入、计划游标回写和前端配置字段第一阶段；同步计划不再只能指向单个导出 JSON 文件。

本阶段新增：

- 拉取参数扩展：
  - `PullCloudCostRecordForm` 和 `Create/UpdateCloudCostSyncScheduleForm` 新增 `sourceIndexUrl`、`cursor`、`maxFiles`。
  - 同步任务和同步计划参数会保存上述字段，接口响应中 `sourceIndexUrl` 仅返回脱敏预览。
- 索引 URL 模式：
  - `sourceIndexUrl` 优先于 `sourceUrl`，拉取模式记为 `export_index`。
  - 索引 JSON 支持数组、`files/items/exports/objects` 包装对象。
  - 文件项支持字符串 URL，或 `{url/sourceUrl/downloadUrl/href, key/name/path, cursor/updatedAt/lastModified}` 对象。
  - 支持相对 URL，按索引 URL 解析为完整文件地址。
  - 按 `cursor` 过滤已处理文件，按 `maxFiles` 控制单次处理数量，默认 20，最大 100。
  - 没有新文件时任务仍成功完成，`imported=0`、`fileCount=0`，避免定时计划被误判失败。
- 任务和计划结果：
  - 任务结果写入 `sourceIndexUrl` 预览、`cursor`、`nextCursor` 和 `fileCount`。
  - 成本拉取事件 payload 写入索引 URL 预览、游标、下一游标和文件数。
  - 计划运行成功后从任务结果读取 `nextCursor` 并写回计划 `params.cursor`，下一次运行自动增量推进。
- 前端成本中心增强：
  - “拉取账单”和“账单同步计划”表单新增“账单索引 URL”“增量游标”“单次最大文件数”。
  - 任务/计划列表优先展示 `sourceIndexUrlPreview`，没有索引 URL 时继续展示 `sourceUrlPreview`。
  - 任务详情新增“索引 URL”字段。

验证：

- `docker compose build iac-portal` 成功。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功；重启期间遇到 Consul 服务锁短暂占用，等待后 `/api/v1/check` 恢复 `success=true`。
- 使用本地索引 endpoint：
  - `index.json` 包含 `2026-09-01` 和 `2026-09-02` 两个文件，每个文件各 1 条 GCP Billing Export 成本记录。
  - 创建一次性同步任务，传 `sourceIndexUrl=http://host.docker.internal:18083/index.json?signature=codex-secret`、`maxFiles=10`，任务 `complete`，`mode=export_index`，`imported=2`，`fileCount=2`，`nextCursor=2026-09-02`，索引 URL 响应为 `http://host.docker.internal:18083/index.json?...`。
  - 使用 `cursor=2026-09-02` 重复创建任务，任务 `complete`，`imported=0`，未重复导入。
- 使用同步计划验证游标回写：
  - 创建禁用状态计划 `codex-cost-index-schedule`，传 `sourceIndexUrl` 和 `maxFiles=1`。
  - 第一次手动运行导入 1 条，任务 `nextCursor=2026-09-01`，计划 `params.cursor` 回写为 `2026-09-01`。
  - 第二次手动运行导入 1 条，任务 `cursor=2026-09-01`、`nextCursor=2026-09-02`，计划 `params.cursor` 回写为 `2026-09-02`。
  - 第三次手动运行无新文件，任务 `complete`、`imported=0`、`fileCount=0`，未重复导入。
- 内置浏览器验证成本中心页面：
  - 切换账期到 `2026-09` 后，计划表可展示索引 URL 脱敏预览。
  - “新建账单同步计划”弹窗展示“账单索引 URL”“增量游标”“单次最大文件数”和“账单导出 URL”字段。
- 验证后已清理本轮临时成本记录、成本建议、事件、同步任务、同步计划和任务日志，剩余临时数据计数均为 `0`；临时本地 HTTP endpoint 已停止并删除。

验证限制：

- 本阶段实现的是平台可控的索引 URL 协议，适合离线对象存储清单、预签名索引或内部镜像服务；Azure Blob、GCS、S3 等云厂商对象存储原生 List API 已在 16.99 完成第一阶段。
- 游标当前按索引文件项的 `cursor/key/url` 字符串顺序推进；生产可继续扩展为按 lastModified 时间、ETag、分区日期和 provider 原生 continuation token。

待继续：

- 对象存储列表 API 继续补齐真实云环境联调、provider SDK/REST 错误映射、ETag 和更复杂目录规则。
- 在真实腾讯云/华为云账单环境补充端到端联调，并扩展 CSV 字段差异、Parquet/Excel、分区目录和 provider 错误码映射。
- 成本同步计划通知静默、通知窗口、负责人和路由已在 16.190 完成第一阶段，按错误类型路由和失败次数升级策略已在 16.191 完成第一阶段；继续增强真实通知通道端到端联调和更细的企业升级策略。

### 16.99 2026-06-21 V1.2 P2 成本对象存储原生列表 API 第一阶段

状态：已完成 Azure Blob、GCS、S3 对象存储原生列表 API 第一阶段，并接入成本拉取任务、同步计划和成本中心前端配置入口。

已完成：

- 成本拉取新增 `export_object_list` 模式，优先于导出索引 URL 和单文件 URL。
- 新增请求参数：
  - `sourceObjectProvider`：支持 `s3`、`gcs`、`azure_blob`。
  - `sourceObjectEndpoint`：对象存储列表 API endpoint。
  - `sourceObjectBucket`：S3/GCS Bucket 或 Azure Blob Container。
  - `sourceObjectPrefix`：对象前缀，用于过滤账单文件。
  - `sourceObjectBaseUrl`：可选的文件下载 Base URL，用于列表 endpoint 与下载 endpoint 分离场景。
- 支持三类对象存储列表解析：
  - S3：`ListObjectsV2` XML，解析 `Contents/Key/LastModified`。
  - GCS：Objects JSON list，解析 `items[].name/updated/mediaLink`。
  - Azure Blob：Container list XML，解析 `Blob/Name/Url/Properties/Last-Modified`。
- 列表结果复用 16.98 的多文件导入、`cursor`、`maxFiles`、`fileCount`、`nextCursor` 和任务结果记录。
- 支持从云账号凭证中读取对象存储配置，兼容 `CLOUD_COST_*`、`CLOUD_BILLING_*`、`AZURE_BILLING_*`、`AZURE_BLOB_*`、`GCP_BILLING_*`、`GCS_*` 等 key 前缀。
- 成本同步任务和同步计划参数会保存对象存储配置；编辑计划时 `sourceObjectEndpoint`、`sourceObjectBaseUrl` 留空可保留原值，避免敏感 URL 回显。
- 任务列表、计划列表和任务详情新增对象存储 Endpoint 脱敏预览。
- 成本中心前端“拉取账单”和“新建/编辑同步计划”弹窗新增对象存储类型、Endpoint、Bucket/Container、对象前缀和文件 Base URL 字段。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_cost.go`、`backend/portal/models/forms/cloud_cost.go`、`backend/portal/models/resps/cloud_cost.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`。
- 使用本地 mock endpoint 验证 S3：
  - `sourceObjectProvider=s3`、`sourceObjectEndpoint=http://host.docker.internal:18084/s3`。
  - 创建同步任务 `complete`，`mode=export_object_list`，`imported=1`，`fileCount=1`，`sourceObjectProvider=s3`。
- 使用本地 mock endpoint 验证 GCS：
  - `sourceObjectProvider=gcs`、`sourceObjectEndpoint=http://host.docker.internal:18084/gcs`。
  - 创建同步任务 `complete`，`mode=export_object_list`，`imported=1`，`fileCount=1`，`sourceObjectProvider=gcs`。
- 使用本地 mock endpoint 验证 Azure Blob：
  - `sourceObjectProvider=azure_blob`、`sourceObjectEndpoint=http://host.docker.internal:18084/azure`。
  - 创建同步任务 `complete`，`mode=export_object_list`，`imported=1`，`fileCount=1`，`sourceObjectProvider=azure_blob`。
- 使用同步计划验证对象存储配置保存和编辑保留：
  - 创建计划后返回 `sourceObjectEndpointPreview=http://host.docker.internal:18084/s3`。
  - 编辑计划时留空 `sourceObjectEndpoint`，返回预览仍保留原 Endpoint，Bucket 和 Prefix 正常保留。
- 验证后已清理本轮临时成本记录、成本建议、事件、同步任务、同步计划和任务日志，剩余临时数据计数均为 `0`；临时本地 HTTP endpoint 已停止。

验证限制：

- 第一阶段使用 REST/XML/JSON 协议解析，不引入云厂商 SDK。
- AWS SigV4、Azure SAS/SharedKey、GCP OAuth/Service Account 等自动签名授权已在 16.101 完成第一阶段；真实生产云账号仍需继续做端到端联调和 provider 错误映射。
- provider 原生分页 token、ETag、分区日期和更复杂的账单目录规则仍待继续。

待继续：

- 对象存储列表 API 继续补齐真实云环境联调、provider SDK/REST 错误映射、ETag 和更复杂目录规则。
- 在真实腾讯云/华为云账单环境补充端到端联调，并扩展 CSV 字段差异、Parquet/Excel、分区目录和 provider 错误码映射。
- 成本同步计划通知静默、通知窗口、负责人和路由已在 16.190 完成第一阶段，按错误类型路由和失败次数升级策略已在 16.191 完成第一阶段；继续增强真实通知通道端到端联调和更细的企业升级策略。

### 16.100 2026-06-21 V1.2 P2 成本同步计划失败退避、通知事件与自动暂停第一阶段

状态：已完成成本同步计划失败重试策略第一阶段；计划运行失败后可按计划参数记录失败次数、计算下一次退避执行时间、写入失败/自动暂停事件，并在超过阈值后自动停用计划。

已完成：

- 同步计划表单参数新增：
  - `maxRetryAttempts`：最大失败重试次数，后端限制 `0-10`，`0` 表示不启用计划级短间隔重试。
  - `retryBackoffSeconds`：基础退避秒数，后端限制 `60-86400`。
  - `notifyOnFailure`：失败时写入成本同步计划失败事件。
  - `autoPauseOnFailure`：超过最大重试次数后自动停用计划。
- 同步计划运行逻辑增强：
  - 任务返回 `failed` 时，计划会递增 `failureCount`，写入 `lastFailureAt`、`lastFailureReason`。
  - 未超过最大重试次数时，按 `retryBackoffSeconds * 2^(failureCount-1)` 计算 `nextRetryAt`，并覆盖计划 `nextSyncAt`，最长 24 小时。
  - 成功运行后清空失败状态，并继续按任务结果回写增量游标。
  - 超过最大重试次数且开启自动暂停时，计划 `status` 写为 `disable`，记录 `autoPausedAt` 和 `autoPauseReason`，`nextSyncAt` 在前端展示为 `-`。
  - 手动重新启用计划时，清空 `failureCount`、`lastFailureAt`、`lastFailureReason`、`nextRetryAt`、`autoPausedAt` 和 `autoPauseReason`，避免立即再次触发自动暂停。
- 事件中心增强：
  - 首次或中间失败写入 `cost.sync.schedule.failed`，payload 包含计划 ID、计划名称、provider、source、账期、失败次数、最大重试次数、退避秒数、下一次重试时间、任务 ID 和失败原因。
  - 自动暂停写入 `cost.sync.schedule.auto_paused`，事件状态为 `disable`，payload 标记 `autoPaused=true`。
- 前端成本中心增强：
  - “新建/编辑账单同步计划”弹窗新增“失败最大重试次数”“失败退避秒数”“失败时发送通知事件”“超过重试次数后自动暂停”。
  - 计划列表“最近同步”列展示失败次数、下一次重试时间和自动暂停标记。

验证：

- 使用 Docker Go 镜像通过 `gofmt` 格式化 `backend/portal/apps/cloud_cost.go`、`backend/portal/models/forms/cloud_cost.go`、`backend/portal/models/resps/cloud_cost.go`。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`。
- API 验证失败退避：
  - 创建计划 `codex-retry-autopause-20260621`，`maxRetryAttempts=1`、`retryBackoffSeconds=60`、`notifyOnFailure=true`、`autoPauseOnFailure=true`。
  - 第一次手动运行生成失败任务，计划保持 `enable`，`failureCount=1`，`nextRetryAt` 和 `nextSyncAt` 推进到约 60 秒后。
  - 事件中心写入 `cost.sync.schedule.failed`，payload 记录失败次数、下一次重试时间、任务状态和错误信息。
- API 验证自动暂停：
  - 第二次手动运行继续失败后，计划写为 `disable`，`failureCount=2`，`autoPausedAt` 非空，`nextRetryAt` 清空。
  - 事件中心写入 `cost.sync.schedule.auto_paused`，事件 `status=disable`，payload 标记 `autoPaused=true`。
- API 验证重新启用：
  - 对自动暂停计划调用更新接口并设置 `status=enable` 后，响应 `failureCount=0`，`autoPausedAt` 和 `nextRetryAt` 均清空，原始 `sourceUrl` 保留。
- `docker compose build iac-web` 成功，仅存在既有 webpack bundle size 警告。
- `docker compose up -d iac-web` 成功。
- 内置浏览器验证成本中心页面：
  - `/org/org-d8qk6fsd6t1s73fu2kr0/m-cloud-costs` 正常加载，无控制台错误。
  - “账单同步计划”列表可展示失败计划的最近同步状态。
  - “新建账单同步计划”弹窗展示“失败最大重试次数”“失败退避秒数”“失败时发送通知事件”“超过重试次数后自动暂停”，默认值为 `3`、`300` 秒且两个开关默认选中。
- 验证后已清理本轮临时同步计划、同步任务、任务日志和事件，剩余临时数据计数均为 `0`。

验证限制：

- 本阶段失败通知先落入平台事件中心；外部通知/Webhook/ITSM 分发可复用后续事件通知策略继续扩展。
- 后端无法区分 API 调用中 `maxRetryAttempts=0` 是未传还是显式禁用，因此默认重试值主要由前端新建计划默认值提供；纯 API 创建如需启用计划级重试，应显式传入 `maxRetryAttempts`。
- 退避按计划整体失败计数计算，尚未细分到 provider 错误类型、HTTP 状态码、对象文件或区域级错误。

待继续：

- 对象存储列表 API 继续补齐真实云环境联调、provider SDK/REST 错误映射、ETag 和更复杂目录规则。
- 在真实腾讯云/华为云账单环境补充端到端联调，并扩展 CSV 字段差异、Parquet/Excel、分区目录和 provider 错误码映射。
- 成本同步计划通知静默、通知窗口、负责人和路由已在 16.190 完成第一阶段，按错误类型路由和失败次数升级策略已在 16.191 完成第一阶段；继续增强真实通知通道端到端联调和更细的企业升级策略。

### 16.216 2026-06-23 V1.2 P0 ITSM 自助目录策略组织级配置第一阶段

状态：已完成 ITSM 自助目录策略组织级配置第一阶段；自助目录从固定内置策略升级为可由组织级系统配置覆盖的策略模型，支撑自助运维目录上架状态、允许角色、适用范围和 SLA 的按组织调整。

已完成：

- 后端新增组织级自助目录策略配置能力：
  - 策略配置存储在 `iac_system_cfg`，使用组织维度 key 前缀 `CLOUD_ITSM_CATALOG_POLICY_<orgId>`。
  - 新增 `PUT /api/v1/cloud/itsm/catalog-policies/:key`，支持按服务项 key 更新或恢复默认策略。
  - 后端校验目录 key、启用状态、SLA 分钟、允许角色和适用范围，保存前做去重和规范化。
- 自助目录响应增强：
  - 返回 `policyConfigured`、`policySource`、`enabled`、`available` 等字段，前端可区分默认策略和组织级覆盖策略。
  - 当组织级策略停用服务项时，目录状态展示为不可用，并给出“策略已停用”原因。
  - 自助工单提交时快照当前策略名称、说明、角色、范围和 SLA，已创建工单不受后续策略调整影响。
- 前端 ITSM 自助目录增强：
  - “自助目录”表格展示策略来源、上架状态、允许角色、适用范围和 SLA。
  - 每个服务项新增“策略”入口，可打开“配置自助目录策略”弹窗。
  - 策略弹窗支持启停、SLA 分钟、策略名称、策略说明、允许角色、适用范围和恢复默认。
- 测试补齐：
  - 修复已有 ITSM 自助目录测试中的策略函数调用签名。
  - 新增 `TestCloudItsmCatalogPolicyOverride`，覆盖组织级策略覆盖、停用、SLA、角色/范围规范化和策略来源标记。
  - 修复 `cloud_cost.go` 中 Go 1.26 vet 对动态 `fmt.Errorf` 格式串的报错，不改变业务行为。

验证：

- Docker Go 1.26 镜像内执行目标测试通过：
  - `go test ./portal/apps -run "TestCloudItsm(CatalogPolicyForSelfService|CatalogPolicyOverride|SelfServicePolicyPayload)|TestCloudEventItsm"`。
- `git diff --check` 覆盖本阶段相关后端、前端和 PRD 文件，通过。
- `docker compose build iac-portal` 成功，后端 Go 编译通过。
- `docker compose up -d iac-portal` 成功，`/api/v1/check` 返回 `success=true`。
- 内置浏览器验证 ITSM 页面：
  - `/org/org-d8sm0ghqn3ks73blu1ig/m-cloud-itsm` 正常加载。
  - “自助目录”页签展示 14 个“策略”入口。
  - 点击 `refresh_metadata` 服务项策略入口后，弹窗展示“配置自助目录策略”“策略状态”“SLA 分钟”“恢复默认”“策略名称”“策略说明”“允许角色”“适用范围”。

验证限制：

- 第一阶段采用组织级 `iac_system_cfg` JSON 覆盖策略，尚未拆分为独立策略表、版本历史和审计差异。
- 团队/组织架构维度目标趋势仍待继续，当前以项目、环境、资源和角色范围为主。
- 外部 ITSM 真实系统端到端联调、独立队列报表细分和更多死信重放操作仍需继续增强。

待继续：

- 补齐真实组织架构团队维度目标趋势，并把自助覆盖率、自动化处理率和 SLA 达成率细化到团队。
- 扩展自助目录策略的版本历史、审批变更留痕和差异审计。
- 继续对接外部 ITSM 真实系统联调、独立队列报表和死信重放页面。

### 16.217 2026-06-23 V1.2 P0 ITSM 团队维度目标趋势第一阶段

状态：已完成 ITSM 团队维度目标趋势第一阶段；自助运维 80% 覆盖、工单自动化 60% 和 SLA 达成率不再只按项目/申请类型展示，也可按团队维度展示，支撑业务团队自助化目标跟踪。

已完成：

- 后端 `CloudItsmOverview` 响应新增 `teamTrends`。
- 团队维度复用 `CloudItsmDimensionMetricResp`，与项目/申请类型保持一致口径：
  - `ticketTotal`：近 30 天团队相关工单总数。
  - `selfServiceTicketTotal` / `selfServiceCoverageRate`：团队自助工单数和自助占比。
  - `automatedTicketTotal` / `ticketAutomationRate`：团队自动化处理工单数和自动化处理率。
  - `selfServiceSlaTicketTotal` / `selfServiceSlaMetRate`：团队 SLA 工单数和 SLA 达成率。
- 团队归属提取顺序：
  - 优先读取工单 payload、operation params、requester、externalPayload 中的 `team`、`teamId`、`teamName`、`businessTeam`、`businessUnit`、`department`、`ownerTeam`、`requestTeam`、`oncallTeam`、`costCenter` 等字段。
  - 其次读取项目 LDAP OU 授权记录 `iac_ldap_ou_project` 的 `OU/DN`，用于真实组织架构映射。
  - 再次读取申请人 `iac_user.company` 作为团队/部门归属。
  - 最后按项目名称兜底；无项目且无团队字段时归入“未标记团队”。
- 前端 ITSM “自助目录”页签新增“团队目标趋势”表，与“项目目标趋势”“申请类型趋势”并列展示。

验证：

- Docker Go 1.26 镜像内执行目标测试通过：
  - `go test ./portal/apps -run "TestCloudItsm(DimensionMetricAddTicket|TicketTeamDimension|CatalogPolicyForSelfService|CatalogPolicyOverride|SelfServicePolicyPayload)|TestCloudEventItsm"`。
- 新增 `TestCloudItsmTicketTeamDimension`，覆盖显式团队对象、requester 部门、项目兜底和未标记团队兜底。
- `git diff --check` 覆盖本阶段相关后端、前端和 PRD 文件，通过。
- `docker compose build iac-portal iac-web` 成功：
  - 后端完成 Swagger 生成和 Go 编译。
  - 前端完成 vendor 和业务包构建，仅保留既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功，`/api/v1/check` 返回 `success=true`。
- 内置浏览器验证 ITSM 页面：
  - `/org/org-d8sm0ghqn3ks73blu1ig/m-cloud-itsm` 正常加载。
  - “自助目录”页签展示“团队目标趋势”“项目目标趋势”“申请类型趋势”。
  - 当前验证组织暂无工单，团队趋势表显示空数据，但页面结构和后端构建链路已验证。

验证限制：

- 第一阶段不新增独立团队主数据表，团队来自工单扩展字段、项目 LDAP OU、申请人公司字段和项目兜底。
- 真实企业组织架构、团队层级、人员多团队归属和值班组映射仍需要根据目标系统继续接入。
- 当前本地验证组织没有真实团队工单数据，非空团队趋势已通过单元测试覆盖聚合输入路径，生产仍需接入真实 ITSM/LDAP 数据做端到端验证。

待继续：

- 接入企业组织架构团队主数据或 LDAP/IdP 团队同步，把团队维度从启发式提取升级为强归属。
- 扩展团队层级钻取、团队负责人和通知/升级路由。
- 扩展自助目录策略的版本历史、审批变更留痕和差异审计。
- 继续对接外部 ITSM 真实系统联调、独立队列报表和死信重放页面。

### 16.218 2026-06-23 V1.2 P0 ITSM 自助目录策略版本历史/差异审计第一阶段

状态：已完成 ITSM 自助目录策略版本历史和差异审计第一阶段；组织级自助目录策略不再只是覆盖当前配置，而是每次更新/恢复默认都会形成版本、操作人、时间、字段级差异和事件中心审计记录。

已完成：

- 后端新增自助目录策略历史存储：
  - 历史记录存储在 `iac_system_cfg`，使用组织维度 key 前缀 `CLOUD_ITSM_CATALOG_POLICY_HISTORY_<orgId>`。
  - 每个服务项最多保留最近 50 条历史记录，避免无限增长。
  - 历史记录包含 `version`、`action`、`key`、`policyName`、`updatedAt`、`updatedBy`、`beforeSnapshot`、`afterSnapshot` 和 `diff`。
- 后端新增字段级差异审计：
  - 差异字段覆盖 `enabled`、`policyName`、`policyDescription`、`requiredRoles`、`allowedScopes` 和 `slaMinutes`。
  - 组织级策略覆盖会在当前策略响应中返回 `policyVersion`、`policyUpdatedAt`、`policyUpdatedBy` 和 `policyLastDiff`。
  - 恢复默认也会写入历史版本，返回本次 reset 的版本和差异。
- 后端新增查询接口：
  - 新增 `GET /api/v1/cloud/itsm/catalog-policies/:key/history`。
  - 支持 `limit` 参数，默认返回 20 条，最大 100 条，按最新版本优先返回。
- 事件中心审计：
  - 策略更新写入 `itsm.catalog_policy.updated`。
  - 恢复默认写入 `itsm.catalog_policy.reset`。
  - 事件 payload 复用历史记录，便于后续通知订阅、审批留痕和外部审计。
- 前端 ITSM 自助目录增强：
  - 自助目录策略列展示当前策略版本号。
  - “配置自助目录策略”弹窗新增“策略变更历史”表。
  - 历史表展示版本、动作、变更人、变更时间和差异字段。
  - 事件类型中文文案新增“ITSM自助目录策略更新”和“ITSM自助目录策略恢复默认”。

验证：

- Docker Go 1.26 镜像内执行目标测试通过：
  - `go test ./portal/apps -run "TestCloudItsm(CatalogPolicyForSelfService|CatalogPolicyOverride|CatalogPolicyHistoryDiff|SelfServicePolicyPayload)|TestCloudEventItsm"`。
- 新增/扩展测试：
  - `TestCloudItsmCatalogPolicyOverride` 覆盖策略版本、更新时间、操作人和最近差异回填。
  - `TestCloudItsmCatalogPolicyHistoryDiff` 覆盖历史 entry、字段级差异、响应转换和版本递增。
- `git diff --check` 覆盖本阶段相关后端、前端和 PRD 文件，通过。
- `docker compose build iac-portal iac-web` 成功：
  - 后端完成 Swagger 生成和 Go 编译。
  - 前端完成 vendor 和业务包构建，仅保留既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功，`/api/v1/check` 返回 `success=true`。
- 内置浏览器验证 ITSM 页面：
  - `/org/org-d8sm0ghqn3ks73blu1ig/m-cloud-itsm` 正常加载。
  - “自助目录”页签展示 14 个“策略”入口。
  - 点击 `refresh_metadata` 服务项策略入口后，弹窗展示“策略变更历史”表；当前验证组织暂无历史记录时展示空态。

验证限制：

- 第一阶段历史存储仍沿用 `iac_system_cfg` JSON；后续如需强查询、强审计和审批链路，可拆分为独立策略版本表。
- 字段级 diff 已覆盖核心策略字段，尚未扩展到审批人、审批规则、通知路由和团队例外策略。
- 本地浏览器验证未创建样例历史记录，避免在验证组织留下额外审计数据；写入链路由目标单测和容器构建覆盖。

待继续：

- 将自助目录策略变更接入审批流，支持策略变更前审批、审批意见和 PR/工单证据留痕。
- 扩展自助目录策略的团队/项目例外规则、通知/升级路由和策略版本对比视图。
- 继续对接外部 ITSM 真实系统联调、独立队列报表和死信重放页面。

### 16.219 2026-06-23 V1.2 P0 ITSM 失败补偿队列报表细分第一阶段

状态：已完成 ITSM 失败补偿队列报表细分第一阶段；失败提交补偿不再只提供总数和明细列表，而是新增按连接器、失败原因、失败年龄和最近死信的独立报表视图，支撑外部 ITSM 联调期间快速定位补偿积压来源。

已完成：

- 后端新增失败补偿队列报表接口：
  - 新增 `GET /api/v1/cloud/itsm/tickets/retry-queue/report`。
  - 报表返回 `connectorBreakdown`、`reasonBreakdown`、`ageBuckets` 和 `recentDeadLetters`。
  - 连接器维度展示连接器 ID、名称、provider、失败总数、到期可重试、等待退避、死信、跳过和下一次重试时间。
  - 失败原因维度覆盖 `due`、`not_due`、`max_attempts_reached`、`connector_missing`、`connector_disabled`、`disabled`、`no_endpoint`、`missing_request_payload`、`external_identity_present`、`dead_letter`、`status_not_failed` 和 `unknown`。
  - 失败年龄维度按 `<1h`、`1-6h`、`6-24h`、`>24h` 和 `unknown` 聚合。
  - 最近死信保留最新 8 条，便于页面直接查看和重放。
- 后端队列 item 构造增强：
  - `cloudItsmSubmitRetryQueueItem` 在测试聚合路径下支持 nil `ServiceContext`，避免单元测试依赖数据库 lookup。
  - 线上路径仍保留连接器、操作任务、项目、环境和创建人的名称回填。
- 前端 ITSM “失败补偿队列”页签增强：
  - 新增“按连接器”“按失败原因”“按失败年龄”三张细分报表。
  - 新增“最近死信”表，展示标题、连接器、原因、尝试次数、更新时间，并支持直接重放。
  - 批量重试、单条重放、自助申请、连接器保存/删除和刷新操作后同步刷新队列 summary、报表和列表。

验证：

- Docker Go 1.26 镜像内执行目标测试通过：
  - `go test ./portal/apps -run "TestCloudItsm(SubmitRetryQueueStatus|SubmitRetryQueueReportBreakdown|ForcedDeadLetterReplayEligibility|CatalogPolicyHistoryDiff|CatalogPolicyOverride)|TestCloudEventItsm"`。
- 新增 `TestCloudItsmSubmitRetryQueueReportBreakdown`，覆盖到期可重试、等待退避、达到最大尝试次数死信、连接器缺失跳过、连接器维度统计、失败原因统计、失败年龄桶顺序和最近死信列表。
- `git diff --check` 覆盖本阶段相关后端、前端和 PRD 文件，通过。
- `docker compose build iac-portal iac-web` 成功：
  - 后端完成 Swagger 生成和 Go 编译。
  - 前端完成 vendor 和业务包构建，仅保留既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功，`/api/v1/check` 返回 `success=true`。
- 内置浏览器验证 ITSM 页面：
  - `/org/org-d8sm0ghqn3ks73blu1ig/m-cloud-itsm` 正常加载。
  - “失败补偿队列”页签展示“按连接器”“按失败原因”“按失败年龄”“最近死信”。
  - 当前验证组织暂无失败队列数据，报表表格和死信表展示空态，页面无横向溢出。

验证限制：

- 第一阶段仍基于本地失败工单队列表做报表聚合，真实 Jira/ServiceNow 等外部 ITSM 端到端失败样本仍需接入后继续验证。
- 最近死信支持单条重放，尚未扩展批量死信重放、死信关闭、人工确认和重放审批。
- 报表维度暂按连接器、失败原因和失败年龄聚合，尚未扩展团队、项目、申请类型、错误码和外部系统响应码维度。

待继续：

- 对接真实外部 ITSM 系统，补充端到端提交失败、退避、死信、重放和状态同步样本。
- 扩展死信批量重放、人工确认关闭、审批留痕和重放操作审计。
- 将失败补偿队列报表继续扩展到团队、项目、申请类型、错误码和外部响应码维度。

### 16.220 2026-06-23 V1.2 P0 ITSM 死信批量重放/人工关闭审计第一阶段

状态：已完成 ITSM 死信批量重放、人工关闭和审计第一阶段；失败补偿队列中的死信不再只能单条重放，运维人员可以批量选择死信进行强制重放，或在业务确认无需继续提交时人工关闭并形成事件中心审计。

已完成：

- 后端新增死信批量处置接口：
  - 新增 `POST /api/v1/cloud/itsm/tickets/retry-queue/dead-letters/action`。
  - 单次最多处理 50 条死信，支持 `action=replay` 和 `action=close`。
  - 仅允许处置 `dead_letter` 队列状态的工单，非死信、无权限或不存在的 ID 会进入 `skipped/errors` 明细。
  - `replay` 会复用单条失败提交重放逻辑，并在死信场景下使用强制重放 eligibility。
  - `close` 会把工单状态改为 `canceled`，写入 `closed_at`、`last_synced_at` 和 `responsePayload.submitRetry` 审计字段。
- 死信人工关闭审计字段：
  - `deadLetter=true`。
  - `deadLetterClosed=true`。
  - `deadLetterClosedAt`。
  - `deadLetterClosedBy`。
  - `deadLetterCloseReason`。
  - `deadLetterAction=close`。
- 连接器缺失/停用死信处置边界增强：
  - `submitRetry.deadLetter=true` 现在优先作为死信状态信号。
  - 当连接器已删除或停用时，队列仍识别为 `dead_letter`，人工关闭可以完成。
  - 缺失或停用连接器的死信重放仍会被连接器校验拦截，避免误发外部请求。
- 事件中心审计：
  - 单条人工关闭写入 `itsm.ticket.dead_letter_closed`。
  - 批量重放写入 `itsm.dead_letter.batch_replayed`。
  - 批量关闭写入 `itsm.dead_letter.batch_closed`。
  - 前端事件中心补充以上事件类型中文文案。
- 前端 ITSM “失败补偿队列”页签增强：
  - “最近死信”表支持多选。
  - 最近死信区域新增“批量重放”“人工关闭”。
  - 主队列表新增死信多选，并提供“批量重放死信”“人工关闭死信”。
  - “人工关闭死信”弹窗要求填写关闭原因，提交后刷新队列列表、summary、报表和工单概览。

验证：

- Docker Go 1.26 镜像内执行目标测试通过：
  - `go test ./portal/apps -run "TestCloudItsm(SubmitRetryQueueStatus|SubmitRetryQueueReportBreakdown|DeadLetterClosePayload|DeadLetterActionResultErrors|ForcedDeadLetterReplayEligibility|CatalogPolicyHistoryDiff|CatalogPolicyOverride)|TestCloudEventItsm"`。
- 新增/扩展测试：
  - `TestCloudItsmDeadLetterClosePayload` 覆盖人工关闭 payload 审计字段和默认关闭原因。
  - `TestCloudItsmDeadLetterActionResultErrors` 覆盖批量操作结果和 errors 明细。
  - `TestCloudItsmSubmitRetryQueueStatus` 补充连接器缺失时 `deadLetter=true` 仍识别为 `dead_letter` 的边界。
- `docker compose build iac-portal` 成功，后端完成 Swagger 生成和 Go 编译。
- `docker compose up -d iac-portal` 成功；`/api/v1/check` 返回 `success=true`、`build=docker-compose`、`version=v1.3.5`。
- API 端到端验证：
  - 插入临时工单 `cit-codex-dl-miss-0623`，连接器 ID 指向不存在的 `citc-missing-codex`，但 `response_payload.submitRetry.deadLetter=true`。
  - 调用批量人工关闭接口返回 `closed=1`、`failed=0`、`skipped=0`。
  - 数据库确认工单 `status=canceled`，`deadLetterClosed=true`，`deadLetterCloseReason=codex missing connector close validation`，`deadLetterClosedBy` 为当前管理员用户，`closed_at` 已写入。
  - 事件中心写入 `itsm.ticket.dead_letter_closed`，payload `ticketId` 指向该临时工单。
  - 验证后已清理临时工单和事件，剩余临时数据计数均为 `0`。

验证限制：

- 第一阶段只做死信批量重放和人工关闭，不引入独立死信审批流。
- 外部 Jira/ServiceNow/通用 HTTP 的真实失败样本仍需接入后继续做端到端重放验证。
- 人工关闭原因已审计，但尚未扩展关闭原因字典、二次确认审批和关闭通知策略。

待继续：

- 将死信重放/关闭接入审批流，支持审批意见、审批人和外部证据留痕。
- 扩展死信维度到团队、项目、申请类型、错误码和外部系统响应码。
- 对接真实外部 ITSM 系统，补齐提交失败、退避、死信、重放、关闭和状态同步的完整样本。

### 16.221 2026-06-23 V1.2 P0 云资产列表响应式排版修复

状态：已完成云资产列表响应式排版修复；多云资产筛选项和宽表格在窄视口下不再把页面挤乱，表格横向宽度由自身滚动区域承接。

已完成：

- 云资产列表复用 `resource-query` 容器时新增云资产专用布局类。
- 筛选栏从宽度平分的 flex 布局升级为云资产模式下的自适应 grid：
  - 中窄视口下可稳定展示两列左右筛选控件，避免 150px 级别小格压缩输入框。
  - 筛选控件宽度在 `807px` 视口下约 `276px`，操作按钮自然换行。
- 资产表格增加外层宽度约束容器：
  - 页面主体不再被 `2622px` 宽表格撑出横向滚动。
  - 表格内部 `.ant-table-content` 保留横向滚动，继续支持所有资产列。
- 样式约束仅作用于云资产列表，不改变普通 CMDB 资产查询页的行为。

验证：

- `git diff --check` 覆盖 `resource-query` 前端改动，通过。
- `docker compose build iac-web` 成功，前端 production build 通过，仅保留既有 webpack bundle size 警告。
- `docker compose up -d iac-web` 成功。
- 内置浏览器验证 `/org/org-d8sm0ghqn3ks73blu1ig/m-cloud-assets`：
  - 页面视口宽度 `807px` 时，document 宽度保持 `807px`，没有被宽表格撑出整页横向滚动。
  - 筛选栏为 `grid`，两列宽度约 `276px`。
  - 表格容器可视宽度约 `559px`，表格内容宽度约 `2622px`，横向滚动由 `.ant-table-content` 承接。
  - 页面展示“资产列表 / 应用依赖 / 云采集”和资产覆盖、Kubernetes 集群、治理报表入口，无 JS 渲染异常。

验证限制：

- 本阶段修复的是现有宽列排版问题，不调整资产列字段数量和列宽策略。
- 后续若资产列继续增加，可考虑列显隐配置、固定列和表格密度切换。

### 16.222 2026-06-23 V1.2 P0 ITSM 死信处置审批留痕第一阶段

状态：已完成 ITSM 死信重放/关闭接入操作任务审批流第一阶段；死信处置现在既可直接批量执行，也可先提交审批，审批通过后复用统一死信批量处置逻辑执行，并在操作任务、审计和事件中心留下审批人、审批意见、申请原因和外部证据链接。

已完成：

- 后端新增死信处置审批申请接口：
  - `POST /api/v1/cloud/itsm/tickets/retry-queue/dead-letters/approval`。
  - 支持 `action=replay` 和 `action=close`。
  - 单次最多 50 条，创建审批前校验所选工单当前为 `dead_letter`。
  - 支持 `reason` 和 `evidenceUrl`，写入 `operation.params.itsmDeadLetter.evidence`。
- 审批流复用 `iac_cloud_operation`：
  - 新增操作动作 `itsm_dead_letter`。
  - 审批单状态为 `approving`，类型为 `self_service`。
  - 审批通过后写入 `approval.approverId`、`approval.comment`、`approval.approvedAt`。
  - 审批驳回写入 `rejected` 状态和操作任务审计。
- 审批通过执行：
  - `close` 会执行死信人工关闭逻辑，工单更新为 `canceled`。
  - `replay` 会按死信强制重放路径执行。
  - 执行结果写入 `operation.result`，包括 `closed/replayed/submitted/failed/skipped/errors`。
- 事件中心审计：
  - `itsm.dead_letter.approval_requested`。
  - `itsm.dead_letter.approval_rejected`。
  - `itsm.dead_letter.approval_executed`。
  - `itsm.dead_letter.approval_execute_failed`。
- 前端增强：
  - ITSM “失败补偿队列”的“最近死信”区域新增“重放审批”“关闭审批”。
  - 主队列工具栏新增“提交重放审批”“提交关闭审批”。
  - 新增审批申请弹窗，要求填写申请原因，可选填写外部证据链接。
  - 操作任务页补充 `itsm_dead_letter` 和 `self_service` 的中文展示。
  - 事件中心和通知事件类型字典补充死信审批相关事件中文文案。

验证：

- Docker Go 1.26 镜像内执行目标测试通过：
  - `go test ./portal/apps -run "TestCloudItsm(SubmitRetryQueueStatus|SubmitRetryQueueReportBreakdown|DeadLetterClosePayload|DeadLetterActionResultErrors|DeadLetterApprovalPayloadHelpers|ForcedDeadLetterReplayEligibility|CatalogPolicyHistoryDiff|CatalogPolicyOverride)|TestCloudEventItsm"`。
- 新增 `TestCloudItsmDeadLetterApprovalPayloadHelpers`，覆盖审批 payload 的 ticketIds JSON 往返解析、项目/环境作用域折叠和资源 ID 摘要。
- `git diff --check` 通过。
- `docker compose build iac-portal iac-web` 成功：
  - 后端完成 Swagger 生成和 Go 编译。
  - 前端完成 vendor 和业务包构建，仅保留既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功；`/api/v1/check` 返回 `success=true`。
- API 端到端验证：
  - 插入临时死信工单 `cit-codex-dlappr-0623`。
  - 调用审批申请接口返回操作任务 `cop-d8t0btfcvpms73bhsc6g`，状态为 `approving`，params 中包含申请原因和 `https://example.com/change/codex-dead-letter` 外部证据链接。
  - 调用 `POST /api/v1/cloud/operations/{id}/approve` 审批通过后，操作任务状态变为 `complete`，result 中 `action=close`、`closed=1`，审批意见为 `codex approval execution validation`。
  - 数据库确认临时工单 `status=canceled`，`deadLetterClosed=true`，关闭原因为 `codex close approval validation`，关闭人为当前管理员用户。
  - 操作审计包含创建审批、审批通过、开始执行和完成四段记录。
  - 事件中心写入审批申请、死信关闭、操作完成和审批执行完成事件。
  - 验证后已清理临时工单、操作任务、步骤、审计和事件，剩余临时数据计数均为 `0`。
- 内置浏览器验证：
  - `/org/org-d8sm0ghqn3ks73blu1ig/m-cloud-assets` 在 `807px` 视口下 document 宽度保持 `807px`，宽表格由内部横向滚动承接。
  - `/org/org-d8sm0ghqn3ks73blu1ig/m-cloud-itsm` 的“失败补偿队列”页签可见“重放审批”“关闭审批”“提交重放审批”“提交关闭审批”。

验证限制：

- 当前审批执行复用平台内置操作任务审批，不对接外部审批系统。
- 真实 Jira/ServiceNow/通用 HTTP 的重放成功路径仍需要外部系统样本继续验证。
- 审批申请已支持外部证据链接；多证据、审批快照和通知策略元数据已在 16.224 完成第一阶段。

### 16.223 2026-06-23 V1.2 P0 ITSM 失败补偿队列报表多维细分第一阶段

状态：已完成 ITSM 失败补偿队列报表多维细分第一阶段；失败提交补偿报表在连接器、失败原因和失败年龄之外，新增团队、项目、申请类型、错误码和外部响应码维度，方便外部 ITSM 联调和死信治理按责任组织、业务上下文和外部系统反馈快速定位。

已完成：

- 后端 `GET /api/v1/cloud/itsm/tickets/retry-queue/report` 响应新增：
  - `projectBreakdown`：按项目聚合失败补偿队列，未关联项目统一归入 `unassigned/未关联项目`。
  - `teamBreakdown`：复用 ITSM overview 的团队推导逻辑，优先从 payload 中的 `team/businessTeam/requestTeam/ownerTeam` 等字段提取，无法识别时按项目或未标记团队兜底。
  - `requestTypeBreakdown`：按自助目录/操作类型聚合，并复用目录中文名称，如 GitOps/IaC 变更申请、权限申请等。
  - `errorCodeBreakdown`：从响应载荷中的 `errorCode/code/error.code/json.error.code/submitRetry.errorCode` 等路径提取外部错误码；无专用错误码时按 HTTP 状态码或重试原因兜底。
  - `externalResponseCodeBreakdown`：从 `statusCode/httpStatus/response.statusCode/json.statusCode/submitRetry.lastStatusCode` 等路径提取外部 HTTP 响应码，统一展示为 `HTTP xxx`。
- 报表聚合仍复用现有队列 item 构造和 due/future/dead_letter/skipped 计数逻辑：
  - 新维度同时保留失败总数、到期可重试、等待退避、死信、跳过、最早到期时间和下次重试时间。
  - 单测路径支持 `ServiceContext=nil`，项目名回退项目 ID，避免测试依赖数据库 lookup。
- 前端 ITSM “失败补偿队列”页签增强：
  - 新增“按项目”“按团队”“按申请类型”“按错误码”“按外部响应码”五张细分报表。
  - 报表网格改为 `auto-fit + minmax(320px, 1fr)`，支持 8 个维度在不同屏幕宽度下自动排布。

验证：

- Docker Go 1.26 镜像内执行目标测试通过：
  - `go test ./portal/apps -run "TestCloudItsm(SubmitRetryQueueStatus|SubmitRetryQueueReportBreakdown|DeadLetterClosePayload|DeadLetterActionResultErrors|DeadLetterApprovalPayloadHelpers|ForcedDeadLetterReplayEligibility|CatalogPolicyHistoryDiff|CatalogPolicyOverride)|TestCloudEventItsm"`。
- `TestCloudItsmSubmitRetryQueueReportBreakdown` 已扩展覆盖：
  - 项目维度 `p-platform` 和 `unassigned`。
  - 团队维度 `team-platform/平台团队`。
  - 申请类型维度 `gitops_iac_change/GitOps/IaC 变更申请`。
  - 错误码维度 `ITSM_TIMEOUT`、`RATE_LIMIT`、`MAX_ATTEMPTS`。
  - 外部响应码维度 `HTTP 503` 和未返回响应码。
- `git diff --check` 覆盖本阶段相关后端、前端和 PRD 文件，通过。
- `docker compose build iac-portal iac-web` 成功：
  - 后端完成 Swagger 生成和 Go 编译。
  - 前端完成 vendor 和业务包构建，仅保留既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功；`/api/v1/check` 返回 `success=true`。
- API 验证：
  - 使用本地管理员登录 token 和 `IaC-Org-Id: org-d8sm0ghqn3ks73blu1ig` 请求 `/api/v1/cloud/itsm/tickets/retry-queue/report`。
  - 响应中 `connectorBreakdown`、`reasonBreakdown`、`ageBuckets`、`projectBreakdown`、`teamBreakdown`、`requestTypeBreakdown`、`errorCodeBreakdown`、`externalResponseCodeBreakdown`、`recentDeadLetters` 均为数组字段。
  - 当前验证组织暂无失败补偿队列数据，所有维度数组为空态。
- 内置浏览器验证：
  - `/org/org-d8sm0ghqn3ks73blu1ig/m-cloud-itsm` 正常加载。
  - “失败补偿队列”页签可见“按连接器”“按失败原因”“按失败年龄”“按项目”“按团队”“按申请类型”“按错误码”“按外部响应码”“最近死信”。
  - `807px` 视口下 document 宽度保持 `807px`，没有页面级横向溢出。

验证限制：

- 错误码和响应码解析覆盖通用 HTTP/Jira/ServiceNow 常见 JSON 路径；真实外部系统若返回数组型错误结构或 provider 专用字段，还需要在真实样本接入后继续扩展映射。
- 当前报表仍基于平台内失败工单队列聚合；真实外部 ITSM 端到端失败、重放成功和状态同步样本仍待继续补测。

### 16.224 2026-06-23 V1.2 P0 ITSM 死信审批多证据、快照和通知策略第一阶段

状态：已完成 ITSM 死信审批证据留痕增强第一阶段；死信重放/关闭审批不再只有单个外部链接，而是支持主证据链接、补充证据列表、审批时刻票据快照和通知策略元数据，审批驳回、审批执行和事件中心可继续复用同一份 payload。

已完成：

- 后端审批申请表单新增 `evidenceItems`：
  - 每条证据支持 `type`、`label`、`url`、`note`。
  - 兼容旧字段 `evidenceUrl`，会自动合并为第一条 `link` 证据。
  - URL 统一校验 `http/https`，单次最多 10 条证据。
- 审批 payload 增强：
  - `operation.params.itsmDeadLetter.evidence.items` 保存证据列表。
  - `operation.params.itsmDeadLetter.evidence.snapshot` 固化审批时刻的 action、reason、precheck 和票据快照。
  - 票据快照包含工单 ID、连接器、项目/环境、外部单号、状态、重试状态、错误码和外部响应码。
  - `snapshotVersion=v1`、`snapshotGeneratedAt` 用于后续审计兼容。
- 通知策略元数据：
  - `operation.params.itsmDeadLetter.notificationStrategy` 和顶层 `operation.params.notificationStrategy` 均写入。
  - 包含事件类型、通知渠道、owner roles、证据快照开关、证据条数、票据数和消息模板。
- 前端审批弹窗增强：
  - `外部证据链接` 调整为 `主证据链接`。
  - 新增“补充证据”列表，可添加证据标题、类型、链接和说明。
  - 提交前过滤空证据行，并通过 `evidenceItems` 发送给后端。

验证：

- Docker Go 1.26 镜像内执行目标测试通过：
  - `go test ./portal/apps -run "TestCloudItsm(SubmitRetryQueueStatus|SubmitRetryQueueReportBreakdown|DeadLetterClosePayload|DeadLetterActionResultErrors|DeadLetterApprovalPayloadHelpers|DeadLetterApprovalEvidencePayload|DeadLetterApprovalEvidenceRejectsInvalidURL|ForcedDeadLetterReplayEligibility|CatalogPolicyHistoryDiff|CatalogPolicyOverride)|TestCloudEventItsm"`。
- 新增单元测试覆盖：
  - 主证据链接和补充证据合并为 `evidence.items`。
  - 非 `http/https` 证据链接被拒绝。
  - 证据条数上限生效。
  - 审批快照包含票据重试原因、HTTP 502 错误码和外部响应码。
  - 通知策略包含 4 类死信审批事件和 `includeEvidenceSnapshot=true`。
- `git diff --check` 覆盖本阶段后端、前端和 PRD 文件，通过。
- `docker compose build iac-portal iac-web` 成功：
  - 后端完成 Swagger 生成和 Go 编译。
  - 前端完成 vendor 和业务包构建，仅保留既有 webpack bundle size 警告。
- `docker compose up -d iac-portal iac-web` 成功；`/api/v1/check` 返回 `{"success":true,"build":"docker-compose","version":"v1.3.5"}`。
- 内置浏览器端到端验证：
  - 插入临时死信工单 `cit-codex-evidence-0623`。
  - `/org/org-d8sm0ghqn3ks73blu1ig/m-cloud-itsm` “失败补偿队列”出现死信记录，报表中错误码和外部响应码均显示 `HTTP 502`。
  - 选中死信后“重放审批”可用。
  - 审批弹窗展示“主证据链接”“补充证据”“添加证据”，可填写补充证据标题、链接和说明。
  - 提交后生成操作任务 `approving`，数据库确认：
    - `evidence.itemCount=2`。
    - `evidence.url=https://gitlab.example.com/platform/iac/-/merge_requests/42`。
    - `evidence.snapshot.ticketCount=1`。
    - `evidence.snapshot.tickets[0].errorCode.id=http_502`。
    - `notificationStrategy.includeEvidenceSnapshot=true`。
    - `notificationStrategy.evidenceItemCount=2`。
  - 验证后已清理临时死信工单、操作任务和相关事件，剩余临时数据计数为 `0`。

验证限制：

- 本阶段保存的是平台审批 payload 中的证据元数据和快照；外部 ITSM 系统原生附件上传、附件下载和附件生命周期管理仍待真实系统联调后继续扩展。
- 通知策略元数据已写入操作参数，实际企业微信/钉钉/Slack/邮件路由仍复用后续通知策略能力配置。

### 16.225 2026-06-23 V1.2 P0 操作任务死信审批审计展示第一阶段

状态：已完成操作任务详情页的 ITSM 死信审批审计展示第一阶段；死信审批任务不再只依赖原始 JSON 参数排查，审批人和审计人员可以在操作任务详情中直接查看证据列表、审批快照、票据快照和通知策略。

已完成：

- 操作任务详情识别 `action=itsm_dead_letter` 的自助运维任务：
  - 类型展示补充 `self_service=自助运维`。
  - 动作展示补充 `itsm_dead_letter=ITSM 死信处置`。
  - `approving` 状态的死信处置任务展示“审批通过”和“驳回任务”，与后端审批执行能力一致。
- 新增“ITSM 死信审批留痕”展示区：
  - 汇总处置动作、票据数、强制执行、申请人、申请时间、主证据和申请原因。
  - 证据列表展示证据标题、类型、链接和说明，支持主证据链接、PR/评审、外部工单、变更记录等类型中文化。
  - 审批快照展示快照版本、生成时间、预检查 JSON 和票据快照。
  - 票据快照展示工单标题、状态、重试原因、错误码、外部响应码、外部单号和外部链接。
  - 通知策略展示启用状态、级别、渠道、路由、Owner 角色、事件类型、证据快照开关、证据数、票据数和消息模板。
- 前端样式增强：
  - 证据链接支持长 URL 自动换行。
  - 证据/票据宽表在抽屉内横向滚动，不撑开页面。
  - 通知策略标签支持换行展示。

验证：

- `git diff --check` 覆盖操作任务前端组件和样式文件，通过。
- `docker compose build iac-web` 成功，前端完成 vendor 和业务包构建，仅保留既有 webpack bundle size 警告。
- `docker compose up -d iac-web` 成功，前端容器重启并运行。
- 内置浏览器验证 `/org/org-d8sm0ghqn3ks73blu1ig/m-cloud-operations?operationId=cop-codex-audit0623`：
  - 插入临时 `self_service/itsm_dead_letter/approving` 操作任务，params 中包含 2 条证据、审批快照、票据快照和通知策略。
  - 详情抽屉可见“ITSM 死信审批留痕”“证据列表”“审批快照”“通知策略”。
  - 详情抽屉可见“审批通过”和“驳回任务”。
  - 证据链接 `https://gitlab.example.com/platform/iac/-/merge_requests/42`、错误码 `HTTP 502`、通知策略事件类型均在页面展示。
  - 验证后已删除临时操作任务，剩余计数为 `0`。

验证限制：

- 本阶段只做平台操作任务详情页展示，不新增外部 ITSM 附件上传、下载或保留策略。
- 临时样本验证覆盖前端展示与现有操作详情 API 返回结构；真实 Jira/ServiceNow 等外部系统仍需在端到端联调时补充真实 payload 样本回归。

### 16.226 2026-06-23 V1.2 P0 前端 vendor 体积和 Docker 构建速度优化第一阶段

状态：已完成前端 vendor 体积和 Docker 构建缓存优化第一阶段；在不改变页面业务逻辑的前提下，裁剪全量 locale 和未使用全局依赖，并让 Docker 构建能够复用 `npm ci` 层。

已完成：

- 裁剪 `vendor/react/vendors.js`：
  - `webpack.vendor.babel.js` 改为遵循依赖包 `browser` 字段，避免 `react-intl/locale-data/index.js` 和 `intl-relativeformat/lib/locales.js` 的全量 locale 被打进 vendor。
  - 将 `intl-relativeformat` 精确别名到 `intl-relativeformat/lib/main`，保留英文默认数据和运行时核心能力。
  - 使用 `ContextReplacementPlugin` 将 `moment` locale 限定为 `zh-cn`。
  - 从 `app/vendor.js` 移除未被源码使用的全量 `lodash` 全局导出；业务代码继续使用 `lodash/foo` 子模块导入。
- 主应用 webpack 配置同步补充：
  - `resolve.aliasFields=['browser']`、`mainFields=['browser','module','main']`。
  - `intl-relativeformat` 精确别名和 `moment/locale` 裁剪在主包构建中同样生效。
  - 移除 `lodash` 精确 external，避免未来误用 `import 'lodash'` 时依赖不存在的全局变量。
- Dockerfile 构建缓存优化：
  - 先复制 `package.json/package-lock.json` 并执行 `npm ci --legacy-peer-deps`。
  - 再复制源码并执行 `npm run build:vendor && npm run build`。
  - 源码小改但依赖不变时，`npm ci` 层可命中 Docker cache。

验证：

- `git diff --check` 覆盖 `frontend/Dockerfile`、`frontend/app/vendor.js`、`webpack.base.babel.js` 和 `webpack.vendor.babel.js`，通过。
- `docker compose build iac-web` 成功：
  - `vendor/react/vendors.js` 从最近一次构建日志中的 `2.54 MiB` 降至 `368 KiB`。
  - copied `vendor/` 目录从最近一次构建日志中的 `3.45 MiB` 降至 `1.27 MiB`。
  - 主包 `js/vendor` 从最近一次构建日志中的 `4.79 MiB` 降至 `4.48 MiB`。
  - `app` 入口从 `85.2 KiB` 降至 `81.1 KiB`。
  - vendor DLL 编译耗时从最近一次日志约 `22.6s` 降至约 `8.2s`。
- 重复执行 `docker compose build iac-web` 命中缓存，耗时约 `3.6s`，`npm ci` 和前端 build 层均复用缓存。
- `docker compose up -d iac-web` 成功，`iac-web` 运行正常。
- `/api/v1/check` 返回 `{"success":true,"build":"docker-compose","version":"v1.3.5"}`。
- 内置浏览器刷新 `/org/org-d8sm0ghqn3ks73blu1ig/m-org-ct`：
  - 页面正常显示“组织/组织设置：云模板”“云模板”“新建云模板”等内容。
  - 浏览器控制台未捕获 error。

验证限制：

- 本阶段优先做 vendor/locale/Docker 缓存优化，尚未重构路由级代码拆分、图表库按需加载、CodeMirror 按需加载或替换 `moment`。
- `js/vendor` 仍有 `4.48 MiB`，后续可继续拆分 `@ant-design/icons`、`@antv/g6`、CodeMirror、ECharts 和 AntD 相关入口。

### 16.227 2026-06-23 V1.2 P0 前端路由与生产代码分包优化第一阶段

状态：已完成前端路由与生产代码分包优化第一阶段；在不改变业务页面和菜单路径的前提下，为所有页面级动态路由补充稳定 chunk 名，并移除生产构建中把所有依赖强制合并为单一 `vendor` 的配置。

已完成：

- 路由声明优化：
  - `frontend/app/routes.js` 新增 `routeLoadable` helper，统一页面级懒加载兜底配置。
  - 为组织选择、合规、组织壳层、项目环境、多云总览、云账号、云资产、操作任务、风险、成本、事件、ITSM、组织设置和系统/用户页面补充 `webpackChunkName`。
  - `/org/:orgId/m-cloud-assets` 和 `/org/:orgId/m-other-resource` 继续复用同一 `route-cloud-assets` chunk，避免同一资源查询页面重复产物。
- 生产分包策略优化：
  - `webpack.prod.babel.js` 移除 `splitChunks.name='vendor'` 的单包聚合策略。
  - 新增 `vendor-antd`、`vendor-common`、`vendor-async`、`vendor-graph`、`vendor-editor` 和 `route-common` 分组。
  - 图谱库 `@antv/g6`/`d3`/`dagre`/`graphlib`、编辑器 `codemirror`/`react-codemirror2` 只在异步路由侧拆包。
  - 初始入口保留运行时、AntD 公共依赖、初始公共依赖和 app 壳层，路由页面代码继续按需加载。

验证：

- `git diff --check` 覆盖 `frontend/app/routes.js` 和 `frontend/internals/webpack/webpack.prod.babel.js`，通过。
- `docker compose -f backend/docker/docker-compose.yml build iac-web` 成功：
  - Docker 依赖层命中缓存，`npm ci` 未重跑。
  - vendor DLL 仍保持 `368 KiB`。
  - 应用入口从上一阶段的单一 `js/vendor 4.48 MiB` 改为：
    - `vendor-antd`：`1.36 MiB`。
    - `vendor-common`：`275 KiB`。
    - `app`：`93.8 KiB`。
    - `runtime`：`4.97 KiB`。
  - `Entrypoint app` 总计约 `1.73 MiB`，不再把 `vendor-graph`、`vendor-editor` 和其他异步依赖注入首屏。
  - 典型异步路由 chunk：
    - `route-cloud-assets`：约 `188 KiB`。
    - `route-org-ct`：约 `24 KiB`。
    - `route-cloud-operations`：约 `24 KiB`。
  - 生产 webpack 编译通过，仅保留体积 warning；本次完整 web 镜像构建耗时约 `81s`。
- `docker compose -f backend/docker/docker-compose.yml up -d iac-web` 成功，`iac-web` 运行正常。
- `/api/v1/check` 返回 `{"success":true,"build":"docker-compose","version":"v1.3.5"}`。
- 静态文件验证：
  - `/index.html` 仅注入 `runtime`、`vendor-antd`、`vendor-common` 和 `app`。
  - `/js/route-cloud-assets.*.chunk.js`、`/js/route-org-ct.*.chunk.js`、`/js/route-cloud-operations.*.chunk.js` 均返回 HTTP 200。
- 内置浏览器验证：
  - `/org/org-d8sm0ghqn3ks73blu1ig/m-org-ct` 正常渲染“云模板”“新建云模板”“导入”“导出”和表格列。
  - `/org/org-d8sm0ghqn3ks73blu1ig/m-cloud-assets` 正常渲染“云资产”“资产列表”“应用依赖”“云采集”和资产表格列。
  - `/org/org-d8sm0ghqn3ks73blu1ig/m-cloud-operations` 正常渲染“操作任务”“刷新”和任务表格列。
  - 三个路由浏览器控制台均未捕获 error。

验证限制：

- 本阶段主要解决生产 `splitChunks` 单 vendor 聚合和路由产物可读性问题；AntD 本身仍是初始公共依赖，后续可继续推进 icon 按需、布局壳层 AntD 使用收敛和登录页独立依赖拆分。
- 图谱和编辑器已被拆入异步 vendor chunk，但页面组件内部仍可以继续按交互动作做二级懒加载，例如资源详情图谱打开时再加载 G6，策略/模板编辑器打开时再加载 CodeMirror。
- 当前仍保留全局 ECharts vendor 文件，后续若需要进一步压首屏，可评估把 ECharts 从静态 vendor 调整为按页面加载。
