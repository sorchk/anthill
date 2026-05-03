package tunnel

import (
	"net"

	tls "github.com/refraction-networking/utls"
)

type TLSFingerprintConfig struct {
	Enabled  bool   `json:"enabled"`
	Profile  string `json:"profile"`
}

func CreateFingerprintTLSConn(rawConn net.Conn, cfg *TLSFingerprintConfig) (net.Conn, error) {
	if cfg == nil || !cfg.Enabled {
		return rawConn, nil
	}

	var browserID tls.ClientHelloID
	switch cfg.Profile {
	case "chrome":
		browserID = tls.HelloChrome_Auto
	case "firefox":
		browserID = tls.HelloFirefox_Auto
	case "edge":
		browserID = tls.HelloEdge_Auto
	case "safari":
		browserID = tls.HelloSafari_Auto
	default:
		browserID = tls.HelloChrome_Auto
	}

	uconn := tls.UClient(rawConn, &tls.Config{
		ServerName: "example.com",
	}, browserID)

	if err := uconn.Handshake(); err != nil {
		return nil, err
	}

	return uconn, nil
}