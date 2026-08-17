# T03 — Android 云端打包

## 目标

与 T02 / T04 同一套云端 Job 模型，提交到 GitHub Actions，下载 debug APK 到 `builds/android/`。

## 约定（已落地）

- 复用 `internal/cloud/common`、`internal/cloud/github.RunCloudJob`
- `platform: "android"` → workflow `build-android.yml`
- 通用模板：`android-shell/`（Gradle + WebView，占位符由 CI 注入）
- 独立产物：`builds/android/`
- 图标：`ci-assets` 分支上的 `android/{jobId}/`（任务结束后清理）
- 第一版只出 **debug 签名 APK**（可侧载）；不上架、不打 AAB、不提交 keystore

## Checklist

- [x] 本任务目录
- [x] `android-shell/` 通用 WebView 模板
- [x] `.github/workflows/build-android.yml`（debug / release / aab）
- [x] `specFor(android)` + `/api/cloud/jobs` 接通
- [x] GUI：Android 选项 + 产物格式联动
- [x] UA / 注入 / Safe Domains / 外链 / 文件相机下载 / 推送 API  
- [x] 下载文件名优化（Disposition / 拒脚本名 / blob·data）  
- [x] 刘海安全区 + 系统分享 + 本地通知通道  
- [x] 文档 `docs/android.md`

## 非目标（本版）

- 不在 Windows 本机装 Android SDK / 不跑本地 Gradle
- 不用 Capacitor / Tauri Android
- 不接完整 FCM（仅 JS 占位）
- 不上架商店审核流程
- 不把 keystore 提交进 git（只用 Actions Secrets）
