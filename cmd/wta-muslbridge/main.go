package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/x2v0/w2a/go/internal/musl"
)

func main() {
	var (
		ndk     = flag.String("ndk", "", "Android NDK path (autodetect if empty)")
		work    = flag.String("work", "build/musl-bridge", "work directory")
		out     = flag.String("out", "build/generated/jniLibs", "output directory")
		version = flag.String("musl", "1.2.5", "musl version")
		tarball = flag.String("tarball", "", "local musl tarball (skip download)")
		patch   = flag.String("patch", "scripts/patches/musl-1.2.5-wta-exec-bridge.patch", "patch file")
		shim    = flag.String("shim", "scripts/musl-bridge/wta_mulxc3.c", "x80 shim file")
		abis    = flag.String("abis", "arm64-v8a,x86_64", "comma-separated ABI list")
	)
	flag.Parse()

	cfg := musl.Config{
		MuslVersion: *version,
		TarballPath: *tarball,
		WorkDir:     *work,
		OutputDir:   *out,
		NDKPath:     *ndk,
		PatchPath:   *patch,
		ShimPath:    *shim,
	}
	for _, a := range strings.Split(*abis, ",") {
		a = strings.TrimSpace(a)
		if a != "" {
			cfg.ABIs = append(cfg.ABIs, a)
		}
	}
	if len(cfg.ABIs) == 0 {
		cfg.ABIs = []string{"arm64-v8a", "x86_64"}
	}

	if err := musl.Build(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "wta-muslbridge: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("wta-muslbridge: done")
}
