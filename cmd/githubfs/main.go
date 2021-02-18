package main

import (
	"fmt"
	"os"

	"githubfs/internal/githubfuse"
)

func usage() {
	fmt.Printf("Usage: %s REPO MOUNTPOINT\n\n", os.Args[0])
	fmt.Printf("Example: %s someorganization/example /mnt\n", os.Args[0])
}

func main() {
	if len(os.Args[1:]) != 2 {
		usage()
		os.Exit(1)
	}

	repo := os.Args[1]
	mountpoint := os.Args[2]

	err := githubfuse.Serve(repo, mountpoint)
	if err != nil {
		os.Exit(1)
	}
}
