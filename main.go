package main

import (
	"fmt"
	"os"
	"bufio"
	"strings"
	"slices"
	"path/filepath"
)

type entry struct {
	name      string
	action    string
	reinstall bool
}
var verbose bool
var yesassume bool
var noassume bool

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"
)

func main() {

	// Si aucun argument → afficher l'aide
	if len(os.Args) < 2 {
		cmdHelp()
		return
	}
	var filteredArgs []string
	for _, arg := range os.Args[1:] {
		if arg == "--verbose" || arg == "-v" {
			verbose = true
		} else if arg == "--yes" || arg == "-y" {
			yesassume = true
		} else if arg == "--no" || arg == "-n" {
			noassume = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	os.Args = append([]string{os.Args[0]}, filteredArgs...)

	// Lire la commande tapée
	commande := os.Args[1]

	// Choisir quoi faire selon la commande
	if commande == "install" || commande == "-i" {
		cmdInstall()
	} else if commande == "remove" || commande == "-r" {
		cmdRemove()
	} else if commande == "update" || commande == "-u" {
		cmdUpdate()
	} else if commande == "search" || commande == "-S" {
		cmdSearch()
	} else if commande == "list" || commande == "-l" {
		cmdList()
	} else if commande == "info" || commande == "-I" {
		cmdInfo()
	} else if commande == "verify" || commande == "-V" {
		cmdVerify()
	} else if commande == "clean" || commande == "-c" {
		cmdClean()
	} else if commande == "help" || commande == "-h" {
		cmdHelp()
	} else if commande == "--sync" || commande == "-s" {
		if os.Getuid() != 0 {
			fmt.Println(ColorRed + "error:", ColorReset + "nopile --sync must be run as administrator")
			return
		}
		dbSync()
	} else if commande == "who" || commande == "-w" {
		cmdWho()
	} else {
		// Commande inconnue
		fmt.Println(ColorRed + "error:", ColorReset + "unknown command:", commande)
		fmt.Println("use \"nopile help\" for see help")
		os.Exit(1)
	}
}



func cmdInstall() {
	if len(os.Args) < 3 {
		fmt.Println(ColorRed + "error:", ColorReset + "missing package name")
		fmt.Println("usage: nopile install <package> --binary|--compile|--local [--force]")
		os.Exit(1)
	}
	groups, ok, force := parseInstall(os.Args[2:])
	if !ok {
		os.Exit(1)
	}
	if len(groups) == 0 {
		fmt.Println(ColorRed + "error:", ColorReset + "missing package name")
		fmt.Println("usage: nopile install <package> --binary|--compile|--local [--force]")
		os.Exit(1)
	}

	var toInstall []entry
	seen     := map[string]bool{}
	explicit := map[string]bool{}

	// Marquer les paquets explicites
	for _, g := range groups {
		for _, name := range g.packages {
			explicit[name] = true
		}
	}

	for _, g := range groups {
		for _, name := range g.packages {
			if g.action != "local" {
				// Résoudre les dépendances — skip si installé ET pas explicite
				resolveDeps(name, seen, explicit, &toInstall)
			}
			// Toujours ajouter les paquets explicites
			if !seen[name] {
				seen[name] = true
				toInstall = append(toInstall, entry{name, g.action, false})
			} else {
				// Déjà dans toInstall comme dépendance → mettre à jour l'action
				for i := range toInstall {
					if toInstall[i].name == name {
						toInstall[i].action = g.action
						break
					}
				}
			}
		}
	}

	// Vérifier que tous les paquets existent et ont la bonne méthode
	valid := true
	for i := range toInstall {
		// Pour --local on skip dbFind
		if toInstall[i].action == "local" {
			if _, err := os.Stat(toInstall[i].name); err != nil {
				fmt.Println(ColorRed + "error:", ColorReset + "file not found:", toInstall[i].name)
				valid = false
			}
			continue
		}
		pkg := dbFind(toInstall[i].name)
		if pkg == nil {
			fmt.Println(ColorRed + "error:", ColorReset + "package not found:", toInstall[i].name)
			valid = false
			continue
		}
		if toInstall[i].action == "binary" && pkg.Binary == "" {
			fmt.Println(ColorRed + "error:", ColorReset + "no binary available for:", toInstall[i].name)
			fmt.Println("try: nopile install", toInstall[i].name, "--compile")
			valid = false
		}
		if toInstall[i].action == "compile" && pkg.Compile == "" {
			fmt.Println(ColorRed + "error:", ColorReset + "no compile available for:", toInstall[i].name)
			fmt.Println("try: nopile install", toInstall[i].name, "--binary")
			valid = false
		}
		// Fichier .nopile présent = déjà installé
		_, err := os.Stat(NOPILE_PKG_DIR + "/" + toInstall[i].name + ".nopile")
		if err == nil {
			toInstall[i].reinstall = true
		}
	}
	if !valid {
		os.Exit(1)
	}

	// Séparer pour l'affichage
	var toBinary []entry
	var toCompile []entry
	var toLocal []entry
	for _, e := range toInstall {
		if e.action == "compile" {
			toCompile = append(toCompile, e)
		} else if e.action == "local" {
			toLocal = append(toLocal, e)
		} else {
			toBinary = append(toBinary, e)
		}
	}

	fmt.Println()
	if len(toBinary) > 0 {
		fmt.Println("packages to install:", len(toBinary))
		fmt.Println()
		for _, e := range toBinary {
			label := "install  "
			if e.reinstall {
				label = "reinstall"
			}
			if pkg := dbFind(e.name); pkg != nil {
				fmt.Printf("  %s %-20s %s\n", label, e.name, pkg.Version)
			} else {
				fmt.Printf("  %s %-20s (unknown)\n", label, e.name)
			}
		}
		fmt.Println()
	}
	if len(toCompile) > 0 {
		fmt.Println("packages to compile:", len(toCompile))
		fmt.Println()
		for _, e := range toCompile {
			label := "install  "
			if e.reinstall {
				label = "reinstall"
			}
			if pkg := dbFind(e.name); pkg != nil {
				fmt.Printf("  %s %-20s %s\n", label, e.name, pkg.Version)
			} else {
				fmt.Printf("  %s %-20s unknown\n", label, e.name)
			}
		}
		fmt.Println()
	}

	if len(toLocal) > 0 {
		fmt.Println("packages to install (local):", len(toLocal))
		fmt.Println()
		for _, e := range toLocal {
			label := "install  "
			if e.reinstall {
				label = "reinstall"
			}
			fmt.Printf("  %s %-20s\n", label, filepath.Base(e.name))
		}
		fmt.Println()
	}

	fmt.Print("proceed? [y/N] ")

	if !noassume && !yesassume {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("aborted.")
			os.Exit(1)
		}
	}

	if noassume {
		fmt.Println("n")
		fmt.Println("aborted.")
		os.Exit(1)
	} else if yesassume {
		fmt.Println("y")
	}

	if !downloadAll(toInstall) {
		fmt.Println(ColorRed + "error:", ColorReset + "transaction aborted")
		os.Exit(2)
	}

	if !verifyAll(toInstall) {
		fmt.Println(ColorRed + "error:", ColorReset + "transaction aborted")
		os.Exit(2)
	}

	if !extractAll(toInstall) {
		fmt.Println(ColorRed + "error:", ColorReset + "transaction aborted")
		os.Exit(1)
	}

	// Installation séquentielle
	fmt.Println("==> Installing packages")
	total  := 0
	for _, e := range toInstall {
		if e.action != "local" {
			total++
		}
	}
	for i, e := range toInstall {
		var ok bool
		if e.action == "binary" {
			fmt.Printf("[%d/%d] Installing %s\n", i+1, total, e.name)
			ok = installBinary(e.name, force, e.reinstall)
		} else if e.action == "compile" {
			fmt.Printf("[%d/%d] Compiling and installing %s\n", i+1, total, e.name)
			ok = installCompile(e.name, force, e.reinstall)
		} else if e.action == "local" {
			ok = installLocal(e.name, force, e.reinstall)
		}
		if !ok {
			fmt.Println("transaction aborted")
			os.Exit(1)
		}
	}

	if !postAll(toInstall) {
		fmt.Println("warning: some post-transaction failed")
	}
	cleanAll(toInstall)

	// Marquer les paquets explicites (pas les dépendances)
	for _, g := range groups {
		for _, name := range g.packages {
			if g.action != "local" {
				explicitAdd(name)
			}
			// Pour --local c'est installLocal qui appelle explicitAdd
		}
	}
}

func cmdRemove() {
	if len(os.Args) < 3 {
		fmt.Println(ColorRed + "error:", ColorReset + "missing package name")
		fmt.Println("usage: nopile remove <package> [options]")
		os.Exit(1)
	}

	var names []string
	deps   := false
	config := false
	for _, arg := range os.Args[2:] {
		if arg == "--config" || arg == "-c" { config = true; continue }
		if arg == "--deps"   || arg == "-d" { deps = true; continue }
		if arg == "--all"    || arg == "-a" { config = true; deps = true; continue }
		if arg[0] == '-' {
			fmt.Println(ColorRed + "error:", ColorReset + "unknown option:", arg)
			fmt.Println("usage: nopile remove <package> [options]")
			os.Exit(1)
		}
		names = append(names, arg)
	}

	// Trouver les orphelins si --deps
	var orphans []string
	if deps {
		orphans = findOrphans(names)
	}

	// Vérifier que tous les paquets sont installés
	valid := true
	for _, name := range names {
		nopilePath := NOPILE_PKG_DIR + "/" + name + ".nopile"
		if _, err := os.Stat(nopilePath); err != nil {
			fmt.Println(ColorRed + "error:", ColorReset + "package not installed:", name)
			valid = false
		}
	}
	if !valid {
		return
	}

	// Vérifier les dependents
	allNames := append(names, orphans...)
	valid = true
	entries, _ := os.ReadDir(NOPILE_PKG_DIR)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".nopile") {
			continue
		}
		pkgName := strings.TrimSuffix(e.Name(), ".nopile")
		if slices.Contains(allNames, pkgName) {
			continue
		}
		pkg := dbFind(pkgName)
		if pkg == nil { continue }
		for _, dep := range pkg.Depends {
			if slices.Contains(allNames, dep) {
				fmt.Printf(ColorRed + "error:", ColorReset + "removing %s breaks dependency required by %s\n", dep, pkgName)
				valid = false
			}
		}
	}
	if !valid {
		return
	}

	// Afficher la transaction
	fmt.Println()
	fmt.Println("packages to remove:", len(names))
	fmt.Println()
	for _, name := range names {
		pkg := dbFind(name)
		if pkg != nil {
			fmt.Printf("  remove %-20s %s\n", name, pkg.Version)
		} else {
			fmt.Printf("  remove %-20s unknown\n", name)
		}
	}
	if len(orphans) > 0 {
		fmt.Println()
		fmt.Println("orphan dependencies to remove:", len(orphans))
		fmt.Println()
		for _, name := range orphans {
			pkg := dbFind(name)
			if pkg != nil {
				fmt.Printf("  remove %-20s %s\n", name, pkg.Version)
			} else {
				fmt.Printf("  remove %-20s unknown\n", name)
			}
		}
	}
	fmt.Println()
	fmt.Print("proceed? [y/N] ")

	if !noassume && !yesassume {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("aborted.")
			os.Exit(1)
		}
	}

	if noassume {
		fmt.Println("n")
		fmt.Println("aborted.")
		os.Exit(1)
	} else if yesassume {
		fmt.Println("y")
	}

	// Supprimer
	allToRemove := append(names, orphans...)
	total := len(allToRemove)
	fmt.Println("==> Removing packages")

	for i, name := range allToRemove {
		fmt.Printf("[%d/%d] Removing %s\n", i+1, total, name) // on affiche une progression claire

		if !removePackage(name, config) {
			fmt.Println("transaction aborted")
			return
		}
		explicitRemove(name)
	}
}

func cmdUpdate() {
	if len(os.Args) > 3 {
		fmt.Println(ColorRed + "error:", ColorReset + "invalid syntax")
		fmt.Println("usage: nopile update [--sync]")
		os.Exit(1)
	}

	// --sync → mettre à jour la db d'abord
	if len(os.Args) == 3 {
		if os.Args[2] != "--sync" && os.Args[2] != "-s" {
			fmt.Println(ColorRed + "error:", ColorReset + "unknown option:", os.Args[2])
			fmt.Println("usage: nopile update [--sync]")
			os.Exit(1)
		}
		dbSync()
	}

	// 1. Lister tous les paquets installés
	entries, err := os.ReadDir(NOPILE_PKG_DIR)
	if err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + "no packages installed")
		os.Exit(1)
	}

	// 2. Trouver les paquets à mettre à jour
	type updateEntry struct {
		name       string
		oldVersion string
		newVersion string
		action     string
		oldFiles   []string
	}
	var toUpdate []updateEntry
	var newDeps  []entry
	var orphans  []string
	seen := map[string]bool{}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".nopile") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".nopile")
		pkg  := dbFind(name)
		if pkg == nil { continue }

		// Lire la version installée depuis le .nopile
		data, err := os.ReadFile(NOPILE_PKG_DIR + "/" + name + ".nopile")
		if err != nil { continue }

		installedVersion := ""
		oldDepends := []string{}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "version: ") {
				installedVersion = strings.TrimPrefix(line, "version: ")
			}
			if strings.HasPrefix(line, "depends: ") {
				oldDepends = strings.Fields(strings.TrimPrefix(line, "depends: "))
			}
		}

		if installedVersion == "" || installedVersion == pkg.Version {
			continue
		}

		// Dépendances retirées → orphelins potentiels
		newDepSet := map[string]bool{}
		for _, dep := range pkg.Depends {
			newDepSet[dep] = true
		}
		for _, dep := range oldDepends {
			if !newDepSet[dep] && !explicitIsExplicit(dep) {
				subOrphans := findOrphans([]string{name})
				for _, o := range subOrphans {
					if !seen[o] {
						seen[o] = true
						orphans = append(orphans, o)
					}
				}
			}
		}

		var oldFiles []string
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if len(line) > 0 && line[0] == '/' {
				oldFiles = append(oldFiles, line)
			}
		}


		action := "binary"
		if pkg.Binary == "" && pkg.Compile != "" {
			action = "compile"
		}

		toUpdate = append(toUpdate, updateEntry{
			name:       name,
			oldVersion: installedVersion,
			newVersion: pkg.Version,
			action:     action,
			oldFiles:   oldFiles,
		})
	}

	if len(toUpdate) == 0 {
		fmt.Println("system is up to date")
		return
	}

	for _, u := range toUpdate {
		seen[u.name] = true
	}

	for _, u := range toUpdate {
		pkg := dbFind(u.name)
		if pkg == nil { continue }

		// Nouvelles dépendances
		for _, dep := range pkg.Depends {
			if !seen[dep] && !isInstalled(dep) {
				seen[dep] = true
				newDeps = append(newDeps, entry{dep, "binary", false})
			}
		}
	}

	// 4. Afficher la transaction
	fmt.Println()
	fmt.Println("packages to update:", len(toUpdate))
	fmt.Println()
	for _, u := range toUpdate {
		fmt.Printf("  %-20s %s → %s\n", u.name, u.oldVersion, u.newVersion)
	}
	if len(newDeps) > 0 {
		fmt.Println()
		fmt.Println("new dependencies:", len(newDeps))
		fmt.Println()
		for _, d := range newDeps {
			pkg := dbFind(d.name)
			if pkg != nil {
				fmt.Printf("  + %-20s %s\n", d.name, pkg.Version)
			}
		}
	}
	if len(orphans) > 0 {
		fmt.Println()
		fmt.Println("orphan dependencies to remove:", len(orphans))
		fmt.Println()
		for _, o := range orphans {
			pkg := dbFind(o)
			if pkg != nil {
				fmt.Printf("  - %-20s %s\n", o, pkg.Version)
			}
		}
	}
	fmt.Println()
	fmt.Print("proceed? [y/N] ")

	if !noassume && !yesassume {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("aborted.")
			os.Exit(1)
		}
	}

	if noassume {
		fmt.Println("n")
		fmt.Println("aborted.")
		os.Exit(0)
	} else if yesassume {
		fmt.Println("y")
	}

	// 5. Installer les nouvelles dépendances d'abord
	for _, d := range newDeps {
		if !installBinary(d.name, false, false) {
			fmt.Println("transaction aborted")
			os.Exit(1)
		}
	}

	// Convertir toUpdate en []entry pour les fonctions parallèles
	var updateEntries []entry
	for _, u := range toUpdate {
		updateEntries = append(updateEntries, entry{u.name, u.action, true})
	}
	for _, d := range newDeps {
		updateEntries = append(updateEntries, entry{d.name, d.action, false})
	}

	// Trier toUpdate — dépendances d'abord
	sorted := []updateEntry{}
	seenSort := map[string]bool{}

	var sortDeps func(name string)
	sortDeps = func(name string) {
		if seenSort[name] { return }
		seenSort[name] = true
		pkg := dbFind(name)
		if pkg != nil {
			for _, dep := range pkg.Depends {
				sortDeps(dep)
			}
		}
		for _, u := range toUpdate {
			if u.name == name {
				sorted = append(sorted, u)
				break
			}
		}
	}

	for _, u := range toUpdate {
		sortDeps(u.name)
	}
	toUpdate = sorted

	// Phases parallèles
	if !downloadAll(updateEntries) {
		fmt.Println("transaction aborted")
		os.Exit(1)
	}

	if !verifyAll(updateEntries) {
		fmt.Println("transaction aborted")
		os.Exit(1)
	}

	if !extractAll(updateEntries) {
		fmt.Println("transaction aborted")
		os.Exit(1)
	}

	// 5. Installer les nouvelles dépendances d'abord
	for _, d := range newDeps {
		if !installBinary(d.name, false, false) {
			fmt.Println("transaction aborted")
			os.Exit(1)
		}
	}

	// 6. Mettre à jour les paquets
	for _, u := range toUpdate {
		fmt.Println("updating:", u.name, u.oldVersion, "→", u.newVersion)
		var ok bool
		if u.action == "binary" {
			ok = installBinary(u.name, false, true)
		} else {
			ok = installCompile(u.name, false, true)
		}
		if !ok {
			fmt.Println("transaction aborted")
			os.Exit(1)
		}

		// Supprimer les fichiers obsolètes
		if len(u.oldFiles) > 0 {
			newData, err := os.ReadFile(NOPILE_PKG_DIR + "/" + u.name + ".nopile")
			if err == nil {
				newSet := map[string]bool{}
				for _, line := range strings.Split(string(newData), "\n") {
					line = strings.TrimSpace(line)
					if len(line) > 0 && line[0] == '/' {
						newSet[line] = true
					}
				}
				for _, oldFile := range u.oldFiles {
					if !newSet[oldFile] {
						fmt.Println("removing deprecated file:", oldFile)
						os.Remove(oldFile)
					}
				}
			}
		}
	}

	// Post-transactions et nettoyage parallèles
	if !postAll(updateEntries) {
		fmt.Println(ColorYellow + "warning:", ColorReset + "some post-transactions failed")
	}
	cleanAll(updateEntries)

	// 7. Supprimer les orphelins
	for _, o := range orphans {
		if !removePackage(o, false) {
			fmt.Println(ColorYellow + "warning:", ColorReset + "failed to remove orphan:", o)
		}
		explicitRemove(o)
	}
	fmt.Println("system updated successfully")
}

func cmdSearch() {
	if len(os.Args) < 3 {
		fmt.Println(ColorRed + "error:", ColorReset + "missing search term")
		fmt.Println("usage: nopile search <term> [<term>...]")
		os.Exit(1)
	}
	terms := os.Args[2:]
	pkgs  := dbLoad()

	for _, term := range terms {
		term  = strings.ToLower(term)
		found := false
		for _, pkg := range pkgs {
			if strings.Contains(strings.ToLower(pkg.Nom), term) {
				if isInstalled(pkg.Nom) {
					fmt.Printf("%-20s %-10s [installed]\n", pkg.Nom, pkg.Version)
				} else {
					fmt.Printf("%-20s %s\n", pkg.Nom, pkg.Version)
				}
				found = true
			}
		}
		if !found {
			fmt.Println(ColorRed + "error:", ColorReset + "no package found for:", term)
		}
	}
}

func cmdList() {
	if len(os.Args) != 2 {
		fmt.Println(ColorRed + "error:", ColorReset + "invalid syntax")
		fmt.Println("usage: nopile list")
		os.Exit(1)
	}
	entries, err := os.ReadDir(NOPILE_PKG_DIR)
	if err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + "no packages installed")
		os.Exit(1)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".nopile") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".nopile")
		pkg  := dbFind(name)
		if pkg != nil {
			fmt.Printf("%-20s %s\n", name, pkg.Version)
		} else {
			fmt.Printf("%-20s (unknown)\n", name)
		}
	}
}

func cmdInfo() {
	if len(os.Args) < 3 {
		fmt.Println(ColorRed + "error:", ColorReset + "missing package name")
		fmt.Println("usage: nopile info <package>")
		os.Exit(1)
	}
	for _, name := range os.Args[2:] {
		if name[0] == '-' {
			fmt.Println(ColorRed + "error:", ColorReset + "invalid syntax")
			fmt.Println("usage: nopile info <package>")
			os.Exit(1)
		}
		pkg := dbFind(name)
		if pkg == nil {
			fmt.Println(ColorRed + "error:", ColorReset + "package not found:", name)
			continue
		}
		fmt.Println("name:     ", pkg.Nom)
		fmt.Println("version:  ", pkg.Version)
		fmt.Println("installed:", isInstalled(name))
		if len(pkg.Depends) > 0 {
			fmt.Println("depends:  ", strings.Join(pkg.Depends, " "))
		}
		if pkg.Binary != "" {
			fmt.Println("binary:   ", pkg.Binary)
		}
		if pkg.Compile != "" {
			fmt.Println("nopbuild: ", pkg.Compile)
		}
		fmt.Println()
	}
}

func cmdVerify() {
	if len(os.Args) < 3 {
		fmt.Println(ColorRed + "error:", ColorReset + "missing package name")
		fmt.Println("usage: nopile verify <package> [<package>...]")
		os.Exit(1)
	}

	for _, name := range os.Args[2:] {
		if name[0] == '-' {
			fmt.Println(ColorRed + "error:", ColorReset + "unknown option:", name)
			os.Exit(1)
		}
		if !isInstalled(name) {
			fmt.Println(ColorRed + "error:", ColorReset + "package not installed:", name)
			continue
		}
		data, err := os.ReadFile(NOPILE_PKG_DIR + "/" + name + ".nopile")
		if err != nil {
			fmt.Println(ColorRed + "fatal:", ColorReset + "cannot read .nopile:", name)
			continue
		}

		var missing  []string
		var modified []string
		currentFile := ""

		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || line[0] == '#' || line[0] == '[' { continue }
			if strings.HasPrefix(line, "md5sum ") {
				if currentFile == "" { continue }
				expected := strings.TrimPrefix(line, "md5sum ")
				actual, err := md5File(currentFile)
				if err != nil {
					missing = append(missing, currentFile)
				} else if actual != expected {
					modified = append(modified, currentFile)
				}
				currentFile = ""
			} else if line[0] == '/' {
				currentFile = line
			}
		}

		if len(missing) == 0 && len(modified) == 0 {
			fmt.Println("intact:  ", name)
		} else {
			fmt.Println("modified:", name)
			for _, f := range missing {
				fmt.Println("  missing: ", f)
			}
			for _, f := range modified {
				fmt.Println("  modified:", f)
			}
		}
	}
}

func cmdClean() {
	if len(os.Args) != 2 {
		fmt.Println(ColorRed + "error:", ColorReset + "invalid syntax")
		fmt.Println("usage: nopile clean")
		os.Exit(1)
	}
	// Nettoyer le cache
	entries, err := os.ReadDir(NOPILE_CACHE)
	if err != nil {
		fmt.Println(ColorRed + "error:", ColorReset + "cache directory not found")
		os.Exit(1)
	}
	cleaned := 0
	for _, e := range entries {
		path := NOPILE_CACHE + "/" + e.Name()
		err  := os.Remove(path)
		if err == nil {
			vlog("removed:", e.Name())
			cleaned++
		}
	}
	os.RemoveAll("/var/tmp/nopile/")
	fmt.Println("cleaned", cleaned, "files")
}

func cmdWho() {
	if len(os.Args) < 3 {
		fmt.Println(ColorRed + "error:", ColorReset + "missing file")
		fmt.Println("usage: nopile who <file> [<file>...]")
		return
	}
	for _, file := range os.Args[2:] {
		owner := findFileOwner(file)
		if owner == "" {
			fmt.Println(ColorRed + "error:", ColorReset + "no package owns:", file)
			continue
		}
		fmt.Println(file, "is owned by", owner)
	}
}

func cmdHelp() {
    fmt.Print(`
    Nopile 1.0.0 - Noapila OS package manager

    Usage:
    nopile  <command> [options]

    Commands:
 -i install <package>    install a package
 -r remove  <package>    remove a package
 -u update  <options>    update installed packages
 -s search  <package>    search packages
 -l list                 list installed packages
 -I info    <package>    show package details
 -V verify  <package>    verify installed files
 -c clean                clean files
 -h help                 show this help
 -w who     <file>       show who own the file

    Install options:
 -c --compile            compile the packages from source
 -b --binary             install the binary package
 -l --local <file>       install from a local file
 -f --force              overwrite conflicts files

    Remove options:
 -d --deps               delete orphans dependency of the package
 -c --config             delete associated config files
 -a --all                alias of --deps and --config

    Update options:
 -s --sync               sync the database

    Global options
 -v --verbose            enable verbosity
 -y --yes                automatically responds "y" to question
 -n --no                 automatically responds "n" to question

`)
}

func resolveDeps(name string, seen map[string]bool, explicit map[string]bool, result *[]entry) {
	pkg := dbFind(name)
	if pkg == nil { return }
	for _, dep := range pkg.Depends {
		if !seen[dep] {
			seen[dep] = true
			resolveDeps(dep, seen, explicit, result)
			if !isInstalled(dep) || explicit[dep] {
				*result = append(*result, entry{dep, "binary", false})
			}
		}
	}
}


