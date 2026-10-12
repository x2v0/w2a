#!/usr/bin/env python3
"""Clean up Python references in modules-*.yml workflows."""
import re
import shutil
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent
CHECK = REPO / ".github/workflows/modules-check.yml"
PUBLISH = REPO / ".github/workflows/modules-publish.yml"

def patch_check():
    src = CHECK.read_text(encoding="utf-8")
    if "# go-patch-7" in src:
        print("  [skip] modules-check.yml already patched")
        return False
    orig = src

    # Header comment
    src = src.replace(
        "# Runs the module catalog validator at `.github/scripts/ci/validate_modules.py`.",
        "# Runs the module catalog validator (Go: `go/bin/wta-validatemodules`)."
    )

    # paths triggers: replace python lines with go equivalents
    src = src.replace("      - '.github/scripts/ci/validate_modules.py'",
                      "      - 'go/bin/wta-validatemodules'\n      - 'go/cmd/wta-validatemodules/**'")
    src = src.replace("      - '.github/scripts/ci/generate_submissions.py'",
                      "      - 'go/cmd/wta-gensubmissions/**'\n      - 'go/internal/modules/**'")

    src = "# go-patch-7\n" + src
    CHECK.write_text(src, encoding="utf-8")
    shutil.copy2(CHECK, str(CHECK) + ".prepatch7")
    print("  [ok] modules-check.yml cleaned")
    return src != orig

def patch_publish():
    src = PUBLISH.read_text(encoding="utf-8")
    if "# go-patch-7" in src:
        print("  [skip] modules-publish.yml already patched")
        return False

    src = src.replace("      - '.github/scripts/ci/generate_submissions.py'",
                      "      - 'go/cmd/wta-gensubmissions/**'\n      - 'go/internal/modules/**'")

    src = "# go-patch-7\n" + src
    PUBLISH.write_text(src, encoding="utf-8")
    shutil.copy2(PUBLISH, str(PUBLISH) + ".prepatch7")
    print("  [ok] modules-publish.yml cleaned")
    return True

def main():
    print("== patch modules-check.yml ==")
    patch_check()
    print("== patch modules-publish.yml ==")
    patch_publish()
    return 0

if __name__ == "__main__":
    sys.exit(main())
