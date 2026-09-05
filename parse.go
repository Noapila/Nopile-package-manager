package main

import "fmt"

type group struct {
	packages []string
	action  string
}

func parseInstall(args []string) ([]group, bool, bool) {
	var groups []group
	var courant []string
	force := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--force" || arg == "-f" {
			if force {
				fmt.Println(ColorRed + "error:", ColorReset + "invalid syntax")
				fmt.Println("usage: nopile install <package> --binary|--compile|--local [--force]")
				return nil, false, false
			}
			force = true
			continue
		}
		if arg == "--compile" || arg == "-c" || arg == "--binary" || arg == "-b" || arg == "--local" || arg == "-l" {
			if len(courant) == 0 {
				fmt.Println(ColorRed + "error:", ColorReset + "invalid syntax")
				fmt.Println("usage: nopile install <package> --binary|--compile|--local [--force]")
				return nil, false, false
			}
			var action string
			switch arg {
				case "-c":
					action = "compile"
				case "-b":
					action = "binary"
				case "-l":
					action = "local"
				default:
					// Pour les flags longs, on garde le comportement actuel
					action = arg[2:]
			}
			groups = append(groups, group{packages: courant, action: action})
			courant = []string{}
		} else {
			courant = append(courant, arg)
		}
	}
	if len(courant) > 0 {
		groups = append(groups, group{packages: courant, action: "binary"})
	}
	return groups, true, force
}
