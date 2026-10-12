#!/usr/bin/env python3
"""Gate syncShellTemplateApk and syncStackBundles with WTA_ASSETS_PREBUILT env."""
import shutil
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent
GRADLE = REPO / "app/build.gradle.kts"
MARKER = "// go-patch-4"

def patch():
    if not GRADLE.exists():
        print(f"  [MISS] {GRADLE}"); return False
    src = GRADLE.read_text(encoding="utf-8")
    if MARKER in src:
        print("  [skip] already patched"); return False

    a1 = 'tasks.register<Copy>("syncShellTemplateApk") {'
    if a1 not in src:
        print("  [MISS] syncShellTemplateApk"); return False
    i1 = src.find(a1) + len(a1)
    ins1 = ('\n    // Skip Gradle copy when prebuilt asset is provided by ./go/bin/wta-shelltemplate\n'
            '    enabled = System.getenv("WTA_ASSETS_PREBUILT") != "1"')
    src = src[:i1] + ins1 + src[i1:]
    print("  [ok] syncShellTemplateApk gated")

    a2 = 'tasks.register<Copy>("syncStackBundles") {'
    if a2 not in src:
        print("  [MISS] syncStackBundles"); return False
    i2 = src.find(a2) + len(a2)
    ins2 = ('\n    // Skip Gradle copy when prebuilt asset is provided by ./go/bin/wta-stackbundle\n'
            '    enabled = System.getenv("WTA_ASSETS_PREBUILT") != "1"')
    src = src[:i2] + ins2 + src[i2:]
    print("  [ok] syncStackBundles gated")

    src = MARKER + "\n" + src
    shutil.copy2(GRADLE, str(GRADLE) + ".gopatch4")
    print(f"  backup: {GRADLE.name}.gopatch4")
    GRADLE.write_text(src, encoding="utf-8")
    return True

def main():
    print("== patch app/build.gradle.kts ==")
    patch()
    print("\nReview: git diff app/build.gradle.kts")
    return 0

if __name__ == "__main__":
    sys.exit(main())
