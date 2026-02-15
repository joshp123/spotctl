package spotify

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CallbackResult struct {
	Code  string
	State string
	Error string
}

type LocalCallbackServer struct {
	RedirectURL string

	ln     net.Listener
	srv    *http.Server
	result chan CallbackResult
}

// StartLocalCallbackServer starts a local HTTPS server for a redirect_uri like:
//
//	https://localhost:8899/callback
//
// Note: uses a self-signed cert; browsers will show a warning you must accept.
func StartLocalCallbackServer(redirectURI string) (*LocalCallbackServer, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect uri: %w", err)
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("redirect uri must be https (Spotify requires secure redirect URIs): %s", redirectURI)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("redirect uri missing host: %s", redirectURI)
	}
	if u.Path == "" {
		u.Path = "/callback"
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		return nil, fmt.Errorf("redirect uri must include an explicit port (e.g. https://localhost:8899/callback)")
	}
	addr := net.JoinHostPort(host, port)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", addr, err)
	}

	c, err := selfSignedCert(host)
	if err != nil {
		_ = ln.Close()
		return nil, err
	}
	cert := &c

	resCh := make(chan CallbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		res := CallbackResult{
			Code:  q.Get("code"),
			State: q.Get("state"),
			Error: q.Get("error"),
		}
		select {
		case resCh <- res:
		default:
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body><h1>Spotify auth complete</h1><p>You can close this tab.</p></body></html>"))
	})

	srv := &http.Server{Handler: mux}
	srv.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{*cert},
		MinVersion:   tls.VersionTLS12,
	}

	go func() {
		_ = srv.ServeTLS(ln, "", "")
	}()

	u2 := *u
	u2.RawQuery = ""

	return &LocalCallbackServer{
		RedirectURL: u2.String(),
		ln:          ln,
		srv:         srv,
		result:      resCh,
	}, nil
}

func (s *LocalCallbackServer) Wait(ctx context.Context) (CallbackResult, error) {
	select {
	case <-ctx.Done():
		return CallbackResult{}, ctx.Err()
	case res := <-s.result:
		return res, nil
	}
}

func (s *LocalCallbackServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.srv.Shutdown(ctx)
	if s.ln != nil {
		_ = s.ln.Close()
	}
	return nil
}

func selfSignedCert(host string) (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	notBefore := time.Now().Add(-1 * time.Minute)
	notAfter := time.Now().Add(2 * time.Hour)

	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return tls.Certificate{}, err
	}

	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// SANs
	if ip := net.ParseIP(host); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{host}
		// Convenience: if user uses localhost, include 127.0.0.1 too.
		if strings.EqualFold(host, "localhost") {
			tmpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})

	c, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, err
	}
	return c, nil
}
