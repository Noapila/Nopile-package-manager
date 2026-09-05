package main

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

func md5File(path string) (string, error) {
	// Ouvrir le fichier
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Créer un calculateur md5
	h := md5.New()

	// Lire le fichier et calculer le md5
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	// Convertir en string hexadécimale
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
