package middleware

import (
	"log"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

func getClientIP(c *gin.Context) string {
	clientIP := c.ClientIP()

	if clientIP == "127.0.0.1" || clientIP == "::1" {
		return clientIP
	}

	xForwardedFor := c.GetHeader("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		clientIP = strings.TrimSpace(ips[0])
	}

	xRealIP := c.GetHeader("X-Real-IP")
	if xRealIP != "" {
		clientIP = xRealIP
	}

	return clientIP
}

func getMACAddresses(ip string) ([]string, error) {
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		interfaces, err := net.Interfaces()
		if err != nil {
			return nil, err
		}

		var macs []string
		for _, iface := range interfaces {
			if len(iface.HardwareAddr) > 0 {
				macs = append(macs, iface.HardwareAddr.String())
			}
		}
		return macs, nil
	}

	return []string{}, nil
}

func getAdminMACs() []string {
	adminMACStr := os.Getenv("QUIZ_ADMIN_MAC_ADDR")
	if adminMACStr == "" {
		log.Println("Warning: QUIZ_ADMIN_MAC_ADDR environment variable is not set")
		return []string{}
	}

	macs := strings.Split(adminMACStr, ",")
	var cleanMACs []string
	for _, mac := range macs {
		cleanMAC := strings.TrimSpace(mac)
		if cleanMAC != "" {
			cleanMACs = append(cleanMACs, strings.ToLower(cleanMAC))
		}
	}
	return cleanMACs
}

func getAdminIPs() []string {
	adminIPStr := os.Getenv("QUIZ_ADMIN_IP_ADDR")
	if adminIPStr == "" {
		return []string{}
	}

	ips := strings.Split(adminIPStr, ",")
	var cleanIPs []string
	for _, ip := range ips {
		cleanIP := strings.TrimSpace(ip)
		if cleanIP != "" {
			cleanIPs = append(cleanIPs, cleanIP)
		}
	}
	return cleanIPs
}

func getAdminToken() string {
	return strings.TrimSpace(os.Getenv("QUIZ_ADMIN_TOKEN"))
}

func isAdminIP(clientIP string) bool {
	adminIPs := getAdminIPs()
	if len(adminIPs) == 0 {
		return false
	}

	for _, adminIP := range adminIPs {
		if clientIP == adminIP {
			return true
		}
		// Support CIDR notation for IP ranges
		if strings.Contains(adminIP, "/") {
			_, network, err := net.ParseCIDR(adminIP)
			if err == nil {
				ip := net.ParseIP(clientIP)
				if ip != nil && network.Contains(ip) {
					return true
				}
			}
		}
	}
	return false
}

func hasValidAdminToken(c *gin.Context) bool {
	adminToken := getAdminToken()
	if adminToken == "" {
		return false
	}

	// Check Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			return token == adminToken
		}
	}

	// Check query parameter
	queryToken := c.Query("token")
	if queryToken == adminToken {
		return true
	}

	// Check form parameter
	formToken := c.PostForm("token")
	if formToken == adminToken {
		return true
	}

	return false
}

func isAdminMAC(clientMACs []string) bool {
	adminMACs := getAdminMACs()
	if len(adminMACs) == 0 {
		return false
	}

	for _, clientMAC := range clientMACs {
		clientMACLower := strings.ToLower(clientMAC)
		return slices.Contains(adminMACs, clientMACLower)
	}
	return false
}

func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := getClientIP(c)
		log.Printf("Admin access attempt from IP: %s", clientIP)

		// Method 1: Token-based authentication (highest priority)
		if hasValidAdminToken(c) {
			log.Printf("Admin access granted for IP %s via TOKEN authentication", clientIP)
			c.Next()
			return
		}

		// Method 2: IP-based authentication
		if isAdminIP(clientIP) {
			log.Printf("Admin access granted for IP %s via IP authentication", clientIP)
			c.Next()
			return
		}

		// Method 3: MAC-based authentication (fallback for native execution)
		clientMACs, err := getMACAddresses(clientIP)
		if err == nil && len(clientMACs) > 0 {
			log.Printf("Client MACs: %v", clientMACs)
			if isAdminMAC(clientMACs) {
				log.Printf("Admin access granted for IP %s via MAC authentication", clientIP)
				c.Next()
				return
			}
		}

		log.Printf("Access denied for IP %s - No valid authentication method", clientIP)
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied - Admin authentication required",
			"hint":  "Use token, IP whitelist, or MAC address authentication",
		})
		c.Abort()
	}
}

func ScreenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := getClientIP(c)
		log.Printf("Screen access attempt from IP: %s", clientIP)

		// Method 1: Token-based authentication (highest priority)
		if hasValidAdminToken(c) {
			log.Printf("Screen access granted for IP %s via TOKEN authentication", clientIP)
			c.Next()
			return
		}

		// Method 2: IP-based authentication
		if isAdminIP(clientIP) {
			log.Printf("Screen access granted for IP %s via IP authentication", clientIP)
			c.Next()
			return
		}

		// Method 3: MAC-based authentication (fallback for native execution)
		clientMACs, err := getMACAddresses(clientIP)
		if err == nil && len(clientMACs) > 0 {
			if isAdminMAC(clientMACs) {
				log.Printf("Screen access granted for IP %s via MAC authentication", clientIP)
				c.Next()
				return
			}
		}

		log.Printf("Screen access denied for IP %s - No valid authentication method", clientIP)
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied - Screen authentication required",
			"hint":  "Use token, IP whitelist, or MAC address authentication",
		})
		c.Abort()
	}
}

func LogMACInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := getClientIP(c)
		clientMACs, err := getMACAddresses(clientIP)

		if err != nil {
			log.Printf("Request from IP: %s (MAC address lookup failed: %v)", clientIP, err)
		} else {
			log.Printf("Request from IP: %s, MACs: %v", clientIP, clientMACs)
		}

		adminMACs := getAdminMACs()
		log.Printf("Configured admin MACs: %v", adminMACs)

		c.Next()
	}
}

func init() {
	log.Println("Multi-method authentication middleware initialized")

	// Check MAC addresses
	adminMACs := getAdminMACs()
	if len(adminMACs) > 0 {
		log.Printf("✅ MAC authentication enabled: %v", adminMACs)
	} else {
		log.Println("❌ MAC authentication disabled: No QUIZ_ADMIN_MAC_ADDR configured")
	}

	// Check IP addresses
	adminIPs := getAdminIPs()
	if len(adminIPs) > 0 {
		log.Printf("✅ IP authentication enabled: %v", adminIPs)
	} else {
		log.Println("❌ IP authentication disabled: No QUIZ_ADMIN_IP_ADDR configured")
	}

	// Check token
	adminToken := getAdminToken()
	if adminToken != "" {
		log.Printf("✅ Token authentication enabled: %s***", adminToken[:min(len(adminToken), 4)])
	} else {
		log.Println("❌ Token authentication disabled: No QUIZ_ADMIN_TOKEN configured")
	}

	if len(adminMACs) == 0 && len(adminIPs) == 0 && adminToken == "" {
		log.Println("⚠️  WARNING: No authentication methods configured! Admin access will be denied.")
		log.Println("   Set at least one of: QUIZ_ADMIN_MAC_ADDR, QUIZ_ADMIN_IP_ADDR, or QUIZ_ADMIN_TOKEN")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
