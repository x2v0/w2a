package stack

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/x2v0/w2a/go/internal/apk"
)

// dexNameRe matches "classes.dex" or "classesN.dex".
var dexNameRe = regexp.MustCompile(`^classes(\d*)\.dex$`)

// MergeDex injects bundle DEX files into the target APK, renaming to avoid
// collisions. Existing classes.dex, classes2.dex, ... are preserved; new
// files get the next available "classesN.dex" name.
//
// NOTE: This does NOT update the multidex manifest metadata
// (android:multidex / MultiDexApplication). If the target's targetSdk < 21,
// the caller must ensure MultiDex is enabled.
// Returns the list of names injected.
func MergeDex(target *apk.APK, b *Bundle) ([]string, error) {
	// Collect existing dex names.
	used := map[int]bool{1: true} // "classes.dex" -> index 1
	for _, e := range target.Archive.Entries {
		base := e.Name
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		}
		if m := dexNameRe.FindStringSubmatch(base); m != nil {
			if m[1] == "" {
				used[1] = true
			} else if n, err := strconv.Atoi(m[1]); err == nil {
				used[n] = true
			}
		}
	}

	// Find next free index for each incoming dex.
	incoming := b.DexNames()
	var injected []string
	for _, name := range incoming {
		// Where to place it in the archive?
		slot := nextFreeSlot(used)
		var dst string
		if slot == 1 {
			dst = "classes.dex"
		} else {
			dst = fmt.Sprintf("classes%d.dex", slot)
		}
		used[slot] = true
		target.Archive.ReplaceDeflated(dst, b.DexFiles[name])
		injected = append(injected, dst)
	}
	return injected, nil
}

func nextFreeSlot(used map[int]bool) int {
	keys := make([]int, 0, len(used))
	for k := range used {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	n := 1
	for used[n] {
		n++
	}
	return n
}
