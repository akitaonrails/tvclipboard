package config

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// cliFlags holds parsed CLI flag values
type cliFlags struct {
	portFlag    string
	baseURLFlag string
	expiresFlag int
	keyFlag      string
	helpFlag     bool
}

var cfg = cliFlags{}

// Config holds the application configuration
type Config struct {
	Port            string
	PublicURL       string
	SessionTimeout  time.Duration
	PrivateKeyHex   string
	LocalIP         string
	showHelp        bool
}

// Load loads configuration from environment variables and CLI flags
func Load() *Config {
	// Parse CLI flags
	flag.StringVar(&cfg.portFlag, "port", "", "Server port (default: 3333, env: PORT)")
	flag.StringVar(&cfg.baseURLFlag, "base-url", "", "Public base URL for QR codes (e.g., https://example.com, env: TVCLIPBOARD_PUBLIC_URL)")
	flag.IntVar(&cfg.expiresFlag, "expires", 0, "Session timeout in minutes (default: 10, env: TVCLIPBOARD_SESSION_TIMEOUT)")
	flag.StringVar(&cfg.keyFlag, "key", "", "Private key hex string (env: TVCLIPBOARD_PRIVATE_KEY)")
	flag.BoolVar(&cfg.helpFlag, "help", false, "Show this help message")
	flag.Parse()

	if cfg.helpFlag {
		printUsage()
		os.Exit(0)
	}

	// Load from environment variables (fallback to CLI flags if set)
	port := cfg.portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "3333"
	}

	timeoutMinutes := cfg.expiresFlag
	if timeoutMinutes == 0 {
		timeoutStr := os.Getenv("TVCLIPBOARD_SESSION_TIMEOUT")
		var err error
		timeoutMinutes, err = strconv.Atoi(timeoutStr)
		if err != nil || timeoutMinutes <= 0 {
			timeoutMinutes = 10
		}
	}

	privateKeyHex := cfg.keyFlag
	if privateKeyHex == "" {
		privateKeyHex = os.Getenv("TVCLIPBOARD_PRIVATE_KEY")
	}

	publicURL := cfg.baseURLFlag
	if publicURL == "" {
		publicURL = os.Getenv("TVCLIPBOARD_PUBLIC_URL")
	}

	config := &Config{
		Port:            port,
		PublicURL:       publicURL,
		SessionTimeout:  time.Duration(timeoutMinutes) * time.Minute,
		PrivateKeyHex:   privateKeyHex,
		LocalIP:         getLocalIP(),
		showHelp:        cfg.helpFlag,
	}

	return config
}

// printUsage displays help information
func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
	fmt.Fprintf(os.Stderr, "  PORT                        Server port (default: 3333)\n")
	fmt.Fprintf(os.Stderr, "  TVCLIPBOARD_PUBLIC_URL      Public base URL for QR codes (default: auto-detected local IP)\n")
	fmt.Fprintf(os.Stderr, "  TVCLIPBOARD_SESSION_TIMEOUT  Session timeout in minutes (default: 10)\n")
	fmt.Fprintf(os.Stderr, "  TVCLIPBOARD_PRIVATE_KEY      Private key hex string (auto-generated if not set)\n")
	fmt.Fprintf(os.Stderr, "\nCLI flags override environment variables.\n")
}

// getLocalIP returns the local IP address
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "localhost"
}

// GetQRHost returns the host to use for QR codes
// If PublicURL is set, uses that; otherwise uses LocalIP
func (c *Config) GetQRHost() string {
	if c.PublicURL != "" {
		parsed, err := url.Parse(c.PublicURL)
		if err == nil {
			return parsed.Hostname()
		}
		return c.PublicURL
	}
	return c.LocalIP
}

// GetQRScheme returns the scheme (http or https) for QR codes
// If PublicURL is set and includes scheme, uses that; otherwise defaults to http
func (c *Config) GetQRScheme() string {
	if c.PublicURL != "" {
		parsed, err := url.Parse(c.PublicURL)
		if err == nil && parsed.Scheme != "" {
			return parsed.Scheme
		}
		if strings.HasPrefix(c.PublicURL, "https://") {
			return "https"
		}
	}
	return "http"
}

// LogStartup logs the server startup information
func (c *Config) LogStartup() {
	log.Printf("Server starting on port %s\n", c.Port)
	log.Printf("Session timeout: %v minutes\n", int(c.SessionTimeout.Minutes()))
	log.Printf("Local access: http://localhost:%s\n", c.Port)

	if c.PublicURL != "" {
		log.Printf("Public access: %s\n", c.PublicURL)
		log.Printf("QR code will use: %s?mode=client\n", c.PublicURL)
	} else if c.LocalIP != "localhost" {
		log.Printf("Network access: http://%s:%s\n", c.LocalIP, c.Port)
		log.Printf("QR code will use: http://%s:%s?mode=client\n", c.LocalIP, c.Port)
	}

	log.Printf("Open in browser and scan QR code with your phone\n")
}
