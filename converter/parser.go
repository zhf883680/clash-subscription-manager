package converter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func ParseSSLink(link string) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("parse ss url: %w", err)
	}

	server := parsed.Hostname()
	portValue := parsed.Port()
	methodPassword := ""

	if parsed.User != nil {
		methodPassword = parsed.User.Username()
	} else if parsed.Host == "" && parsed.Opaque != "" {
		opaque := strings.TrimPrefix(parsed.Opaque, "//")
		methodPassword, server, portValue = splitSSOpaque(opaque)
	}

	decoded, ok := decodeBase64String(methodPassword)
	if !ok {
		decoded = methodPassword
	}

	parts := strings.SplitN(decoded, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ss credentials")
	}

	port, err := strconv.Atoi(portValue)
	if err != nil {
		return nil, fmt.Errorf("invalid ss port: %w", err)
	}

	name, _ := url.PathUnescape(strings.TrimPrefix(parsed.Fragment, "#"))
	if name == "" {
		name = net.JoinHostPort(server, portValue)
	}

	pluginName, pluginMode, pluginHost, pluginPath, pluginTLS, pluginMux := parseSSPluginOptions(parsed.Query().Get("plugin"))

	return &ProxyNode{
		Name:       name,
		Type:       string(TypeSS),
		Server:     server,
		Port:       port,
		Cipher:     parts[0],
		Password:   parts[1],
		Plugin:     pluginName,
		PluginMode: pluginMode,
		PluginHost: pluginHost,
		PluginPath: pluginPath,
		PluginTLS:  pluginTLS,
		PluginMux:  pluginMux,
		UDP:        true,
	}, nil
}

func ParseSSRLink(link string) (*ProxyNode, error) {
	raw := strings.TrimPrefix(strings.TrimSpace(link), "ssr://")
	decoded, ok := decodeBase64String(raw)
	if !ok {
		return nil, fmt.Errorf("invalid ssr payload")
	}

	mainPart := decoded
	queryPart := ""
	if idx := strings.Index(decoded, "/?"); idx >= 0 {
		mainPart = decoded[:idx]
		queryPart = decoded[idx+2:]
	}

	parts := strings.Split(mainPart, ":")
	if len(parts) != 6 {
		return nil, fmt.Errorf("invalid ssr payload")
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid ssr port: %w", err)
	}
	password, ok := decodeBase64String(parts[5])
	if !ok {
		return nil, fmt.Errorf("invalid ssr password")
	}

	queryValues, err := url.ParseQuery(queryPart)
	if err != nil {
		return nil, fmt.Errorf("parse ssr query: %w", err)
	}

	name := decodeSSRQueryValue(queryValues.Get("remarks"))
	if name == "" {
		name = net.JoinHostPort(parts[0], parts[1])
	}

	return &ProxyNode{
		Name:          name,
		Type:          "ssr",
		Server:        parts[0],
		Port:          port,
		Cipher:        parts[3],
		Password:      password,
		Protocol:      parts[2],
		ProtocolParam: decodeSSRQueryValue(queryValues.Get("protoparam")),
		Obfs:          parts[4],
		ObfsParam:     decodeSSRQueryValue(queryValues.Get("obfsparam")),
		UDP:           true,
	}, nil
}

func ParseVMessLink(link string) (*ProxyNode, error) {
	raw := strings.TrimPrefix(strings.TrimSpace(link), "vmess://")
	decoded, ok := decodeBase64String(raw)
	if !ok {
		return nil, fmt.Errorf("invalid vmess payload")
	}

	var payload struct {
		PS   string `json:"ps"`
		Add  string `json:"add"`
		Port string `json:"port"`
		ID   string `json:"id"`
		AID  string `json:"aid"`
		Net  string `json:"net"`
		Host string `json:"host"`
		Path string `json:"path"`
		TLS  string `json:"tls"`
		SNI  string `json:"sni"`
		SCY  string `json:"scy"`
	}
	if err := json.Unmarshal([]byte(decoded), &payload); err != nil {
		return nil, fmt.Errorf("parse vmess payload: %w", err)
	}

	port, err := strconv.Atoi(payload.Port)
	if err != nil {
		return nil, fmt.Errorf("invalid vmess port: %w", err)
	}
	alterID, _ := strconv.Atoi(payload.AID)
	network := normalizeNetwork(payload.Net)
	name := payload.PS
	if name == "" {
		name = net.JoinHostPort(payload.Add, payload.Port)
	}

	return &ProxyNode{
		Name:       name,
		Type:       string(TypeVMess),
		Server:     payload.Add,
		Port:       port,
		UUID:       payload.ID,
		AlterID:    alterID,
		Cipher:     firstNonEmpty(payload.SCY, "auto"),
		Network:    network,
		TLS:        strings.EqualFold(payload.TLS, "tls"),
		ServerName: firstNonEmpty(payload.SNI, payload.Host),
		Host:       payload.Host,
		Path:       payload.Path,
		ServiceName: func() string {
			if network == "grpc" {
				return payload.Path
			}
			return ""
		}(),
		UDP:        true,
	}, nil
}

func ParseTrojanLink(link string) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("parse trojan url: %w", err)
	}
	port, err := portOrDefault(parsed, 443)
	if err != nil {
		return nil, err
	}

	query := parsed.Query()
	name, _ := url.PathUnescape(strings.TrimPrefix(parsed.Fragment, "#"))
	if name == "" {
		name = net.JoinHostPort(parsed.Hostname(), strconv.Itoa(port))
	}

	return &ProxyNode{
		Name:       name,
		Type:       string(TypeTrojan),
		Server:     parsed.Hostname(),
		Port:       port,
		Password:   parsed.User.Username(),
		Network:    normalizeNetwork(query.Get("type")),
		TLS:        true,
		ServerName: firstNonEmpty(query.Get("sni"), query.Get("peer"), query.Get("host")),
		Host:       query.Get("host"),
		Path:       query.Get("path"),
		ServiceName: firstNonEmpty(
			query.Get("serviceName"),
			query.Get("grpc-service-name"),
		),
		SkipCertVerify: parseOptionalBoolDefault(firstNonEmpty(query.Get("allowInsecure"), query.Get("insecure")), nil),
		TFO:            parseOptionalBoolDefault(firstNonEmpty(query.Get("tfo"), query.Get("fast-open")), nil),
		UDP:        true,
	}, nil
}

func ParseVLESSLink(link string) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("parse vless url: %w", err)
	}
	port, err := portOrDefault(parsed, 443)
	if err != nil {
		return nil, err
	}

	query := parsed.Query()
	name, _ := url.PathUnescape(strings.TrimPrefix(parsed.Fragment, "#"))
	if name == "" {
		name = net.JoinHostPort(parsed.Hostname(), strconv.Itoa(port))
	}

	security := strings.ToLower(query.Get("security"))
	skipCertVerify := true
	tfo := false

	if value, ok := parseOptionalBool(firstNonEmpty(query.Get("allowInsecure"), query.Get("insecure"))); ok {
		skipCertVerify = value
	}
	if value, ok := parseOptionalBool(firstNonEmpty(query.Get("tfo"), query.Get("fast-open"))); ok {
		tfo = value
	}

	return &ProxyNode{
		Name:              name,
		Type:              string(TypeVLESS),
		Server:            parsed.Hostname(),
		Port:              port,
		UUID:              parsed.User.Username(),
		Network:           normalizeNetwork(query.Get("type")),
		TLS:               security == "tls" || security == "reality",
		ServerName:        firstNonEmpty(query.Get("sni"), query.Get("host")),
		Host:              query.Get("host"),
		Path:              query.Get("path"),
		ClientFingerprint: query.Get("fp"),
		PublicKey:         query.Get("pbk"),
		ShortID:           query.Get("sid"),
		ServiceName:       query.Get("serviceName"),
		Flow:              query.Get("flow"),
		SkipCertVerify:    &skipCertVerify,
		TFO:               &tfo,
		UDP:               true,
	}, nil
}

func parseLine(line string) (*ProxyNode, error) {
	switch {
	case strings.HasPrefix(line, "ss://"):
		return ParseSSLink(line)
	case strings.HasPrefix(line, "ssr://"):
		return ParseSSRLink(line)
	case strings.HasPrefix(line, "vmess://"):
		return ParseVMessLink(line)
	case strings.HasPrefix(line, "trojan://"):
		return ParseTrojanLink(line)
	case strings.HasPrefix(line, "vless://"):
		return ParseVLESSLink(line)
	default:
		return nil, ErrUnsupportedSubscription
	}
}

func decodeBase64String(value string) (string, bool) {
	value = strings.TrimSpace(value)
	for _, decode := range []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		base64.URLEncoding.DecodeString,
		base64.RawURLEncoding.DecodeString,
	} {
		decoded, err := decode(value)
		if err == nil {
			return string(decoded), true
		}
	}
	return "", false
}

func splitSSOpaque(value string) (string, string, string) {
	at := strings.LastIndex(value, "@")
	if at == -1 {
		return value, "", ""
	}
	methodPassword := value[:at]
	hostPort := value[at+1:]
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return methodPassword, "", ""
	}
	return methodPassword, host, port
}

func normalizeNetwork(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "tcp":
		return "tcp"
	case "websocket":
		return "ws"
	default:
		return value
	}
}

func portOrDefault(parsed *url.URL, fallback int) (int, error) {
	if parsed.Port() == "" {
		return fallback, nil
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		return 0, fmt.Errorf("invalid port: %w", err)
	}
	return port, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseOptionalBool(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func parseOptionalBoolDefault(value string, fallback *bool) *bool {
	if parsed, ok := parseOptionalBool(value); ok {
		return &parsed
	}
	return fallback
}

func parseSSPluginOptions(value string) (string, string, string, string, *bool, *bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", "", "", nil, nil
	}

	parts := strings.Split(value, ";")
	name := strings.TrimSpace(parts[0])
	options := map[string]string{}
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, ok := strings.Cut(part, "=")
		if ok {
			options[strings.TrimSpace(key)] = strings.TrimSpace(val)
			continue
		}
		options[part] = "true"
	}

	var pluginTLS *bool
	var pluginMux *bool
	if parsed, ok := parseOptionalBool(options["tls"]); ok {
		pluginTLS = &parsed
	} else if _, ok := options["tls"]; ok {
		value := true
		pluginTLS = &value
	}
	if parsed, ok := parseOptionalBool(options["mux"]); ok {
		pluginMux = &parsed
	} else if _, ok := options["mux"]; ok {
		value := true
		pluginMux = &value
	}

	return name, options["mode"], options["host"], options["path"], pluginTLS, pluginMux
}

func decodeSSRQueryValue(value string) string {
	if value == "" {
		return ""
	}
	decoded, ok := decodeBase64String(value)
	if !ok {
		return value
	}
	return strings.TrimSpace(decoded)
}
