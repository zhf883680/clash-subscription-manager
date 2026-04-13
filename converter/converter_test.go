package converter

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestConvertToClashRejectsUnknownContent(t *testing.T) {
	_, _, err := ConvertToClash([]byte("not-a-subscription"))
	if err == nil {
		t.Fatal("expected error for unsupported content")
	}
}

func TestConvertToClashPassesThroughClashYAML(t *testing.T) {
	input := []byte(strings.TrimSpace(`
proxies:
  - name: demo
    type: ss
    server: example.com
    port: 443
    cipher: aes-256-gcm
    password: pass
`))

	got, summary, err := ConvertToClash(input)
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if summary.DetectedType != "clash" {
		t.Fatalf("DetectedType = %q, want %q", summary.DetectedType, "clash")
	}
	if summary.OutputType != "clash" {
		t.Fatalf("OutputType = %q, want %q", summary.OutputType, "clash")
	}
	if summary.NodeCount != 1 {
		t.Fatalf("NodeCount = %d, want 1", summary.NodeCount)
	}
	if string(got) != string(input) {
		t.Fatalf("converted content changed unexpectedly:\n%s", string(got))
	}
}

func TestConvertToClashDecodesBase64ClashYAML(t *testing.T) {
	raw := strings.TrimSpace(`
proxies:
  - name: demo
    type: trojan
    server: trojan.example.com
    port: 443
    password: secret
`)

	got, summary, err := ConvertToClash([]byte(base64.StdEncoding.EncodeToString([]byte(raw))))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if summary.DetectedType != "clash" {
		t.Fatalf("DetectedType = %q, want %q", summary.DetectedType, "clash")
	}
	if !bytes.Contains(got, []byte("type: trojan")) {
		t.Fatalf("converted content missing trojan proxy:\n%s", string(got))
	}
}

func TestConvertToClashDecodesBase64WrappedProtocolLines(t *testing.T) {
	raw := strings.Join([]string{
		"trojan://secret@trojan.example.com:443#trojan-demo",
		"ss://YWVzLTI1Ni1nY206cGFzcw==@ss.example.com:443#ss-demo",
	}, "\n")

	got, summary, err := ConvertToClash([]byte(base64.StdEncoding.EncodeToString([]byte(raw))))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if summary.DetectedType != "mixed" {
		t.Fatalf("DetectedType = %q, want %q", summary.DetectedType, "mixed")
	}
	if summary.NodeCount != 2 {
		t.Fatalf("NodeCount = %d, want 2", summary.NodeCount)
	}
	if !bytes.Contains(got, []byte("type: trojan")) {
		t.Fatalf("converted content missing trojan proxy:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("type: ss")) {
		t.Fatalf("converted content missing ss proxy:\n%s", string(got))
	}
}

func TestConvertToClashConvertsSSSubscription(t *testing.T) {
	got, summary, err := ConvertToClash([]byte("ss://YWVzLTI1Ni1nY206cGFzcw==@example.com:443#demo"))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if summary.DetectedType != "ss" {
		t.Fatalf("DetectedType = %q, want %q", summary.DetectedType, "ss")
	}
	if summary.NodeCount != 1 {
		t.Fatalf("NodeCount = %d, want 1", summary.NodeCount)
	}
	if !bytes.Contains(got, []byte("type: ss")) {
		t.Fatalf("converted content missing ss proxy:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("server: example.com")) {
		t.Fatalf("converted content missing server:\n%s", string(got))
	}
}

func TestConvertToClashConvertsVMessSubscription(t *testing.T) {
	vmessJSON := `{"v":"2","ps":"vmess-demo","add":"vmess.example.com","port":"443","id":"123e4567-e89b-12d3-a456-426614174000","aid":"0","net":"ws","type":"none","host":"cdn.example.com","path":"/ws","tls":"tls"}`
	input := "vmess://" + base64.StdEncoding.EncodeToString([]byte(vmessJSON))

	got, summary, err := ConvertToClash([]byte(input))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if summary.DetectedType != "vmess" {
		t.Fatalf("DetectedType = %q, want %q", summary.DetectedType, "vmess")
	}
	if !bytes.Contains(got, []byte("type: vmess")) {
		t.Fatalf("converted content missing vmess proxy:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("network: ws")) {
		t.Fatalf("converted content missing ws network:\n%s", string(got))
	}
}

func TestConvertToClashConvertsTrojanSubscription(t *testing.T) {
	input := "trojan://secret@trojan.example.com:443?type=ws&host=cdn.example.com&path=%2Fws&sni=edge.example.com#trojan-demo"

	got, summary, err := ConvertToClash([]byte(input))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if summary.DetectedType != "trojan" {
		t.Fatalf("DetectedType = %q, want %q", summary.DetectedType, "trojan")
	}
	if !bytes.Contains(got, []byte("type: trojan")) {
		t.Fatalf("converted content missing trojan proxy:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("servername: edge.example.com")) {
		t.Fatalf("converted content missing sni:\n%s", string(got))
	}
}

func TestConvertToClashConvertsVLESSSubscription(t *testing.T) {
	input := "vless://123e4567-e89b-12d3-a456-426614174000@vless.example.com:443?encryption=none&security=tls&type=ws&host=cdn.example.com&path=%2Fvless&sni=front.example.com#vless-demo"

	got, summary, err := ConvertToClash([]byte(input))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if summary.DetectedType != "vless" {
		t.Fatalf("DetectedType = %q, want %q", summary.DetectedType, "vless")
	}
	if !bytes.Contains(got, []byte("type: vless")) {
		t.Fatalf("converted content missing vless proxy:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("servername: front.example.com")) {
		t.Fatalf("converted content missing sni:\n%s", string(got))
	}
}

func TestConvertToClashConvertsMixedSubscriptionLines(t *testing.T) {
	vmessJSON := `{"v":"2","ps":"mix-vmess","add":"mix-vmess.example.com","port":"8443","id":"123e4567-e89b-12d3-a456-426614174000","aid":"0","net":"tcp","type":"none","host":"","path":"","tls":""}`
	input := strings.Join([]string{
		"ss://YWVzLTI1Ni1nY206cGFzcw==@ss.example.com:443#mix-ss",
		"vmess://" + base64.StdEncoding.EncodeToString([]byte(vmessJSON)),
		"trojan://secret@trojan.example.com:443#mix-trojan",
	}, "\n")

	got, summary, err := ConvertToClash([]byte(input))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if summary.DetectedType != "mixed" {
		t.Fatalf("DetectedType = %q, want %q", summary.DetectedType, "mixed")
	}
	if summary.NodeCount != 3 {
		t.Fatalf("NodeCount = %d, want 3", summary.NodeCount)
	}
	if bytes.Count(got, []byte("\n    - ")) != 3 {
		t.Fatalf("expected 3 proxies in output:\n%s", string(got))
	}
}
