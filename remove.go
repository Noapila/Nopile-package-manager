package main

import (
	"fmt"
	"os"
	"strings"
	"bufio"
)

func isInstalled(name string) bool {
	_, err := os.Stat(NOPILE_PKG_DIR + "/" + name + ".nopile")
	return err == nil
}

func removePackage(name string, config bool) bool {
	if !isInstalled(name) {
		fmt.Println(ColorRed + "error:", ColorReset + "package not installed:", name)
		return false
	}
	nopilePath := NOPILE_PKG_DIR + "/" + name + ".nopile"
	data, err := os.ReadFile(nopilePath)
	if err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + "cannot read .nopile")
		return false
	}

	var modifiedFiles []string
	var installedDirs  []string
	currentFile := ""
	inDirs    := false
	inConfig  := false

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		if line == "[DIRS]" {
			inDirs   = true
			inConfig = false
			continue
		}
		if line == "[CONFIG]" {
			inConfig = true
			inDirs   = false
			continue
		}
		if line[0] == '[' {
			inDirs      = false
			inConfig    = false
			currentFile = ""
			continue
		}
		if inDirs {
			installedDirs = append(installedDirs, line)
			continue
		}
		if strings.HasPrefix(line, "md5sum ") {
			if currentFile == "" { continue }
			expectedMd5 := strings.TrimPrefix(line, "md5sum ")
			actualMd5, err := md5File(currentFile)
			if err != nil {
				fmt.Println(ColorYellow + "warning:", ColorReset + "cannot check:", currentFile)
				currentFile = ""
				continue
			}
			if inConfig {
				if config {
					os.Remove(currentFile)
					vlog(ColorGray + "removed config:", currentFile)
				} else if actualMd5 != expectedMd5 {
					fmt.Println(ColorBlue + "info:", ColorReset + "skipping modified config:", currentFile)
				} else {
					fmt.Println(ColorBlue + "info:", ColorReset + "skipping config:", currentFile)
				}
			} else {
				if actualMd5 != expectedMd5 {
					modifiedFiles = append(modifiedFiles, currentFile)
				} else {
					os.Remove(currentFile)
					vlog(ColorGray + "removed:", currentFile)
				}
			}
			currentFile = ""
		} else if line[0] == '/' {
			currentFile = line
		}
	} // ← fermeture de la boucle for

	// ── Étape 2 : supprimer les dossiers en ordre inverse ─────────────────
	for i := len(installedDirs) - 1; i >= 0; i-- {
		dir := installedDirs[i]
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		if len(entries) == 0 {
			os.Remove(dir)
			vlog(ColorGray + "removed dir:", dir)
		} else {
			fmt.Println(ColorYellow + "warning:", ColorReset + "keeping non-empty dir:", dir)
		}
	}

	// ── Étape 3 : fichiers modifiés ───────────────────────────────────────
	if len(modifiedFiles) > 0 {
		fmt.Println("\nmodified files detected:")
		for _, f := range modifiedFiles {
			fmt.Println(" ", f)
		}
		fmt.Print("\nremove modified files? [y/N] ")
		if !noassume && !yesassume {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer == "y" || answer == "Y" {
				for _, f := range modifiedFiles {
					os.Remove(f)
					vlog("removed:", f)
				}
			}
		}
		if yesassume {
			fmt.Println("y")
			for _, f := range modifiedFiles {
				os.Remove(f)
				vlog("removed:", f)
			}
		}
	}

	// ── Étape 4 : supprimer le .nopile ────────────────────────────────────
	os.Remove(nopilePath)
	vlog(name, "removed successfully")
	return true
}
