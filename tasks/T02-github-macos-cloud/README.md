# T02 — GitHub macOS 云端打包

## 目标

Windows GUI 选好参数 → 提交 GitHub Actions → 下载 DMG 到 `builds/macos/`。

## 验收标准

- [x] 可在界面保存 GitHub 仓库与 PAT，并可「测试连接」
- [x] 提交任务时创建独立目录 `data/cloud-jobs/{jobId}/`
- [x] 本地图标随任务上传到仓库 `ci-assets/macos/{jobId}/`，并作为 workflow `icon` 输入
- [x] 成功触发 `.github/workflows/build-macos.yml`
- [x] 可轮询状态；成功后 Artifact 落到 `builds/macos/`
- [x] 失败时界面可见原因（不泄露 Token）

## 实现落点

| 模块 | 路径 |
|------|------|
| Job 模型 / 落盘 | `internal/cloud/common` |
| GitHub API | `internal/cloud/github` |
| HTTP | `/api/cloud/*` |
| UI | 表单「提交 macOS 云端」+ 云端设置 + 云端任务列表 |
| Workflow | `.github/workflows/build-macos.yml` |
| 参数模板 | `configs/macos/` |

## 子步骤 Checklist

- [x] T02.1 settings 存取（`data/github.json`）
- [x] T02.2 UI 设置 + 测连
- [x] T02.3 创建 job 目录 + request.json
- [x] T02.4 图标 Contents API 上传
- [x] T02.5 workflow_dispatch
- [x] T02.6 轮询 + 下载 Artifact
- [x] T02.7 任务列表 / 打开目录

## 使用前仍需

1. 将本仓库推到 GitHub（含 workflow）
2. 创建 PAT（`repo` + `workflow`）
3. 在 GUI「云端打包 · GitHub 设置」保存并测试连接
4. 点「提交 macOS 云端」

## 授权

优先：PAT 表单 → 后端保存。  
二期：Device Flow（仍放 `internal/cloud/github/`）。

## 非目标

- Apple 签名 / 公证
- 在 Windows 本机编出 macOS 包
