# Clash Subscription Manager

A small web app for managing Clash subscription links and cached config files.

**[简体中文版 README](README.zh-CN.md)** | English

## Features

- Add Clash subscriptions from a URL
- Save downloaded configs locally
- Update subscription URL, headers, and cached file
- Copy local download URLs from the web UI
- Delete subscriptions and cached files together

## Run locally

```bash
go run .
```

The app reads configuration from `config.yaml` and listens on port `8080` by default.

## Docker

Build:

```bash
docker build -t zhf883680/clash-subscription-manager:latest .
```

Run:

```bash
docker run --rm -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  zhf883680/clash-subscription-manager:latest
```

## Project Links

- **GitHub Repository**: https://github.com/zhf883680/clash-subscription-manager
- **Docker Hub Image**: https://hub.docker.com/r/zhf883680/clash-subscription-manager

