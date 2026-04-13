package converter

import "errors"

var ErrUnsupportedSubscription = errors.New("unsupported subscription format")

type SubscriptionType string

const (
	TypeUnknown SubscriptionType = "unknown"
	TypeClash   SubscriptionType = "clash"
	TypeSS      SubscriptionType = "ss"
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
	ClientFingerprint string
	PublicKey         string
	ShortID           string
	ServiceName       string
}

type ConversionSummary struct {
	DetectedType string
	OutputType   string
	NodeCount    int
}
