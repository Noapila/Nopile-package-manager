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
const VERSION        = "1.0.0"

type database struct {
	Nom       string
	Version   string
	Binary    string
	BinaryMd5 string
	Compile   string
	Depends   []string
}

func dbLoad() []database {
	var paquets []database

	f, err := os.Open(NOPILE_DB)
	if err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + "database not found, run: nopile --sync")
		return nil
	}
	defer f.Close()

	var pkg database
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ligne := scanner.Text()

		if ligne == "" || ligne[0] == '#' {
			continue
		}
		if ligne[0] == '[' {
			if pkg.Nom != "" {
				paquets = append(paquets, pkg)
			}
			pkg = database{}
			continue
		}

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
		if cle == "compile"       { pkg.Compile   = val }
		if cle == "depends"       { pkg.Depends   = strings.Fields(val) }
	}

	if pkg.Nom != "" {
		paquets = append(paquets, pkg)
	}
	return paquets
}

func dbSync() {
	fmt.Println("syncing database...")
	ok := netDownload(NOPILE_REPO+"/sync/nopile.db", NOPILE_DB)
	if ok {
		fmt.Println("database synced")
	} else {
		fmt.Println(ColorRed + "fatal:", ColorReset + "failed to sync database")
	}
}
