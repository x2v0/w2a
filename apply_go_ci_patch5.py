#!/usr/bin/env python3
"""Add wta-buildapk smoke-test step to android-ci.yml (after Build musl-bridge)."""
import shutil
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent
CI = REPO / ".github/workflows/android-ci.yml"
MARKER = "# go-patch-5"

def backup(p):
    b = Path(str(p) + ".gopatch5")
    shutil.copy2(p, b)
    print(f"  backup: {b.name}")

def main():
    if not CI.exists():
        print(f"  [MISS] {CI}"); return 1
    src = CI.read_text(encoding="utf-8")
    if MARKER in src:
        print("  [skip] already patched"); return 0

    # Anchor: right before "Build standard debug APK"
    anchor = "      - name: Build standard debug APK\n"
    if anchor not in src:
        print("  [MISS] 'Build standard debug APK' anchor not found"); return 2

    smoke_step = (
        "      - name: Smoke test wta-buildapk\n"
        "        run: |\n"
        "          if [ -f app/src/main/assets/template/webview_shell.apk ]; then\n"
        "            ./go/bin/wta-buildapk \\\n"
        "              -template app/src/main/assets/template/webview_shell.apk \\\n"
        "              -url https://example.com \\\n"
        "              -app-name \"Smoke Test\" \\\n"
        "              -package com.smoketest.app \\\n"
        "              -out build/wta-smoke.apk\n"
        "            unzip -l build/wta-smoke.apk | head -5\n"
        "          else\n"
        "            echo 'shell-template.apk not present - skipping smoke test'\n"
        "          fi\n\n"
        "      - name: Upload wta-buildapk smoke artifact\n"
        "        if: hashFiles('build/wta-smoke.apk') != ''\n"
        "        uses: actions/upload-artifact@v4\n"
        "        with:\n"
        "          name: wta-buildapk-smoke-${{ github.run_number }}\n"
        "          path: build/wta-smoke.apk\n"
        "          if-no-files-found: warn\n"
        "          retention-days: 14\n\n"
    )

    src = src.replace(anchor, smoke_step + anchor, 1)
    src = MARKER + "\n" + src
    backup(CI)
    CI.write_text(src, encoding="utf-8")
    print("  [ok] android-ci.yml: smoke test added")
    print("\nReview:")
    print("  git diff .github/workflows/android-ci.yml | grep -A2 'wta-buildapk'")
    return 0

if __name__ == "__main__":
    sys.exit(main())
