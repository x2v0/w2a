// Command wta-stackbundle copies feature-stack ZIPs into app assets.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/x2v0/w2a/go/internal/assetsync"
)

func main() {
	var (
		srcGlob = flag.String("src", "feature-stacks/*/build/outputs/stack/*.zip", "glob for stack zips")
		dst     = flag.String("dst", "app/src/main/assets/stacks", "destination directory")
	)
	flag.Parse()

	n, err := assetsync.CopyMatching(*srcGlob, *dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wta-stackbundle: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("copied %d stack zip(s) to %s\n", n, *dst)
}
