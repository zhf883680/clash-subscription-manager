package converter

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func ConvertToClash(content []byte) ([]byte, ConversionSummary, error) {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return nil, ConversionSummary{}, ErrUnsupportedSubscription
	}

	normalized := trimmed
	if decoded, ok := decodeBase64Content(trimmed); ok {
		normalized = decoded
	}

	if isClashFormat(normalized) {
		return normalized, ConversionSummary{
			DetectedType: string(TypeClash),
			OutputType:   string(TypeClash),
			NodeCount:    countClashNodes(normalized),
		}, nil
	}

	detected := DetectSubscriptionType(normalized)
	if detected == TypeUnknown {
		return nil, ConversionSummary{}, ErrUnsupportedSubscription
	}

	lines := splitSubscriptionLines(normalized)
	nodes := make([]*ProxyNode, 0, len(lines))
	for _, line := range lines {
		node, err := parseLine(line)
		if err != nil {
			continue
		}
		nodes = append(nodes, node)
	}
	if len(nodes) == 0 {
		return nil, ConversionSummary{}, fmt.Errorf("%w: no valid proxy nodes found", ErrUnsupportedSubscription)
	}

	out, err := BuildClashConfig(nodes)
	if err != nil {
		return nil, ConversionSummary{}, err
	}
	return out, ConversionSummary{
		DetectedType: string(detected),
		OutputType:   string(TypeClash),
		NodeCount:    len(nodes),
	}, nil
}

func ConvertNodesTextToClash(input string) ([]byte, ConversionSummary, error) {
	lines := strings.Split(input, "\n")
	nodes := make([]*ProxyNode, 0, len(lines))
	types := make(map[string]struct{})

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		node, err := parseLine(trimmed)
		if err != nil {
			continue
		}

		nodes = append(nodes, node)
		types[node.Type] = struct{}{}
	}

	if len(nodes) == 0 {
		return nil, ConversionSummary{}, fmt.Errorf("%w: no valid proxy nodes found", ErrUnsupportedSubscription)
	}

	out, err := BuildClashConfig(nodes)
	if err != nil {
		return nil, ConversionSummary{}, err
	}

	detectedType := string(TypeMixed)
	if len(types) == 1 {
		for nodeType := range types {
			detectedType = nodeType
		}
	}

	return out, ConversionSummary{
		DetectedType: detectedType,
		OutputType:   string(TypeClash),
		NodeCount:    len(nodes),
	}, nil
}

func countClashNodes(content []byte) int {
	var parsed struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		return 0
	}
	return len(parsed.Proxies)
}
