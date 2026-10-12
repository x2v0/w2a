#!/usr/bin/env python3
"""Fix WtaAlertDialog -> AlertDialog in download block. Adds import."""
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent
TARGET = REPO / "app/src/main/java/com/webtoapp/ui/screens/FileManagerScreen.kt"

def main() -> int:
    if not TARGET.exists():
        print(f"ERROR: {TARGET} not found"); return 1
    src = TARGET.read_text(encoding="utf-8")

    if "import androidx.compose.material3.AlertDialog" in src and \
       "AlertDialog(\n            onDismissRequest = { if (!downloading) showDownloadDialog = false }," in src:
        print("Already fixed."); return 0

    # 1. Add import
    if "import androidx.compose.material3.AlertDialog\n" not in src:
        anchor = "import androidx.compose.material3.CircularProgressIndicator\n"
        if anchor not in src:
            print("ABORT: CircularProgressIndicator import not found"); return 2
        src = src.replace(anchor,
                          "import androidx.compose.material3.AlertDialog\n" + anchor, 1)
        print("  [ok] import AlertDialog")
    else:
        print("  [skip] import already present")

    # 2. Replace WtaAlertDialog with AlertDialog ONLY in the download block
    old = ("        WtaAlertDialog(\n"
           "            onDismissRequest = { if (!downloading) showDownloadDialog = false },")
    new = ("        AlertDialog(\n"
           "            onDismissRequest = { if (!downloading) showDownloadDialog = false },")
    if old not in src:
        if new in src:
            print("  [skip] dialog already AlertDialog")
        else:
            print("ABORT: download dialog anchor not found")
            print("       expected exactly:")
            print(repr(old))
            return 3
    else:
        src = src.replace(old, new, 1)
        print("  [ok] WtaAlertDialog -> AlertDialog")

    TARGET.write_text(src, encoding="utf-8")
    print(f"\nWrote: {TARGET}")
    print("\nNext:")
    print("  git add app/src/main/java/com/webtoapp/ui/screens/FileManagerScreen.kt apply_patch_v5_fix.py")
    print('  git commit -m "fix(filemanager): use standard Material3 AlertDialog for download"')
    print("  git push")
    return 0

if __name__ == "__main__":
    sys.exit(main())
    
