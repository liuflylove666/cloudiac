# CloudIaC 离线依赖镜像方案

## 当前可用性判断

`https://exchange.cloudiac.org/v1/mirrors/providers/` 当前不能作为可靠依赖源使用。

本地验证结果：

- `curl -I -L https://exchange.cloudiac.org/v1/mirrors/providers/` 返回 `Could not resolve host: exchange.cloudiac.org`。
- `nslookup exchange.cloudiac.org` 返回 `NXDOMAIN`。
- 同一网络下 `https://registry.terraform.io/` 可正常返回 200，说明不是本机完全不可联网，而是该域名不可解析。

因此 CloudIaC 部署和执行链路应改为依赖内网镜像，不再直接依赖公网 `exchange.cloudiac.org`、GitHub repo 或 Terraform Registry。`https://github.com/accurics/terrascan/` 作为官方上游下载地址保留，用于在有公网的环境预先同步二进制包。

## 离线化目标

离线化拆成三类资产：

- Terraform providers：放到 worker 镜像内 `/cloudiac/terraform/plugins`，或发布成内网 network mirror。
- Git 代码和云模板：镜像到本地 GitLab，CloudIaC 的 VCS 地址改为本地 GitLab。
- 容器镜像和二进制包：镜像到 Harbor；`terrascan`、`tfenv`、worker/portal/web/base 镜像都走内网。

## 推荐目录

默认下载目录不放在源码树里，避免把大体积二进制和 bare git repo 混入项目提交：

```bash
/data/cloudiac-offline-deps
  git/
    tfutils/tfenv.git
    cloudiac/*.git
  artifacts/
    terrascan/1.9.0/
  terraform/
    providers/
  images/
```

也可以通过脚本参数指定到任意本地目录。

## 下载 GitHub 相关依赖

在有公网访问的机器上执行：

```bash
bash deploy/offline-mirror/mirror-github-deps.sh /data/cloudiac-offline-deps
```

脚本会处理：

- `https://github.com/tfutils/tfenv.git`
- `https://github.com/idcos/cloudiac.git`
- `https://github.com/idcos/cloudiac-web.git`
- `https://github.com/idcos/cloudiac-docs.git`
- `https://github.com/idcos/cloudiac-example.git`
- `https://github.com/accurics/terrascan/releases/download/v1.9.0/terrascan_1.9.0_Linux_x86_64.tar.gz`
- 如果存在 `backend/repos.list`，会按清单镜像内置模板仓库；清单中使用相对路径时必须显式传入 `REPO_BASE`，例如本地 GitLab 模板组地址。

示例：

```bash
REPO_BASE=http://gitlab.local/cloudiac-templates \
bash deploy/offline-mirror/mirror-github-deps.sh /data/cloudiac-offline-deps
```

如果 `idcos` 仓库需要 GitHub 凭证：

```bash
GITHUB_TOKEN=ghp_xxx \
bash deploy/offline-mirror/mirror-github-deps.sh /data/cloudiac-offline-deps
```

如果希望能先下载可访问仓库，再汇总失败项：

```bash
CONTINUE_ON_ERROR=true \
bash deploy/offline-mirror/mirror-github-deps.sh /data/cloudiac-offline-deps
```

也可以使用总控脚本一次完成“下载 + 可选上传”：

```bash
CONTINUE_ON_ERROR=true \
ALLOW_PARTIAL_MIRROR=true \
bash deploy/offline-mirror/sync-offline-deps.sh /data/cloudiac-offline-deps
```

如果同时需要上传到本地 GitLab 和 Harbor/OCI：

```bash
CONTINUE_ON_ERROR=true \
ALLOW_PARTIAL_MIRROR=true \
GIT_REMOTE_BASE=http://gitlab.local/iac-mirrors \
TERRASCAN_OCI_REF=harbor.local/cloudiac-offline/terrascan:1.9.0 \
bash deploy/offline-mirror/sync-offline-deps.sh /data/cloudiac-offline-deps
```

## 上传到本地 GitLab

推荐使用脚本批量推送 `/data/cloudiac-offline-deps/git/**/*.git`：

```bash
GIT_REMOTE_BASE=http://gitlab.local/iac-mirrors \
bash deploy/offline-mirror/push-git-mirrors.sh /data/cloudiac-offline-deps
```

推送前可以先 dry run：

```bash
DRY_RUN=true \
GIT_REMOTE_BASE=http://gitlab.local/iac-mirrors \
bash deploy/offline-mirror/push-git-mirrors.sh /data/cloudiac-offline-deps
```

脚本会保留离线目录下的相对路径，例如：

```bash
/data/cloudiac-offline-deps/git/tfutils/tfenv.git
  -> http://gitlab.local/iac-mirrors/tfutils/tfenv.git
/data/cloudiac-offline-deps/git/idcos/cloudiac-docs.git
  -> http://gitlab.local/iac-mirrors/idcos/cloudiac-docs.git
```

GitLab 项目/分组需要提前创建，除非你的 GitLab 已允许 push-to-create。内置云模板仓库也按同样方式推送到本地 GitLab。推送后在 CloudIaC 中新增/修改代码仓库，地址使用本地 GitLab API 地址和 token，不再使用 GitHub。

## 上传到 Harbor

容器镜像建议直接用 Harbor 作为镜像仓库：

```bash
docker pull cloudiac/ct-worker:latest
docker tag cloudiac/ct-worker:latest harbor.local/cloudiac/ct-worker:latest
docker push harbor.local/cloudiac/ct-worker:latest
```

`terrascan` 这类二进制包如果 Harbor 开启 OCI artifact，可以用脚本上传，脚本内部依赖 `oras`：

```bash
TERRASCAN_OCI_REF=harbor.local/cloudiac-offline/terrascan:1.9.0 \
bash deploy/offline-mirror/push-artifacts-oci.sh /data/cloudiac-offline-deps
```

如果 Harbor 不承载普通文件，就放到 GitLab Package Registry、Nexus 或内网 Nginx 静态目录。

## 使用 AWS ECR 作为镜像仓库

系统设置页支持把镜像仓库类型配置为 `AWS ECR`，并填写 AWS 账号、区域和可选仓库前缀。

ECR 镜像仓库前缀格式：

```bash
123456789012.dkr.ecr.us-east-1.amazonaws.com/
```

如果设置仓库前缀 `platform`，则生成：

```bash
123456789012.dkr.ecr.us-east-1.amazonaws.com/platform/
```

部署侧可以把该地址作为 `DOCKER_REGISTRY`，例如：

```bash
DOCKER_REGISTRY=123456789012.dkr.ecr.us-east-1.amazonaws.com/ \
docker compose --env-file .env -f backend/docker/docker-compose.yml up -d
```

私有 ECR 拉取镜像前，需要在运行 Docker daemon 的机器上登录：

```bash
aws ecr get-login-password --region us-east-1 \
  | docker login --username AWS --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com
```

## Terraform providers 离线镜像

CloudIaC 代码里已有离线机制：

- `backend/assets/terraformrc-offline`：只允许从 `/cloudiac/terraform/plugins` 安装 provider。
- `backend/runner/task.go`：`RUNNER_OFFLINE_MODE=true` 时会让 Terraform 禁止直连官方 registry。
- `backend/scripts/generate-providers-mirror.sh`：可生成 provider filesystem mirror。

生成方式：

```bash
TARGET_DIR=/data/cloudiac-offline-deps/terraform/providers \
PLATFORM=linux_amd64 \
bash backend/scripts/generate-providers-mirror.sh
```

如果模板仓库已经迁移到本地 GitLab，可以把这些仓库 clone 到一个本地目录后逐个执行：

```bash
terraform providers mirror -platform=linux_amd64 /data/cloudiac-offline-deps/terraform/providers
```

部署时有两种选择：

- 打包进 worker 镜像：把 `/data/cloudiac-offline-deps/terraform/providers` 复制到镜像 `/cloudiac/terraform/plugins`。
- 做内网 network mirror：把该目录发布到内网静态服务，并把系统 Registry 地址配置为内网地址。

## CloudIaC 配置建议

- 设置 `RUNNER_OFFLINE_MODE=true`。
- worker 镜像构建不再依赖不可控公网 `git clone github.com`，改为从离线目录 `COPY` 或从 Harbor/GitLab 下载。
- `backend/docker/base/worker/Dockerfile*` 中保留 `accurics/terrascan` 官方 release 地址作为同步种子；生产离线部署时再把已下载的 terrascan 包发布到 Harbor、GitLab Package Registry、Nexus 或内网静态服务。
- 云模板、应用代码、策略仓库统一迁移到本地 GitLab，不保留公网 idcos GitHub 地址作为默认仓库前缀。
- 系统 Registry 地址改为内网 Registry/Mirror 地址，使 `GetRegistryMirrorUrl` 返回内网 `/v1/mirrors/providers/`。

## 后续开发项

- 增加“离线依赖管理”页面：展示 provider mirror、Git 模板镜像、Harbor 镜像、工具包版本和健康检查状态。
- 增加一键健康检查：DNS、HTTP 200、provider index、GitLab clone、Harbor pull、Terraform init dry-run。
- 在 worker Dockerfile 中增加 `OFFLINE_DEPS_DIR` 构建参数，支持纯离线构建。
- 在 PRD 中把公网依赖下线列为部署必选项，而不是可选优化项。
