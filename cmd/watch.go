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
	verbose := fs.Bool("verbose", false, "Log room names on startup")
	fs.Usage = func() {
		fmt.Println("Usage: sba-showroom watch [--campaign <nogi|hinata|sakura>] [URL ...]")
		fmt.Println()
		fmt.Println("At least one --campaign or room URL is required.")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	roomURLs := fs.Args()
	if len(campaigns) == 0 && len(roomURLs) == 0 {
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

	if err := showroom.Watch(showroom.WatchOptions{Campaigns: unique, RoomURLs: roomURLs, Verbose: *verbose}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
