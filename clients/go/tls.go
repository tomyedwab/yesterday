package yesterdaygo

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// configureTLSForLocalhost configures TLS settings for localhost domains
// Returns a custom TLS config if the baseURL is a localhost domain and certificates are found,
// otherwise returns nil to use default TLS settings
func configureTLSForLocalhost(baseURL string) (*tls.Config, error) {
	// Parse the base URL to extract hostname
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	hostname := parsedURL.Hostname()
	
	// Only apply custom TLS config for .localhost domains
	if !strings.HasSuffix(hostname, ".localhost") {
		return nil, nil // Not a localhost domain, use default TLS
	}

	// Check if CERTS_DIR environment variable is set
	certsDir := os.Getenv("CERTS_DIR")
	if certsDir == "" {
		log.Printf("CERTS_DIR not set, using default TLS verification for %s", hostname)
		return nil, nil // No certificate dir specified, use default TLS
	}

	// Load certificates from the specified directory
	caCertPool, err := loadCertificatesFromDir(certsDir)
	if err != nil {
		log.Printf("Failed to load certificates from %s: %v", certsDir, err)
		return nil, nil // Fall back to default TLS verification
	}

	if caCertPool == nil {
		log.Printf("No certificates found in %s, using default TLS verification", certsDir)
		return nil, nil // No certificates found, use default TLS
	}

	log.Printf("Successfully loaded certificates from %s for %s", certsDir, hostname)

	// Create custom TLS config with loaded certificates
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	return tlsConfig, nil
}

// loadCertificatesFromDir loads all certificate files from the specified directory
// and returns a certificate pool, or nil if no certificates are found
func loadCertificatesFromDir(certsDir string) (*x509.CertPool, error) {
	// Check if directory exists
	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("certificates directory does not exist: %s", certsDir)
	}

	// Read directory contents
	files, err := ioutil.ReadDir(certsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificates directory: %w", err)
	}

	// Create certificate pool
	caCertPool := x509.NewCertPool()
	certificatesLoaded := 0

	// Process each file in the directory
	for _, file := range files {
		if file.IsDir() {
			continue // Skip directories
		}

		// Check if file has a certificate extension
		filename := file.Name()
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".crt" && ext != ".pem" && ext != ".cer" {
			continue // Skip non-certificate files
		}

		// Load certificate file
		certPath := filepath.Join(certsDir, filename)
		cert, err := loadCertificateFile(certPath)
		if err != nil {
			log.Printf("Failed to load certificate %s: %v", certPath, err)
			continue // Skip this certificate but continue with others
		}

		// Add certificate to pool
		if caCertPool.AppendCertsFromPEM(cert) {
			certificatesLoaded++
			log.Printf("Loaded certificate: %s", certPath)
		} else {
			log.Printf("Failed to parse certificate: %s", certPath)
		}
	}

	if certificatesLoaded == 0 {
		return nil, nil // No certificates were successfully loaded
	}

	log.Printf("Successfully loaded %d certificate(s) from %s", certificatesLoaded, certsDir)
	return caCertPool, nil
}

// loadCertificateFile reads and validates a certificate file
func loadCertificateFile(certPath string) ([]byte, error) {
	// Check if file exists and is readable
	if _, err := os.Stat(certPath); err != nil {
		return nil, fmt.Errorf("certificate file not accessible: %w", err)
	}

	// Read certificate file
	certData, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	// Basic validation - check if it looks like a PEM certificate
	certStr := string(certData)
	if !strings.Contains(certStr, "-----BEGIN CERTIFICATE-----") {
		return nil, fmt.Errorf("file does not appear to contain a PEM certificate")
	}

	return certData, nil
}

// applyTLSConfigToClient applies the TLS configuration to an HTTP client
func applyTLSConfigToClient(httpClient *http.Client, tlsConfig *tls.Config) {
	if tlsConfig == nil {
		return // No TLS config to apply
	}

	// Get or create transport
	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		// Create new transport if none exists or it's not an http.Transport
		transport = &http.Transport{}
		httpClient.Transport = transport
	}

	// Apply TLS configuration
	transport.TLSClientConfig = tlsConfig
}
