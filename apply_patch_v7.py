#!/usr/bin/env python3
"""
Patch v7 — write to PUBLIC /storage/emulated/0/Download/MyFiles/
so Termux can access files.

Changes:
  1. AndroidManifest.xml: add MANAGE_EXTERNAL_STORAGE permission
  2. appFilesRoot(): use public Downloads when permission granted,
                     fallback to app-specific otherwise
  3. Add helper hasAllFilesAccess()
  4. MainActivity: request permission on first launch

Usage: python3 apply_patch_v7.py
"""
import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent

MANIFEST = REPO / "app/src/main/AndroidManifest.xml"
STORAGE = REPO / "app/src/main/java/com/webtoapp/core/storage/AppStorage.kt"
MAIN_ACTIVITY = REPO / "app/src/main/java/com/webtoapp/ui/MainActivity.kt"


NEW_STORAGE = '''package com.webtoapp.core.storage

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.os.Environment
import android.provider.Settings
import java.io.File

/**
 * Central app storage for user-visible files.
 *
 * If MANAGE_EXTERNAL_STORAGE is granted (Android 11+), files go to:
 *     /storage/emulated/0/Download/MyFiles/
 * which is accessible from Termux as:
 *     /sdcard/Download/MyFiles/
 *
 * Otherwise falls back to the app-specific Downloads directory:
 *     Android/data/<pkg>/files/Download/MyFiles/
 */
fun appFilesRoot(context: Context): File {
    val root = if (hasAllFilesAccess()) {
        File(
            Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_DOWNLOADS),
            "MyFiles"
        )
    } else {
        (context.getExternalFilesDir(Environment.DIRECTORY_DOWNLOADS) ?: context.filesDir)
            .resolve("MyFiles")
    }
    root.mkdirs()
    return root
}

fun appStorageSection(context: Context, name: String): File =
    File(appFilesRoot(context), name).also { it.mkdirs() }

/** true if we can write to public shared storage. */
fun hasAllFilesAccess(): Boolean =
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
        Environment.isExternalStorageManager()
    } else {
        true // pre-R uses WRITE_EXTERNAL_STORAGE
    }

/** Opens system settings page where user grants MANAGE_EXTERNAL_STORAGE. */
fun requestAllFilesAccess(context: Context) {
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R && !hasAllFilesAccess()) {
        try {
            val intent = Intent(Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION).apply {
                data = Uri.parse("package:${context.packageName}")
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
            context.startActivity(intent)
        } catch (_: Exception) {
            // Fallback: open the generic list of apps
            try {
                context.startActivity(
                    Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION)
                        .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                )
            } catch (_: Exception) { /* give up */ }
        }
    }
}
'''

PERMISSION_LINE = (
    '    <uses-permission android:name="android.permission.MANAGE_EXTERNAL_STORAGE" '
    'android:minSdkVersion="30" />\n'
)


def patch_manifest() -> bool:
    if not MANIFEST.exists():
        print(f"  [MISS] {MANIFEST}"); return False
    src = MANIFEST.read_text(encoding="utf-8")
    if "MANAGE_EXTERNAL_STORAGE" in src:
        print("  [skip] manifest already has MANAGE_EXTERNAL_STORAGE")
        return False

    # Insert right before </manifest>
    if "</manifest>" not in src:
        print("  [MISS] </manifest> not found"); return False
    src = src.replace("</manifest>", PERMISSION_LINE + "</manifest>", 1)
    MANIFEST.write_text(src, encoding="utf-8")
    print("  [ok]   manifest: MANAGE_EXTERNAL_STORAGE added")
    return True


def patch_storage() -> bool:
    STORAGE.parent.mkdir(parents=True, exist_ok=True)
    if STORAGE.exists():
        old = STORAGE.read_text(encoding="utf-8")
        if "MANAGE_EXTERNAL_STORAGE" in old or "isExternalStorageManager" in old:
            print("  [skip] AppStorage.kt already patched")
            return False
    STORAGE.write_text(NEW_STORAGE, encoding="utf-8")
    print(f"  [ok]   rewritten: {STORAGE.relative_to(REPO)}")
    return True


def patch_main_activity() -> bool:
    """Add requestAllFilesAccess call in MainActivity.onCreate (once, before content)."""
    if not MAIN_ACTIVITY.exists():
        print(f"  [MISS] {MAIN_ACTIVITY}"); return False
    src = MAIN_ACTIVITY.read_text(encoding="utf-8")

    if "requestAllFilesAccess" in src:
        print("  [skip] MainActivity already calls requestAllFilesAccess")
        return False

    # Find `override fun onCreate(savedInstanceState: Bundle?) {` and insert after it
    anchor = "override fun onCreate(savedInstanceState: Bundle?) {"
    idx = src.find(anchor)
    if idx < 0:
        print("  [MISS] onCreate anchor not found in MainActivity")
        return False

    insert_after = idx + len(anchor)
    snippet = (
        "\n        com.webtoapp.core.storage.requestAllFilesAccess(this)"
    )
    src = src[:insert_after] + snippet + src[insert_after:]

    MAIN_ACTIVITY.write_text(src, encoding="utf-8")
    print("  [ok]   MainActivity: requestAllFilesAccess() added")
    return True


def main() -> int:
    print("Patching…")
    changed = False
    changed |= patch_manifest()
    changed |= patch_storage()
    changed |= patch_main_activity()

    print(f"\nDone. {'Changes applied.' if changed else 'Nothing to change.'}")
    print("\nNext:")
    print("  git add -A")
    print('  git commit -m "feat(storage): use public Download/MyFiles for Termux access"')
    print("  git push")
    print("\nAfter install:")
    print("  - Grant 'All files access' permission on first launch")
    print("  - Files appear in /sdcard/Download/MyFiles/ (visible from Termux)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
    
