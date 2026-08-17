# Pake GUI

把网页打成桌面应用的本地工作台：

- **Windows**：本机 `pake-cli`，或经 GitHub Actions 云端出 **exe**（免装 VS）  
- **macOS**：经 GitHub Actions 云端出 DMG（无需本机 Mac）  
- **Android**：经 GitHub Actions 云端出 debug APK（无需本机 Android SDK，可侧载）

仓库：https://github.com/DDZH-DEV/pake-gui

## 快速开始

```bat
build.bat
```

或双击 `PakeGUI.exe`。调试：`PakeGUI.exe -browser`

更多说明 → [docs/getting-started.md](./docs/getting-started.md)

## 功能概览

| 能力 | 说明 |
|------|------|
| Tab 界面 | 打包 / 云端 / 任务 / 环境 |
| 本机打包 | 图标上传、窗口选项（分平台标注）、Targets 下拉、注入 JS/CSS 扫描 |
| 云端 Windows exe | GitHub Actions、`--iterative-build`、下载到 `builds/windows/` |
| 云端 macOS | GitHub Device Flow 授权、提交 Actions、轮询下载到 `builds/macos/` |
| 云端 Android APK | GitHub Actions、WebView 模板、下载到 `builds/android/` |
| 任务回填 | 本机记录可「回填·本机 / 回填·云端」；云端记录可回填重提 |
| 单实例 / 日志 | `data/app.log`；子进程无黑窗闪烁 |

## 文档

| 文档 | 说明 |
|------|------|
| [docs/README.md](./docs/README.md) | 文档总索引 |
| [快速开始](./docs/getting-started.md) | 安装与 Tab 说明 |
| [Windows 本机打包](./docs/windows.md) | 环境、选项、MSI 注意 |
| [macOS 云端打包](./docs/macos.md) | Actions 打 DMG |
| [Android 云端打包](./docs/android.md) | Actions 打 debug APK（可侧载） |
| [GitHub 授权](./docs/github-oauth.md) | Client ID / Device Flow / PAT |
| [常见问题](./docs/troubleshooting.md) | 排障 |
| [开发计划](./docs/dev-plan-cloud-and-platforms.md) | 目录与任务拆分 |

任务规格（开发向）：[tasks/README.md](./tasks/README.md)

## 目录要点

```text
builds/{windows,macos,android}/   # 产物
configs/{windows,macos,android}/  # 参数模板
data/                             # 运行时（gitignore）：日志、图标、注入脚本、云端任务、Token
.github/workflows/                # build-macos.yml / build-windows.yml / build-android.yml
android-shell/                    # Android WebView 模板（云端 Gradle 构建）
docs/                             # 用户与开发文档
tasks/                            # T01–T04 任务规格
```

## 开发构建

```bat
go build -ldflags="-H windowsgui -s -w" -o PakeGUI.exe .
```

要求：Go、Windows 上 WebView2（缺失时可引导安装）。
