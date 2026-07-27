package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Exécute une commande shell et affiche la sortie en temps réel
func runCmd(cmd string, dir string) bool {
	parts := strings.Fields(cmd)
	c := exec.Command(parts[0], parts[1:]...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	return err == nil
}

func runNopbuild(path string, dir string) bool {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("fatal: install.nopbuild not found")
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Ignorer commentaires et sections
		if line == "" || line[0] == '#' {
			continue
		}
		if line[0] == '[' {
			continue
		}

		// Exécuter la commande
		fmt.Println("→", line)
		if !runCmd(line, dir) {
			fmt.Println("fatal: command failed:", line)
			return false
		}
	}
	return true
}

func dbFind(name string) *database {
	pkgs := dbLoad()
	for _, pkg := range pkgs {
		if pkg.Nom == name {
			return &pkg
		}
	}
	return nil
}

func installBinary(name string) bool {
	// 1. Chercher dans le db
	pkg := dbFind(name)
	if pkg == nil {
		fmt.Println("error: package not found:", name)
		return false
	}
	if pkg.Binary == "" {
		fmt.Println("error: no binary available for:", name)
		fmt.Println("try: nopile install", name, "--compile")
		return false
	}

	fmt.Println("installing:", name, pkg.Version)

	// 2. Créer le dossier temporaire
	tmpDir  := "/var/tmp/nopile/" + name
	os.MkdirAll(tmpDir, 0755)
	os.MkdirAll(NOPILE_CACHE, 0755)

	// 3. Télécharger le tarball
	// Garder le vrai nom du fichier depuis l'URL
	parts := strings.Split(pkg.Binary, "/")
	filename := parts[len(parts)-1]
	tarball := NOPILE_CACHE + "/" + filename
	os.MkdirAll(NOPILE_PKG_DIR, 0755)

	fmt.Println("downloading", filename)
	if !netDownload(pkg.Binary, tarball) {
		return false
	}

	// 4. Extraire dans tmpDir
	fmt.Println("extracting...")
	if !runCmd("tar -xf "+tarball+" -C /var/tmp/nopile/", "/") {
		fmt.Println("fatal: extraction failed")
		return false
	}

	// 5. Exécuter le install.nopbuild
	nopbuild := tmpDir + "/install.nopbuild"
	fmt.Println("executing install instructions...")
	if !runNopbuild(nopbuild, tmpDir) {
		return false
	}

	// 6. Copier le .nopile dans /var/lib/nopile/packages/
	nopileFile := tmpDir + "/" + name + ".nopile"
	data, err := os.ReadFile(nopileFile)
	if err != nil {
		fmt.Println("fatal: .nopile not found in package")
		return false
	}
	err = os.WriteFile(NOPILE_PKG_DIR+"/"+name+".nopile", data, 0644)
	if err != nil {
		fmt.Println("fatal: failed to install .nopile")
		return false
	}
	fmt.Println(name, pkg.Version, "installed successfully")
	return true
}
