# T01 — 本地 Windows 打包

## 目标

在 Windows 上用 Pake GUI 调用本机 `pake-cli`，产出 exe / msi。

## 现状

- Go + WebView2 桌面壳、本地 HTTP API、图标上传、单实例、日志等已具备  
- Tab「打包 / 环境 / 任务」  
- Targets 下拉（自动 / x64 / arm64）  
- 注入 JS/CSS：`data/inject/` 扫描 + 上传多选  
- 窗口与行为按平台分组标注  
- 本机历史支持「回填·本机 / 回填·云端」  
- 代码主路径：`internal/pake`、`internal/server`

## 已知问题

- 中文应用名打 MSI 时 WiX `light.exe` 易失败 → 用英文名或「快速迭代构建」只要 exe  

## 建议收尾

- [x] 产物默认落到 `builds/windows/`  
- [x] UI 对 MSI + 非 ASCII 名称：自动转拼音打包名 + 提示  
- [ ] 本地 history 标记 `source: local`  

## 相关路径

- `builds/windows/`（目标）  
- `configs/windows/`（模板，可选）  
- `data/inject/`（注入脚本）  
- 用户文档：[docs/windows.md](../../docs/windows.md)
