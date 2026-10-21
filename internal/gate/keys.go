package gate

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
)

func validatePublicKey(path string, private *rsa.PrivateKey) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		data = pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&private.PublicKey)})
		return createKeyFile(path, data, 0644)
	}
	if err != nil {
		return err
	}
	block, trailing := pem.Decode(data)
	if block == nil || block.Type != "RSA PUBLIC KEY" || len(bytes.TrimSpace(trailing)) != 0 {
		return errors.New("invalid server public key")
	}
	public, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return err
	}
	if public.E != private.E || public.N.Cmp(private.N) != 0 {
		return errors.New("server public key does not match private key")
	}
	return nil
}

func createKeyFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}
