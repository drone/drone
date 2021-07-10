// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
)

var (
	tlsVersion = map[string]uint16{
		"TLS10": tls.VersionTLS10,
		"TLS11": tls.VersionTLS11,
		"TLS12": tls.VersionTLS12,
		"TLS13": tls.VersionTLS13,
	}

	tlsCurve = map[string]tls.CurveID{
		"CurveP256": tls.CurveP256,
		"CurveP384": tls.CurveP384,
		"CurveP521": tls.CurveP521,
		"X25519":    tls.X25519,
	}

	tlsCipher = map[string]uint16{
		"RSA_WITH_RC4_128_SHA":                tls.TLS_RSA_WITH_RC4_128_SHA,
		"RSA_WITH_3DES_EDE_CBC_SHA":           tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		"RSA_WITH_AES_128_CBC_SHA":            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"RSA_WITH_AES_256_CBC_SHA":            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"RSA_WITH_AES_128_CBC_SHA256":         tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		"RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"ECDHE_ECDSA_WITH_RC4_128_SHA":        tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		"ECDHE_ECDSA_WITH_AES_128_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"ECDHE_ECDSA_WITH_AES_256_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"ECDHE_RSA_WITH_RC4_128_SHA":          tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		"ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":     tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		"ECDHE_RSA_WITH_AES_128_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"ECDHE_RSA_WITH_AES_256_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"ECDHE_ECDSA_WITH_AES_128_CBC_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		"ECDHE_RSA_WITH_AES_128_CBC_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		"ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"ECDHE_RSA_WITH_CHACHA20_POLY1305":    tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		"ECDHE_ECDSA_WITH_CHACHA20_POLY1305":  tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		"AES_128_GCM_SHA256":                  tls.TLS_AES_128_GCM_SHA256,
		"AES_256_GCM_SHA384":                  tls.TLS_AES_256_GCM_SHA384,
		"CHACHA20_POLY1305_SHA256":            tls.TLS_CHACHA20_POLY1305_SHA256,
		"FALLBACK_SCSV":                       tls.TLS_FALLBACK_SCSV,
	}
)

// A Server defines parameters for running an HTTP server.
type Server struct {
	Acme            bool
	Email           string
	Addr            string
	Cert            string
	Key             string
	Host            string
	TLSMinVersion   string
	TLSMaxVersion   string
	TLSCurves       []string
	TLSCipherSuites []string
	Handler         http.Handler
}

func (s Server) buildTLSConfig() (*tls.Config, error) {
	cfg := &tls.Config{}
	cfg.PreferServerCipherSuites = true
	cfg.NextProtos = []string{"h2", "http/1.1"}
	if s.TLSMinVersion != "" {
		if version, found := tlsVersion[s.TLSMinVersion]; found {
			cfg.MinVersion = version
		} else {
			return nil, fmt.Errorf("Unknown TLS min version: %s", s.TLSMinVersion)
		}
	}
	if s.TLSMaxVersion != "" {
		if version, found := tlsVersion[s.TLSMaxVersion]; found {
			cfg.MaxVersion = version
		} else {
			return nil, fmt.Errorf("Unknown TLS max version: %s", s.TLSMaxVersion)
		}
	}
	for _, curve := range s.TLSCurves {
		if curve != "" {
			if curveId, found := tlsCurve[curve]; found {
				cfg.CurvePreferences = append(cfg.CurvePreferences, curveId)
			} else {
				return nil, fmt.Errorf("Unknown TLS curve: %s", curve)
			}
		}
	}
	for _, cipher := range s.TLSCipherSuites {
		if cipher != "" {
			if cipherId, found := tlsCipher[cipher]; found {
				cfg.CipherSuites = append(cfg.CipherSuites, cipherId)
			} else {
				return nil, fmt.Errorf("Unknown TLS cipher: %s", cipher)
			}
		}
	}
	return cfg, nil
}

const timeoutGracefulShutdown = 5 * time.Second

// ListenAndServe initializes a server to respond to HTTP network requests.
func (s Server) ListenAndServe(ctx context.Context) error {
	if s.Acme {
		return s.listenAndServeAcme(ctx)
	} else if s.Key != "" {
		return s.listenAndServeTLS(ctx)
	}
	err := s.listenAndServe(ctx)
	if err == http.ErrServerClosed {
		err = nil
	}
	return err
}

func (s Server) listenAndServe(ctx context.Context) error {
	var g errgroup.Group
	s1 := &http.Server{
		Addr:    s.Addr,
		Handler: s.Handler,
	}
	g.Go(func() error {
		<-ctx.Done()

		ctxShutdown, cancelFunc := context.WithTimeout(context.Background(), timeoutGracefulShutdown)
		defer cancelFunc()

		return s1.Shutdown(ctxShutdown)
	})
	g.Go(s1.ListenAndServe)
	return g.Wait()
}

func (s Server) listenAndServeTLS(ctx context.Context) error {
	var g errgroup.Group

	cfg, err := s.buildTLSConfig()
	if err != nil {
		return err
	}
	s1 := &http.Server{
		Addr:    ":http",
		Handler: http.HandlerFunc(redirect),
	}
	s2 := &http.Server{
		Addr:      ":https",
		Handler:   s.Handler,
		TLSConfig: cfg,
	}
	g.Go(s1.ListenAndServe)
	g.Go(func() error {
		return s2.ListenAndServeTLS(
			s.Cert,
			s.Key,
		)
	})
	g.Go(func() error {
		<-ctx.Done()

		var gShutdown errgroup.Group
		ctxShutdown, cancelFunc := context.WithTimeout(context.Background(), timeoutGracefulShutdown)
		defer cancelFunc()

		gShutdown.Go(func() error {
			return s1.Shutdown(ctxShutdown)
		})
		gShutdown.Go(func() error {
			return s2.Shutdown(ctxShutdown)
		})

		return gShutdown.Wait()
	})
	return g.Wait()
}

func (s Server) listenAndServeAcme(ctx context.Context) error {
	var g errgroup.Group

	cfg, err := s.buildTLSConfig()
	if err != nil {
		return err
	}
	c := cacheDir()
	m := &autocert.Manager{
		Email:      s.Email,
		Cache:      autocert.DirCache(c),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.Host),
	}
	s1 := &http.Server{
		Addr:    ":http",
		Handler: m.HTTPHandler(s.Handler),
	}
	cfg.GetCertificate = m.GetCertificate
	s2 := &http.Server{
		Addr:      ":https",
		Handler:   s.Handler,
		TLSConfig: cfg,
	}
	g.Go(s1.ListenAndServe)
	g.Go(func() error {
		return s2.ListenAndServeTLS("", "")
	})
	g.Go(func() error {
		<-ctx.Done()

		var gShutdown errgroup.Group
		ctxShutdown, cancelFunc := context.WithTimeout(context.Background(), timeoutGracefulShutdown)
		defer cancelFunc()

		gShutdown.Go(func() error {
			return s1.Shutdown(ctxShutdown)
		})
		gShutdown.Go(func() error {
			return s2.Shutdown(ctxShutdown)
		})

		return gShutdown.Wait()
	})
	return g.Wait()
}

func redirect(w http.ResponseWriter, req *http.Request) {
	target := "https://" + req.Host + req.URL.Path
	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
}

func cacheDir() string {
	const base = "golang-autocert"
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, base)
	}
	return filepath.Join(os.Getenv("HOME"), ".cache", base)
}
