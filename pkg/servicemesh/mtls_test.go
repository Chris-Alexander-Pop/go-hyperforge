package servicemesh_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery"
)

func TestMTLSConfigDisabled(t *testing.T) {
	cfg := servicemesh.MTLSConfig{Enabled: false}
	tlsCfg, err := cfg.TLSConfig()
	if err != nil || tlsCfg != nil {
		t.Fatalf("expected nil tls config, got %v %v", tlsCfg, err)
	}
	client, err := cfg.HTTPClient(nil)
	if err != nil || client == nil {
		t.Fatalf("HTTPClient: %v %v", client, err)
	}
}

func TestMTLSConfigWithCerts(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, caFile := writeTestCerts(t, dir)

	cfg := servicemesh.MTLSConfig{
		Enabled:  true,
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}
	tlsCfg, err := cfg.TLSConfig()
	if err != nil {
		t.Fatalf("TLSConfig: %v", err)
	}
	if len(tlsCfg.Certificates) != 1 || tlsCfg.RootCAs == nil {
		t.Fatal("expected cert and CA pool")
	}

	client, err := discovery.WithMTLS(nil, cfg)
	if err != nil {
		t.Fatalf("WithMTLS: %v", err)
	}
	tr, ok := client.Transport.(*http.Transport)
	if !ok || tr.TLSClientConfig == nil {
		t.Fatal("expected TLS transport")
	}
}

func TestDialTLSPlain(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, _ := ln.Accept()
		if c != nil {
			_ = c.Close()
		}
	}()

	conn, err := servicemesh.DialTLS("tcp", ln.Addr().String(), servicemesh.MTLSConfig{}, time.Second)
	if err != nil {
		t.Fatalf("DialTLS: %v", err)
	}
	_ = conn.Close()
}

func TestHTTPClientHitsServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := servicemesh.MTLSConfig{Enabled: false}.HTTPClient(nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}

func writeTestCerts(t *testing.T, dir string) (certFile, keyFile, caFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	caFile = filepath.Join(dir, "ca.pem")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(certFile, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caFile, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	// Ensure key pair loads.
	if _, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile, caFile
}
