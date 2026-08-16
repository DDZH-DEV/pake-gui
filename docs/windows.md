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
4. 可先点 **预览命令**  
5. 点 **开始本机打包**，右侧看日志  
6. 产物一般在程序目录下的 `builds/`（建议逐步归入 `builds/windows/`）

参数模板示例：`configs/windows/`（可自行添加 JSON）。

## 名称与 MSI 注意

- 应用名含**中文**时，WiX 打 **MSI** 常失败（`light.exe` / 代码页问题）  
- 处理办法（任选）：
  1. 名称改用英文，如 `HunliPai-AI`
  2. 安装语言设为 `zh-CN`（若 CLI/界面提供 installer language）
  3. 勾选 **快速迭代构建**（只要 exe，不打安装包）
- 编译本身可能已成功：可在 pake 缓存目录找到 `pake-*.exe`，或使用 GUI 拷到 `builds/` 的副本

## 常用选项说明

| 选项 | 说明 |
|------|------|
| 隐藏窗口装饰 | Windows/Linux 无边框 |
| 快速迭代构建 | 更快，通常不生成 MSI/DMG |
| 保留原始二进制 | 安装包旁保留 exe |
| 本地文件模式 | 打包本地静态站点时勾选 |
| WASM | Flutter Web 等需要时开启 |

完整 CLI 参数见 [Pake CLI 文档](https://github.com/tw93/Pake/blob/main/docs/cli-usage.md)。

## 日志与排障

- GUI：**打开日志** → `data/app.log`  
- 黑窗闪烁：新版本已用 `CREATE_NO_WINDOW` 隐藏；请使用最新 `PakeGUI.exe`  
- 更多：见 [troubleshooting.md](./troubleshooting.md)

## 局限

- **不能**在 Windows 上交叉编译出可用的 macOS / iOS 包  
- macOS 请走 [macos.md](./macos.md) 云端构建  
- Android 不能靠本机 `pake` 一键出 APK，见 [android.md](./android.md)
