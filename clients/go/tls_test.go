package yesterdaygo

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigureTLSForLocalhost(t *testing.T) {
	tests := []struct {
		name           string
		baseURL        string
		setupCertsDir  bool
		createCert     bool
		expectTLSConfig bool
		expectError    bool
	}{
		{
			name:            "non-localhost domain",
			baseURL:         "https://api.example.com",
			setupCertsDir:   true,
			createCert:      true,
			expectTLSConfig: false,
			expectError:     false,
		},
		{
			name:            "localhost domain without CERTS_DIR",
			baseURL:         "https://api.yesterday.localhost",
			setupCertsDir:   false,
			createCert:      false,
			expectTLSConfig: false,
			expectError:     false,
		},
		{
			name:            "localhost domain with CERTS_DIR but no certs",
			baseURL:         "https://api.yesterday.localhost",
			setupCertsDir:   true,
			createCert:      false,
			expectTLSConfig: false,
			expectError:     false,
		},
		{
			name:            "localhost domain with CERTS_DIR and cert file",
			baseURL:         "https://api.yesterday.localhost",
			setupCertsDir:   true,
			createCert:      true,
			expectTLSConfig: false, // Will be false because our test cert is not valid
			expectError:     false,
		},
		{
			name:            "localhost domain with valid cert",
			baseURL:         "https://api.yesterday.localhost",
			setupCertsDir:   true,
			createCert:      true,
			expectTLSConfig: true,
			expectError:     false,
		},
		{
			name:            "invalid URL",
			baseURL:         "://invalid-url",
			setupCertsDir:   false,
			createCert:      false,
			expectTLSConfig: false,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			originalCertsDir := os.Getenv("CERTS_DIR")
			defer os.Setenv("CERTS_DIR", originalCertsDir)

			var tempDir string
			if tt.setupCertsDir {
				// Create temporary directory for certificates
				var err error
				tempDir, err = os.MkdirTemp("", "tls-test-*")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				defer os.RemoveAll(tempDir)

				os.Setenv("CERTS_DIR", tempDir)

				if tt.createCert {
					var certContent string
					if tt.name == "localhost domain with valid cert" {
						// Generate a valid test certificate programmatically
						certContent = generateValidTestCert(t)
					} else {
						// Create an incomplete certificate file for testing error handling
						certContent = `-----BEGIN CERTIFICATE-----
MIICXTCCAcagAwIBAgIJAPIAxxxxxxxxMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMjMwMTAxMDAwMDAwWhcNMjQwMTAxMDAwMDAwWjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKA
-----END CERTIFICATE-----`
					}
					certPath := filepath.Join(tempDir, "test.crt")
					if err := os.WriteFile(certPath, []byte(certContent), 0644); err != nil {
						t.Fatalf("Failed to write cert file: %v", err)
					}
				}
			} else {
				os.Unsetenv("CERTS_DIR")
			}

			// Test the function
			tlsConfig, err := configureTLSForLocalhost(tt.baseURL)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check TLS config expectation
			if tt.expectTLSConfig {
				if tlsConfig == nil {
					t.Errorf("Expected TLS config but got nil")
				} else if tlsConfig.RootCAs == nil {
					t.Errorf("Expected RootCAs to be set in TLS config")
				}
			} else {
				if tlsConfig != nil {
					t.Errorf("Expected nil TLS config but got: %+v", tlsConfig)
				}
			}
		})
	}
}

// generateValidTestCert creates a valid self-signed certificate for testing
func generateValidTestCert(t *testing.T) string {
	// Generate a private key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test Org"},
			Country:       []string{"US"},
			Province:      []string{"CA"},
			Locality:      []string{"Test City"},
			StreetAddress: []string{"Test Street"},
			PostalCode:    []string{"12345"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  nil,
		DNSNames:     []string{"*.localhost", "localhost"},
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Encode certificate to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return string(certPEM)
}

func TestApplyTLSConfigToClient(t *testing.T) {
	// Test with nil TLS config
	client := &http.Client{Timeout: 30 * time.Second}
	applyTLSConfigToClient(client, nil)
	
	// Should not have modified the client
	if client.Transport != nil {
		t.Errorf("Expected nil transport but got: %+v", client.Transport)
	}

	// Test with actual TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Just for testing
	}
	
	applyTLSConfigToClient(client, tlsConfig)
	
	// Should have created transport with TLS config
	if client.Transport == nil {
		t.Errorf("Expected transport to be created")
		return
	}
	
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Errorf("Expected http.Transport but got: %T", client.Transport)
		return
	}
	
	if transport.TLSClientConfig != tlsConfig {
		t.Errorf("Expected TLS config to be applied")
	}
}

func TestNewClientWithLocalhostDomain(t *testing.T) {
	// Test that NewClient doesn't fail with localhost domain
	client := NewClient("https://api.yesterday.localhost")
	
	if client == nil {
		t.Errorf("Expected client to be created")
		return
	}
	
	if client.GetBaseURL() != "https://api.yesterday.localhost" {
		t.Errorf("Expected baseURL to be set correctly")
	}
	
	// Client should have HTTP client configured
	if client.GetHTTPClient() == nil {
		t.Errorf("Expected HTTP client to be configured")
	}
}
