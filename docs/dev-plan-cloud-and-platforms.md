# 多平台云端打包：开发建议

目标：在 Windows GUI 选参数后，把任务提交到 GitHub Actions（先 macOS DMG，后 Android），本地只负责任务编排与产物下载。  
原则：**按平台 / 按任务分目录**，避免多次任务与多平台文件混在一起。

> 面向用户的平台说明请优先看：[README.md](./README.md)、[windows.md](./windows.md)、[macos.md](./macos.md)、[android.md](./android.md)。  
> 项目根目录还有总览：[../README.md](../README.md)。  
> 本文偏**开发与目录约定**。

---

## 1. 推荐目录结构

```text
pake-gui/
├── tasks/                          # 任务规格书（给人看，按任务编号）
│   ├── README.md                   # 任务索引与状态
│   ├── T01-local-windows/          # 本地 Windows 打包（已基本完成）
│   ├── T02-github-macos-cloud/     # 下一阶段：GUI → Actions → DMG
│   └── T03-android-cloud/          # 安卓云端 debug APK
│
├── internal/
│   ├── pake/                       # 现有：本地 pake-cli 调用（Windows 主路径）
│   ├── cloud/                      # 云端编排（与本地打包解耦）
│   │   ├── common/                 # Job 模型、状态机、产物落盘约定
│   │   ├── github/                 # Token、workflow_dispatch、Contents、Artifacts
│   │   └── android/                # 文档包：说明走共享 RunCloudJob + android-shell
│   └── server/                     # HTTP API + 前端；只做路由，不塞平台细节
│
├── configs/                        # 可复用的打包参数模板（按平台）
│   ├── windows/
│   ├── macos/
│   └── android/                    # 预留
│
├── builds/                         # 最终产物（按平台分子目录；运行时创建）
│   ├── windows/
│   ├── macos/
│   └── android/
│
├── data/                           # 本地运行时（已在 .gitignore）
│   ├── cloud-jobs/                 # 每次云端任务一个子文件夹
│   │   └── {jobId}/
│   │       ├── request.json        # 提交参数快照
│   │       ├── icon.*              # 本任务图标副本
│   │       ├── remote.json         # run_id / 远程图标路径等
│   │       ├── status.json         # queued|running|success|failed
│   │       └── logs.txt            # 轮询摘要
│   ├── github.json                 # 仓库、Token（勿提交）
│   └── icons/                      # 通用图标缓存（现有）
│
└── .github/workflows/
    ├── build-macos.yml             # 已有
    ├── build-windows.yml           # Windows exe（iterative-build）
    └── build-android.yml           # Android debug APK（android-shell）
```

### 为什么这样分

| 层级 | 作用 |
|------|------|
| `tasks/Txx-*` | 需求、验收、接口草案，多次迭代不挤在一个 md 里 |
| `internal/cloud/*` | 代码按「云厂商 / 平台」分，避免 `server.go` 膨胀 |
| `data/cloud-jobs/{jobId}/` | 每次提交独立目录，历史任务可追溯、可重试 |
| `builds/{platform}/` | 下载的 DMG / APK 不混放 |
| `configs/{platform}/` | 模板与真实任务数据分离 |

---

## 2. 任务拆分（建议实施顺序）

### T01 — 本地 Windows（现状归档）

- **状态**：已基本可用（Pake GUI + 本地 `pake`）
- **目录**：`tasks/T01-local-windows/`
- **后续小改**：产物默认落到 `builds/windows/`；中文名打 MSI 提示英文名 / `zh-CN`

### T02 — GitHub macOS 云端（下一主任务）

- **目录**：`tasks/T02-github-macos-cloud/`
- **建议子步骤**（实现时在该目录写 checklist）：

| 步骤 | 内容 | 落点 |
|------|------|------|
| T02.1 | GitHub 设置 API：保存 `owner/repo` + PAT | `internal/cloud/github` + `data/github.json` |
| T02.2 | 界面「云端设置」：Token、仓库、测试连接 | `server/web` 独立区块，勿和本地打包按钮缠在一起 |
| T02.3 | 提交任务：参数 → 创建 `data/cloud-jobs/{id}/` | `internal/cloud/common` |
| T02.4 | 图标：复制到 job 目录 → Contents API 上传 → 得到 raw URL | `internal/cloud/github` |
| T02.5 | `workflow_dispatch` 触发 `build-macos.yml` | 同上 |
| T02.6 | 轮询 run 状态 + 下载 Artifact → `builds/macos/` | 同上 |
| T02.7 | UI：进度、失败原因、打开产物目录 | `/api/cloud/*` |

**授权建议（先做简单可用）**

1. HTML 表单填写 **Fine-grained / classic PAT**（`repo` + `workflow`）
2. Go 后端写入 `data/github.json`（不进 git）
3. 以后再加 Device Flow / OAuth（仍归 `internal/cloud/github/oauth.go`）

**图标建议**

- 本地仍用现有上传控件
- 提交云端时：**按 job 复制**，再上传到仓库路径：  
  `ci-assets/macos/{jobId}/icon.png`  
  （不要覆盖同一文件名，避免并发任务互相踩）

### T03 — Android 云端（debug APK）

- **目录**：`tasks/T03-android-cloud/` + `android-shell/` + `builds/android/`
- **已选定方案**：自建 WebView 模板 + GitHub Actions `assembleDebug`；不在 Windows 本机硬编 APK
- **产物**：debug 签名 APK（可侧载）；不上架、不打 AAB

---

## 3. API 形状建议（按平台前缀，便于扩展）

```text
POST   /api/cloud/github/settings     # 保存仓库与 Token
GET    /api/cloud/github/settings     # 脱敏回显
POST   /api/cloud/github/test         # 测 Token / Actions 权限

POST   /api/cloud/jobs                # body: { platform: "macos"|"windows"|"android", ...options }
GET    /api/cloud/jobs                # 列表（读 data/cloud-jobs/*/status.json）
GET    /api/cloud/jobs/{id}           # 详情
POST   /api/cloud/jobs/{id}/cancel    # 能取消则取消远程 run
POST   /api/cloud/jobs/{id}/retry
GET    /api/cloud/jobs/{id}/download  # 或直接打开 builds/{platform}/
```

`platform` 字段从第一天就带上，安卓接入时只加分支，不改路由体系。

---

## 4. Job 目录约定（强制）

每次点「提交云端打包」：

```text
data/cloud-jobs/20260817-0031-a1b2/
  request.json      # url/name/icon/platform/...
  icon.png          # 本任务图标
  remote.json       # { "runId": 123, "iconRemotePath": "ci-assets/macos/.../icon.png" }
  status.json       # { "state": "running", "updatedAt": "..." }
  logs.txt
```

产物：

```text
builds/macos/HunliPai-AI-1.0.0.dmg
builds/android/HunliPai-AI-1.0.0.apk   # 未来
builds/windows/...                     # 本地包也可逐步迁入
```

---

## 5. 前端 UI 建议（避免混在一起）

- **本地打包**：保持现有主按钮（Windows）
- **云端打包**：单独卡片/折叠区  
  - 平台：`macOS (GitHub)` / `Android (GitHub)`（后者 disabled +「预留」）  
  - GitHub 设置入口  
  - 「提交云端任务」与「开始本地打包」分开
- **任务列表**：只列 `data/cloud-jobs/*`，与本地 history 分开展示或加 `source: local|cloud` 标签

---

## 6. 安全注意

- Token 只存 `data/`，`.gitignore` 已忽略 `data/`
- API 继续走现有 localhost + token 中间件
- 上传到 GitHub 的图标视为半公开（raw URL）；不要上传含隐私截图
- 日志里对 Token 打码

---

## 7. 建议排期

| 阶段 | 产出 | 预估 |
|------|------|------|
| P0 | 落地本目录骨架 + 任务规格书 | 0.5d |
| P1 | T02.1–T02.2 设置与测连 | 1d |
| P2 | T02.3–T02.5 提交 + 图标上传 + 触发 | 1–2d |
| P3 | T02.6–T02.7 轮询下载 + UI | 1d |
| P4 | T03 仅占位（空模块 + disabled UI + workflow stub） | 0.5d |

---

## 8. 明确不做（本阶段）

- Windows 交叉编译 macOS / Android
- 在 HTML 里完成完整 OAuth（可二期）
- Apple 签名公证、Google Play 上架
- 把所有逻辑塞进单个 `server.go`

---

## 9. 下一步

1. 按本文创建目录骨架与 `tasks/*` 规格书  
2. 开始实现 **T02**（GitHub macOS 云端）  
3. **T03** 只占位，等 macOS 闭环后再填安卓方案
