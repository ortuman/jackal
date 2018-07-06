/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

const localhostDomain = "localhost"

const (
	selfSignedCertKeyRSABits = 2048
	selfSignedCertFolder     = "./.cert/"
	selfSignedCertPrivateKey = selfSignedCertFolder + "key.pem"
	selfSignedCertFile       = selfSignedCertFolder + "cert.pem"
)

// LoadCertificate loads a certificate given a private key and certificate PEM files.
// Only in case the associated domain is localhost and no files are specified
// a self signed certificate will be automatically generated.
func LoadCertificate(keyFile, certFile, domain string) (tls.Certificate, error) {
	if len(certFile) == 0 || len(keyFile) == 0 {
		switch domain {
		case localhostDomain:
			if !selfSignedCertificateExists() {
				err := generateSelfSignedCertificate(selfSignedCertPrivateKey, selfSignedCertFile, domain)
				if err != nil {
					return tls.Certificate{}, err
				}
			}
			keyFile = selfSignedCertPrivateKey
			certFile = selfSignedCertFile

		default:
			return tls.Certificate{}, fmt.Errorf("must specify a private key and a server certificate for the domain '%s'", domain)
		}
	}
	cer, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return tls.Certificate{}, err
	}
	return cer, nil
}

func generateSelfSignedCertificate(keyFile, certFile, domain string) error {
	if err := os.MkdirAll(selfSignedCertFolder, os.ModePerm); err != nil {
		return err
	}
	// generate template
	notBefore := time.Now()
	notAfter := notBefore.Add(1825 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return err
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{domain},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain},
	}
	// obtain private key
	priv, err := rsa.GenerateKey(rand.Reader, selfSignedCertKeyRSABits)
	if err != nil {
		return err
	}
	// create and encode certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}
	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	// encode private key
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer keyOut.Close()
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return nil
}

func selfSignedCertificateExists() bool {
	keySt, _ := os.Stat(selfSignedCertPrivateKey)
	certSt, _ := os.Stat(selfSignedCertPrivateKey)
	return keySt != nil && certSt != nil
}
