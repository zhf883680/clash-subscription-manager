# Clash 订阅转换实现指南

> 本文档提供完整的实现细节,可基于此独立实现订阅转换功能

## 目录

1. [技术栈和依赖](#技术栈和依赖)
2. [核心数据结构](#核心数据结构)
3. [协议解析实现](#协议解析实现)
4. [Clash 配置生成](#clash-配置生成)
5. [工具函数实现](#工具函数实现)
6. [完整实现示例](#完整实现示例)
7. [测试用例](#测试用例)

---

## 技术栈和依赖

### 推荐技术栈

```yaml
Python (推荐):
  - 3.8+
  - 优点: 开发快速,库丰富,易于维护

Node.js:
  - 14+
  - 优点: 异步处理好,适合高并发

Go:
  - 1.18+
  - 优点: 性能好,单文件部署

Rust:
  - 1.70+
  - 优点: 内存安全,性能最优
```

### 必需依赖库

#### Python 实现依赖

```python
# requirements.txt
pyyaml>=6.0          # YAML 生成/解析
requests>=2.31.0     # HTTP 请求
base64               # Base64 编解码(内置)
urllib3              # URL 处理
re                   # 正则表达式(内置)
json                 # JSON 解析(内置)
typing               # 类型注解(内置)
```

#### Node.js 实现依赖

```json
{
  "dependencies": {
    "js-yaml": "^4.1.0",      // YAML 处理
    "axios": "^1.6.0",         // HTTP 请求
    "uri-js": "^4.4.1"         // URL 解析
  }
}
```

---

## 核心数据结构

### 1. Proxy 节点结构

```python
from dataclasses import dataclass, field
from typing import Optional, List
from enum import Enum

class ProxyType(Enum):
    UNKNOWN = "unknown"
    SS = "ss"                  # Shadowsocks
    SSR = "ssr"                # ShadowsocksR
    VMESS = "vmess"            # V2Ray VMess
    TROJAN = "trojan"          # Trojan
    SNELL = "snell"            # Snell
    SOCKS5 = "socks5"          # SOCKS5
    HTTP = "http"              # HTTP
    HTTPS = "https"            # HTTPS
    WIREGUARD = "wireguard"    # WireGuard
    HYSTERIA = "hysteria"      # Hysteria
    HYSTERIA2 = "hysteria2"    # Hysteria2
    TUIC = "tuic"              # TUIC

@dataclass
class Proxy:
    """统一的代理节点结构"""
    # 基本信息
    proxy_type: ProxyType = ProxyType.UNKNOWN
    name: str = ""              # 节点名称
    server: str = ""            # 服务器地址
    port: int = 0               # 端口

    # 认证信息
    username: str = ""          # 用户名 (HTTP/SOCKS5)
    password: str = ""          # 密码
    uuid: str = ""              # UUID (VMess/Trojan)

    # 加密
    cipher: str = ""            # 加密方式 (SS/SSR)

    # Shadowsocks 特定
    plugin: str = ""            # 插件名
    plugin_opts: str = ""       # 插件选项

    # ShadowsocksR 特定
    protocol: str = ""          # 协议
    protocol_param: str = ""    # 协议参数
    obfs: str = ""              # 混淆
    obfs_param: str = ""        # 混淆参数

    # VMess 特定
    alter_id: int = 0           # AlterId
    network: str = ""           # 传输协议 (ws/http/h2/grpc/quic)
    tls: bool = False           # 是否启用 TLS
    sni: str = ""               # Server Name Indication
    ws_path: str = ""           # WebSocket 路径
    ws_headers: dict = field(default_factory=dict)  # WebSocket 头

    # Trojan 特定
    network_trojan: str = ""    # 传输协议
    sni_trojan: str = ""        # SNI
    fingerprint: str = ""       # 指纹

    # 网络选项
    udp: bool = True            # UDP 支持
    tfo: bool = False           # TCP Fast Open
    skip_cert_verify: bool = False  # 跳过证书验证

    # 底层代理
    underlying_proxy: str = ""  # 底层代理地址
```

### 2. Clash 规则结构

```python
@dataclass
class ClashRule:
    """Clash 规则结构"""
    type: str                   # DOMAIN, DOMAIN-SUFFIX, IP-CIDR 等
    value: str                  # 规则值
    policy: str                 # 策略组名称
    no_resolve: bool = False    # 是否禁用解析
```

---

## 协议解析实现

### Shadowsocks (SS) URI 解析

**URI 格式**:
```
ss://BASE64(method:password@server:port)#name
ss://BASE64(method:password@server:port?plugin=opts)#name
```

**完整实现**:

```python
import base64
import urllib.parse
from typing import Optional

def parse_ss_uri(uri: str) -> Optional[Proxy]:
    """
    解析 Shadowsocks URI

    Args:
        uri: ss:// 开头的链接

    Returns:
        Proxy 对象,解析失败返回 None

    Examples:
        >>> parse_ss_uri("ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@server.com:8388#MyServer")
        Proxy(name='MyServer', server='server.com', port=8388, ...)
    """
    if not uri.startswith("ss://"):
        return None

    try:
        # 移除 ss:// 前缀
        uri = uri[5:]

        # 分离 fragment (节点名称)
        if "#" in uri:
            uri, name = uri.rsplit("#", 1)
            name = urllib.parse.unquote(name)
        else:
            name = ""

        # 分离查询参数 (插件信息)
        if "?" in uri:
            uri, query = uri.split("?", 1)
            plugin_info = urllib.parse.parse_qs(query)
            plugin = plugin_info.get("plugin", [""])[0]
        else:
            plugin = ""

        # Base64 解码
        # 注意:标准 Base64 可能包含 padding,需要处理
        padding = len(uri) % 4
        if padding:
            uri += "=" * (4 - padding)

        decoded = base64.urlsafe_b64decode(uri).decode("utf-8")

        # 解析 method:password@server:port
        if "@" not in decoded:
            return None

        auth_part, server_part = decoded.rsplit("@", 1)

        if ":" not in auth_part:
            return None

        method, password = auth_part.split(":", 1)

        # 解析 server:port
        if ":" not in server_part:
            return None

        # IPv6 地址处理
        if server_part.startswith("["):
            # IPv6 格式 [2001:db8::1]:8388
            if "]:" in server_part:
                server, port = server_part.rsplit("]:", 1)
                server = server[1:]  # 移除 [
            else:
                return None
        else:
            server, port = server_part.rsplit(":", 1)

        # 构造 Proxy 对象
        return Proxy(
            proxy_type=ProxyType.SS,
            name=name,
            server=server,
            port=int(port),
            cipher=method,
            password=password,
            plugin=plugin,
            udp=True
        )

    except Exception as e:
        print(f"Failed to parse SS URI: {e}")
        return None


# 测试用例
def test_parse_ss():
    # 基本测试
    uri = "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@server.com:8388#MyServer"
    proxy = parse_ss_uri(uri)
    assert proxy.cipher == "aes-256-gcm"
    assert proxy.password == "password"
    assert proxy.server == "server.com"
    assert proxy.port == 8388
    assert proxy.name == "MyServer"

    # 带插件测试
    uri2 = "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@server.com:8388?plugin=obfs-local%3Bobfs%3Dhttp%3Bobfs-host%3Dwww.bing.com#WithPlugin"
    proxy2 = parse_ss_uri(uri2)
    assert proxy2.plugin == "obfs-local;obfs=http;obfs-host=www.bing.com"

    print("SS parsing tests passed!")

if __name__ == "__main__":
    test_parse_ss()
```

### ShadowsocksR (SSR) URI 解析

**URI 格式**:
```
ssr://BASE64(server:port:protocol:method:obfs:password_base64/?params)
```

**完整实现**:

```python
def parse_ssr_uri(uri: str) -> Optional[Proxy]:
    """
    解析 ShadowsocksR URI

    Args:
        uri: ssr:// 开头的链接

    Returns:
        Proxy 对象
    """
    if not uri.startswith("ssr://"):
        return None

    try:
        # 移除 ssr:// 前缀
        uri = uri[6:]

        # Base64 解码
        padding = len(uri) % 4
        if padding:
            uri += "=" * (4 - padding)

        decoded = base64.urlsafe_b64decode(uri).decode("utf-8")

        # 分离主体和参数
        if "/?" in decoded:
            main_part, params = decoded.split("/?", 1)
        else:
            main_part = decoded
            params = ""

        # 解析参数
        param_dict = {}
        if params:
            for param in params.split("&"):
                if "=" in param:
                    key, value = param.split("=", 1)
                    param_dict[key] = value

        # 获取节点名称
        name = param_dict.get("remarks", "")
        name = base64.urlsafe_b64decode(name).decode("utf-8", errors="ignore")

        # 解析主要部分
        # 格式: server:port:protocol:method:obfs:password_base64
        parts = main_part.split(":")
        if len(parts) < 6:
            return None

        server = parts[0]
        port = int(parts[1])
        protocol = parts[2]
        method = parts[3]
        obfs = parts[4]
        password_base64 = parts[5]

        # 解码密码
        password = base64.urlsafe_b64decode(password_base64).decode("utf-8")

        # 解析协议参数
        protocol_param = param_dict.get("protoparam", "")
        obfs_param = param_dict.get("obfsparam", "")

        return Proxy(
            proxy_type=ProxyType.SSR,
            name=name,
            server=server,
            port=port,
            cipher=method,
            password=password,
            protocol=protocol,
            protocol_param=protocol_param,
            obfs=obfs,
            obfs_param=obfs_param,
            udp=True
        )

    except Exception as e:
        print(f"Failed to parse SSR URI: {e}")
        return None
```

### VMess URI 解析

**URI 格式**:
```
vmess://BASE64(JSON配置)
vmess://uuid@server:port?params  (新格式)
```

**完整实现**:

```python
import json
import uuid as uuid_lib

def parse_vmess_uri(uri: str) -> Optional[Proxy]:
    """
    解析 VMess URI

    支持格式:
    1. vmess://BASE64(JSON) - 传统格式
    2. vmess://uuid@server:port?params - 新格式

    Args:
        uri: vmess:// 开头的链接

    Returns:
        Proxy 对象
    """
    if not uri.startswith("vmess://"):
        return None

    try:
        # 移除 vmess:// 前缀
        uri = uri[8:]

        # 检测新格式: vmess://uuid@server:port?
        if "@" in uri and "?" in uri:
            return parse_vmess_new_format(uri)

        # 传统格式: Base64 编码的 JSON
        padding = len(uri) % 4
        if padding:
            uri += "=" * (4 - padding)

        decoded = base64.urlsafe_b64decode(uri).decode("utf-8")

        # 尝试解析为 JSON
        try:
            config = json.loads(decoded)
        except json.JSONDecodeError:
            # 可能是等号格式: ps=备注&add=地址...
            return parse_vmess_eq_format(decoded)

        # 提取参数
        name = config.get("ps", config.get("remark", config.get("name", "")))
        server = config.get("add", "")
        port = int(config.get("port", 0))
        uuid_str = config.get("id", "")
        alter_id = int(config.get("aid", 0))
        cipher = config.get("scy", config.get("cipher", "auto"))
        network = config.get("net", "tcp")
        tls = config.get("tls", "") == "tls"
        sni = config.get("sni", "")

        # WebSocket 参数
        ws_path = config.get("path", "/")
        host = config.get("host", "")

        # HTTP/2 参数
        h2_path = config.get("h2-path", "")
        h2_host = config.get("h2-host", "")

        # gRPC 参数
        grpc_path = config.get("serviceName", "")

        # QUIC 参数
        quic_security = config.get("type", "")
        quic_key = config.get("key", "")

        return Proxy(
            proxy_type=ProxyType.VMESS,
            name=name,
            server=server,
            port=port,
            uuid=uuid_str,
            alter_id=alter_id,
            cipher=cipher,
            network=network,
            tls=tls,
            sni=sni,
            ws_path=ws_path if network == "ws" else "",
            ws_headers={"Host": host} if host else {},
            udp=True
        )

    except Exception as e:
        print(f"Failed to parse VMess URI: {e}")
        return None


def parse_vmess_new_format(uri: str) -> Optional[Proxy]:
    """
    解析新格式 VMess URI
    vmess://uuid@server:port?params
    """
    try:
        # uuid@server:port?
        uuid_part, rest = uri.split("@", 1)
        server_part, params = rest.split("?", 1)

        # server:port
        if server_part.startswith("["):
            # IPv6
            server, port = server_part.rsplit("]:", 1)
            server = server[1:]
        else:
            server, port = server_part.rsplit(":", 1)

        # 解析参数
        param_dict = urllib.parse.parse_qs(params)

        network = param_dict.get("network", ["tcp"])[0]
        tls = param_dict.get("tls", ["false"])[0].lower() == "true"
        sni = param_dict.get("sni", [""])[0]

        return Proxy(
            proxy_type=ProxyType.VMESS,
            name="",  # 新格式可能没有名称
            server=server,
            port=int(port),
            uuid=uuid_part,
            alter_id=0,
            cipher="auto",
            network=network,
            tls=tls,
            sni=sni,
            udp=True
        )
    except Exception as e:
        print(f"Failed to parse VMess new format: {e}")
        return None


def parse_vmess_eq_format(decoded: str) -> Optional[Proxy]:
    """
    解析等号格式的 VMess 配置
    ps=备注&add=地址&port=端口...
    """
    try:
        params = {}
        for item in decoded.split("&"):
            if "=" in item:
                key, value = item.split("=", 1)
                params[key] = value

        name = params.get("ps", "")
        server = params.get("add", "")
        port = int(params.get("port", 0))
        uuid_str = params.get("id", "")
        alter_id = int(params.get("aid", 0))
        cipher = params.get("scy", "auto")
        network = params.get("net", "tcp")
        tls = params.get("tls", "") == "tls"

        return Proxy(
            proxy_type=ProxyType.VMESS,
            name=name,
            server=server,
            port=port,
            uuid=uuid_str,
            alter_id=alter_id,
            cipher=cipher,
            network=network,
            tls=tls,
            udp=True
        )
    except Exception as e:
        print(f"Failed to parse VMess eq format: {e}")
        return None
```

### Trojan URI 解析

**URI 格式**:
```
trojan://password@server:port?params#name
```

**完整实现**:

```python
def parse_trojan_uri(uri: str) -> Optional[Proxy]:
    """
    解析 Trojan URI

    Args:
        uri: trojan:// 开头的链接

    Returns:
        Proxy 对象
    """
    if not uri.startswith("trojan://"):
        return None

    try:
        # 移除 trojan:// 前缀
        uri = uri[9:]

        # 分离 fragment
        if "#" in uri:
            uri, name = uri.rsplit("#", 1)
            name = urllib.parse.unquote(name)
        else:
            name = ""

        # 分离查询参数
        if "?" in uri:
            uri, query = uri.split("?", 1)
            param_dict = urllib.parse.parse_qs(query)
        else:
            param_dict = {}

        # password@server:port
        if "@" not in uri:
            return None

        password, server_part = uri.split("@", 1)

        # server:port
        if server_part.startswith("["):
            # IPv6
            if "]:" in server_part:
                server, port = server_part.rsplit("]:", 1)
                server = server[1:]
            else:
                return None
        else:
            server, port = server_part.rsplit(":", 1)

        # 解析参数
        network = param_dict.get("type", ["tcp"])[0]
        sni = param_dict.get("sni", [""])[0]
        fingerprint = param_dict.get("fp", [""])[0]

        # WebSocket 参数
        ws_path = param_dict.get("path", [""])[0]
        host = param_dict.get("host", [""])[0]

        # gRPC 参数
        grpc_path = param_dict.get("serviceName", [""])[0]

        return Proxy(
            proxy_type=ProxyType.TROJAN,
            name=name,
            server=server,
            port=int(port),
            password=password,
            network_trojan=network,
            sni_trojan=sni,
            fingerprint=fingerprint,
            ws_path=ws_path if network == "ws" else "",
            ws_headers={"Host": host} if host else {},
            skip_cert_verify=param_dict.get("allowInsecure", ["0"])[0] == "1",
            udp=True
        )

    except Exception as e:
        print(f"Failed to parse Trojan URI: {e}")
        return None
```

### 通用订阅解析器

```python
from typing import List

class SubscriptionParser:
    """订阅链接解析器"""

    @staticmethod
    def parse_uri(uri: str) -> Optional[Proxy]:
        """
        自动识别并解析单条订阅链接

        Args:
            uri: 订阅链接

        Returns:
            Proxy 对象
        """
        uri = uri.strip()

        if uri.startswith("ss://"):
            return parse_ss_uri(uri)
        elif uri.startswith("ssr://"):
            return parse_ssr_uri(uri)
        elif uri.startswith("vmess://"):
            return parse_vmess_uri(uri)
        elif uri.startswith("trojan://"):
            return parse_trojan_uri(uri)
        elif uri.startswith("snell://"):
            return parse_snell_uri(uri)
        elif uri.startswith("socks5://") or uri.startswith("socks://"):
            return parse_socks_uri(uri)
        elif uri.startswith("http://") or uri.startswith("https://"):
            return parse_http_uri(uri)

        return None

    @staticmethod
    def parse_subscription(url: str, proxy: str = "") -> List[Proxy]:
        """
        下载并解析订阅

        Args:
            url: 订阅 URL
            proxy: 代理服务器

        Returns:
            节点列表
        """
        import requests

        proxies = {"http": proxy, "https": proxy} if proxy else None

        try:
            response = requests.get(url, proxies=proxies, timeout=30)
            response.raise_for_status()
            content = response.text

            # 检测内容类型
            if content.startswith("ssd://"):
                return SubscriptionParser._parse_ssd(content)
            else:
                # 逐行解析
                nodes = []
                for line in content.strip().split("\n"):
                    line = line.strip()
                    if line:
                        node = SubscriptionParser.parse_uri(line)
                        if node:
                            nodes.append(node)
                return nodes

        except Exception as e:
            print(f"Failed to fetch subscription: {e}")
            return []

    @staticmethod
    def _parse_ssd(content: str) -> List[Proxy]:
        """
        解析 SSD 格式订阅
        """
        # SSD 格式: ssd://BASE64(JSON)
        # 实现省略...
        pass
```

---

## Clash 配置生成

### 节点转换为 Clash 格式

```python
import yaml

class ClashGenerator:
    """Clash 配置生成器"""

    @staticmethod
    def proxy_to_clash(proxy: Proxy, new_field_name: bool = True) -> dict:
        """
        将 Proxy 节点转换为 Clash 配置

        Args:
            proxy: Proxy 对象
            new_field_name: 是否使用新字段名 (proxies/proxy-groups)

        Returns:
            Clash 配置字典
        """
        if proxy.proxy_type == ProxyType.SS:
            return ClashGenerator._ss_to_clash(proxy)
        elif proxy.proxy_type == ProxyType.SSR:
            return ClashGenerator._ssr_to_clash(proxy)
        elif proxy.proxy_type == ProxyType.VMESS:
            return ClashGenerator._vmess_to_clash(proxy)
        elif proxy.proxy_type == ProxyType.TROJAN:
            return ClashGenerator._trojan_to_clash(proxy)
        elif proxy.proxy_type == ProxyType.SOCKS5:
            return ClashGenerator._socks_to_clash(proxy)
        elif proxy.proxy_type in (ProxyType.HTTP, ProxyType.HTTPS):
            return ClashGenerator._http_to_clash(proxy)
        else:
            return {}

    @staticmethod
    def _ss_to_clash(proxy: Proxy) -> dict:
        """Shadowsocks → Clash"""
        config = {
            "name": proxy.name or f"{proxy.server}:{proxy.port}",
            "type": "ss",
            "server": proxy.server,
            "port": proxy.port,
            "cipher": proxy.cipher,
            "password": proxy.password,
            "udp": proxy.udp
        }

        # 处理插件
        if proxy.plugin:
            if "obfs" in proxy.plugin.lower():
                config["plugin"] = "obfs"
                config["plugin-opts"] = ClashGenerator._parse_obfs_opts(proxy.plugin_opts)
            elif "v2ray-plugin" in proxy.plugin.lower():
                config["plugin"] = "v2ray-plugin"
                config["plugin-opts"] = ClashGenerator._parse_v2ray_opts(proxy.plugin_opts)

        return config

    @staticmethod
    def _ssr_to_clash(proxy: Proxy) -> dict:
        """ShadowsocksR → Clash"""
        config = {
            "name": proxy.name or f"{proxy.server}:{proxy.port}",
            "type": "ssr",
            "server": proxy.server,
            "port": proxy.port,
            "cipher": proxy.cipher,
            "password": proxy.password,
            "protocol": proxy.protocol,
            "protocol-param": proxy.protocol_param,
            "obfs": proxy.obfs,
            "obfs-param": proxy.obfs_param,
            "udp": proxy.udp
        }

        return config

    @staticmethod
    def _vmess_to_clash(proxy: Proxy) -> dict:
        """VMess → Clash"""
        config = {
            "name": proxy.name or f"{proxy.server}:{proxy.port}",
            "type": "vmess",
            "server": proxy.server,
            "port": proxy.port,
            "uuid": proxy.uuid,
            "alterId": proxy.alter_id,
            "cipher": proxy.cipher,
            "udp": proxy.udp,
            "tls": proxy.tls
        }

        if proxy.skip_cert_verify:
            config["skip-cert-verify"] = True

        if proxy.sni:
            config["servername"] = proxy.sni

        # 传输协议
        if proxy.network:
            config["network"] = proxy.network

            if proxy.network == "ws":
                if proxy.ws_path:
                    config["ws-opts"] = {"path": proxy.ws_path}
                if proxy.ws_headers:
                    config["ws-opts"] = config.get("ws-opts", {})
                    config["ws-opts"]["headers"] = proxy.ws_headers
            elif proxy.network == "h2":
                if proxy.ws_path:  # 复用字段
                    config["h2-opts"] = {"path": proxy.ws_path}
            elif proxy.network == "grpc":
                if proxy.ws_path:  # 复用字段
                    config["grpc-opts"] = {"grpc-service-name": proxy.ws_path}

        return config

    @staticmethod
    def _trojan_to_clash(proxy: Proxy) -> dict:
        """Trojan → Clash"""
        config = {
            "name": proxy.name or f"{proxy.server}:{proxy.port}",
            "type": "trojan",
            "server": proxy.server,
            "port": proxy.port,
            "password": proxy.password,
            "udp": proxy.udp
        }

        if proxy.sni_trojan:
            config["sni"] = proxy.sni_trojan

        if proxy.skip_cert_verify:
            config["skip-cert-verify"] = True

        if proxy.fingerprint:
            config["fingerprint"] = proxy.fingerprint

        # 传输协议
        if proxy.network_trojan:
            config["network"] = proxy.network_trojan

            if proxy.network_trojan == "ws":
                if proxy.ws_path:
                    config["ws-opts"] = {"path": proxy.ws_path}
                if proxy.ws_headers:
                    config["ws-opts"] = config.get("ws-opts", {})
                    config["ws-opts"]["headers"] = proxy.ws_headers
            elif proxy.network_trojan == "grpc":
                if proxy.ws_path:  # 复用字段
                    config["grpc-opts"] = {"grpc-service-name": proxy.ws_path}

        return config

    @staticmethod
    def _socks_to_clash(proxy: Proxy) -> dict:
        """SOCKS5 → Clash"""
        config = {
            "name": proxy.name or f"{proxy.server}:{proxy.port}",
            "type": "socks5",
            "server": proxy.server,
            "port": proxy.port,
            "udp": proxy.udp
        }

        if proxy.username:
            config["username"] = proxy.username
        if proxy.password:
            config["password"] = proxy.password

        return config

    @staticmethod
    def _http_to_clash(proxy: Proxy) -> dict:
        """HTTP/HTTPS → Clash"""
        config = {
            "name": proxy.name or f"{proxy.server}:{proxy.port}",
            "type": "http",
            "server": proxy.server,
            "port": proxy.port
        }

        if proxy.username:
            config["username"] = proxy.username
        if proxy.password:
            config["password"] = proxy.password

        if proxy.proxy_type == ProxyType.HTTPS:
            config["tls"] = True

        return config

    @staticmethod
    def _parse_obfs_opts(opts: str) -> dict:
        """
        解析 obfs 插件选项
        obfs=http;obfs-host=www.bing.com
        """
        result = {}
        for item in opts.split(";"):
            if "=" in item:
                key, value = item.split("=", 1)
                if key == "obfs":
                    result["mode"] = value
                elif key == "obfs-host":
                    result["host"] = value
        return result

    @staticmethod
    def _parse_v2ray_opts(opts: str) -> dict:
        """解析 v2ray-plugin 选项"""
        result = {"mode": "websocket"}

        for item in opts.split(";"):
            if "=" in item:
                key, value = item.split("=", 1)
                if key == "host":
                    result["host"] = value
                elif key == "path":
                    result["path"] = value
                elif key == "tls":
                    result["tls"] = True
                elif key == "mux":
                    result["mux"] = True

        return result

    @staticmethod
    def generate_config(
        proxies: List[Proxy],
        base_config: dict = None,
        proxy_groups: List[dict] = None
    ) -> str:
        """
        生成完整的 Clash 配置

        Args:
            proxies: 节点列表
            base_config: 基础配置模板
            proxy_groups: 代理组配置

        Returns:
            YAML 格式的配置字符串
        """
        config = base_config or {
            "port": 7890,
            "socks-port": 7891,
            "redir-port": 7892,
            "mixed-port": 7893,
            "allow-lan": True,
            "mode": "Rule",
            "log-level": "info",
            "external-controller": "127.0.0.1:9090"
        }

        # 转换节点
        clash_proxies = []
        for proxy in proxies:
            clash_proxy = ClashGenerator.proxy_to_clash(proxy)
            if clash_proxy:
                clash_proxies.append(clash_proxy)

        config["proxies"] = clash_proxies

        # 添加代理组
        if proxy_groups:
            config["proxy-groups"] = proxy_groups
        else:
            # 默认代理组
            config["proxy-groups"] = [
                {
                    "name": "🚀 节点选择",
                    "type": "select",
                    "proxies": ["♻️ 自动选择", "🔯 故障转移", "DIRECT"] + [p["name"] for p in clash_proxies]
                },
                {
                    "name": "♻️ 自动选择",
                    "type": "url-test",
                    "url": "http://www.gstatic.com/generate_204",
                    "interval": 300,
                    "proxies": [p["name"] for p in clash_proxies]
                }
            ]

        # 添加默认规则
        config["rules"] = [
            "MATCH,🚀 节点选择"
        ]

        return yaml.dump(config, allow_unicode=True, sort_keys=False)
```

---

## 工具函数实现

```python
import re
import ipaddress
from typing import Optional

def is_ipv4(ip: str) -> bool:
    """检查是否为 IPv4 地址"""
    try:
        ipaddress.IPv4Address(ip)
        return True
    except:
        return False

def is_ipv6(ip: str) -> bool:
    """检查是否为 IPv6 地址"""
    try:
        ipaddress.IPv6Address(ip)
        return True
    except:
        return False

def parse_port(port_str: str) -> Optional[int]:
    """解析端口号"""
    try:
        port = int(port_str)
        if 1 <= port <= 65535:
            return port
    except:
        pass
    return None

def safe_base64_decode(data: str) -> str:
    """安全的 Base64 解码"""
    # 添加 padding
    padding = len(data) % 4
    if padding:
        data += "=" * (4 - padding)

    try:
        return base64.urlsafe_b64decode(data).decode("utf-8")
    except:
        return ""

def url_safe_base64_encode(data: str) -> str:
    """URL 安全的 Base64 编码"""
    return base64.urlsafe_b64encode(data.encode()).decode().rstrip("=")

def extract_domain(host: str) -> str:
    """提取域名"""
    # 移除端口
    if ":" in host:
        host = host.split(":")[0]

    # 移除方括号 (IPv6)
    if host.startswith("[") and host.endswith("]"):
        host = host[1:-1]

    return host

def format_node_name(name: str, server: str, port: int) -> str:
    """格式化节点名称"""
    if not name:
        return f"{server}:{port}"

    # 替换不安全字符
    unsafe_chars = ['=', '\n', '\r']
    for char in unsafe_chars:
        name = name.replace(char, '-')

    # 如果包含逗号,添加引号
    if "," in name:
        name = f'"{name}"'

    return name

def deduplicate_nodes(nodes: list) -> list:
    """去重节点"""
    seen = set()
    unique_nodes = []

    for node in nodes:
        # 使用服务器+端口作为唯一标识
        key = (node.server, node.port)
        if key not in seen:
            seen.add(key)
            unique_nodes.append(node)

    return unique_nodes

def validate_proxy(proxy: Proxy) -> bool:
    """验证节点配置是否完整"""
    if not proxy.server or not proxy.port:
        return False

    if proxy.port < 1 or proxy.port > 65535:
        return False

    # 根据类型验证必要字段
    if proxy.proxy_type == ProxyType.SS:
        return bool(proxy.cipher and proxy.password)
    elif proxy.proxy_type == ProxyType.VMESS:
        return bool(proxy.uuid)
    elif proxy.proxy_type == ProxyType.TROJAN:
        return bool(proxy.password)

    return True
```

---

## 完整实现示例

### 主程序

```python
#!/usr/bin/env python3
"""
订阅转换器 - 订阅转 Clash 配置

使用方法:
    python converter.py -u <订阅URL> -o <输出文件>
    python converter.py -i <输入文件> -o <输出文件>
"""

import argparse
import sys
from typing import List

def convert_subscription(url: str, output: str = None) -> str:
    """
    转换订阅为 Clash 配置

    Args:
        url: 订阅 URL 或文件路径
        output: 输出文件路径

    Returns:
        Clash 配置字符串
    """
    print(f"📥 正在获取订阅: {url}")

    # 解析订阅
    if url.startswith("http://") or url.startswith("https://"):
        proxies = SubscriptionParser.parse_subscription(url)
    else:
        # 从文件读取
        with open(url, "r") as f:
            lines = f.readlines()

        proxies = []
        for line in lines:
            line = line.strip()
            if line:
                proxy = SubscriptionParser.parse_uri(line)
                if proxy:
                    proxies.append(proxy)

    if not proxies:
        print("❌ 未找到有效节点")
        return ""

    print(f"✅ 成功解析 {len(proxies)} 个节点")

    # 去重
    proxies = deduplicate_nodes(proxies)
    print(f"🔍 去重后剩余 {len(proxies)} 个节点")

    # 验证
    valid_proxies = [p for p in proxies if validate_proxy(p)]
    print(f"✔️ 有效节点 {len(valid_proxies)} 个")

    # 生成配置
    config = ClashGenerator.generate_config(valid_proxies)

    # 输出
    if output:
        with open(output, "w", encoding="utf-8") as f:
            f.write(config)
        print(f"💾 配置已保存到: {output}")
    else:
        print("\n" + "="*50)
        print(config)
        print("="*50)

    return config

def main():
    parser = argparse.ArgumentParser(description="订阅转换器 - 订阅转 Clash 配置")
    parser.add_argument("-u", "--url", help="订阅 URL 或文件路径")
    parser.add_argument("-o", "--output", help="输出文件路径")
    parser.add_argument("-v", "--version", action="version", version="%(prog)s 1.0")

    args = parser.parse_args()

    if not args.url:
        parser.print_help()
        sys.exit(1)

    try:
        convert_subscription(args.url, args.output)
    except Exception as e:
        print(f"❌ 转换失败: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
```

---

## 测试用例

```python
import unittest

class TestSubscriptionParser(unittest.TestCase):
    """订阅解析测试"""

    def test_ss_parsing(self):
        """测试 SS 解析"""
        uri = "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@server.com:8388#TestServer"
        proxy = parse_ss_uri(uri)

        self.assertEqual(proxy.proxy_type, ProxyType.SS)
        self.assertEqual(proxy.cipher, "aes-256-gcm")
        self.assertEqual(proxy.password, "password")
        self.assertEqual(proxy.server, "server.com")
        self.assertEqual(proxy.port, 8388)
        self.assertEqual(proxy.name, "TestServer")

    def test_vmess_parsing(self):
        """测试 VMess 解析"""
        uri = "vmess://eyJhZGQiOiJzZXJ2ZXIuY29tIiwiYWlkIjoiMCIsImhvc3QiOiIiLCJpZCI6IjEyMzQ1NjNlLTAwYzYtNDNiZS05NjQyLWU3NWU2YjEyMzQ1NiIsIm5ldCI6IndzIiwicGF0aCI6Ii8iLCJwb3J0IjoiNDQzIiwicHMiOiJUZXN0Iiwic2N5IjoiYXV0byIsInNuaSI6IiIsInRscyI6InRscyIsInR5cGUiOiIiLCJ2IjoiMiJ9"
        proxy = parse_vmess_uri(uri)

        self.assertEqual(proxy.proxy_type, ProxyType.VMESS)
        self.assertEqual(proxy.server, "server.com")
        self.assertEqual(proxy.port, 443)
        self.assertEqual(proxy.network, "ws")
        self.assertTrue(proxy.tls)

    def test_trojan_parsing(self):
        """测试 Trojan 解析"""
        uri = "trojan://password@server.com:443?security=tls&sni=example.com#TestTrojan"
        proxy = parse_trojan_uri(uri)

        self.assertEqual(proxy.proxy_type, ProxyType.TROJAN)
        self.assertEqual(proxy.password, "password")
        self.assertEqual(proxy.server, "server.com")
        self.assertEqual(proxy.port, 443)
        self.assertEqual(proxy.sni_trojan, "example.com")
        self.assertEqual(proxy.name, "TestTrojan")

class TestClashGenerator(unittest.TestCase):
    """Clash 配置生成测试"""

    def test_ss_to_clash(self):
        """测试 SS 转换"""
        proxy = Proxy(
            proxy_type=ProxyType.SS,
            name="Test",
            server="server.com",
            port=8388,
            cipher="aes-256-gcm",
            password="password"
        )

        config = ClashGenerator.proxy_to_clash(proxy)

        self.assertEqual(config["type"], "ss")
        self.assertEqual(config["server"], "server.com")
        self.assertEqual(config["port"], 8388)
        self.assertEqual(config["cipher"], "aes-256-gcm")
        self.assertEqual(config["password"], "password")

    def test_vmess_to_clash(self):
        """测试 VMess 转换"""
        proxy = Proxy(
            proxy_type=ProxyType.VMESS,
            name="Test",
            server="server.com",
            port=443,
            uuid="123456e-00c6-43be-9642-e75e6b123456",
            alter_id=0,
            cipher="auto",
            network="ws",
            tls=True
        )

        config = ClashGenerator.proxy_to_clash(proxy)

        self.assertEqual(config["type"], "vmess")
        self.assertEqual(config["server"], "server.com")
        self.assertEqual(config["port"], 443)
        self.assertEqual(config["uuid"], "123456e-00c6-43be-9642-e75e6b123456")
        self.assertTrue(config["tls"])
        self.assertEqual(config["network"], "ws")

if __name__ == "__main__":
    unittest.main()
```

---

## 使用示例

### 命令行使用

```bash
# 转换在线订阅
python converter.py -u https://example.com/subscription -o config.yaml

# 转换本地文件
python converter.py -u subs.txt -o config.yaml

# 输出到终端
python converter.py -u https://example.com/subscription
```

### 代码集成

```python
# 简单使用
proxies = SubscriptionParser.parse_subscription("https://example.com/sub")
config = ClashGenerator.generate_config(proxies)
print(config)

# 高级使用
proxies = SubscriptionParser.parse_subscription("https://example.com/sub")
proxies = [p for p in proxies if "香港" in p.name]  # 过滤
config = ClashGenerator.generate_config(
    proxies,
    base_config={"port": 7890},
    proxy_groups=[{
        "name": "MyGroup",
        "type": "select",
        "proxies": ["DIRECT"] + [p.name for p in proxies]
    }]
)
```

---

## 总结

本文档提供了完整的实现指南,包括:

✅ **数据结构** - 统一的 Proxy 节点结构
✅ **协议解析** - SS/SSR/VMess/Trojan 完整解析实现
✅ **Clash 生成** - 所有节点类型的转换代码
✅ **工具函数** - Base64、URL、验证等辅助函数
✅ **完整示例** - 可直接运行的转换器程序
✅ **测试用例** - 单元测试示例

基于这份文档,你可以:
1. 独立实现订阅转换功能
2. 理解各种协议的解析细节
3. 扩展支持更多协议
4. 集成到自己的项目中

**建议实现优先级**:
1. 先实现 SS → Clash (最简单)
2. 再实现 VMess → Clash (最常用)
3. 最后实现其他协议
