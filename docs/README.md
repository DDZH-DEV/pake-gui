# Pake GUI 文档

把网页打成桌面应用的本地工作台：Windows 本机打包 + GitHub Actions 云端 macOS（Android 预留）。

仓库：https://github.com/DDZH-DEV/pake-gui

## 快速入口

| 文档 | 说明 |
|------|------|
| [快速开始](./getting-started.md) | 安装、启动、界面 Tab 说明 |
| [Windows 本机打包](./windows.md) | 环境、用法、MSI/中文名注意 |
| [macOS 云端打包](./macos.md) | GitHub Actions 打 DMG |
| [Android 云端打包](./android.md) | 预留方案与目录约定 |
| [GitHub 授权](./github-oauth.md) | Device Flow / Client ID / PAT |
| [常见问题](./troubleshooting.md) | 黑窗、MSI 失败、Gatekeeper 等 |
| [多平台开发计划](./dev-plan-cloud-and-platforms.md) | 目录结构与任务拆分（开发向） |

## 平台能力一览

| 平台 | 模式 | GUI | 产物目录 | 状态 |
|------|------|-----|----------|------|
| Windows | 本机 `pake-cli` | 「打包」Tab | `builds/windows/`（或 `builds/`） | 可用 |
| macOS | GitHub Actions | 「云端」Tab | `builds/macos/` | 可用（需推仓 + OAuth） |
| Android | GitHub Actions（预留） | 选项禁用 | `builds/android/` | 占位 |

## 相关目录

- 任务规格：`tasks/T01-*` / `T02-*` / `T03-*`
- 参数模板：`configs/windows|macos|android/`
- 工作流：`.github/workflows/build-macos.yml`、`build-android.yml`
