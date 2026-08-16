//go:build !windows

package main

import (
	"fmt"
	"log"
)

func fatalf(format string, args ...any) {
	log.Fatal(fmt.Sprintf(format, args...))
}

func messageBoxInfo(title, text string) {
	fmt.Printf("%s: %s\n", title, text)
}
