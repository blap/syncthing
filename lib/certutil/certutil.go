// Copyright (C) 2025 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package certutil provides certificate utility functions
package certutil

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/syncthing/syncthing/lib/rand"
)

// ParseCertificate parses a certificate from its DER-encoded bytes
func ParseCertificate(der []byte) (*x509.Certificate, error) {
	return x509.ParseCertificate(der)
}

// generateCertificate generates a PEM formatted key pair and self-signed
// certificate in memory. The compatible flag indicates whether we aim for
// compatibility (browsers) or maximum efficiency/security (sync
// connections).
func generateCertificate(commonName string, lifetimeDays int, compatible bool) (*pem.Block, *pem.Block, error) {
	var pub, priv any
	var err error
	var sigAlgo x509.SignatureAlgorithm
	if compatible {
		// For browser connections we prefer ECDSA-P256
		sigAlgo = x509.ECDSAWithSHA256
		var pk *ecdsa.PrivateKey
		pk, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err == nil {
			priv = pk
			pub = pk.Public()
		}
	} else {
		// For sync connections we use Ed25519
		sigAlgo = x509.PureEd25519
		pub, priv, err = ed25519.GenerateKey(rand.Reader)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}

	notBefore := time.Now().Truncate(24 * time.Hour)
	notAfter := notBefore.Add(time.Duration(lifetimeDays*24) * time.Hour)

	// NOTE: update lib/api.shouldRegenerateCertificate() appropriately if
	// you add or change attributes in here, especially DNSNames or
	// IPAddresses.
	template := x509.Certificate{
		SerialNumber: new(big.Int).SetUint64(rand.Uint64()),
		Subject: pkix.Name{
			CommonName:         commonName,
			Organization:       []string{"Syncthing"},
			OrganizationalUnit: []string{"Automatically Generated"},
		},
		DNSNames:              []string{commonName},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		SignatureAlgorithm:    sigAlgo,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, pub, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("create cert: %w", err)
	}

	certBlock := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}
	keyBlock, err := pemBlockForKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("save key: %w", err)
	}

	return certBlock, keyBlock, nil
}

// NewCertificate generates and returns a new TLS certificate, saved to the
// given PEM files. The compatible flag indicates whether we aim for
// compatibility (browsers) or maximum efficiency/security (sync
// connections).
func NewCertificate(certFile, keyFile string, commonName string, lifetimeDays int, compatible bool) (tls.Certificate, error) {
	certBlock, keyBlock, err := generateCertificate(commonName, lifetimeDays, compatible)
	if err != nil {
		return tls.Certificate{}, err
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("save cert: %w", err)
	}
	if err = pem.Encode(certOut, certBlock); err != nil {
		return tls.Certificate{}, fmt.Errorf("save cert: %w", err)
	}
	if err = certOut.Close(); err != nil {
		return tls.Certificate{}, fmt.Errorf("save cert: %w", err)
	}

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("save key: %w", err)
	}
	if err = pem.Encode(keyOut, keyBlock); err != nil {
		return tls.Certificate{}, fmt.Errorf("save key: %w", err)
	}
	if err = keyOut.Close(); err != nil {
		return tls.Certificate{}, fmt.Errorf("save key: %w", err)
	}

	return tls.X509KeyPair(pem.EncodeToMemory(certBlock), pem.EncodeToMemory(keyBlock))
}

// NewCertificateInMemory generates and returns a new TLS certificate, kept
// only in memory.
func NewCertificateInMemory(commonName string, lifetimeDays int) (tls.Certificate, error) {
	certBlock, keyBlock, err := generateCertificate(commonName, lifetimeDays, false)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(pem.EncodeToMemory(certBlock), pem.EncodeToMemory(keyBlock))
}

func pemBlockForKey(priv interface{}) (*pem.Block, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, err
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
	case ed25519.PrivateKey:
		bs, err := x509.MarshalPKCS8PrivateKey(k)
		if err != nil {
			return nil, err
		}
		return &pem.Block{Type: "PRIVATE KEY", Bytes: bs}, nil
	default:
		return nil, fmt.Errorf("unknown key type: %T", priv)
	}
}
