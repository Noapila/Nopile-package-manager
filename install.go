package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"runtime"
	"io"
	"sync"
)

type nopbuild struct {
	Name        string
	Version     string
	SourceURL   string
	SourceSha   string
	Depends     []string
	BuildCmds   []string
	InstallCmds []string
	PostCmds    []string
	ConfigDirs  []string
}

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
		fmt.Println(ColorRed + "error:", ColorReset + "install.nopbuild not found")
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		if line[0] == '[' {
			continue
		}
		line = strings.ReplaceAll(line, "$(nproc)", fmt.Sprintf("%d", runtime.NumCPU()))
		if strings.HasPrefix(line, "install ") {
			vlog(ColorGray + "nopbuild:", line + ColorReset)
			parts := strings.Fields(line)
			if len(parts) != 3 {
				fmt.Println(ColorRed + "error:", ColorReset + "invalid install syntax:", line)
				return false
			}
			if !copyFile(filepath.Join(dir, parts[1]), parts[2], 0755) {
				fmt.Println(ColorRed + "error:", ColorReset + "cannot install:", parts[2])
				return false
			}
			continue
		}
		if strings.HasPrefix(line, "config ") {
			vlog(ColorGray + "nopbuild:", line + ColorReset)
			parts := strings.Fields(line)
			if len(parts) != 3 {
				fmt.Println(ColorRed + "error:", ColorReset + "invalid config syntax:", line)
				return false
			}
			if _, err := os.Stat(parts[2]); err == nil {
				fmt.Println(ColorBlue + "info:", ColorReset + "config file already present:", parts[2])
				continue
			}
			if !copyFile(filepath.Join(dir, parts[1]), parts[2], 0644) {
				fmt.Println(ColorRed + "error:", ColorReset + "cannot install config:", parts[2])
				return false
			}
			continue
		}
		vlog(ColorGray + "nopbuild:", line + ColorReset)
		if !runCmd(line, dir) {
			fmt.Println(ColorRed + "error:", ColorReset + "command failed:", line)
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

func installBinary(name string, force bool, reinstall bool) bool {
	pkg := dbFind(name)

	tmpDir   := "/var/tmp/nopile/binary/" + name
	parts    := strings.Split(pkg.Binary, "/")
	filename := parts[len(parts)-1]
	tarball  := NOPILE_CACHE + "/" + filename

	if _, err := os.Stat(tarball); err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + "tarball not found:", filename)
		return false
	}

	// Lire le .nopile
	nopileInTarball := tmpDir + "/" + name + ".nopile"
	data, err := os.ReadFile(nopileInTarball)
	if err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + ".nopile not found in package")
		return false
	}

	// Vérifier les conflits
	conflict := false
	inConfig := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "[CONFIG]" { inConfig = true; continue }
		if len(line) > 0 && line[0] == '[' { inConfig = false; continue }
		if inConfig { continue }
		if len(line) == 0 || line[0] != '/' { continue }
		if _, statErr := os.Stat(line); statErr == nil {
			owner := findFileOwner(line)
			if owner == name {
				continue
			} else if owner != "" {
				if !force {
					fmt.Println(ColorRed + "error:", ColorReset + "conflict with package:", owner, "->", line)
					conflict = true
				} else {
					fmt.Println(ColorYellow + "warning:", ColorReset + "conflict with package:", owner, "->", line "it will be overwrited")
				}
			} else {
				if reinstall { continue }
				if !force {
					fmt.Println(ColorRed + "error:", ColorReset + "file already exists:", line)
					conflict = true
				} else {
					fmt.Println(ColorYellow + "warning:", ColorReset + "conflict with" line, "it will be overwrited")
				}
			}
		}
	}
	if conflict {
		if !force {
			fmt.Println("aborting")
			return false
		}
		fmt.Println("overwriting conflicts files")
	}

	// Exécuter le nopbuild
	nopbuild := tmpDir + "/install.nopbuild"
	vlog(ColorGray + "executing nopbuild" + ColorReset)
	if !runNopbuild(nopbuild, tmpDir) {
		return false
	}

	// Copier le .nopile
	data, err = os.ReadFile(tmpDir + "/" + name + ".nopile")
	if err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + ".nopile not found in package")
		return false
	}
	err = os.WriteFile(NOPILE_PKG_DIR+"/"+name+".nopile", data, 0644)
	if err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + "failed to install .nopile")
		return false
	}

	vlog(name, pkg.Version, "installed successfully")
	return true
}

func readNopbuild(path string) (*nopbuild, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	nb := &nopbuild{}
	section := ""
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		if line[0] == '[' {
			section = line[1 : len(line)-1]
			continue
		}

		switch section {
			case "INFO":
				parts := strings.SplitN(line, ": ", 2)
				if len(parts) != 2 {
					continue
				}
				switch parts[0] {
					case "name":          nb.Name      = parts[1]
					case "version":       nb.Version   = parts[1]
					case "source-url":    nb.SourceURL = parts[1]
					case "source-sha256": nb.SourceSha = parts[1]
					case "depends":
						for _, dep := range strings.Fields(parts[1]) {
							nb.Depends = append(nb.Depends, dep)
						}
				}
					case "BUILD":
						nb.BuildCmds = append(nb.BuildCmds, line)
					case "INSTALL":
						nb.InstallCmds = append(nb.InstallCmds, line)
					case "POST":
						nb.PostCmds = append(nb.PostCmds, line)
					case "CONFIG":
						nb.ConfigDirs = append(nb.ConfigDirs, line)
		}
	}
	return nb, nil
}

func installCompile(name string, force bool, reinstall bool) bool {
	pkg := dbFind(name)
	if pkg == nil {
		fmt.Println(ColorRed + "error:", ColorReset + "package not found:", name)
		return false
	}
	if pkg.Compile == "" {
		fmt.Println(ColorRed + "error:", ColorReset + "no compilation file available for:", name)
		fmt.Println("try: nopile install", name, "--binary")
		return false
	}

	vlog("preparing compilation for:", name)
	srcDir  := "/var/tmp/nopile/sources/" + name
	rootDir := srcDir + "/root"

	nopbuildPath := NOPILE_CACHE + "/" + name + ".nopbuild"
	nb, err := readNopbuild(nopbuildPath)
	if err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + "cannot read nopbuild")
		return false
	}

	os.MkdirAll(rootDir, 0755)

	vlog(ColorGray + "executing nopbuild" + ColorReset)
	skippedConfigure := false

	for _, cmd := range nb.BuildCmds {
		cmd = strings.ReplaceAll(cmd, "$(nproc)", fmt.Sprintf("%d", runtime.NumCPU()))

		if strings.HasPrefix(cmd, "./configure") {
			_, hasMakefile     := os.Stat(filepath.Join(srcDir, "Makefile"))
			_, hasConfigStatus := os.Stat(filepath.Join(srcDir, "config.status"))
			if hasMakefile == nil && hasConfigStatus == nil {
				vlog("nopbuild: skipping configure (already configured)")
				skippedConfigure = true
				continue
			}
			skippedConfigure = false
		}

		vlog(ColorGray + "nopbuild:", cmd + ColorReset)
		if !runCmd(cmd, srcDir) {
			if skippedConfigure && strings.HasPrefix(cmd, "make") {
				fmt.Println(ColorYellow + "warning:", ColorReset + "build failed, re-trying...")
				for _, buildCmd := range nb.BuildCmds {
					if strings.HasPrefix(buildCmd, "./configure") {
						vlog(ColorGray + "nopbuild:", buildCmd + ColorReset)
						if !runCmd(buildCmd, srcDir) {
							fmt.Println(ColorRed + "error:", ColorReset + "configure failed")
							return false
						}
						break
					}
				}
				vlog(ColorGray + "nopbuild:", cmd + ColorReset)
				if !runCmd(cmd, srcDir) {
					fmt.Println(ColorRed + "error:", ColorReset + "build failed:", cmd)
					return false
				}
			} else {
				fmt.Println(ColorRed + "error:", ColorReset + "build failed:", cmd)
				return false
			}
		}
	}

	vlog(ColorGray + "cleaning root/..." + ColorReset)
	os.RemoveAll(rootDir)
	os.MkdirAll(rootDir, 0755)

	vlog(ColorGray + "installing into root/..." + ColorReset)
	for _, cmd := range nb.InstallCmds {
		cmd = strings.ReplaceAll(cmd, "DESTDIR=root", "DESTDIR="+rootDir)
		fmt.Println("nopbuild:", cmd)
		if !runCmd(cmd, srcDir) {
			fmt.Println(ColorRed + "error:", ColorReset + "install failed:", cmd)
			return false
		}
	}

	vlog(ColorGray + "scanning files..." + ColorReset)
	var installedFiles []string
	var installedDirs  []string

	excluded := map[string]bool{
		"/usr/share/info/dir": true,
	}
	filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil { return err }
		realPath := strings.TrimPrefix(path, rootDir)
		if realPath == "" { return nil }

		// Exclure les fichiers système
		if excluded[realPath] {
			return nil
		}
		if d.IsDir() {
			_, statErr := os.Stat(realPath)
			if statErr != nil {
				installedDirs = append(installedDirs, realPath)
			}
		} else {
			hash, err := md5File(path)
			if err != nil {
				fmt.Println(ColorYellow + "warning:", ColorReset + "cannot read md5:", realPath)
				return nil
			}
			installedFiles = append(installedFiles, realPath+"|"+hash)
		}
		return nil
	})

	vlog(ColorGray + "found", len(installedFiles), "files" + ColorReset)
	vlog(ColorGray + "found", len(installedDirs), "directories" + ColorReset)

	// Vérifier les conflits
	conflict := false
	for _, entry := range installedFiles {
		realPath := strings.Split(entry, "|")[0]
		_, err := os.Stat(realPath)
		if err == nil {
			owner := findFileOwner(realPath)
			if owner == name {
				continue
			} else if owner != "" {
				fmt.Println(ColorRed + "error:", ColorReset + "conflict with package:", owner, "->", realPath)
				conflict = true
			} else {
				// Fichier sans propriétaire
				if reinstall {
					// En réinstallation on écrase sans erreur
					continue
				}
				fmt.Println(ColorRed + "error:", ColorReset + "file already exists:", realPath)
				conflict = true
			}
		}
	}
	if conflict {
		if !force {
			fmt.Println("aborting")
			return false
		}
		fmt.Println("forcing installation")
	}

	// 9. Créer le .nopile
	nopilePath := NOPILE_PKG_DIR + "/" + name + ".nopile"
	nf, err := os.Create(nopilePath)
	if err != nil {
		fmt.Println(ColorRed + "fatal:", ColorReset + "cannot create .nopile")
		return false
	}
	defer nf.Close()

	fmt.Fprintf(nf, "version: %s\n", nb.Version)
	fmt.Fprintf(nf, "source:  %s\n\n", nb.SourceURL)
	if len(nb.Depends) > 0 {
		fmt.Fprintf(nf, "depends: %s\n", strings.Join(nb.Depends, " "))
	}
	fmt.Fprintf(nf, "\n")

	for _, entry := range installedFiles {
		parts := strings.Split(entry, "|")
		path  := parts[0]
		hash  := parts[1]
		fname := filepath.Base(path)
		fmt.Fprintf(nf, "[%s]\n%s\nmd5sum %s\n\n", fname, path, hash)
	}

	fmt.Fprintf(nf, "[DIRS]\n")
	for _, d := range installedDirs {
		fmt.Fprintf(nf, "%s\n", d)
	}

	// 10. Installer les fichiers
	vlog(ColorGray + "installing files..." + ColorReset)
	for _, entry := range installedFiles {
		parts    := strings.Split(entry, "|")
		realPath := parts[0]
		src      := rootDir + realPath

		os.MkdirAll(strings.TrimSuffix(realPath, "/"+filepath.Base(realPath)), 0755)

		data, err := os.ReadFile(src)
		if err != nil {
			fmt.Println(ColorRed + "error:", ColorReset + "cannot read:", src)
			return false
		}
		err = os.WriteFile(realPath, data, 0755)
		if err != nil {
			fmt.Println(ColorRed + "error:", ColorReset + "cannot install:", realPath)
			return false
		}
		vlog(ColorGray + "installed:", realPath + ColorReset)
	}
	vlog(name, nb.Version, "installed successfully")
	return true
}

func installLocal(path string, force bool, reinstall bool) bool {
	// Vérifier que le fichier existe
	if _, err := os.Stat(path); err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + "file not found:", path)
		return false
	}

	// Extraire dans tmpDir
	name := strings.TrimSuffix(filepath.Base(path), ".tar.gz")
	name  = strings.TrimSuffix(name, ".tar.bz2")
	name  = strings.TrimSuffix(name, ".tar.zst")
	name  = strings.TrimSuffix(name, ".tar.xz")

	fmt.Println("==> Installing", name, "(local)" )

	tmpDir := "/var/tmp/nopile/binary/" + name
	os.MkdirAll(tmpDir, 0755)

	fmt.Println("[1/2] Extracting", name)
	if !extractTarball(path, "/var/tmp/nopile/binary/", 0) {
		fmt.Println(ColorRed + "error:", ColorReset + "extraction failed")
		return false
	}

	// Lire le .nopile
	nopileInTarball := tmpDir + "/" + name + ".nopile"
	data, err := os.ReadFile(nopileInTarball)
	if err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + ".nopile not found in package")
		return false
	}

	// Extraire le vrai nom depuis le .nopile
	pkgName := name
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name: ") {
			pkgName = strings.TrimPrefix(line, "name: ")
			break
		}
	}
	// Vérifier les conflits
	conflict := false
	inConfig := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "[CONFIG]" { inConfig = true; continue }
		if len(line) > 0 && line[0] == '[' { inConfig = false; continue }
		if inConfig { continue }
		if len(line) == 0 || line[0] != '/' { continue }
		if _, statErr := os.Stat(line); statErr == nil {
			owner := findFileOwner(line)
			if owner == name {
				continue
			} else if owner != "" {
				fmt.Println(ColorRed + "error:", ColorReset + "conflict with package:", owner, "->", line)
				conflict = true
			} else {
				fmt.Println(ColorRed + "error:", ColorReset + "file already exists:", line)
				conflict = true
			}
		}
	}
	if conflict {
		if !force {
			fmt.Println("use --force to override")
			return false
		}
		fmt.Println("forcing installation")
	}

	fmt.Println("[2/2] Installing", name)

	// Exécuter le nopbuild
	nopbuild := tmpDir + "/install.nopbuild"
	vlog("executing nopbuild")
	if !runNopbuild(nopbuild, tmpDir) {
		return false
	}

	// Copier le .nopile
	err = os.WriteFile(NOPILE_PKG_DIR+"/"+name+".nopile", data, 0644)
	if err != nil {
		fmt.Println(ColorRed + "fatal:", ColorReset + "failed to install .nopile")
		return false
	}

	explicitAdd(pkgName)
	vlog(pkgName, "installed successfully")
	return true
}

func findFileOwner(filePath string) string {
	entries, err := os.ReadDir(NOPILE_PKG_DIR)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".nopile") {
			continue
		}

		nopilePath := NOPILE_PKG_DIR + "/" + entry.Name()
		data, err := os.ReadFile(nopilePath)
		if err != nil {
			continue
		}

		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == filePath {
				// Retourner le nom du paquet (sans .nopile)
				return strings.TrimSuffix(entry.Name(), ".nopile")
			}
		}
	}
	return ""
}

func copyFile(src string, dst string, perm os.FileMode) bool {
	srcFile, err := os.Open(src)
	if err != nil {
		return false
	}
	defer srcFile.Close()

	os.MkdirAll(filepath.Dir(dst), 0755)
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return false
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err == nil
}

func downloadAll(toInstall []entry) bool {
	var wg      sync.WaitGroup
	var mu      sync.Mutex
	var counter int
	failed := false
	total  := 0

	for _, e := range toInstall {
		if e.action != "local" {
			total++
		}
	}

	fmt.Printf("==> Downloading packages\n")

	for _, e := range toInstall {
		if e.action == "local" { continue }
		pkg := dbFind(e.name)
		if pkg == nil { continue }
		wg.Add(1)

		go func(e entry, pkg *database) {
			defer wg.Done()
			mu.Lock()
			counter++
			n := counter
			mu.Unlock()
			fmt.Printf("[%d/%d] Downloading %s\n", n, total, e.name)
			if e.action == "binary" {
				parts    := strings.Split(pkg.Binary, "/")
				filename := parts[len(parts)-1]
				tarball  := NOPILE_CACHE + "/" + filename
				if !netDownloadCached(pkg.Binary, tarball, pkg.BinaryMd5) {
					mu.Lock()
					fmt.Println(ColorRed + "fatal:", ColorReset + "failed to download:", e.name)
					failed = true
					mu.Unlock()
				}
			} else if e.action == "compile" {
				nopbuildPath := NOPILE_CACHE + "/" + e.name + ".nopbuild"
				if !netDownloadCached(pkg.Compile, nopbuildPath, "") {
					mu.Lock()
					fmt.Println(ColorRed + "fatal:", ColorReset + "failed to download nopbuild:", e.name)
					failed = true
					mu.Unlock()
					return
				}
				nb, err := readNopbuild(nopbuildPath)
				if err != nil { return }
				parts         := strings.Split(nb.SourceURL, "/")
				sourceTarball := NOPILE_CACHE + "/" + parts[len(parts)-1]
				if !netDownloadCached(nb.SourceURL, sourceTarball, nb.SourceSha) {
					mu.Lock()
					fmt.Println(ColorRed + "fatal:", ColorReset + "failed to download sources:", e.name)
					failed = true
					mu.Unlock()
				}
			}
		}(e, pkg)
	}
	wg.Wait()
	return !failed
}

func extractAll(toInstall []entry) bool {
	var wg      sync.WaitGroup
	var mu      sync.Mutex
	var counter int
	failed := false
	total  := 0

	for _, e := range toInstall {
		if e.action != "local" {
			total++
		}
	}

	fmt.Printf("==> Extracting packages\n")

	for _, e := range toInstall {
		if e.action == "local" { continue }
		wg.Add(1)
		go func(e entry) {
			defer wg.Done()
			pkg := dbFind(e.name)
			if pkg == nil { return }
			mu.Lock()
			counter++
			n := counter
			mu.Unlock()
			fmt.Printf("[%d/%d] Extracting %s\n", n, total, e.name)

			if e.action == "binary" {
				parts   := strings.Split(pkg.Binary, "/")
				tarball := NOPILE_CACHE + "/" + parts[len(parts)-1]
				tmpDir  := "/var/tmp/nopile/binary/" + e.name
				os.MkdirAll(tmpDir, 0755)
				if !extractTarball(tarball, "/var/tmp/nopile/binary/", 0) {
					mu.Lock()
					fmt.Println(ColorRed + "error:", ColorReset + "extraction failed:", e.name)
					failed = true
					mu.Unlock()
				}
			} else if e.action == "compile" {
				nopbuildPath  := NOPILE_CACHE + "/" + e.name + ".nopbuild"
				nb, err       := readNopbuild(nopbuildPath)
				if err != nil { return }
				parts         := strings.Split(nb.SourceURL, "/")
				sourceTarball := NOPILE_CACHE + "/" + parts[len(parts)-1]
				srcDir        := "/var/tmp/nopile/sources/" + e.name
				os.MkdirAll(srcDir, 0755)
				if !extractTarball(sourceTarball, srcDir, 1) {
					mu.Lock()
					fmt.Println(ColorRed + "error:", ColorReset + "extraction failed:", e.name)
					failed = true
					mu.Unlock()
				}
			}
		}(e)
	}
	wg.Wait()
	return !failed
}

func verifyAll(toInstall []entry) bool {
	var wg      sync.WaitGroup
	var mu      sync.Mutex
	var counter int
	failed := false
	total  := 0

	for _, e := range toInstall {
		if e.action != "local" {
			total++
		}
	}

	fmt.Printf("==> Verifying packages\n")

	for _, e := range toInstall {
		if e.action == "local" { continue }
		wg.Add(1)
		go func(e entry) {
			defer wg.Done()
			pkg := dbFind(e.name)
			if pkg == nil { return }
			mu.Lock()
			counter++
			n := counter
			mu.Unlock()
			fmt.Printf("[%d/%d] Verifying %s\n", n, total, e.name)

			if e.action == "binary" && pkg.BinaryMd5 != "" {
				parts   := strings.Split(pkg.Binary, "/")
				tarball := NOPILE_CACHE + "/" + parts[len(parts)-1]
				actual, err := md5File(tarball)
				if err != nil || actual != pkg.BinaryMd5 {
					mu.Lock()
					fmt.Println(ColorRed + "fatal:", ColorReset + "checksum mismatch:", e.name)
					failed = true
					mu.Unlock()
				}
			} else if e.action == "compile" {
				nopbuildPath  := NOPILE_CACHE + "/" + e.name + ".nopbuild"
				nb, err       := readNopbuild(nopbuildPath)
				if err != nil || nb.SourceSha == "" { return }
				parts         := strings.Split(nb.SourceURL, "/")
				sourceTarball := NOPILE_CACHE + "/" + parts[len(parts)-1]
				actual, err   := sha256File(sourceTarball)
				if err != nil || actual != nb.SourceSha {
					mu.Lock()
					fmt.Println(ColorRed + "fatal:", ColorReset + "checksum mismatch:", e.name)
					failed = true
					mu.Unlock()
				}
			}
		}(e)
	}
	wg.Wait()
	return !failed
}

func postAll(toInstall []entry) bool {
	var wg      sync.WaitGroup
	var mu      sync.Mutex
	var counter int
	failed := false

	// Compter seulement les paquets avec des post-transactions
	total := 0
	for _, e := range toInstall {
		if e.action != "compile" { continue }
		nopbuildPath := NOPILE_CACHE + "/" + e.name + ".nopbuild"
		nb, err := readNopbuild(nopbuildPath)
		if err == nil && len(nb.PostCmds) > 0 {
			total++
		}
	}

	for _, e := range toInstall {
		if e.action != "compile" { continue }
		wg.Add(1)
		go func(e entry) {
			defer wg.Done()
			nopbuildPath := NOPILE_CACHE + "/" + e.name + ".nopbuild"
			nb, err      := readNopbuild(nopbuildPath)
			if err != nil || len(nb.PostCmds) == 0 { return }

			mu.Lock()
			counter++
			n := counter
			mu.Unlock()

			fmt.Printf("==> Executing post install\n")
			fmt.Printf("[%d/%d] Executing post-transaction for %s\n", n, total, e.name)
			for _, cmd := range nb.PostCmds {
				cmd = strings.ReplaceAll(cmd, "$(nproc)", fmt.Sprintf("%d", runtime.NumCPU()))
				vlog(ColorGray + "post:", cmd)
				if !runCmd(cmd, "/") {
					fmt.Printf(ColorRed + "error:", ColorReset + "post-transaction failed for %s: %s\n", e.name, cmd)
					mu.Lock()
					failed = true
					mu.Unlock()
				}
			}
		}(e)
	}
	wg.Wait()
	return !failed
}

func cleanAll(toInstall []entry) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var counter int
	total := len(toInstall)
	fmt.Printf("==> Cleaning files\n")

	for _, e := range toInstall {
		wg.Add(1)
		go func(e entry) {
			defer wg.Done()
			mu.Lock()
			counter++
			n := counter
			mu.Unlock()

			// Affiche le nom du paquet (pas le chemin complet)
			displayName := e.name
			if e.action == "local" {
				displayName = filepath.Base(e.name)
			}
			fmt.Printf("[%d/%d] Cleaning %s\n", n, total, displayName)

			if e.action == "binary" {
				os.RemoveAll("/var/tmp/nopile/binary/" + e.name)
			} else if e.action == "compile" {
				os.RemoveAll("/var/tmp/nopile/sources/" + e.name)
			} else if e.action == "local" {
				name := filepath.Base(e.name)
				os.RemoveAll("/var/tmp/nopile/binary/" + name)
			}
		}(e)
	}
	wg.Wait()
}
