package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const NOPILE_DB      = "/var/lib/nopile/nopile.db"
const NOPILE_PKG_DIR = "/var/lib/nopile/packages"
const NOPILE_CACHE   = "/var/cache/nopile"
const NOPILE_REPO    = "https://raw.githubusercontent.com/Noapila/Nopile-package-manager-repo/main"
const VERSION        = "0.1.0"

type database struct {
	Nom       string
	Version   string
	Binary    string
	BinaryMd5 string
	Compile   string
}

func dbLoad() []database {
	// 1. Créer une liste vide
	var paquets []database

	// 2. Ouvrir le fichier
	f, err := os.Open(NOPILE_DB)
	if err != nil {
		fmt.Println("error: database not found, run: nopile --sync")
		return nil
	}
	defer f.Close()

	// 3. Lire ligne par ligne
	var pkg database
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ligne := scanner.Text()

		// Ignorer commentaires et lignes vides
		if ligne == "" || ligne[0] == '#' {
			continue
		}

		// Nouvelle section [test] → nouveau paquet
		if ligne[0] == '[' {
			// Si pkg a un nom c'est qu'on vient de finir un paquet → l'ajouter
			if pkg.Nom != "" {
				paquets = append(paquets, pkg)
			}
			// Repartir avec un paquet vide
			pkg = database{}
			continue
		}

		// Remplir le paquet courant
		parts := strings.SplitN(ligne, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		cle := parts[0]
		val := parts[1]

		if cle == "name"          { pkg.Nom       = val }
		if cle == "version"       { pkg.Version   = val }
		if cle == "binary"        { pkg.Binary    = val }
		if cle == "binary-md5sum" { pkg.BinaryMd5 = val }
		if cle == "compile"       { pkg.Compile = val }
	}

	// Ajouter le dernier paquet
	if pkg.Nom != "" {
		paquets = append(paquets, pkg)
	}

	return paquets
}

func dbSync() {
	fmt.Println("syncing database...")
	ok := netDownload(NOPILE_REPO + "/sync/nopile.db", NOPILE_DB)
	if ok {
		fmt.Println("database synced")
	} else {
		fmt.Println("fatal: failed to sync database")
	}
}
