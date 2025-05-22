package certmanager

type TLSCipher interface {
	LoadCertificate(certFilepath string) error
	Cipher(plaintext []byte) (ciphertext []byte, err error)
}

type TLSDecipher interface {
	LoadPrivateKey(keyFilepath string) error
	Decrypt(ciphertext []byte) (plaintext []byte, err error)
}
