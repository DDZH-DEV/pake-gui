# 快速开始

## 1. 运行 Pake GUI（Windows）

```bat
cd c:\Users\Administrator\Desktop\pake-gui
build.bat
```

或直接双击已生成的 `PakeGUI.exe`。

调试可用：

```bat
PakeGUI.exe -browser
```

## 2. 界面结构（Tab）

| Tab | 用途 |
|-----|------|
| **打包** | 填写网址、名称、图标、窗口选项；**本机 Windows 打包**；看构建日志 |
| **云端** | GitHub 授权与仓库设置；**提交 macOS 云端任务**；看云端进度 |
| **任务** | 本机历史 + 云端任务列表 |
| **环境** | Node / npm / Rust / pake-cli 检测；一键安装 pake-cli |

云端打包会复用「打包」页填写的网址、名称、图标等字段。

## 3. 按目标选路径

### 只要 Windows 安装包 / exe

1. 「环境」确认 Node、pake-cli、Rust、MSVC 就绪  
2. 「打包」填参数 → **开始本机打包**  
3. 详见 [windows.md](./windows.md)

### 要 macOS DMG（无需本机 Mac）

1. 代码已推到 GitHub，Actions 可用  
2. 配置 OAuth（见 [github-oauth.md](./github-oauth.md)）  
3. 「云端」授权 → **提交 macOS 云端**  
4. 详见 [macos.md](./macos.md)

### 要 Android APK

当前未实现，见 [android.md](./android.md)。

## 4. 默认仓库

云端默认：

- Owner：`DDZH-DEV`
- Repo：`pake-gui`
- Workflow：`build-macos.yml`

可在「云端」Tab 修改。
