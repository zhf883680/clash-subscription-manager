# Clash 订阅管理器

一个用于管理 Clash 订阅链接、原始节点文本和缓存配置文件的小型 Web 应用，支持自动识别订阅格式并转换为 Clash YAML。

**[English Version](README.md)** | 简体中文

## 页面截图

![订阅管理页面](img/截屏2026-04-12%2013.11.54.png)

![模板管理页面](img/截屏2026-04-12%2013.13.04.png)

![模板编辑页面](img/截屏2026-04-12%2013.13.17.png)

## 功能特性

- 支持两种新增订阅方式：通过 URL 下载，或直接粘贴一个/多个原始节点链接并合并为一条订阅
- 支持直接导入 Clash 订阅，以及自动转换 SS、SSR、VMess、Trojan、VLESS 链接
- 在本地保存转换后的配置文件
- 仅修改订阅 URL、节点筛选 Filter、请求头等信息，不刷新缓存文件
- 按需手动更新订阅并刷新缓存文件
- 通过 Web 界面复制本地下载链接
- 管理多份 Clash 模板
- 在页面中直接编辑模板 YAML
- 模板可选择使用哪些订阅，默认全选
- 下载渲染后的模板配置时，自动用该模板选中的订阅填充 `proxy-providers`
- 支持为每个订阅单独设置可选的 `filter`，并在渲染 `proxy-providers` 时写入
- 同时删除订阅和缓存文件

## 本地运行

```bash
go run .
```

应用从 `config.yaml` 读取配置，默认监听端口 `8080`。
启动时会输出当前版本，例如：

```text
starting clash-subscription-manager v1.0.13 on port 8080
```

## 模板说明

- 页面默认先显示“模板管理”，模板页布局为左侧编辑器、右侧模板列表；订阅页布局为左侧订阅列表、右侧新增订阅。
- 模板支持多份创建、编辑、删除。
- 订阅编辑支持”仅保存修改”和”保存并更新”两种方式。
- 页面里编辑的是基础 YAML；下载模板时，系统会自动根据该模板勾选的订阅生成 `proxy-providers`。
- 新建模板默认会选中全部订阅，也可以按模板单独缩小使用范围。
- 如果订阅填写了 `filter`，渲染后的对应 provider 会带上该字段；未填写时不会输出。
- `复制全节点地址` 会输出展开后的 `proxies` 配置，适用于 Shadowrocket、Loon 等软件。
- 模板下载地址：`/api/templates/{id}/render`

## 订阅说明

- 新增订阅支持 `订阅地址` 和 `节点文本` 两种来源。
- 选择 `节点文本` 时，可以直接粘贴多行节点链接，例如 `ss://...`、`vmess://...`、`trojan://...`、`vless://...`、`ssr://...`。
- 多个节点会在一次提交中合并生成一条订阅记录。
- 节点文本模式会自动忽略空行和无法识别的无效行；如果全部内容都无效，会返回错误。
- `自定义请求头` 仅在 `订阅地址` 模式下生效；节点文本模式不需要请求头。

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

## 感谢
[tindy2013/subconverter](https://github.com/tindy2013/subconverter)
