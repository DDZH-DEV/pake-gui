# T04 — GitHub Windows 云端 exe

## 目标

本机不必安装 Visual Studio / MSVC。GUI 提交 GitHub Actions（`windows-latest`），只出 **exe**（patch pake WinBuilder → `tauri --no-bundle`；仅靠 `--iterative-build` 在 Windows 上仍会打 MSI）。

## 验收

- [x] `.github/workflows/build-windows.yml`（`workflow_dispatch` + 自路径 push 注册）  
- [x] 与 macOS 共用 Job / OAuth / 轮询 / 下载  
- [x] 图标上传到 `ci-assets` 分支 `windows/{jobId}/`（构建后清理）  
- [x] GUI「云端目标」可选 Windows exe  
- [x] 产物落到 `builds/windows/`  

## 使用前

必须把 `build-windows.yml` push 到默认分支，等 Actions 登记后再提交（否则会 404）。

用户文档：[docs/windows.md](../../docs/windows.md)
