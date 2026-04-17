# GitHub Pages 设置指南

本指南帮助你将 ReignX Demo 页面部署到 GitHub Pages。

## 📋 前置要求

- GitHub 账号
- ReignX 仓库已推送到 GitHub
- 仓库设置权限（需要能修改仓库设置）

## 🚀 快速设置

### 方法 1：使用 GitHub Web 界面（推荐）

1. **访问仓库设置**
   - 打开 https://github.com/liqcui/reignx
   - 点击 `Settings` 标签页

2. **配置 GitHub Pages**
   - 在左侧菜单找到 `Pages`
   - 在 `Build and deployment` 部分：
     - Source: 选择 `Deploy from a branch`
     - Branch: 选择 `main` 分支
     - Folder: 选择 `/docs`
   - 点击 `Save`

3. **等待部署**
   - GitHub Actions 会自动部署
   - 通常需要 1-3 分钟
   - 部署完成后会显示 URL：`https://liqcui.github.io/reignx/`

4. **验证部署**
   - 访问 https://liqcui.github.io/reignx/
   - 应该能看到 ReignX Demo 页面

### 方法 2：使用 GitHub Actions（自动化）

我们已经为你准备好了所有文件：

```
reignx/
├── docs/
│   ├── index.html    # Demo 页面（已复制）
│   └── CNAME         # 自定义域名配置（可选）
├── demo/
│   └── index.html    # 原始 Demo 文件
└── README.md         # 已添加 Demo 链接
```

只需推送到 GitHub：

```bash
cd /Users/liqcui/goproject/github.com/liqcui/reignx

# 添加 docs 目录
git add docs/
git commit -m "Add GitHub Pages demo site"
git push origin main
```

然后在 GitHub 设置中启用 Pages（见方法 1 的步骤 2）。

## 🎨 自定义域名（可选）

如果你有自己的域名，可以配置自定义域名：

### 1. 配置 DNS

在你的域名提供商添加 DNS 记录：

```
类型: CNAME
主机: reignx (或你想要的子域名)
值: liqcui.github.io
```

或使用 A 记录指向 GitHub Pages IP：

```
类型: A
主机: @ 或 reignx
值: 185.199.108.153
值: 185.199.109.153
值: 185.199.110.153
值: 185.199.111.153
```

### 2. 更新 CNAME 文件

编辑 `docs/CNAME`:

```bash
echo "reignx.yourdomain.com" > docs/CNAME
git add docs/CNAME
git commit -m "Add custom domain"
git push origin main
```

### 3. 在 GitHub 设置中配置

- 进入仓库 Settings → Pages
- 在 `Custom domain` 输入你的域名
- 勾选 `Enforce HTTPS`
- 点击 `Save`

## 📝 更新 Demo 页面

当你修改 demo 页面时：

```bash
# 1. 修改源文件
vim demo/index.html

# 2. 复制到 docs 目录
cp demo/index.html docs/index.html

# 3. 提交并推送
git add demo/index.html docs/index.html
git commit -m "Update demo page"
git push origin main

# 4. 等待 GitHub Actions 自动部署（1-3 分钟）
```

## 🔍 故障排查

### 问题 1：404 Not Found

**原因**：Pages 未正确配置或文件路径错误

**解决**：
1. 检查 Settings → Pages 是否启用
2. 确认选择了正确的分支（main）和目录（/docs）
3. 确认 `docs/index.html` 文件存在
4. 等待 3-5 分钟让部署完成

### 问题 2：页面显示不正常

**原因**：CSS/JS 路径问题

**解决**：
1. 检查 `index.html` 中的资源路径
2. 确保所有资源使用相对路径
3. 如果使用图片，确保图片也在 docs 目录中

### 问题 3：自定义域名不工作

**原因**：DNS 配置未生效或 HTTPS 证书未就绪

**解决**：
1. 使用 `dig` 检查 DNS：`dig yourdomain.com`
2. 等待 DNS 传播（可能需要 24-48 小时）
3. HTTPS 证书生成可能需要几个小时
4. 暂时取消勾选 `Enforce HTTPS` 进行测试

### 问题 4：更新未生效

**原因**：浏览器缓存或 CDN 缓存

**解决**：
1. 强制刷新浏览器（Cmd+Shift+R on Mac）
2. 清除浏览器缓存
3. 使用隐身模式访问
4. 等待几分钟让 CDN 缓存更新

## 📊 查看部署状态

### GitHub Actions

1. 访问 https://github.com/liqcui/reignx/actions
2. 查看最新的 `pages build and deployment` 工作流
3. 如果失败，点击查看错误日志

### 部署历史

1. Settings → Pages
2. 查看 `Your site is live at` 下的部署信息
3. 点击链接验证网站

## 🎯 最佳实践

### 1. 保持 demo 和 docs 同步

创建一个更新脚本 `update-demo.sh`:

```bash
#!/bin/bash
# 更新 GitHub Pages demo

set -e

echo "Copying demo to docs..."
cp demo/index.html docs/index.html

echo "Committing changes..."
git add demo/index.html docs/index.html
git commit -m "Update demo page: $(date +%Y-%m-%d)"

echo "Pushing to GitHub..."
git push origin main

echo "✅ Demo updated! Visit https://liqcui.github.io/reignx/"
echo "⏳ Deployment will complete in 1-3 minutes"
```

### 2. 添加 README 到 docs 目录

创建 `docs/README.md`:

```markdown
# ReignX Demo Site

This directory contains the GitHub Pages site for ReignX.

**Live URL**: https://liqcui.github.io/reignx/

## Source

The demo page is generated from `demo/index.html`.

## Updating

To update the demo:
1. Edit `demo/index.html`
2. Run `cp demo/index.html docs/index.html`
3. Commit and push changes
```

### 3. 添加分析（可选）

在 `docs/index.html` 中添加 Google Analytics 或其他分析工具：

```html
<!-- Google Analytics -->
<script async src="https://www.googletagmanager.com/gtag/js?id=GA_MEASUREMENT_ID"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());
  gtag('config', 'GA_MEASUREMENT_ID');
</script>
```

## 📧 获取帮助

如果遇到问题：

1. 查看 [GitHub Pages 文档](https://docs.github.com/en/pages)
2. 检查 [GitHub Status](https://www.githubstatus.com/)
3. 在 ReignX 仓库提交 Issue

## 🔗 相关链接

- **Demo 页面**: https://liqcui.github.io/reignx/
- **GitHub 仓库**: https://github.com/liqcui/reignx
- **GitHub Pages 文档**: https://docs.github.com/en/pages
- **自定义域名指南**: https://docs.github.com/en/pages/configuring-a-custom-domain-for-your-github-pages-site

## ✅ 检查清单

完成以下步骤确保 GitHub Pages 正确配置：

- [ ] docs 目录存在且包含 index.html
- [ ] 代码已推送到 GitHub
- [ ] Settings → Pages 已启用
- [ ] 分支设置为 main，目录设置为 /docs
- [ ] 等待 3-5 分钟部署完成
- [ ] 访问 https://liqcui.github.io/reignx/ 验证
- [ ] README.md 已添加 demo 链接
- [ ] （可选）配置自定义域名
- [ ] （可选）启用 HTTPS
