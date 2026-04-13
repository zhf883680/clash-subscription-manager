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

func TestConvertToClashConvertsSSSubscriptionWithPluginOptions(t *testing.T) {
	input := "ss://YWVzLTI1Ni1nY206cGFzcw==@example.com:443?plugin=v2ray-plugin%3Bmode%3Dwebsocket%3Bhost%3Dcdn.example.com%3Bpath%3D%2Fws%3Btls%3Bmux#demo-plugin"

	got, _, err := ConvertToClash([]byte(input))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if !bytes.Contains(got, []byte("plugin: v2ray-plugin")) {
		t.Fatalf("converted content missing ss plugin:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("plugin-opts:")) {
		t.Fatalf("converted content missing ss plugin-opts:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("mode: websocket")) {
		t.Fatalf("converted content missing ss plugin mode:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("host: cdn.example.com")) {
		t.Fatalf("converted content missing ss plugin host:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("path: /ws")) {
		t.Fatalf("converted content missing ss plugin path:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("tls: true")) {
		t.Fatalf("converted content missing ss plugin tls flag:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("mux: true")) {
		t.Fatalf("converted content missing ss plugin mux flag:\n%s", string(got))
	}
}

func TestConvertToClashConvertsSSRSubscription(t *testing.T) {
	raw := "example.com:443:origin:aes-256-cfb:plain:" + base64.RawURLEncoding.EncodeToString([]byte("pass")) +
		"/?remarks=" + base64.RawURLEncoding.EncodeToString([]byte("ssr-demo")) +
		"&obfsparam=" + base64.RawURLEncoding.EncodeToString([]byte("cdn.example.com")) +
		"&protoparam=" + base64.RawURLEncoding.EncodeToString([]byte("proto"))
	input := "ssr://" + base64.RawURLEncoding.EncodeToString([]byte(raw))

	got, summary, err := ConvertToClash([]byte(input))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if summary.DetectedType != "ssr" {
		t.Fatalf("DetectedType = %q, want %q", summary.DetectedType, "ssr")
	}
	if !bytes.Contains(got, []byte("type: ssr")) {
		t.Fatalf("converted content missing ssr proxy:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("protocol: origin")) {
		t.Fatalf("converted content missing ssr protocol:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("obfs: plain")) {
		t.Fatalf("converted content missing ssr obfs:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("protocol-param: proto")) {
		t.Fatalf("converted content missing ssr protocol-param:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("obfs-param: cdn.example.com")) {
		t.Fatalf("converted content missing ssr obfs-param:\n%s", string(got))
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

func TestConvertToClashConvertsVMessSubscriptionWithHTTPOptions(t *testing.T) {
	vmessJSON := `{"v":"2","ps":"vmess-http","add":"vmess.example.com","port":"443","id":"123e4567-e89b-12d3-a456-426614174000","aid":"0","net":"http","type":"none","host":"cdn.example.com","path":"/http","tls":"tls","sni":"front.example.com","scy":"zero"}`
	input := "vmess://" + base64.StdEncoding.EncodeToString([]byte(vmessJSON))

	got, _, err := ConvertToClash([]byte(input))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if !bytes.Contains(got, []byte("http-opts:")) {
		t.Fatalf("converted content missing http-opts:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("- /http")) {
		t.Fatalf("converted content missing vmess http path:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("- cdn.example.com")) {
		t.Fatalf("converted content missing vmess http host:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("cipher: zero")) {
		t.Fatalf("converted content missing vmess cipher:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("servername: front.example.com")) {
		t.Fatalf("converted content missing vmess sni:\n%s", string(got))
	}
}

func TestConvertToClashConvertsVMessSubscriptionWithGrpcOptions(t *testing.T) {
	vmessJSON := `{"v":"2","ps":"vmess-grpc","add":"vmess.example.com","port":"443","id":"123e4567-e89b-12d3-a456-426614174000","aid":"0","net":"grpc","path":"vmess-grpc","host":"front.example.com","tls":"tls"}`
	input := "vmess://" + base64.StdEncoding.EncodeToString([]byte(vmessJSON))

	got, _, err := ConvertToClash([]byte(input))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if !bytes.Contains(got, []byte("grpc-opts:")) {
		t.Fatalf("converted content missing vmess grpc-opts:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("grpc-service-name: vmess-grpc")) {
		t.Fatalf("converted content missing vmess grpc service name:\n%s", string(got))
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

func TestConvertToClashConvertsTrojanSubscriptionWithGrpcAndFlags(t *testing.T) {
	input := "trojan://secret@trojan.example.com:443?type=grpc&serviceName=grpc-demo&sni=edge.example.com&allowInsecure=1&tfo=1#trojan-grpc"

	got, _, err := ConvertToClash([]byte(input))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if !bytes.Contains(got, []byte("grpc-opts:")) {
		t.Fatalf("converted content missing trojan grpc-opts:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("grpc-service-name: grpc-demo")) {
		t.Fatalf("converted content missing trojan grpc service name:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("skip-cert-verify: true")) {
		t.Fatalf("converted content missing trojan skip-cert-verify:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("tfo: true")) {
		t.Fatalf("converted content missing trojan tfo:\n%s", string(got))
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

func TestConvertToClashConvertsVLESSSubscriptionWithGrpcAndFlags(t *testing.T) {
	input := "vless://123e4567-e89b-12d3-a456-426614174000@vless.example.com:443?encryption=none&security=tls&type=grpc&serviceName=grpc-vless&sni=front.example.com&allowInsecure=0&tfo=1#vless-grpc"

	got, _, err := ConvertToClash([]byte(input))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if !bytes.Contains(got, []byte("grpc-opts:")) {
		t.Fatalf("converted content missing vless grpc-opts:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("grpc-service-name: grpc-vless")) {
		t.Fatalf("converted content missing vless grpc service name:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("skip-cert-verify: false")) {
		t.Fatalf("converted content missing vless skip-cert-verify false:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("tfo: true")) {
		t.Fatalf("converted content missing vless tfo:\n%s", string(got))
	}
}

func TestConvertToClashConvertsVLESSRealityFlowOptions(t *testing.T) {
	input := "vless://d26c44d9-0ce8-480d-a6db-41161325eee1@96.44.165.5:46464?encryption=none&flow=xtls-rprx-vision&fp=chrome&pbk=vUwf9NWQLZdF08PXR-Zq8sy68pzTdpc4oC-ayCca6DA&security=reality&sid=e875f0d4&sni=digitalassets.tesla.com&spx=%2FlPEPAqyhqXlBCr6&type=tcp#%E7%BE%8E%E5%9B%BD%20VPS-jbirgl3y"

	got, summary, err := ConvertToClash([]byte(input))
	if err != nil {
		t.Fatalf("ConvertToClash() error = %v", err)
	}
	if summary.DetectedType != "vless" {
		t.Fatalf("DetectedType = %q, want %q", summary.DetectedType, "vless")
	}
	if !bytes.Contains(got, []byte("flow: xtls-rprx-vision")) {
		t.Fatalf("converted content missing flow:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("skip-cert-verify: true")) {
		t.Fatalf("converted content missing skip-cert-verify:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("tfo: false")) {
		t.Fatalf("converted content missing tfo:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("client-fingerprint: chrome")) {
		t.Fatalf("converted content missing client fingerprint:\n%s", string(got))
	}
	if !bytes.Contains(got, []byte("servername: digitalassets.tesla.com")) {
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
