package main

import "fmt"

type group struct {
	packages []string
	action  string
}

func parseInstall(args []string) ([]group, bool) {
	var groups []group
	var courant []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--compile" || arg == "--binary" || arg == "--local" {
			// Pas de paquets avant l'option
			if len(courant) == 0 {
				fmt.Println("error: invalid syntax")
				fmt.Println("usage: nopile install <package> --binary|--compile|--local")
				return nil, false
			}
			// Fermer le groupe
			action := arg[2:]   // "--compile" → "compile"
			groups = append(groups, group{packages: courant, action: action})
			courant = []string{}
		} else {
			// C'est un paquet
			courant = append(courant, arg)
		}
	}

	// Des paquets sans option à la fin → binaire par défaut
	if len(courant) > 0 {
		groups = append(groups, group{packages: courant, action: "binary"})
	}

	return groups, true
}
