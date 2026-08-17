# GitHub 授权

云端打包（macOS / Windows exe / Android APK）需要访问你的仓库并触发 Actions。  
推荐 **Device Flow**；也可手动粘贴 PAT。

默认仓库：`DDZH-DEV/pake-gui`

## 方式一：Device Flow（推荐）

### 1. 创建 OAuth App

1. 打开 https://github.com/settings/developers → **OAuth Apps** → New  
2. 建议填写：
   - Application name：`Pake GUI`
   - Homepage URL：`https://github.com/DDZH-DEV/pake-gui`
   - Authorization callback URL：`http://127.0.0.1/`（占位即可）
3. **启用 Device Flow**  
4. 复制 **Client ID**  
5. **不要**把 Client Secret 提交到公开仓库或写进 exe

### 2. 在 GUI 中登录

1. 打开 **云端** Tab  
2. 粘贴 Client ID（或写入 `configs/github-oauth.json` 的 `clientId`）  
   - **不要**填设置页 URL 里的数字（如 `…/applications/3797658` 中的 `3797658`）  
3. 点 **使用 GitHub 授权**  
4. 浏览器应自动打开；若没有，点界面上的链接，或打开 https://github.com/login/device 并输入设备码  
5. 成功后显示「已登录 @用户名」  
6. 点 **测试连接** 确认能读仓库与 Actions

申请的 scope：`repo`、`workflow`、`read:user`。

### 3. 配置文件（可选）

`configs/github-oauth.json`：

```json
{
  "clientId": "你的ClientID",
  "defaultOwner": "DDZH-DEV",
  "defaultRepo": "pake-gui"
}
```

本机登录态与 Token 保存在 `data/github.json`（已 gitignore，勿提交）。

## 方式二：Personal Access Token（备用）

1. GitHub → Settings → Developer settings → Personal access tokens  
2. 权限至少包含：访问目标仓库、`workflow`  
3. 在 **云端** Tab →「高级：手动粘贴 Token」→ **保存设置** → **测试连接**

适合临时调试；日常更推荐 Device Flow。

## 退出登录

云端 Tab → **退出登录**：清除本机 Token，不影响仓库或 OAuth App。

## 安全注意

| 项 | 说明 |
|----|------|
| Token | 只存 `data/github.json`，权限尽量最小 |
| Client ID | 可公开 |
| Client Secret | 禁止进仓库 / GUI 配置 |
| 日志 | 不会打印完整 Token |

## 相关

- macOS 云端流程：[macos.md](./macos.md)  
- 快速开始：[getting-started.md](./getting-started.md)
