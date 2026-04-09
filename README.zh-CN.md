# Clash 订阅管理器

一个用于管理 Clash 订阅链接和缓存配置文件的小型 Web 应用。

**[English Version](README.md)** | 简体中文

## 功能特性

- 通过 URL 添加 Clash 订阅
- 在本地保存下载的配置文件
- 更新订阅 URL、请求头和缓存文件
- 通过 Web 界面复制本地下载链接
- 同时删除订阅和缓存文件

## 本地运行

```bash
go run .
```

应用从 `config.yaml` 读取配置，默认监听端口 `8080`。

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

