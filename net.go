package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

// Télécharge une URL vers un fichier local
func netDownload(url string, dest string) bool {
	// Étape 1 — découper l'URL
	host, path := parseURL(url)

	vlog("connecting to", host)

	// Étape 2 — connexion TCP
	conn, err := tcpConnect(host)
	if err != nil {
		return false
	}

	// Étape 3 — handshake TLS
	tlsConn, err := tlsHandshake(conn, host)
	if err != nil {
		return false
	}
	defer tlsConn.Close()

	// Étape 4 — requête HTTP GET
	err = httpGet(tlsConn, host, path)
	if err != nil {
		return false
	}

	vlog("downloading", dest)

	// Étape 5 — lire la réponse et écrire le fichier
	err = readResponse(tlsConn, dest)
	if err != nil {
		return false
	}

	vlog("done:", dest)
	return true
}

func parseURL(url string) (string, string) {
	// Enlever "https://"
	rest := strings.TrimPrefix(url, "https://")

	// Trouver le premier "/" après le host
	slash := strings.Index(rest, "/")
	if slash == -1 {
		return rest, "/"
	}

	host := rest[:slash]
	path := rest[slash:]
	return host, path
}

func tcpConnect(host string) (net.Conn, error) {
	conn, err := net.Dial("tcp", host+":443")
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "no route to host") {
			fmt.Println(ColorRed + "error:", ColorReset + "no route to host", host)
			fmt.Println("check your internet connection")
		} else if strings.Contains(errMsg, "connection refused") {
			fmt.Println(ColorRed + "fatal:", ColorReset + "connection refused by", host)
		} else if strings.Contains(errMsg, "no such host") {
			fmt.Println(ColorRed + "fatal:", ColorReset + "host not found:", host)
		} else {
			fmt.Println(ColorRed + "error:", ColorReset + "cannot connect to", host)
			fmt.Println("check your internet connection")
		}
		return nil, err
	}
	return conn, nil
}

func tlsHandshake(conn net.Conn, host string) (*tls.Conn, error) {
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName: host,
	})

	err := tlsConn.Handshake()
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "certificate") {
			fmt.Println(ColorRed + "fatal:", ColorReset + "invalid SSL certificate for", host)
			fmt.Println("the server certificate may be expired or untrusted")
		} else if strings.Contains(errMsg, "handshake failure") {
			fmt.Println(ColorRed + "fatal:", ColorReset + "TLS handshake failure with", host)
			fmt.Println("the server may not support TLS")
		} else if strings.Contains(errMsg, "connection reset") {
			fmt.Println(ColorRed + "fatal:", ColorReset + "connection reset during TLS handshake")
			fmt.Println("check your internet connection")
		} else if strings.Contains(errMsg, "timeout") {
			fmt.Println(ColorRed + "fatal:", ColorReset + "TLS handshake timeout with", host)
			fmt.Println("the server may be down or unreachable")
		} else {
			fmt.Println(ColorRed + "fatal:", ColorReset + "TLS handshake failed with", host)
		}
		conn.Close()
		return nil, err
	}

	return tlsConn, nil
}

// Envoie la requête HTTP GET au serveur
func httpGet(conn *tls.Conn, host string, path string) error {
	request := "GET " + path + " HTTP/1.1\r\n" +
	"Host: " + host + "\r\n" +
	"User-Agent: nopile/" + VERSION + "\r\n" +
	"Connection: close\r\n\r\n"

	_, err := conn.Write([]byte(request))
	if err != nil {
		fmt.Println(ColorRed + "fatal:", ColorReset + "failed to send request to", host)
		return err
	}
	return nil
}

// Lit la réponse du serveur et écrit le contenu dans dest
func readResponse(conn *tls.Conn, dest string) error {
	// Lire toute la réponse
	data, err := io.ReadAll(conn)
	if err != nil {
		fmt.Println(ColorRed + "fatal:", ColorReset + "failed to read response")
		return err
	}

	// Vérifier le code HTTP
	response := string(data)
	if strings.Contains(response, "HTTP/1.1 404") {
		fmt.Println(ColorRed + "fatal:", ColorReset + "file not found on server (404)")
		return fmt.Errorf("404")
	} else if strings.Contains(response, "HTTP/1.1 403") {
		fmt.Println(ColorRed + "fatal:", ColorReset + "access denied (403)")
		return fmt.Errorf("403")
	} else if !strings.Contains(response, "HTTP/1.1 200") {
		fmt.Println(ColorRed + "fatal:", ColorReset + "unexpected server response")
		return fmt.Errorf("unexpected response")
	}

	// Trouver la fin des headers
	separator := "\r\n\r\n"
	index := strings.Index(response, separator)
	if index == -1 {
		fmt.Println(ColorRed + "fatal:", ColorReset + "invalid server response")
		return fmt.Errorf("invalid response")
	}

	// Extraire seulement le contenu
	body := data[index+4:]

	// Écrire dans le fichier
	err = os.WriteFile(dest, body, 0644)
	if err != nil {
		fmt.Println(ColorRed + "fatal:", ColorReset + "cannot write file:", dest)
		return err
	}

	return nil
}

func netDownloadCached(url string, dest string, expectedMd5 string) bool {
	if _, err := os.Stat(dest); err == nil {
		if expectedMd5 != "" {
			actual, err := md5File(dest)
			if err == nil && actual == expectedMd5 {
				return true // déjà en cache et valide
			}
		} else {
			return true // en cache, pas de md5 à vérifier
		}
	}
	return netDownload(url, dest)
}
