package main

import "fmt"

func vlog(args ...interface{}) {
	if verbose {
		fmt.Println(args...)
	}
}
