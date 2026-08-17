# macOS 云端打包

在 **GitHub Actions 的 macOS runner** 上调用 `pake-cli`，生成 DMG / `.app`。  
本机无需 Mac、Xcode、Rust。

工作流文件：`.github/workflows/build-macos.yml`

## 前置条件

1. 本仓库已推送到 GitHub（例如 `DDZH-DEV/pake-gui`），且 Actions 已启用  
2. 完成 [GitHub 授权](./github-oauth.md)（Device Flow 或 PAT）  
3. 「打包」Tab 已填好网址、应用名（**建议英文名**）、图标等

## 用 Pake GUI 提交（推荐）

1. 打开 **云端** Tab  
2. 确认 Owner / Repo（默认 `DDZH-DEV` / `pake-gui`）  
3. **使用 GitHub 授权**（或高级选项粘贴 PAT）→ **测试连接**  
4. 选择产物格式 `dmg` 或 `app`；需要 Intel+Apple 再勾 Universal  
5. 点 **提交 macOS 云端**  
6. 同页右侧或 **任务** Tab 查看进度  
7. 成功后产物在 `builds/macos/`

也可从 **任务** Tab 对本机 Windows 记录点 **回填·云端**，带上同一套参数再提交。

### 图标如何传到云端

- 本地图标会先写入任务目录，再通过 Contents API 上传到：  
  `ci-assets/macos/{jobId}/icon.*`  
- Actions `checkout` 后使用**仓库内相对路径**，私有仓库也可用  
- 也可直接在「打包」里填 `https://…` 图标地址

## 在 GitHub 网页手动跑

1. 仓库 → **Actions** → **Build macOS App (Pake)** → **Run workflow**  
2. 填写示例：

| 字段 | 示例 |
|------|------|
| url | `https://example.com/` |
| name | `HunliPai-AI` |
| icon | 公开 URL 或仓库路径；可空 |
| identifier | `com.example.app` |
| targets | `dmg` 或 `app` |
| multi_arch | Universal 时勾选 |
| job_id | GUI 自动填；手动可空 |

3. 首次约 10–20 分钟，之后更快  
4. 成功 run → **Artifacts** → 下载 `{name}-macOS`

参数模板：`configs/macos/hunlipai-ai.json`

## 产物与签名

| 项 | 说明 |
|----|------|
| 默认产物 | DMG（或 `.app`） |
| 签名 | 默认**未签名** |
| 自己用 | 可能需右键打开，或在「隐私与安全性」允许 |
| 对外分发 | 需要 Apple 开发者账号：签名 + 公证（尚未接入 GUI） |

## 命令行触发（可选）

```bat
gh auth login
gh workflow run "Build macOS App (Pake)" -R DDZH-DEV/pake-gui -f url=https://example.com/ -f name=HunliPai-AI
```

## 备选

也可 [Fork tw93/Pake](https://github.com/tw93/Pake/fork)，用官方工作流 `Build App With Pake CLI`（平台选 `macos-latest`）。

## 相关

- 授权细节：[github-oauth.md](./github-oauth.md)  
- 排障：[troubleshooting.md](./troubleshooting.md)  
- 任务规格：`tasks/T02-github-macos-cloud/`
