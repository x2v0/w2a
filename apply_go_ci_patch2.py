#!/usr/bin/env python3
"""Second-pass CI patch: Go tooling in package-apks, robust paths."""
import shutil
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent
CI   = REPO / ".github/workflows/android-ci.yml"
MARKER = "# go-patch-2"

def backup(p):
    b = Path(str(p) + ".gopatch2")
    shutil.copy2(p, b)
    print(f"  backup: {b.name}")

def main():
    if not CI.exists():
        print(f"[MISS] {CI}"); return 1
    src = CI.read_text(encoding="utf-8")
    if MARKER in src:
        print("[skip] already patched"); return 0

    # 1. cache-dependency-path: go/go.sum -> go/go.mod
    if "cache-dependency-path: go/go.sum" in src:
        src = src.replace("cache-dependency-path: go/go.sum",
                          "cache-dependency-path: go/go.mod")
        print("  [ok] cache-dependency-path -> go/go.mod")

    # 2. -ndk "$ANDROID_NDK_HOME" -> explicit versioned path
    old_ndk = '-ndk "$ANDROID_NDK_HOME" \\'
    new_ndk = '-ndk "$ANDROID_HOME/ndk/29.0.14206865" \\'
    if old_ndk in src:
        src = src.replace(old_ndk, new_ndk)
        print("  [ok] -ndk explicit version")
    else:
        print("  [warn] -ndk anchor not found (may already be fixed)")

    # 3. Add Go setup + build in package-apks.
    #    Anchor: the 2nd occurrence of "      - name: Prepare Android tools\n".
    needle = "      - name: Prepare Android tools\n"
    first = src.find(needle)
    if first < 0:
        print("  [MISS] no Prepare Android tools"); return 2
    second = src.find(needle, first + len(needle))
    if second < 0:
        print("  [MISS] no 2nd Prepare Android tools in package-apks"); return 2

    setup_go = (
        "      - name: Prepare Go\n"
        "        uses: actions/setup-go@v5\n"
        "        with:\n"
        "          go-version: \"1.22\"\n"
        "          cache-dependency-path: go/go.mod\n\n"
        "      - name: Build Go tools\n"
        "        working-directory: go\n"
        "        run: make build\n\n"
    )
    src = src[:second] + setup_go + src[second:]
    print("  [ok] package-apks: added Prepare Go + Build Go tools")

    # 4. Add marker comment at top
    src = MARKER + "\n" + src

    backup(CI)
    CI.write_text(src, encoding="utf-8")
    print("\nDone. Review:")
    print("  git diff .github/workflows/android-ci.yml | tail -80")
    return 0

if __name__ == "__main__":
    sys.exit(main())
