# 常见问题

## 启动时闪很多黑色命令窗口

**原因：** 环境检测会执行 `npm` / `node` 等（Windows 上多为 `.cmd`），未隐藏控制台时会闪窗。  

**处理：** 使用已修复的 `PakeGUI.exe`（子进程带 `CREATE_NO_WINDOW`）。用 `build.bat` 重新编译后再开。

---

## Windows：MSI 失败，提示 light.exe

日志类似：

```text
Built application at: ...\pake-xxx.exe
Running light to produce ...msi
failed to run ...\light.exe
```

**原因：** 应用名含中文等非 ASCII 时，WiX MSI 编码易失败；exe 往往已经编成功。  

**处理：**

1. 应用名改英文（如 `HunliPai-AI`）后重打  
2. 或只要 exe：勾选「快速迭代构建」  
3. 从 pake 输出目录拷贝已生成的 exe 到 `builds/`

详见 [windows.md](./windows.md)。

---

## 云端：提示请先保存 GitHub 设置 / 未登录

1. 打开 **云端** Tab  
2. 填写 **OAuth Client ID**（不是设置页 URL 里的数字编号）并 **使用 GitHub 授权**，或粘贴 PAT  
3. Owner/Repo 确认无误 → **测试连接**  
4. 确认仓库已包含 `.github/workflows/build-macos.yml` 且已 push  

详见 [github-oauth.md](./github-oauth.md)。

---

## 云端：触发 workflow 失败 404 Not Found

常见原因：**文件已在默认分支，但 GitHub 还没把它登记进 Actions**（API `/actions/workflows` 里看不到 `build-macos.yml`，只有 Dependabot 等）。Token 权限正常时也会 404。

处理：

1. 确认默认分支上有对应 workflow 文件（`build-macos.yml` / `build-windows.yml` / `build-android.yml`）  
2. **再 push 一次该文件**（当前 workflow 带 `push.paths` 自注册；job 仅在 `workflow_dispatch` 时真正打包）  
3. 打开 https://github.com/DDZH-DEV/pake-gui/actions ，侧边栏应出现对应工作流  
4. GUI 里再点 **测试连接**（应提示 workflow 已注册）后重新提交

---

## 授权：显示「正在打开浏览器」但没打开

1. 确认 Client ID 正确（常见错误：填成了 `3797658` 这类 App 编号）  
2. 新版本会用 ShellExecute 打开，并显示**可点击链接 + 设备码**；也可手动打开 https://github.com/login/device  
3. OAuth App 需**启用 Device Flow**  

---

## 云端：触发成功但找不到 run / 构建失败

- 到 GitHub → Actions 看具体日志  
- 确认默认分支与 GUI 里 Ref 一致（空=仓库默认分支）  
- 首次构建慢属正常；失败可清 Actions cache 后重试  
- 应用名建议英文，避免 artifact 名称异常  

---

## 云端：`Target x86_64-apple-darwin is not installed`

勾了 **Universal（Intel+Apple）** 时，pake 会打 `universal-apple-darwin`。GitHub `macos-latest` 默认只有 Apple Silicon 工具链，缺 Intel 目标就会失败。

处理：使用已安装 `x86_64-apple-darwin` 的 workflow（push 后再提交）；或不勾 Universal，只打当前 runner 架构（更快）。  

---

## macOS：下载的 DMG 打不开 / 已损坏

未签名常见现象。可尝试：

- 右键 → 打开  
- 系统设置 → 隐私与安全性 → 仍要打开  

正式分发需 Apple 签名与公证（尚未接入）。

---

## 环境 Tab：找不到 Rust / cargo

1. 安装 Rust MSVC：https://www.rust-lang.org/tools/install  
2. 安装 VS Build Tools（C++ + Windows SDK）  
3. **关闭并重开** Pake GUI（PATH 需刷新）  
4. 「环境」→ 重新检测  

---

## 能否在 Windows 上打包 macOS / Android？

| 目标 | 结论 |
|------|------|
| macOS | 不能本机交叉编；用 [macos.md](./macos.md) 云端 |
| Windows exe | 可不装 VS，用云端 `build-windows.yml`（公开仓免费） |
| Android | 不能本机一键 APK；用 [android.md](./android.md) 云端 debug APK |

---

## 图标上传失败 / 云端没用上图标

- 本机：确认格式为 png/ico/icns/jpg/webp，≤ 8MB  
- 云端：需有推送仓库内容的权限（OAuth `repo`）  
- 私有仓：GUI 使用仓库内路径 `ci-assets/macos/...`，不要依赖 raw 公网 URL  

---

## 注入 JS/CSS 扫描不到文件

- 默认目录：`data/inject/`（将 `.js` / `.css` 放入后点「扫描」）  
- 也可「上传文件」；自定义路径需在程序 `data` 目录内  

---

## Android：安装 APK 提示未知来源 / 非正式签名

第一版是 **debug 签名**，不能上架。在系统设置里允许该文件管理器「安装未知应用」后再打开 apk。

---

## 历史任务如何接着打 Mac / Win / Android

**任务** Tab → 记录 → **回填** / **回填·云端** → 在云端 Tab 选目标后提交。

---

## 日志在哪

- GUI → **打开日志** → `data/app.log`  
- 云端任务摘要：`data/cloud-jobs/{jobId}/logs.txt`
