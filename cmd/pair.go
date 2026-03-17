package cmd

import (
	"fmt"
	"net"
	"net/url"

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

The QR encodes an oclaw:// URI with the gateway's LAN IP, port, and auth token.
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

		// Build oclaw:// URI using url.URL for correctness
		pairURI := &url.URL{
			Scheme:   "oclaw",
			Host:     net.JoinHostPort(host, port),
			RawQuery: url.Values{"token": {cfg.Token}}.Encode(),
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
		fmt.Printf("  Host:  %s:%s\n", host, port)
		fmt.Printf("  Token: %s\n\n", maskedToken)

		return nil
	},
}

func init() {
	pairCmd.Flags().StringVar(&pairHost, "host", "", "Override auto-detected LAN IP (e.g., 192.168.1.100)")
	rootCmd.AddCommand(pairCmd)
}

// detectLANIP returns the first non-loopback, non-virtual IPv4 address.
// It prefers physical interfaces (en*, eth*) over virtual ones (docker*, veth*, br*, utun*).
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

			// Skip link-local (169.254.x.x)
			if ip4[0] == 169 && ip4[1] == 254 {
				continue
			}

			// Prefer physical interfaces (en0, en1, eth0, etc.)
			name := iface.Name
			if (len(name) >= 2 && name[:2] == "en") || (len(name) >= 3 && name[:3] == "eth") {
				return ip4.String(), nil
			}

			// Store first non-physical as fallback
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
