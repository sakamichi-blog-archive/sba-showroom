package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/sakamichi-blog-archive/sba-showroom/internal/showroom"
)

type campaignFlags []string

func (c *campaignFlags) String() string { return fmt.Sprintf("%v", *c) }
func (c *campaignFlags) Set(v string) error {
	*c = append(*c, v)
	return nil
}

func runWatch(args []string) {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	var campaigns campaignFlags
	fs.Var(&campaigns, "campaign", "Campaign to watch (nogi|hinata|sakura); may be repeated")
	verbose := fs.Bool("verbose", false, "Log room names on startup and excluded rooms")
	fs.Usage = func() {
		fmt.Println("Usage: sba-showroom watch --campaign <nogi|hinata|sakura> [--campaign ...]")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if len(campaigns) == 0 {
		fs.Usage()
		os.Exit(1)
	}

	seen := make(map[string]bool)
	var unique []string
	for _, c := range campaigns {
		if !seen[c] {
			seen[c] = true
			unique = append(unique, c)
		}
	}

	if err := showroom.Watch(showroom.WatchOptions{Campaigns: unique, Verbose: *verbose}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
