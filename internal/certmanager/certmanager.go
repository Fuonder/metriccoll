package certmanager

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"go.uber.org/zap"
	"log"
	"os"
)

type CertManager struct {
	cert *rsa.PublicKey
	key  *rsa.PrivateKey
}

func NewCertManager() (*CertManager, error) {
	manager := &CertManager{}
	logger.Log.Info("Basic Certificate manager loaded")
	return manager, nil
}

func (m *CertManager) LoadCertificate(certFilepath string) error {
	logger.Log.Info("Loading certificate", zap.String("file", certFilepath))
	certPEM, err := os.ReadFile(certFilepath)
	if err != nil {
		logger.Log.Warn("failed to read server certificate: %v", zap.Error(err))
		return err
	}
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		logger.Log.Warn("failed to decode PEM block containing certificate")
		return fmt.Errorf("failed to decode PEM block containing certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		logger.Log.Warn("failed to parse certificate: %v", zap.Error(err))
		return err
	}

	pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		logger.Log.Warn("certificate does not contain an RSA public key")
		return fmt.Errorf("certificate does not contain an RSA public key")
	}
	logger.Log.Info("Loaded certificate", zap.String("file", certFilepath))
	m.cert = pubKey

	return nil
}

func (m *CertManager) LoadPrivateKey(keyFilepath string) error {
	logger.Log.Info("Loading private key", zap.String("file", keyFilepath))
	keyData, err := os.ReadFile(keyFilepath)
	if err != nil {
		log.Fatalf("failed to read private key: %v", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		log.Fatal("failed to decode PEM block containing private key")
	}

	m.key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("failed to parse private key: %v", err)
	}
	logger.Log.Info("Loaded private key", zap.String("file", keyFilepath))
	return nil
}

func (m *CertManager) Cipher(plaintext []byte) (ciphertext []byte, err error) {
	if m.cert == nil {
		logger.Log.Warn("encryption certificate not loaded")
		return []byte{}, err
	}
	logger.Log.Info("Signing buffer with RSA public key")
	ciphertext, err = rsa.EncryptPKCS1v15(rand.Reader, m.cert, plaintext)
	if err != nil {
		logger.Log.Warn("encryption failed: %v", zap.Error(err))
		return []byte{}, err
	}
	logger.Log.Info("Buffer signed successfully")
	return ciphertext, nil
}

func (m *CertManager) Decrypt(ciphertext []byte) (plaintext []byte, err error) {
	if m.key == nil {
		logger.Log.Warn("encryption private key not loaded")
		return []byte{}, err
	}
	logger.Log.Info("Decrypting buffer with RSA private key")
	logger.Log.Debug("ciphertext", zap.Any("ciphertext", ciphertext))
	plaintext, err = rsa.DecryptPKCS1v15(nil, m.key, ciphertext)
	if err != nil {
		logger.Log.Warn("decryption failed: %v", zap.Error(err))
		return []byte{}, fmt.Errorf("decryption failed: %v", err)
	}
	logger.Log.Info("Buffer decrypted successfully")
	return plaintext, nil
}
