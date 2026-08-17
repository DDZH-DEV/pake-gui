# Android 云端打包

在 **GitHub Actions 的 Ubuntu runner** 上，用仓库内通用 WebView 模板 `android-shell/` 生成 APK / AAB。  
本机无需 Android SDK / JDK / Gradle。

工作流文件：`.github/workflows/build-android.yml`

## 能力一览

| 能力 | 说明 |
|------|------|
| debug APK | 默认产物，可侧载 |
| release APK / AAB | 需配置 GitHub Secrets（keystore） |
| User-Agent / Safe Domains / 注入 JS·CSS | 与「本机高级」共用 |
| 屏幕方向、全屏、下拉刷新、进度条 | 「打包 → Android 选项」 |
| 外链策略 | 白名单 / 全部内开 / 全部外开 |
| 选文件 / 相机 / 下载 | 可选开启 |
| 系统分享 / 刘海安全区 | 默认开启（`PakeAndroid` + CSS 变量） |
| 本地通知 API | 可选；FCM 远端推送需自行接 Firebase |

不上架流程、不提交 keystore 到仓库。安装 debug 包时系统会提示「未知来源」，属预期。

## 前置条件

1. 本仓库已推送到 GitHub，且 Actions 已启用  
2. 完成 [GitHub 授权](./github-oauth.md)  
3. 「打包」填好**基本信息**；需要时展开 **Android 选项** / **本机高级**

首次使用前，确认 Actions 侧边栏已出现 **Build Android APK (WebView)**。若提示 workflow 404，再 push 一次 `build-android.yml`。

## 「打包」页怎么填

| 区块 | 用途 |
|------|------|
| **基本信息** | 网址、名称、版本、包名、图标 |
| **Android 选项** | 方向、外链、全屏、刷新、进度条、文件/相机/下载、推送占位 |
| **本机高级** | UA、Safe Domains、注入（Android 云端也会带上） |
| 桌面窗口 / 行为 | 仅桌面，Android 忽略 |

包名留空则按域名反转（如 `hlai.yingdedao.cn` → `cn.yingdedao.hlai`）。

## 产物格式（云端 Tab）

| 值 | 说明 |
|------|------|
| `apk` | debug APK（默认，可侧载） |
| `apk-release` | 正式签名 APK（需 Secrets） |
| `aab` | Play 用的 AAB（需 Secrets） |
| `apk-release,aab` | 同时出两者 |

### Release / AAB 所需 Secrets

在仓库 **Settings → Secrets and variables → Actions** 添加：

| Secret | 说明 |
|--------|------|
| `ANDROID_KEYSTORE_BASE64` | keystore 文件的 base64（整文件） |
| `ANDROID_KEYSTORE_PASSWORD` | keystore 密码 |
| `ANDROID_KEY_ALIAS` | key alias |
| `ANDROID_KEY_PASSWORD` | key 密码 |

生成 base64（PowerShell 示例）：

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("D:\path\release.keystore"))
```

## 用 Pake GUI 提交

1. 「打包」填基本信息，按需开 Android 选项 / 注入  
2. 「云端」授权 → 目标选 **Android APK** → 选产物格式  
3. **提交 Android 云端（APK）**  
4. 产物在 `builds/android/`

## 模板行为摘要

- 返回键：先退 WebView 历史  
- 白名单：启动域名 + Safe Domains；其它 http(s) 按外链策略处理  
- **刘海 / 安全区**：非全屏自动避开状态栏与挖孔；全屏沉浸并向页面注入 `--pake-safe-top/bottom/left/right`（及 `viewport-fit=cover`）  
- **下载**：优先 `Content-Disposition` / 查询参数文件名；避免把 `export.php` 存成脚本名；支持 `blob:` / `data:`  
- **系统分享**：页面可调 `PakeAndroid.share(title, text, url)`  
- **通知**：勾选「本地通知 / 推送 API」后可用 `requestPushPermission()` / `showNotification(title, body)`；FCM 远端推送需另接 `google-services.json`（见下）  
- 注入：页面加载完成后执行 `assets/inject` 下的 `.js` / `.css`  

### H5 调用示例

```js
// 分享
PakeAndroid.share("标题", "摘要", location.href);

// 刘海安全区（px）
const inset = JSON.parse(PakeAndroid.getSafeAreaInsets());
// 或 CSS: padding-top: var(--pake-safe-top);

// 本地通知（需打包时勾选推送能力）
PakeAndroid.requestPushPermission();
PakeAndroid.showNotification("新消息", "内容预览");
```

### 后续接入 FCM（可选）

1. Firebase 控制台创建应用，下载 `google-services.json`  
2. 放入 `android-shell/app/`，并在 Gradle 启用 Google Services 插件与 Messaging 依赖  
3. 实现 `FirebaseMessagingService`，把 token 回写到 `PakeAndroid.getPushToken()`  
4. 通知仍可复用现有 channel `pake_general`  

旧目录本地手搓 SDK 方案已废弃；只保留云端 `android-shell/`。

## 相关

- 授权：[github-oauth.md](./github-oauth.md)  
- 排障：[troubleshooting.md](./troubleshooting.md)  
- 任务：`tasks/T03-android-cloud/README.md`
