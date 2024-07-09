package config

// // NOTE: ONLY PUT STRUCT DEFINITIONS IN THIS FILE
// //
// // After making edits here, run 'make cfgdoc-gen' (or 'make gen')

// Common is common config between full node and miner
type Common struct {
	API API
}

// API contains configs for API endpoint
type API struct {
	ListenAddress       string
	RemoteListenAddress string
	Timeout             Duration
}

// ManagerCfg manager config
type ManagerCfg struct {
	Common
	// database address
	DatabaseAddress string

	DNSServerAddress string
}

// ProviderCfg provider config
type ProviderCfg struct {
	Common
	BaseHostname string
	// Timeout string specifies the duration to wait before timing out
	Timeout string
	// Owner string identifies the owner of the resource
	Owner string
	// ExternalIP If the external IP is empty, it should be automatically obtained
	ExternalIP string
	// IngressHostName specifies the ingress hostname associated with the resource
	IngressHostName string
	// Certificate is the path to the security certificate file
	Certificate string
	// CertificateKey is the path to the key file for the security certificate
	CertificateKey string
	// IngressClassName specifies the class of the ingress resource
	IngressClassName string
	// KubeConfigPath specifies the path to the Kubernetes configuration file
	KubeConfigPath string
	// WalletDir specifies the directory path where the wallet files are stored
	WalletDir string
}
