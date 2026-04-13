package converter

import "errors"

var ErrUnsupportedSubscription = errors.New("unsupported subscription format")

type SubscriptionType string

const (
	TypeUnknown SubscriptionType = "unknown"
	TypeClash   SubscriptionType = "clash"
	TypeSS      SubscriptionType = "ss"
	TypeSSR     SubscriptionType = "ssr"
	TypeVMess   SubscriptionType = "vmess"
	TypeTrojan  SubscriptionType = "trojan"
	TypeVLESS   SubscriptionType = "vless"
	TypeMixed   SubscriptionType = "mixed"
)

type ProxyNode struct {
	Name              string
	Type              string
	Server            string
	Port              int
	Cipher            string
	Password          string
	UUID              string
	AlterID           int
	Network           string
	TLS               bool
	ServerName        string
	Host              string
	Path              string
	UDP               bool
	Protocol          string
	ProtocolParam     string
	Obfs              string
	ObfsParam         string
	Plugin            string
	PluginMode        string
	PluginHost        string
	PluginPath        string
	PluginTLS         *bool
	PluginMux         *bool
	ClientFingerprint string
	PublicKey         string
	ShortID           string
	ServiceName       string
	Flow              string
	SkipCertVerify    *bool
	TFO               *bool
}

type ConversionSummary struct {
	DetectedType string
	OutputType   string
	NodeCount    int
}
