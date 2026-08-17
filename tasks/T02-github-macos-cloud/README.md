# T02 — GitHub macOS 云端打包

## 目标

Windows GUI 选好参数 → 提交 GitHub Actions → 下载 DMG 到 `builds/macos/`。

## 验收标准

- [x] 可在界面保存 GitHub 仓库与授权，并可「测试连接」  
- [x] Device Flow 授权（Client ID）；PAT 作备用  
- [x] 提交任务时创建独立目录 `data/cloud-jobs/{jobId}/`  
- [x] 本地图标随任务上传到仓库 `ci-assets/macos/{jobId}/`  
- [x] 成功触发 `.github/workflows/build-macos.yml`  
- [x] 可轮询状态；成功后 Artifact 落到 `builds/macos/`  
- [x] 失败时界面可见原因（不泄露 Token）  
- [x] 任务列表回填 / 打开产物  

## 实现落点

| 模块 | 路径 |
|------|------|
| Job 模型 / 落盘 | `internal/cloud/common` |
| GitHub API / OAuth | `internal/cloud/github` |
| HTTP | `/api/cloud/*` |
| UI | 「云端」Tab + 「任务」Tab |
| Workflow | `.github/workflows/build-macos.yml` |
| 参数模板 | `configs/macos/` |

## 使用前仍需

1. 将本仓库推到 GitHub（含 workflow）  
2. 创建 OAuth App（启用 Device Flow）并填 Client ID，或使用 PAT  
3. 「云端」授权 → 测试连接 → 提交  

详见 [docs/macos.md](../../docs/macos.md)、[docs/github-oauth.md](../../docs/github-oauth.md)。

## 授权

- 主路径：Device Flow（`oauth_device.go`）  
- 备用：手动 PAT  

## 非目标

- Apple 签名 / 公证  
- 在 Windows 本机编出 macOS 包  
