package cmd

import (
	"fmt"
	"os"
	"runtime/debug"
)

// version is set via -ldflags "-X .../cmd.version=vX.Y.Z" when building release
// binaries (see .github/workflows/publish.yml). It is left at "dev" for local
// builds and go install builds fall back to the module version below.
var version = "dev"

func Execute() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) < 1 {
		printUsage()
		return 1
	}

	switch args[0] {
	case "download":
		runDownload(args[1:])
	case "watch":
		runWatch(args[1:])
	case "version", "-v", "--version":
		printVersion()
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printUsage()
		return 1
	}
	return 0
}

func printVersion() {
	fmt.Println("sba-showroom " + resolveVersion())
}

// resolveVersion falls back to the module version recorded by the Go toolchain
// (populated for `go install .../sba-showroom@vX.Y.Z`) when not overridden by
// -ldflags at build time.
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func printUsage() {
	fmt.Println("Usage: sba-showroom <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  download  Download a SHOWROOM livestream")
	fmt.Println("  watch     Watch campaign rooms and download when live (experimental)")
	fmt.Println("  version   Print version")
	fmt.Println("  help      Show this help")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h, --help     Show this help")
	fmt.Println("  -v, --version  Print version")
}
