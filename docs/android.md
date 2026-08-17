# Android 云端打包

在 **GitHub Actions 的 Ubuntu runner** 上，用仓库内通用 WebView 模板 `android-shell/` 生成 **debug 签名 APK**。  
本机无需 Android SDK / JDK / Gradle。

工作流文件：`.github/workflows/build-android.yml`

## 这一版做什么、不做什么

| 做 | 不做 |
|----|------|
| 把网址包进 WebView APK | Google Play 上架、AAB |
| debug 签名，可侧载安装 | 提交 release keystore |
| 同源页面内打开，外链走系统浏览器 | 婚礼派等产品专用 JS 桥 / 相册 |
| 包名用 Identifier，或按域名生成 | 在 Windows GUI 进程内跑 SDK |

安装时系统会提示「未知来源 / 非正式签名」，属预期。

## 前置条件

1. 本仓库已推送到 GitHub，且 Actions 已启用  
2. 完成 [GitHub 授权](./github-oauth.md)  
3. 「打包」Tab 已填好网址、应用名；**Identifier** 建议填 Android 包名（如 `com.example.app`），留空则按网址域名反转生成（`hlai.yingdedao.cn` → `cn.yingdedao.hlai`）

首次使用前，确认 Actions 侧边栏已出现 **Build Android APK (WebView)**。若提示 workflow 404，再 push 一次 `build-android.yml`（文件带 `push.paths` 自注册）。

## 用 Pake GUI 提交

1. 打开 **云端** Tab 并完成授权  
2. 云端目标选 **Android APK**  
3. 点 **提交 Android 云端（APK）**  
4. 成功后产物在 `builds/android/`，一般是 `{jobId}-debug.apk`

也可从 **任务** Tab 对本机或其它云端记录点 **回填** 后再提交。

### 图标如何传到云端

与 macOS / Windows 相同：本地图标经 Contents API 上传到 `ci-assets/android/{jobId}/icon.*`，构建时转成启动图标。未选图标则用模板默认图标。

## 模板能力（`android-shell/`）

- JavaScript、DOM Storage、第三方 Cookie  
- 返回键：先退 WebView 历史，再退出应用  
- 同主机（含子域）留在应用内；其它 http(s)、mailto、tel、下载用系统应用打开  
- `http://` 网址会打开明文流量（`usesCleartextTraffic`）

旧目录 `android-webview-shell/` 只作参考，**不是**正式模板。

## 在 GitHub 网页手动跑

1. 仓库 → **Actions** → **Build Android APK (WebView)** → **Run workflow**  
2. 填写 url、name；identifier / icon 可选  

## 安装 APK

把 `builds/android/` 里的 apk 拷到手机，允许「安装未知应用」后打开即可。debug 签名不能用于上架。

## 相关

- 授权：[github-oauth.md](./github-oauth.md)  
- 排障：[troubleshooting.md](./troubleshooting.md)  
- 任务规格：`tasks/T03-android-cloud/README.md`
