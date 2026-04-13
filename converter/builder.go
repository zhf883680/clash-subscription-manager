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
		if node.Plugin != "" {
			proxy["plugin"] = normalizeSSPluginName(node.Plugin)
			pluginOpts := map[string]any{}
			if node.PluginMode != "" {
				pluginOpts["mode"] = node.PluginMode
			}
			if node.PluginHost != "" {
				pluginOpts["host"] = node.PluginHost
			}
			if node.PluginPath != "" {
				pluginOpts["path"] = node.PluginPath
			}
			if node.PluginTLS != nil {
				pluginOpts["tls"] = *node.PluginTLS
			}
			if node.PluginMux != nil {
				pluginOpts["mux"] = *node.PluginMux
			}
			if len(pluginOpts) > 0 {
				proxy["plugin-opts"] = pluginOpts
			}
		}
	case string(TypeSSR):
		proxy["cipher"] = node.Cipher
		proxy["password"] = node.Password
		proxy["protocol"] = node.Protocol
		proxy["obfs"] = node.Obfs
		if node.ProtocolParam != "" {
			proxy["protocol-param"] = node.ProtocolParam
		}
		if node.ObfsParam != "" {
			proxy["obfs-param"] = node.ObfsParam
		}
	case string(TypeVMess):
		proxy["uuid"] = node.UUID
		proxy["alterId"] = node.AlterID
		proxy["cipher"] = firstNonEmpty(node.Cipher, "auto")
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
	if node.Flow != "" {
		proxy["flow"] = node.Flow
	}
	if node.SkipCertVerify != nil {
		proxy["skip-cert-verify"] = *node.SkipCertVerify
	}
	if node.TFO != nil {
		proxy["tfo"] = *node.TFO
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
	if node.Network == "http" {
		httpOpts := map[string]any{
			"method": "GET",
		}
		if node.Path != "" {
			httpOpts["path"] = []string{node.Path}
		}
		if node.Host != "" {
			httpOpts["headers"] = map[string]any{
				"Host": []string{node.Host},
			}
		}
		proxy["http-opts"] = httpOpts
	}
	if node.Network == "h2" {
		h2Opts := map[string]any{}
		if node.Path != "" {
			h2Opts["path"] = node.Path
		}
		if node.Host != "" {
			h2Opts["host"] = []string{node.Host}
		}
		if len(h2Opts) > 0 {
			proxy["h2-opts"] = h2Opts
		}
	}
	if node.Network == "grpc" && node.ServiceName != "" {
		proxy["grpc-opts"] = map[string]any{
			"grpc-service-name": node.ServiceName,
		}
	}
	if node.Type == string(TypeTrojan) && node.ServerName != "" {
		proxy["sni"] = node.ServerName
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

func normalizeSSPluginName(value string) string {
	switch value {
	case "simple-obfs", "obfs-local":
		return "obfs"
	default:
		return value
	}
}
