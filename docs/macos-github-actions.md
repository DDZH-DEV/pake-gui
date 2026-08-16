# 用 GitHub Actions 打 macOS 包（方案 B）

本仓库已内置工作流：`.github/workflows/build-macos.yml`。  
在 **macOS 云端 runner** 上调用 `pake-cli`，无需本机 Mac / Xcode。

## 一次性前置

1. 注册 / 登录 [GitHub](https://github.com)
2. 新建空仓库（例如 `pake-gui`），**不要**勾选自动添加 README
3. 把本项目推上去：

```bat
cd c:\Users\Administrator\Desktop\pake-gui
git remote add origin https://github.com/<你的用户名>/pake-gui.git
git push -u origin master
```

4. 打开仓库 → **Settings → Actions → General**  
   确认 Actions 已启用（默认一般可用）

## 在 Pake GUI 里提交（推荐）

1. 推送本仓库到 GitHub（需包含 `.github/workflows/build-macos.yml`）
2. 创建 PAT（权限：`repo` + `workflow`）
3. 打开 Pake GUI → 展开「云端打包 · GitHub 设置」→ 填写 Owner/Repo/Token → 保存 → 测试连接
4. 填好网址、应用名、图标后点「提交 macOS 云端」
5. 右侧「云端任务」查看进度；成功后产物在 `builds/macos/`

本地图标会上传到仓库 `ci-assets/macos/{jobId}/`，Actions checkout 后直接使用该路径（私有仓库也可用）。

## 在 GitHub 网页手动跑

1. 打开仓库 **Actions** 页
2. 左侧选 **Build macOS App (Pake)**
3. **Run workflow**，建议填写：

| 字段 | 示例 |
|------|------|
| url | `https://hlai-pc.yingdedao.cn/` |
| name | `HunliPai-AI`（建议英文，方便下载） |
| icon | 公开图片 URL 或仓库内路径；空则自动抓站点图标 |
| identifier | `com.hlpai.yingdedao` |
| targets | `dmg`（默认）或 `app` |
| multi_arch | 需要 Intel+Apple 通用包再勾 |

4. 等待约 10–20 分钟（首次更慢）
5. 进入成功的 run → **Artifacts** → 下载 `HunliPai-AI-macOS`

参考参数也可看：`configs/macos/hunlipai-ai.json`。

## 说明

- 产物默认 **未签名**。自己用：下载后可能需右键打开或在「隐私与安全性」里允许。
- 要分发给他人、减少 Gatekeeper 拦截，需 Apple 开发者账号做签名与公证（后续再加）。
- 备选：也可 [Fork tw93/Pake](https://github.com/tw93/Pake/fork)，用官方工作流 `Build App With Pake CLI`（选 `macos-latest`）。

## 可选：本机安装 GitHub CLI

便于用命令触发构建（需先 `gh auth login`）：

```bat
winget install --id GitHub.cli -e
gh auth login
gh workflow run "Build macOS App (Pake)" -f url=https://hlai-pc.yingdedao.cn/ -f name=HunliPai-AI -f identifier=com.hlpai.yingdedao
```

