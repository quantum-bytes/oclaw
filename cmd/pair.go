package cmd

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"

	qrcode "github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"

	"github.com/quantum-bytes/oclaw/internal/config"
)

var pairHost string

var pairCmd = &cobra.Command{
	Use:   "pair",
	Short: "Display a QR code for pairing the oclaw mobile app",
	Long: `Generates a QR code containing the gateway connection details.
Scan this code with the oclaw mobile app to connect instantly.

The QR encodes an oclaw:// URI with the gateway's LAN IP, port, auth token,
and device credentials for scope authorization.
Use --host to override the auto-detected LAN IP if the wrong interface is selected.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(flagURL, flagToken, flagAgent)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// Parse gateway URL to extract port
		gwURL, err := url.Parse(cfg.GatewayURL)
		if err != nil {
			return fmt.Errorf("parse gateway URL %q: %w", cfg.GatewayURL, err)
		}

		port := gwURL.Port()
		if port == "" {
			port = "39421"
		}

		// Determine host: explicit flag > auto-detect
		host := pairHost
		if host == "" {
			host, err = detectLANIP()
			if err != nil {
				return fmt.Errorf("detect LAN IP (use --host to specify manually): %w", err)
			}
		}

		// Build query params
		params := url.Values{"token": {cfg.Token}}

		// Load device identity for scope authorization
		deviceID, privKeyB64, err := loadDeviceKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: no device identity found (%v)\n", err)
			fmt.Fprintf(os.Stderr, "  Mobile app will connect without operator scopes\n\n")
		} else {
			params.Set("did", deviceID)
			params.Set("dkey", privKeyB64)
		}

		// Build oclaw:// URI
		pairURI := &url.URL{
			Scheme:   "oclaw",
			Host:     net.JoinHostPort(host, port),
			RawQuery: params.Encode(),
		}
		uri := pairURI.String()

		// Generate QR code as terminal string
		qr, err := qrcode.New(uri, qrcode.Medium)
		if err != nil {
			return fmt.Errorf("generate QR code: %w", err)
		}

		// Mask token for display
		maskedToken := cfg.Token
		if len(maskedToken) > 4 {
			maskedToken = maskedToken[:4] + "****"
		}

		fmt.Println()
		fmt.Println("  Scan this QR code with the oclaw mobile app:")
		fmt.Println()
		fmt.Print(qr.ToSmallString(false))
		fmt.Println()
		fmt.Printf("  Host:   %s:%s\n", host, port)
		fmt.Printf("  Token:  %s\n", maskedToken)
		if deviceID != "" {
			fmt.Printf("  Device: %s...%s\n", deviceID[:8], deviceID[len(deviceID)-8:])
		}
		fmt.Println()

		return nil
	},
}

func init() {
	pairCmd.Flags().StringVar(&pairHost, "host", "", "Override auto-detected LAN IP (e.g., 192.168.1.100)")
	rootCmd.AddCommand(pairCmd)
}

// deviceJSON is the structure of ~/.openclaw/identity/device.json.
type deviceJSON struct {
	DeviceID      string `json:"deviceId"`
	PrivateKeyPem string `json:"privateKeyPem"`
}

// loadDeviceKey reads the device identity and returns the device ID and
// base64-encoded raw Ed25519 private key seed (32 bytes).
func loadDeviceKey() (deviceID, privKeyB64 string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}

	path := filepath.Join(home, ".openclaw", "identity", "device.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	var dev deviceJSON
	if err := json.Unmarshal(data, &dev); err != nil {
		return "", "", fmt.Errorf("parse device.json: %w", err)
	}

	// Parse PEM-encoded private key
	block, _ := pem.Decode([]byte(dev.PrivateKeyPem))
	if block == nil {
		return "", "", fmt.Errorf("no PEM block found in privateKeyPem")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", "", fmt.Errorf("parse PKCS8 key: %w", err)
	}

	edKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return "", "", fmt.Errorf("not an Ed25519 key")
	}

	// Ed25519 private key is 64 bytes; seed is first 32
	seed := edKey.Seed()
	return dev.DeviceID, base64.StdEncoding.EncodeToString(seed), nil
}

// detectLANIP returns the first non-loopback, non-virtual IPv4 address.
// It prefers physical interfaces (en*, eth*) over virtual ones.
func detectLANIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var fallback string

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}

			if ip4[0] == 169 && ip4[1] == 254 {
				continue
			}

			name := iface.Name
			if (len(name) >= 2 && name[:2] == "en") || (len(name) >= 3 && name[:3] == "eth") {
				return ip4.String(), nil
			}

			if fallback == "" {
				fallback = ip4.String()
			}
		}
	}

	if fallback != "" {
		return fallback, nil
	}

	return "", fmt.Errorf("no LAN IPv4 address found")
}
