# T03 — Android 云端打包

## 目标

与 T02 / T04 同一套云端 Job 模型，提交到 GitHub Actions，下载 debug APK 到 `builds/android/`。

## 约定（已落地）

- 复用 `internal/cloud/common`、`internal/cloud/github.RunCloudJob`
- `platform: "android"` → workflow `build-android.yml`
- 通用模板：`android-shell/`（Gradle + WebView，占位符由 CI 注入）
- 独立产物：`builds/android/`
- 图标：`ci-assets/android/{jobId}/`
- 第一版只出 **debug 签名 APK**（可侧载）；不上架、不打 AAB、不提交 keystore

## Checklist

- [x] 本任务目录
- [x] `android-shell/` 通用 WebView 模板
- [x] `.github/workflows/build-android.yml`（`assembleDebug` + `push.paths` 自注册）
- [x] `specFor(android)` + `/api/cloud/jobs` 接通
- [x] GUI 启用 Android 选项，产物目录 `builds/android/`
- [x] 文档 `docs/android.md`

## 非目标（本版）

- 不在 Windows GUI 进程内跑 Android SDK
- 不用 Capacitor / Tauri Android
- 不沿用 `android-webview-shell/` 的产品写死逻辑与 Homebrew `aapt` 脚本
- 不上架、不打正式签名
