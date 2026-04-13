# Clash Subscription Manager

A small web app for managing Clash subscription links and cached config files, with automatic subscription format detection and conversion.

**[简体中文版 README](README.zh-CN.md)** | English

## Screenshots

![Subscription management page](img/截屏2026-04-12%2013.11.54.png)

![Template management page](img/截屏2026-04-12%2013.13.04.png)

![Template editor page](img/截屏2026-04-12%2013.13.17.png)

## Features

- Add subscription URLs and automatically convert supported formats to Clash YAML
- Support direct Clash subscriptions plus automatic conversion for SS, VMess, Trojan, and VLESS links
- Save converted configs locally
- Edit subscription URL, provider filter, and headers without refreshing the cached file
- Refresh a subscription on demand when you want to update the cached file
- Copy local download URLs from the web UI
- Manage multiple Clash templates
- Edit template YAML directly in the web UI
- Choose which subscriptions each template should use, defaulting to all subscriptions
- Download rendered templates with `proxy-providers` generated from the subscriptions selected for that template
- Set an optional per-subscription `filter` value that is injected into rendered `proxy-providers`
- Delete subscriptions and cached files together

## Run locally

```bash
go run .
```

The app reads configuration from `config.yaml` and listens on port `8080` by default.
On startup it logs the current version, for example:

```text
starting clash-subscription-manager v1.0.10 on port 8080
```

## Templates

- Create, edit, and delete multiple templates in the web UI.
- Subscription editing supports both `Save Changes Only` and `Save and Refresh`.
- The saved YAML is the base config. When you download a template, the server replaces `proxy-providers` with entries generated from the subscriptions selected for that template.
- New templates default to selecting all subscriptions, and you can narrow them down per template in the UI.
- If a subscription has a `filter` value, the renderer writes it into that provider entry. Empty `filter` values are omitted.
- `Copy Expanded Proxy URL` outputs flattened `proxies` entries and is suitable for clients such as Shadowrocket and Loon.
- Template render URL: `/api/templates/{id}/render`

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
