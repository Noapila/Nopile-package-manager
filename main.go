package main

import (
	"fmt"
	"os"
//	"strings"
)

func main() {
	// os.Args c'est la liste des arguments tapés dans le terminal
	// os.Args[0] = "nopile" (le nom du programme)
	// os.Args[1] = la commande tapée (install, remove, etc.)

	// Si aucun argument → afficher l'aide
	if len(os.Args) < 2 {
		cmdHelp()
		return
	}

	// Lire la commande tapée
	commande := os.Args[1]

	// Choisir quoi faire selon la commande
	if commande == "install" {
		cmdInstall()
	} else if commande == "remove" {
		cmdRemove()
	} else if commande == "update" {
		cmdUpdate()
	} else if commande == "search" {
		cmdSearch()
	} else if commande == "list" {
		cmdList()
	} else if commande == "info" {
		cmdInfo()
	} else if commande == "verify" {
		cmdVerify()
	} else if commande == "clean" {
		cmdClean()
	} else if commande == "help" {
		cmdHelp()
	} else if commande == "--sync" {
		if os.Getuid() != 0 {
			fmt.Println("error: nopile --sync must be run as administrator")
			return
		}
		dbSync()
	} else {
		// Commande inconnue
		fmt.Println("[nopile] unknown command:", commande)
		fmt.Println("use \"nopile help\" for see help")
	}
}

// ── Chaque fonction est dans son propre fichier ────────────────────────────
// Tu vas créer : install.go, remove.go, update.go, search.go, etc.
// Pour l'instant elles sont vides ici pour que ça compile

func cmdInstall() {
	if len(os.Args) < 3 {
		fmt.Println("error: missing package name")
		fmt.Println("usage: nopile install <package> --binary|--compile|--local")
		return
	}

	groups, ok := parseInstall(os.Args[2:])
	if !ok {
		return
	}

	for _, g := range groups {
		for _, pkg := range g.packages {
			if g.action == "binary" {
				installBinary(pkg)
			} else if g.action == "compile" {
				fmt.Println("compile not yet implemented:", pkg)
			} else if g.action == "local" {
				fmt.Println("local not yet implemented:", pkg)
			}
		}
	}
}
func cmdRemove() {
	if len(os.Args) == 3 && os.Args[2][0] != '-' {
		fmt.Println("[nopile] removing:", os.Args[2])
		return
	}
	fmt.Println("error: invalid syntax")
	fmt.Println("usage: nopile remove <package>")
}

func cmdUpdate() {
	// nopile update
	if len(os.Args) == 2 {
		fmt.Println("[nopile] updating system...")
		return
	}
	// nopile update --sync
	if len(os.Args) == 3 && os.Args[2] == "--sync" {
		fmt.Println("[nopile] syncing database and updating...")
		return
	}
	fmt.Println("error: invalid syntax")
	fmt.Println("usage: nopile update [--sync]")
}

func cmdSearch() {
	if len(os.Args) == 3 && os.Args[2][0] != '-' {
		paquets := dbLoad()
		for _, pkg := range paquets {
			if pkg.Nom == os.Args[2] {
				fmt.Println(pkg.Nom, pkg.Version)
				return
			}
		}
		fmt.Println("error: package not found:", os.Args[2])
		return
	}
	fmt.Println("error: invalid syntax")
	fmt.Println("usage: nopile search <package>")
}

func cmdList() {
	if len(os.Args) == 2 {
		fmt.Println("[nopile] listing installed packages...")
		return
	}
	fmt.Println("error: invalid syntax")
	fmt.Println("usage: nopile list")
}

func cmdInfo() {
	if len(os.Args) == 3 && os.Args[2][0] != '-' {
		fmt.Println("[nopile] info:", os.Args[2])
		return
	}
	fmt.Println("error: invalid syntax")
	fmt.Println("usage: nopile info <package>")
}

func cmdVerify() {
	if len(os.Args) == 3 && os.Args[2][0] != '-' {
		fmt.Println("[nopile] verifying:", os.Args[2])
		return
	}
	fmt.Println("error: invalid syntax")
	fmt.Println("usage: nopile verify <package>")
}

func cmdClean() {
	if len(os.Args) == 2 {
		fmt.Println("[nopile] cleaning cache...")
		return
	}
	fmt.Println("error: invalid syntax")
	fmt.Println("usage: nopile clean")
}

func cmdHelp() {
    fmt.Print(`
    nopile 0.1.0 - Noapila OS package manager

    Usage:
    nopile <command> [options]

    Commands:
    install <package>    install a package
    remove  <package>    remove a package
    update               update installed packages
    search  <package>    search packages
    list                 list installed packages
    info    <package>    show package details
    verify  <package>    verify installed files
    clean                clear cache
    help                 show this help

    Install options:
    --compile            compile the packages from source
    -l <file.tar.gz>     install from a local tarball

    Update options:
    --sync               sync the database

`)
}

