# Windows 本机打包

在 Windows 上直接调用本机 `pake-cli`（底层 Rust + Tauri），生成 exe / MSI 等。

## 环境要求

| 工具 | 说明 |
|------|------|
| Windows 10+ | 建议已带 WebView2；缺失时 GUI 可引导安装 |
| Node.js | 建议 ≥ 18（推荐 22 LTS） |
| pake-cli | `npm i -g pake-cli`，或在「环境」Tab 一键安装 |
| Rust | ≥ 1.85，**MSVC 工具链**（不要选 GNU） |
| VS Build Tools | MSVC C++ 生成工具 + Windows SDK |

检测：打开 Pake GUI → **环境** Tab → 看各工具是否「就绪」。

安装 Rust：https://www.rust-lang.org/tools/install  
装完后**重开**终端 / 重开 Pake GUI。

## 操作步骤

1. 打开 **打包** Tab  
2. 填写：
   - 网址或本地 `dist` / `index.html`
   - 应用名称、版本、窗口尺寸
   - 可选：图标（选择图片 / 拖放 / URL）
3. 按需展开「窗口与行为」「高级选项」  
4. 可先点 **预览命令**（确认参数是否写入 CLI）  
5. 点 **开始本机打包**，右侧看日志  
6. 产物一般在程序目录下的 `builds/`（也可整理到 `builds/windows/`）

参数模板示例：`configs/windows/`（可自行添加 JSON）。

## 窗口与行为（是否生效）

勾选后都会传给 `pake-cli`。界面已按平台分组：

| 分组 | 示例 | 说明 |
|------|------|------|
| 全平台 | 全屏、置顶、托盘、WASM、快速迭代… | Win / macOS / Linux 产物均可相关 |
| Windows / Linux | 隐藏窗口装饰 | macOS 上忽略 |
| 仅 macOS | 隐藏标题栏 | 本机 Win 勾选也会带参数，但只在 macOS 产物上有效；云端打 Mac 时生效 |

- **启动到托盘** 需同时开启「系统托盘」  
- **快速迭代构建**：只要应用、不要 MSI/DMG，更快  

## 高级：Targets 与注入

- **Targets**：下拉选择「自动 / x64 / arm64」（本机 Windows）  
- **注入 JS/CSS**：
  - 默认扫描 `data/inject/`
  - 「扫描」列出 `.js` / `.css` 后勾选
  - 「上传文件」可多选上传到该目录  
  - 扫描路径需在程序 `data` 目录内  

## 名称与 MSI 注意

- 应用名含**中文**时，WiX 打 **MSI** 常失败（`light.exe` / 代码页问题）  
- 处理办法（任选）：
  1. 名称改用英文，如 `HunliPai-AI`
  2. 勾选 **快速迭代构建**（只要 exe）
- 编译本身可能已成功：可在 pake 缓存目录找到 `pake-*.exe`

## 与云端联动

在 **任务** Tab 对本机成功记录点 **回填·云端**，可带着同一套网址/名称/图标去打 macOS。

## 日志与排障

- GUI：**打开日志** → `data/app.log`  
- 黑窗闪烁：请使用带 `CREATE_NO_WINDOW` 的新版 `PakeGUI.exe`  
- 更多：见 [troubleshooting.md](./troubleshooting.md)

## 局限

- **不能**在 Windows 上交叉编译出可用的 macOS / iOS 包  
- macOS 请走 [macos.md](./macos.md) 云端构建  
- Android 不能靠本机 `pake` 一键出 APK，见 [android.md](./android.md)

完整 CLI 参数见 [Pake CLI 文档](https://github.com/tw93/Pake/blob/main/docs/cli-usage.md)。
