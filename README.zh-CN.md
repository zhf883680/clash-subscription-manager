# Clash 订阅管理器

一个用于管理 Clash 订阅链接和缓存配置文件的小型 Web 应用。

**[English Version](README.md)** | 简体中文

## 功能特性

- 通过 URL 添加 Clash 订阅
- 在本地保存下载的配置文件
- 更新订阅 URL、请求头和缓存文件
- 通过 Web 界面复制本地下载链接
- 管理多份 Clash 模板并设置默认模板
- 在页面中直接编辑模板 YAML
- 下载渲染后的模板配置，自动用当前全部订阅填充 `proxy-providers`
- 同时删除订阅和缓存文件

## 本地运行

```bash
go run .
```

应用从 `config.yaml` 读取配置，默认监听端口 `8080`。

## 模板说明

- 模板支持多份创建、编辑、删除，并可设置一个默认模板。
- 页面里编辑的是基础 YAML；下载模板时，系统会自动根据当前全部订阅生成 `proxy-providers`。
- 单个模板下载地址：`/api/templates/{id}/render`
- 默认模板下载地址：`/api/templates/default/render`

## Docker 使用

构建镜像：

```bash
docker build -t zhf883680/clash-subscription-manager:latest .
```

运行容器：

```bash
docker run --rm -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  zhf883680/clash-subscription-manager:latest
```

## 项目链接

- **GitHub 仓库**: https://github.com/zhf883680/clash-subscription-manager
- **Docker Hub 镜像**: https://hub.docker.com/r/zhf883680/clash-subscription-manager
