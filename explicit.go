package main

import (
	"os"
	"strings"
)

const NOPILE_EXPLICIT = "/var/lib/nopile/explicit"

func explicitLoad() []string {
	data, err := os.ReadFile(NOPILE_EXPLICIT)
	if err != nil {
		return []string{}
	}
	var pkgs []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			pkgs = append(pkgs, line)
		}
	}
	return pkgs
}

func explicitAdd(name string) {
	pkgs := explicitLoad()
	for _, p := range pkgs {
		if p == name { return }
	}
	pkgs = append(pkgs, name)
	explicitSave(pkgs)
}

func explicitRemove(name string) {
	pkgs := explicitLoad()
	var newPkgs []string
	for _, p := range pkgs {
		if p != name {
			newPkgs = append(newPkgs, p)
		}
	}
	explicitSave(newPkgs)
}

func explicitSave(pkgs []string) {
	os.MkdirAll(NOPILE_PKG_DIR, 0755)
	data := strings.Join(pkgs, "\n")
	if len(pkgs) > 0 {
		data += "\n"
	}
	os.WriteFile(NOPILE_EXPLICIT, []byte(data), 0644)
}

func explicitIsExplicit(name string) bool {
	for _, p := range explicitLoad() {
		if p == name { return true }
	}
	return false
}

func findOrphans(names []string) []string {
	var orphans []string
	seen := map[string]bool{}
	for _, n := range names { seen[n] = true }

	for _, name := range names {
		pkg := dbFind(name)
		if pkg == nil { continue }
		for _, dep := range pkg.Depends {
			if seen[dep] { continue }
			// Pas dans explicit
			if explicitIsExplicit(dep) { continue }
			// Vérifier que personne d'autre ne l'utilise
			usedByOther := false
			entries, _ := os.ReadDir(NOPILE_PKG_DIR)
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".nopile") { continue }
				pkgName := strings.TrimSuffix(e.Name(), ".nopile")
				if seen[pkgName] { continue }
				p := dbFind(pkgName)
				if p == nil { continue }
				for _, d := range p.Depends {
					if d == dep {
						usedByOther = true
						break
					}
				}
				if usedByOther { break }
			}
			if !usedByOther && isInstalled(dep) {
				seen[dep] = true
				orphans = append(orphans, dep)
				// Récursif — les dépendances des orphelins
				subOrphans := findOrphans([]string{dep})
				for _, o := range subOrphans {
					if !seen[o] {
						seen[o] = true
						orphans = append(orphans, o)
					}
				}
			}
		}
	}
	return orphans
}
