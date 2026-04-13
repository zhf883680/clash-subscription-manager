package converter

import (
	"bytes"
	"encoding/base64"
	"strings"

	"gopkg.in/yaml.v3"
)

func DetectSubscriptionType(content []byte) SubscriptionType {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return TypeUnknown
	}

	if isClashFormat(trimmed) {
		return TypeClash
	}

	if decoded, ok := decodeBase64Content(trimmed); ok && isClashFormat(decoded) {
		return TypeClash
	}

	counts := map[SubscriptionType]int{}
	for _, line := range splitSubscriptionLines(trimmed) {
		switch {
		case strings.HasPrefix(line, "ss://"):
			counts[TypeSS]++
		case strings.HasPrefix(line, "vmess://"):
			counts[TypeVMess]++
		case strings.HasPrefix(line, "trojan://"):
			counts[TypeTrojan]++
		case strings.HasPrefix(line, "vless://"):
			counts[TypeVLESS]++
		}
	}

	if len(counts) == 0 {
		return TypeUnknown
	}
	if len(counts) > 1 {
		return TypeMixed
	}
	for kind := range counts {
		return kind
	}
	return TypeUnknown
}

func isClashFormat(content []byte) bool {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return false
	}
	root := &doc
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		root = doc.Content[0]
	}
	if root == nil || root.Kind != yaml.MappingNode {
		return false
	}

	for index := 0; index+1 < len(root.Content); index += 2 {
		key := root.Content[index]
		value := root.Content[index+1]
		if key.Value == "proxies" && value.Kind == yaml.SequenceNode {
			return true
		}
	}
	return false
}

func decodeBase64Content(content []byte) ([]byte, bool) {
	candidates := []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		base64.URLEncoding.DecodeString,
		base64.RawURLEncoding.DecodeString,
	}

	value := strings.TrimSpace(string(content))
	for _, decode := range candidates {
		decoded, err := decode(value)
		if err == nil && len(bytes.TrimSpace(decoded)) > 0 {
			return bytes.TrimSpace(decoded), true
		}
	}
	return nil, false
}

func splitSubscriptionLines(content []byte) []string {
	lines := strings.Split(string(content), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return result
}
