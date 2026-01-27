package config

import (
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	Port            string
	SessionTimeout  time.Duration
	PrivateKeyHex   string
	LocalIP         string
}

// Load loads configuration from environment variables and defaults
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3333"
	}

	timeoutMinutes, err := strconv.Atoi(os.Getenv("TVCLIPBOARD_SESSION_TIMEOUT"))
	if err != nil || timeoutMinutes <= 0 {
		timeoutMinutes = 10
	}

	privateKeyHex := os.Getenv("TVCLIPBOARD_PRIVATE_KEY")

	cfg := &Config{
		Port:            port,
		SessionTimeout:  time.Duration(timeoutMinutes) * time.Minute,
		PrivateKeyHex:   privateKeyHex,
		LocalIP:         getLocalIP(),
	}

	return cfg
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

// LogStartup logs the server startup information
func (c *Config) LogStartup() {
	log.Printf("Server starting on port %s\n", c.Port)
	log.Printf("Session timeout: %v minutes\n", int(c.SessionTimeout.Minutes()))
	log.Printf("Local access: http://localhost:%s\n", c.Port)
	if c.LocalIP != "localhost" {
		log.Printf("Network access: http://%s:%s\n", c.LocalIP, c.Port)
		log.Printf("QR code will use: http://%s:%s?mode=client\n", c.LocalIP, c.Port)
	}
	log.Printf("Open in browser and scan QR code with your phone\n")
}
