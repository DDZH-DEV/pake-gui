# T03 — Android 云端打包（预留）

## 目标（未来）

与 T02 同一套云端 Job 模型，提交到 GitHub Actions，下载 APK/AAB 到 `builds/android/`。

## 现在就要定的约定

- 复用 `internal/cloud/common`、`internal/cloud/github`
- 新建 `internal/cloud/android`（适配层，可先 stub）
- 独立 workflow：`.github/workflows/build-android.yml`
- 独立配置：`configs/android/`
- 独立产物：`builds/android/`
- API 用同一套 `/api/cloud/jobs`，`platform: "android"`
- UI 平台选项先 **disabled**，文案：「预留」

## 技术方案（实现时再选，勿绑死）

候选：PakePlus 类流水线 / Capacitor CI / 自建 Android WebView 模板仓库。  
共同点：必须在 Linux/Android SDK 的 CI 环境构建，不在 Windows GUI 进程内直接 `gradle`。

## 占位 Checklist

- [x] 本任务目录
- [ ] `internal/cloud/android` stub（`NotImplemented`）
- [ ] `configs/android/.gitkeep`
- [ ] `builds/android/.gitkeep`
- [ ] workflow stub（`if: false` 或仅文档）
- [ ] UI 入口 disabled

## 非目标（当前）

- 不实现真实打包
- 不引入 Android SDK 到本机开发依赖
