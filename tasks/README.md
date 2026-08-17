# 任务索引

按编号分目录维护，避免多平台 / 多次迭代混在一个文档里。

| ID | 目录 | 平台 | 模式 | 状态 | 用户文档 |
|----|------|------|------|------|----------|
| T01 | [T01-local-windows](./T01-local-windows/) | Windows | 本地 pake | 基本完成 | [docs/windows.md](../docs/windows.md) |
| T02 | [T02-github-macos-cloud](./T02-github-macos-cloud/) | macOS | GitHub Actions | 已实现 | [docs/macos.md](../docs/macos.md) |
| T03 | [T03-android-cloud](./T03-android-cloud/) | Android | GitHub Actions（debug APK） | 已实现 | [docs/android.md](../docs/android.md) |
| T04 | [T04-github-windows-cloud](./T04-github-windows-cloud/) | Windows | GitHub Actions（仅 exe） | 已实现 | [docs/windows.md](../docs/windows.md) |

## 文档入口

- 项目说明：[../README.md](../README.md)
- 文档总览：[../docs/README.md](../docs/README.md)
- 开发计划：[../docs/dev-plan-cloud-and-platforms.md](../docs/dev-plan-cloud-and-platforms.md)
- GitHub 授权：[../docs/github-oauth.md](../docs/github-oauth.md)

## 近期已落地（跨任务）

- Tab 布局：打包 / 云端 / 任务 / 环境  
- Device Flow 授权 + 浏览器打开修复  
- 任务回填（本机 ↔ 云端）  
- Targets 下拉、注入目录扫描  
- 云端 Android debug APK（`android-shell` + `build-android.yml`） 
