# Pake GUI 文档

用户说明与开发约定。项目简介见仓库根目录 [README.md](../README.md)。

## 快速入口

| 文档 | 说明 |
|------|------|
| [快速开始](./getting-started.md) | 启动、Tab、按目标选路径 |
| [Windows 打包](./windows.md) | 本机环境，或云端 exe（免装 VS） |
| [macOS 云端打包](./macos.md) | GitHub Actions 打 DMG |
| [Android 云端打包](./android.md) | Actions 打 debug APK（可侧载） |
| [GitHub 授权](./github-oauth.md) | Device Flow、Client ID、PAT |
| [常见问题](./troubleshooting.md) | 黑窗、MSI、授权、Gatekeeper 等 |
| [多平台开发计划](./dev-plan-cloud-and-platforms.md) | 目录结构与任务拆分（开发向） |

## 平台能力一览

| 平台 | 模式 | GUI | 产物目录 | 状态 |
|------|------|-----|----------|------|
| Windows | 本机 `pake-cli` 或 GitHub Actions（仅 exe） | 「打包」/「云端」Tab | `builds/windows/` | 可用 |
| macOS | GitHub Actions | 「云端」Tab | `builds/macos/` | 可用（需推仓 + OAuth） |
| Android | GitHub Actions（`android-shell` WebView） | 「云端」Tab | `builds/android/` | 可用（debug / release APK、AAB） |

## 相关目录

- 任务规格：[`tasks/`](../tasks/README.md)
- 参数模板：`configs/windows|macos|android/`
- 工作流：`.github/workflows/build-macos.yml`、`build-windows.yml`、`build-android.yml`
- Android 模板：`android-shell/`
- 注入脚本默认目录：`data/inject/`（运行时创建，不入库）
