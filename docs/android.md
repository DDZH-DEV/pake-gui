# Android 云端打包（预留）

当前 **未实现** 真实 APK/AAB 构建。GUI「云端目标」中 Android 为禁用项。

## 为什么不能在 Windows 上一键出 APK

- Pake / Tauri 主路径是桌面（Win / macOS / Linux）  
- 网页套壳 APK 通常依赖 **Android SDK / NDK、Gradle、JDK**，需在 Linux CI 或 Android 工程中构建  
- 单纯 `go build` 或本机 `pake` **不能**从网址直接生成可安装 APK

## 已占位内容（勿删，便于后续接入）

| 路径 | 作用 |
|------|------|
| `tasks/T03-android-cloud/` | 任务规格与 checklist |
| `internal/cloud/android/` | 适配层 stub（返回 NotImplemented） |
| `configs/android/` | 参数模板目录 |
| `builds/android/` | 未来产物目录 |
| `.github/workflows/build-android.yml` | 占位工作流（目前会主动失败并提示） |

## 约定（实现时遵守）

与 macOS 共用：

- `internal/cloud/common` Job 模型  
- `internal/cloud/github` 授权与 API  
- `/api/cloud/jobs`，`platform: "android"`

独立：

- workflow、`configs/android/`、`builds/android/`  
- UI 平台选项启用后再开放提交  

## 候选技术方案（未选定）

| 方案 | 说明 |
|------|------|
| PakePlus 类流水线 | 偏「网址 → 移动端」产品线 |
| Capacitor / Cordova CI | 标准 WebView 壳 + GitHub Actions |
| 自建 Android WebView 模板仓 | 用 Actions 注入 URL/图标后 `gradle assemble` |

实现时再选，**不要**在 Windows GUI 进程内直接跑完整 Android 工具链。

## 现阶段建议

1. Windows：本机打包，见 [windows.md](./windows.md)  
2. macOS：云端 DMG，见 [macos.md](./macos.md)  
3. Android：单独立项后再填 T03  

任务索引：`tasks/T03-android-cloud/README.md`
