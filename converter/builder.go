package converter

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func BuildClashConfig(nodes []*ProxyNode) ([]byte, error) {
	proxies := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		proxies = append(proxies, buildProxyMap(node))
	}

	out, err := yaml.Marshal(map[string]any{
		"proxies": proxies,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal clash yaml: %w", err)
	}
	return out, nil
}

func buildProxyMap(node *ProxyNode) map[string]any {
	proxy := map[string]any{
		"name":   node.Name,
		"type":   node.Type,
		"server": node.Server,
		"port":   node.Port,
		"udp":    node.UDP,
	}

	switch node.Type {
	case string(TypeSS):
		proxy["cipher"] = node.Cipher
		proxy["password"] = node.Password
	case string(TypeVMess):
		proxy["uuid"] = node.UUID
		proxy["alterId"] = node.AlterID
		proxy["cipher"] = "auto"
	case string(TypeTrojan):
		proxy["password"] = node.Password
	case string(TypeVLESS):
		proxy["uuid"] = node.UUID
	}

	if node.Network != "" && node.Network != "tcp" {
		proxy["network"] = node.Network
	}
	if node.TLS {
		proxy["tls"] = true
	}
	if node.ServerName != "" {
		proxy["servername"] = node.ServerName
	}
	if node.Network == "ws" {
		wsOpts := map[string]any{}
		if node.Path != "" {
			wsOpts["path"] = node.Path
		}
		if node.Host != "" {
			wsOpts["headers"] = map[string]any{"Host": node.Host}
		}
		if len(wsOpts) > 0 {
			proxy["ws-opts"] = wsOpts
		}
	}
	if node.Network == "grpc" && node.ServiceName != "" {
		proxy["grpc-opts"] = map[string]any{
			"grpc-service-name": node.ServiceName,
		}
	}
	if node.Type == string(TypeVLESS) && node.ClientFingerprint != "" {
		proxy["client-fingerprint"] = node.ClientFingerprint
	}
	if node.Type == string(TypeVLESS) && (node.PublicKey != "" || node.ShortID != "") {
		realityOpts := map[string]any{}
		if node.PublicKey != "" {
			realityOpts["public-key"] = node.PublicKey
		}
		if node.ShortID != "" {
			realityOpts["short-id"] = node.ShortID
		}
		if len(realityOpts) > 0 {
			proxy["reality-opts"] = realityOpts
		}
	}

	return proxy
}
