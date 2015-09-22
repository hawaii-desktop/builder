/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2015 Pier Luigi Fiorini
 *
 * Author(s):
 *    Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * $BEGIN_LICENSE:AGPL3+$
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * $END_LICENSE$
 ***************************************************************************/

/*
 * Copyright 2009 The Go Authors. All rights reserved.
 * Copyright 2014 The Gogs Authors. All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are
 * met:
 *
 *    * Redistributions of source code must retain the above copyright
 * notice, this list of conditions and the following disclaimer.
 *    * Redistributions in binary form must reproduce the above
 * copyright notice, this list of conditions and the following disclaimer
 * in the documentation and/or other materials provided with the
 * distribution.
 *    * Neither the name of Google Inc. nor the names of its
 * contributors may be used to endorse or promote products derived from
 * this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 * "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
 * A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 * OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 * SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 * LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"github.com/codegangsta/cli"
	"github.com/hawaii-desktop/builder/src/logging"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

var CmdCert = cli.Command{
	Name:  "cert",
	Usage: "Generate self-signed certificate",
	Description: `Generate a self-signed X.509 certificate for master or slaves.
Outputs 'cacert.pem' and 'cakey.pem' for Certification Authority,
otherwise 'cert.pem' and 'key.pem'.
Generate a Certification Authority for master and use that to sign certificate for
slaves.`,
	Before: validateArgs,
	Action: runCert,
	Flags: []cli.Flag{
		cli.StringFlag{"host", "", "comma-separated list of host names and IPs to generate the certificate for", ""},
		cli.StringFlag{"ecdsa", "", "ECDSA curve to use to generate a key. Valid values are P224, P256, P384, P521", ""},
		cli.IntFlag{"rsa-bits", 2048, "RSA key size. Ignored if --ecdsa is passed", ""},
		cli.StringFlag{"start-date", "", "start date formatted as 2015-09-22 18:35:23", ""},
		cli.DurationFlag{"duration", 365 * 24 * time.Hour, "how long the certificate will last before expiring", ""},
		cli.BoolFlag{"ca", "whether this certificate should be its own Certification Authority", ""},
	},
}

func validateArgs(ctx *cli.Context) error {
	if ctx.String("host") == "" {
		return errors.New("Missing host name, please specify a host name with the --host argument")
	}
	return nil
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			logging.Fatalf("Unable to marshal ECDSA private key: %v\n", err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func runCert(ctx *cli.Context) {
	// File names
	certPemFileName := "cert.pem"
	keyPemFileName := "key.pem"
	certDerFileName := "cert.der"

	// Private key
	var priv interface{}
	var err error
	switch ctx.String("ecdsa") {
	case "":
		priv, err = rsa.GenerateKey(rand.Reader, ctx.Int("rsa-bits"))
	case "P256":
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "P384":
		priv, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "P521":
		priv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		logging.Fatalf("Unrecognized elliptic curve: %q\n", ctx.String("ecdsa"))
	}
	if err != nil {
		logging.Fatalf("Failed to generate private key: %s\n", err)
	}

	// Determine start date
	var notBefore time.Time
	if ctx.String("start-date") == "" {
		notBefore = time.Now()
	} else {
		notBefore, err = time.Parse("2015-09-22 18:35:23", ctx.String("start-date"))
		if err != nil {
			logging.Fatalf("Failed to parse start date: %s\n", err)
		}
	}

	// Determine end date
	notAfter := notBefore.Add(ctx.Duration("duration"))

	// Generate serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		logging.Fatalf("Failed to generate serial number: %s\n", err)
	}

	// Determine the extended set of actions
	var extKeyUsage []x509.ExtKeyUsage
	if ctx.Bool("ca") {
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else {
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	// Certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"Hawaii"},
			OrganizationalUnit: []string{"Builder"},
			CommonName:         "Builder",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: true,
	}

	// Add hosts to the template
	hosts := strings.Split(ctx.String("host"), ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	// CA
	var parent *x509.Certificate
	if ctx.Bool("ca") {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
		certPemFileName = "cacert.pem"
		keyPemFileName = "cakey.pem"
		certDerFileName = "cacert.der"
		parent = &template
	} else {
		caCertBytes, err := ioutil.ReadFile("cacert.der")
		certs, err := x509.ParseCertificates(caCertBytes)
		if err != nil {
			logging.Fatalf("Could not load CA certificate: %s\n", err)
		}
		parent = certs[0]
	}

	// Create certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, parent, publicKey(priv), priv)
	if err != nil {
		logging.Fatalf("Failed to create certificate: %s\n", err)
	}
	derOut, err := os.Create(certDerFileName)
	if err != nil {
		logging.Fatalf("Failed to open %s for writing: %s\n", certDerFileName, err)
	}
	derOut.Write(derBytes)
	derOut.Close()
	logging.Infoln("Written", derOut.Name())

	// Save cert.pem
	certOut, err := os.Create(certPemFileName)
	if err != nil {
		logging.Fatalf("Failed to open %s for writing: %s\n", certPemFileName, err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()
	logging.Infoln("Written", certOut.Name())

	// Save key.pem
	keyOut, err := os.OpenFile(keyPemFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		logging.Fatalf("Failed to open %s for writing: %v", keyPemFileName, err)
	}
	pem.Encode(keyOut, pemBlockForKey(priv))
	keyOut.Close()
	logging.Infoln("Written", keyOut.Name())
}
