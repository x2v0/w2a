#!/usr/bin/env python3
"""Patch modules-check.yml and modules-publish.yml to use Go binaries."""
import re
import shutil
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent
CHECK   = REPO / ".github/workflows/modules-check.yml"
PUBLISH = REPO / ".github/workflows/modules-publish.yml"
MARKER  = "# go-patch-3"

def backup(p):
    b = Path(str(p) + ".gopatch3")
    shutil.copy2(p, b)
    print(f"  backup: {b.name}")

GO_SETUP = """      - name: Prepare Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.22"
          cache-dependency-path: go/go.mod

      - name: Build Go tools
        working-directory: go
        run: make build

"""

def replace_python_version_step(src: str) -> str:
    """Replace the 'Use system Python 3' step (with optional comment lines) with Go setup."""
    # Match the whole step block: name line, optional comment, run line.
    pattern = re.compile(
        r"^      - name: Use system Python 3\n"
        r"(?:        [#].*\n)*"
        r"        run: python3 --version\n",
        re.MULTILINE,
    )
    if not pattern.search(src):
        # Fallback: just replace the run line.
        if "run: python3 --version" in src:
            src = src.replace(
                "run: python3 --version",
                "run: go version",
                1,
            )
            print("  [warn] replaced only 'run: python3 --version' -> 'run: go version'")
        return src
    src = pattern.sub(GO_SETUP, src, count=1)
    print("  [ok] Python step -> Go setup step")
    return src

def patch_check():
    if not CHECK.exists():
        print(f"  [MISS] {CHECK}"); return False
    src = CHECK.read_text(encoding="utf-8")
    if MARKER in src:
        print("  [skip] modules-check.yml already patched"); return False

    src = replace_python_version_step(src)

    # Replace the script invocation
    old_calls = [
        "python3 .github/scripts/ci/validate_modules.py",
    ]
    for call in old_calls:
        if call in src:
            src = src.replace(call, "./go/bin/wta-validatemodules")
            print(f"  [ok] {call!r} -> './go/bin/wta-validatemodules'")
            break
    else:
        print("  [MISS] validate_modules.py invocation not found")

    src = MARKER + "\n" + src
    backup(CHECK)
    CHECK.write_text(src, encoding="utf-8")
    return True

def patch_publish():
    if not PUBLISH.exists():
        print(f"  [MISS] {PUBLISH}"); return False
    src = PUBLISH.read_text(encoding="utf-8")
    if MARKER in src:
        print("  [skip] modules-publish.yml already patched"); return False

    src = replace_python_version_step(src)

    old_calls = [
        "python3 .github/scripts/ci/generate_submissions.py",
    ]
    for call in old_calls:
        if call in src:
            src = src.replace(call, "./go/bin/wta-gensubmissions")
            print(f"  [ok] {call!r} -> './go/bin/wta-gensubmissions'")
            break
    else:
        print("  [MISS] generate_submissions.py invocation not found")

    src = MARKER + "\n" + src
    backup(PUBLISH)
    PUBLISH.write_text(src, encoding="utf-8")
    return True

def main():
    print("== modules-check.yml ==")
    a = patch_check()
    print("== modules-publish.yml ==")
    b = patch_publish()
    if a or b:
        print("\nReview:")
        print("  git diff .github/workflows/modules-check.yml .github/workflows/modules-publish.yml")
    return 0

if __name__ == "__main__":
    sys.exit(main())
