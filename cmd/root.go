package cmd

import (
	"fmt"
	"os"
)

func Execute() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "download":
		runDownload(os.Args[2:])
	case "watch":
		runWatch(os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: sba-showroom <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  download  Download a SHOWROOM livestream")
	fmt.Println("  watch     Watch campaign rooms and download when live (experimental)")
}
