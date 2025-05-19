package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

const salt = "fuonder-salt-1"

func genSerialNumber() int64 {
	timestamp := time.Now().Unix()
	saltValue := saltToInt64(salt)
	id := timestamp + saltValue
	return id
}

func saltToInt64(s string) int64 {
	var result int64 = 0
	for i, ch := range s {
		shifted := int64(ch) << (i % 8)
		result += shifted
	}
	return result
}

func writeCertificates(certPEM bytes.Buffer, keyPEM bytes.Buffer) {
	err := os.WriteFile("../../certs/server.crt", certPEM.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println("server.crt generated")

	err = os.WriteFile("../../certs/server.key", keyPEM.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println("server.key generated")
}

func main() {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(genSerialNumber()),
		Subject: pkix.Name{
			Organization: []string{"Student.Yandex.Praktikum"},
			Country:      []string{"RU"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 0, 1),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Fatal(err)
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Fatal(err)
	}

	var certPEM bytes.Buffer
	pem.Encode(&certPEM, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	var privateKeyPEM bytes.Buffer
	pem.Encode(&privateKeyPEM, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	writeCertificates(certPEM, privateKeyPEM)
}
