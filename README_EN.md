# Clash Subscription Manager

A small web app for managing Clash subscription links, raw node text, and cached config files, with automatic subscription format detection and conversion.


## Screenshots

![Subscription management page](img/截屏2026-04-12%2013.11.54.png)

![Template management page](img/截屏2026-04-12%2013.13.04.png)

![Template editor page](img/截屏2026-04-12%2013.13.17.png)

## Features

- Add subscriptions in two ways: download from a URL, or paste one or more raw node links and merge them into one subscription
- Support direct Clash subscriptions plus automatic conversion for SS, SSR, VMess, Trojan, and VLESS links
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
starting clash-subscription-manager v1.0.13 on port 8080
```

## Templates

- The UI now opens on `Template Management` first. The template page uses an editor-left/list-right layout, while the subscription page uses a list-left/create-right layout.
- Create, edit, and delete multiple templates in the web UI.
- Subscription editing supports both `Save Changes Only` and `Save and Refresh`.
- The saved YAML is the base config. When you download a template, the server replaces `proxy-providers` with entries generated from the subscriptions selected for that template.
- New templates default to selecting all subscriptions, and you can narrow them down per template in the UI.
- If a subscription has a `filter` value, the renderer writes it into that provider entry. Empty `filter` values are omitted.
- `Copy Expanded Proxy URL` outputs flattened `proxies` entries and is suitable for clients such as Shadowrocket and Loon.
- Template render URL: `/api/templates/{id}/render`

## Subscriptions

- The create form supports both `Subscription URL` and `Node Text` sources.
- In `Node Text` mode, you can paste multiple lines such as `ss://...`, `vmess://...`, `trojan://...`, `vless://...`, and `ssr://...`.
- Multiple pasted nodes are converted and stored as one subscription record in a single submission.
- Blank lines and invalid lines are ignored in node-text mode. If every line is invalid, the request fails.
- `Custom Headers` only apply to URL-based subscriptions and are hidden for node-text mode.

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

## Thanks
[tindy2013/subconverter](https://github.com/tindy2013/subconverter)
