# T01 — 本地 Windows 打包

## 目标

在 Windows 上用 Pake GUI 调用本机 `pake-cli`，产出 exe / msi。

## 现状

- Go + WebView2 桌面壳、本地 HTTP API、图标上传、单实例、日志等已具备
- 代码主路径：`internal/pake`、`internal/server`

## 已知问题

- 中文应用名打 MSI 时 WiX `light.exe` 易失败 → 提示用英文名或安装语言 `zh-CN`，或 `--iterative-build` 只要 exe

## 建议收尾

- [ ] 产物默认落到 `builds/windows/`
- [ ] UI 提示 MSI + 非 ASCII 名称风险
- [ ] 本地 history 标记 `source: local`

## 相关路径

- `builds/windows/`（目标）
- `configs/windows/`（模板，可选）
