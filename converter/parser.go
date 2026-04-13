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

	return &ProxyNode{
		Name:     name,
		Type:     string(TypeSS),
		Server:   server,
		Port:     port,
		Cipher:   parts[0],
		Password: parts[1],
		UDP:      true,
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
		Network:    network,
		TLS:        strings.EqualFold(payload.TLS, "tls"),
		ServerName: payload.Host,
		Host:       payload.Host,
		Path:       payload.Path,
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
		UDP:               true,
	}, nil
}

func parseLine(line string) (*ProxyNode, error) {
	switch {
	case strings.HasPrefix(line, "ss://"):
		return ParseSSLink(line)
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
