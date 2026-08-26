# GitHub Actions 构建方案（公开仓，专职 amd64 构建并推 ACR）

## 目标
- 在 GitHub 新建**公开仓库** `withyou-build`（或自选名），只放 Dockerfile + workflow + backend/frontend 源码
- 每次 push main 或手动触发，GitHub Actions 用 buildx 在 ubuntu-latest runner 上构建 linux/amd64 镜像
- 推到 ACR `registry.cn-beijing.aliyuncs.com/withyou_holmose/deeix-chat-withyou` 三 tag：`v2.2.33-amd64` / `latest-amd64` / `latest`

## 你需要做的步骤（约 10 分钟）

### 1. 新建 GitHub 公开仓库
- 打开 https://github.com/new
- Repository name: 例如 `withyou-build`（或别的）
- 选 **Public**
- 不要勾选 "Add a README" / "Add .gitignore"（空白仓库即可）
- 点 "Create repository"
- 记下 URL：`https://github.com/<你的用户名>/withyou-build.git`

### 2. 配置 Secret（ACR 推送凭证）
- 进新建仓库 → Settings → Secrets and variables → Actions → New repository secret
- 添加两条：
  - Name: `ACR_USERNAME`
    Value: 你的阿里云容器镜像服务用户名（一般是阿里云账号全名或子账号 RAM）
  - Name: `ACR_PASSWORD`
    Value: 对应密码（或 Registry 专用访问凭证的密码）
- 没有凭证？到 https://cr.console.aliyun.com → 个人实例 → 访问凭证 → 设置/重置固定密码

### 3. 推送代码（从本工作区执行）
我会在本机把仓库打包成 `.tmp/build-pipeline/withyou-build-source.tar.gz`，包含：
- `Dockerfile`（全局 ARG GO_IMAGE 化，amd64构建走官方 golang）
- `backend/` 全部源码
- `frontend/out/` 53MB 静态产物（gitignored，已从我工作区现成拷贝）
- `.github/workflows/build-and-publish-amd64.yml`（上面的 workflow）

### 4. 触发构建
- 进仓库 → Actions → "build-and-publish-amd64"
- 点 "Run workflow" → Run workflow（手动触发）
- 等约 5-15 分钟完成

## 关键约束
- Dockerfile 第一个 ARG 必须是全局（已在文件最顶 `ARG GO_IMAGE=...`）
- 默认 GO_IMAGE 为 `golang:1.26-bookworm`（官方源，runner 网络可达）；如遇镜像拉取问题可在 workflow dispatch 输入自定义
- 基础镜像未做 apt 源替换；GHA ubuntu runner 国际网络通常够用，若卡源可在 workflow 加 build args 注入前置 sed 命令
- 仅推 amd64（与 HK / 上海 / 你后续 amd64 服务器对齐）；arm64 如需要另起 workflow
- frontend/out 已在 tar 包内；若你之后改了前端源码需本地 pnpm build 后重新打包 frontend/out 再 push

## 完成后
- HK 服务器：`cd /opt/deeix-chat-withyou && docker compose pull app && docker compose up -d app`
- 验证：`curl -s http://127.0.0.1:8080/api/v1/version` 应回 `0.3.4-41f8558...`
- WECHAT_BOT_PER_CONTACT_SESSIONS=true env 我已加在 `/opt/deeix-chat-withyou/docker-compose.yml`（备份 .bak-20260826）