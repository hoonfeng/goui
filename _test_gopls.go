package main

import (
	"fmt"
	"os/exec"
)

func main() {
	path, err := exec.LookPath("gopls")
	if err != nil {
		fmt.Println("LookPath FAILED:", err)
	} else {
		fmt.Println("Found gopls at:", path)
	}
}
