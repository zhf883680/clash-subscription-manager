package converter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseClashProxies parses a Clash YAML file and extracts proxy nodes.
func ParseClashProxies(data []byte) ([]*ProxyNode, error) {
	var doc struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse clash yaml: %w", err)
	}

	nodes := make([]*ProxyNode, 0, len(doc.Proxies))
	for _, proxy := range doc.Proxies {
		node := clashMapToProxyNode(proxy)
		if node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

func clashMapToProxyNode(m map[string]any) *ProxyNode {
	nodeType, _ := m["type"].(string)
	name, _ := m["name"].(string)
	server, _ := m["server"].(string)
	port := intField(m, "port")

	node := &ProxyNode{
		Name:   name,
		Type:   nodeType,
		Server: server,
		Port:   port,
		UDP:    true,
	}

	switch nodeType {
	case "ss":
		node.Cipher, _ = m["cipher"].(string)
		node.Password, _ = m["password"].(string)
		if opts, ok := m["plugin-opts"].(map[string]any); ok {
			node.Plugin, _ = opts["name"].(string)
			node.PluginMode, _ = opts["mode"].(string)
			node.PluginHost, _ = opts["host"].(string)
			node.PluginPath, _ = opts["path"].(string)
			if v, ok := opts["tls"].(bool); ok {
				node.PluginTLS = &v
			}
			if v, ok := opts["mux"].(bool); ok {
				node.PluginMux = &v
			}
		}
	case "ssr":
		node.Cipher, _ = m["cipher"].(string)
		node.Password, _ = m["password"].(string)
		node.Protocol, _ = m["protocol"].(string)
		node.Obfs, _ = m["obfs"].(string)
		node.ProtocolParam, _ = m["protocol-param"].(string)
		node.ObfsParam, _ = m["obfs-param"].(string)
	case "vmess":
		node.UUID, _ = m["uuid"].(string)
		node.AlterID = intField(m, "alterId")
		node.Cipher, _ = m["cipher"].(string)
	case "trojan":
		node.Password, _ = m["password"].(string)
	case "vless":
		node.UUID, _ = m["uuid"].(string)
		node.Flow, _ = m["flow"].(string)
	}

	if v, ok := m["network"].(string); ok && v != "" {
		node.Network = v
	}
	if v, ok := m["tls"].(bool); ok {
		node.TLS = v
	}
	if v, ok := m["skip-cert-verify"].(bool); ok {
		node.SkipCertVerify = &v
	}
	if v, ok := m["tfo"].(bool); ok {
		node.TFO = &v
	}
	if v, ok := m["servername"].(string); ok {
		node.ServerName = v
	}
	if v, ok := m["sni"].(string); ok && node.ServerName == "" {
		node.ServerName = v
	}
	if wsOpts, ok := m["ws-opts"].(map[string]any); ok {
		node.Network = "ws"
		node.Path, _ = wsOpts["path"].(string)
		if headers, ok := wsOpts["headers"].(map[string]any); ok {
			node.Host, _ = headers["Host"].(string)
		}
	}
	if h2Opts, ok := m["h2-opts"].(map[string]any); ok {
		node.Network = "h2"
		node.Path, _ = h2Opts["path"].(string)
		if hosts, ok := h2Opts["host"].([]any); ok && len(hosts) > 0 {
			node.Host, _ = hosts[0].(string)
		}
	}
	if grpcOpts, ok := m["grpc-opts"].(map[string]any); ok {
		node.Network = "grpc"
		node.ServiceName, _ = grpcOpts["grpc-service-name"].(string)
	}
	if realityOpts, ok := m["reality-opts"].(map[string]any); ok {
		node.PublicKey, _ = realityOpts["public-key"].(string)
		node.ShortID, _ = realityOpts["short-id"].(string)
	}
	if v, ok := m["client-fingerprint"].(string); ok {
		node.ClientFingerprint = v
	}

	return node
}

func intField(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}

// BuildNodeLinks converts ProxyNodes back to their original URI format
// (ss://, vmess://, trojan://, vless://) and base64-encodes the result.
func BuildNodeLinks(nodes []*ProxyNode) ([]byte, error) {
	var lines []string
	for _, node := range nodes {
		link, err := proxyNodeToLink(node)
		if err != nil {
			continue
		}
		lines = append(lines, link)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("no valid nodes to export")
	}
	joined := strings.Join(lines, "\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(joined))
	return []byte(encoded), nil
}

func proxyNodeToLink(node *ProxyNode) (string, error) {
	switch node.Type {
	case string(TypeSS):
		return buildSSLink(node)
	case string(TypeVMess):
		return buildVMessLink(node)
	case string(TypeTrojan):
		return buildTrojanLink(node)
	case string(TypeVLESS):
		return buildVLESSLink(node)
	default:
		return "", fmt.Errorf("unsupported node type: %s", node.Type)
	}
}

func buildSSLink(node *ProxyNode) (string, error) {
	creds := base64.StdEncoding.EncodeToString([]byte(node.Cipher + ":" + node.Password))
	u := url.URL{
		Scheme: "ss",
		User:   url.User(creds),
		Host:   net.JoinHostPort(node.Server, strconv.Itoa(node.Port)),
	}
	if node.Name != "" {
		u.Fragment = node.Name
	}
	return u.String(), nil
}

func buildVMessLink(node *ProxyNode) (string, error) {
	payload := map[string]string{
		"ps":   node.Name,
		"add":  node.Server,
		"port": strconv.Itoa(node.Port),
		"id":   node.UUID,
		"aid":  strconv.Itoa(node.AlterID),
		"scy":  firstNonEmpty(node.Cipher, "auto"),
		"net":  firstNonEmpty(node.Network, "tcp"),
		"type": "none",
		"tls":  "",
	}
	if node.TLS {
		payload["tls"] = "tls"
	}
	if node.ServerName != "" {
		payload["sni"] = node.ServerName
	}
	switch node.Network {
	case "ws":
		payload["path"] = node.Path
		payload["host"] = node.Host
	case "h2":
		payload["path"] = node.Path
		payload["host"] = node.Host
	case "grpc":
		payload["path"] = node.ServiceName
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(jsonBytes)
	return "vmess://" + encoded, nil
}

func buildTrojanLink(node *ProxyNode) (string, error) {
	u := url.URL{
		Scheme: "trojan",
		User:   url.User(node.Password),
		Host:   net.JoinHostPort(node.Server, strconv.Itoa(node.Port)),
	}
	query := u.Query()
	if node.ServerName != "" {
		query.Set("sni", node.ServerName)
	}
	if node.Network != "" && node.Network != "tcp" {
		query.Set("type", node.Network)
	}
	if node.SkipCertVerify != nil {
		query.Set("allowInsecure", strconv.FormatBool(*node.SkipCertVerify))
	}
	if node.Host != "" {
		query.Set("host", node.Host)
	}
	if node.Path != "" {
		query.Set("path", node.Path)
	}
	if node.ServiceName != "" {
		query.Set("serviceName", node.ServiceName)
	}
	u.RawQuery = query.Encode()
	if node.Name != "" {
		u.Fragment = node.Name
	}
	return u.String(), nil
}

func buildVLESSLink(node *ProxyNode) (string, error) {
	u := url.URL{
		Scheme: "vless",
		User:   url.User(node.UUID),
		Host:   net.JoinHostPort(node.Server, strconv.Itoa(node.Port)),
	}
	query := u.Query()
	security := "none"
	if node.TLS {
		security = "tls"
		if node.PublicKey != "" {
			security = "reality"
		}
	}
	query.Set("security", security)
	if node.Network != "" && node.Network != "tcp" {
		query.Set("type", node.Network)
	}
	if node.ServerName != "" {
		query.Set("sni", node.ServerName)
	}
	if node.SkipCertVerify != nil {
		query.Set("allowInsecure", strconv.FormatBool(*node.SkipCertVerify))
	}
	if node.Host != "" {
		query.Set("host", node.Host)
	}
	if node.Path != "" {
		query.Set("path", node.Path)
	}
	if node.Flow != "" {
		query.Set("flow", node.Flow)
	}
	if node.ServiceName != "" {
		query.Set("serviceName", node.ServiceName)
	}
	if node.PublicKey != "" {
		query.Set("pbk", node.PublicKey)
	}
	if node.ShortID != "" {
		query.Set("sid", node.ShortID)
	}
	if node.ClientFingerprint != "" {
		query.Set("fp", node.ClientFingerprint)
	}
	u.RawQuery = query.Encode()
	if node.Name != "" {
		u.Fragment = node.Name
	}
	return u.String(), nil
}
