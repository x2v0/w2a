#!/usr/bin/env python3
"""
Apply all FileManagerScreen.kt patches.

Place this file at the repo root (~/w2a) and run:
    python3 apply_patch.py

It modifies:
    app/src/main/java/com/webtoapp/ui/screens/FileManagerScreen.kt

Patches:
  1. Add Download button to TopAppBar
  2. Add URL download dialog
  3. Move all app storage under Android/data/.../files/Download/MyFiles
  4. Auto-migrate legacy files on first run
  5. Add DOWNLOADS section

Idempotent: safe to run multiple times.
Creates backup: FileManagerScreen.kt.bak
"""

import re
import shutil
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent
TARGET = REPO_ROOT / "app/src/main/java/com/webtoapp/ui/screens/FileManagerScreen.kt"
MARKER = "// WTA_PATCH_FM_V1"


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def read(p: Path) -> str:
    return p.read_text(encoding="utf-8")


def write(p: Path, s: str) -> None:
    p.write_text(s, encoding="utf-8")


def insert_before(text: str, anchor: str, addition: str, desc: str) -> str:
    if addition.strip() and addition.strip() in text:
        print(f"  [skip] {desc}")
        return text
    idx = text.find(anchor)
    if idx < 0:
        print(f"  [MISS] {desc} — anchor not found: {anchor[:80]!r}")
        return text
    print(f"  [ok]   {desc}")
    return text[:idx] + addition + text[idx:]


def insert_after(text: str, anchor: str, addition: str, desc: str) -> str:
    if addition.strip() and addition.strip() in text:
        print(f"  [skip] {desc}")
        return text
    idx = text.find(anchor)
    if idx < 0:
        print(f"  [MISS] {desc} — anchor not found: {anchor[:80]!r}")
        return text
    end = idx + len(anchor)
    print(f"  [ok]   {desc}")
    return text[:end] + addition + text[end:]


def replace_once(text: str, old: str, new: str, desc: str) -> str:
    if new.strip() and new.strip() in text and old.strip() not in text:
        print(f"  [skip] {desc}")
        return text
    cnt = text.count(old)
    if cnt == 0:
        print(f"  [MISS] {desc} — pattern not found: {old[:80]!r}")
        return text
    print(f"  [ok]   {desc}")
    return text.replace(old, new, 1)


def replace_fn(text: str, fn_start: str, new_body: str, desc: str) -> str:
    if new_body.strip() in text and fn_start not in text:
        print(f"  [skip] {desc}")
        return text
    pattern = re.compile(
        r"^" + re.escape(fn_start) + r".*?^\}",
        re.DOTALL | re.MULTILINE,
    )
    if not pattern.search(text):
        print(f"  [MISS] {desc} — function not found: {fn_start[:60]!r}")
        return text
    print(f"  [ok]   {desc}")
    return pattern.sub(new_body.rstrip() + "\n", text, count=1)


# ---------------------------------------------------------------------------
# Patch pieces
# ---------------------------------------------------------------------------

IMPORTS = """
import androidx.compose.material.icons.outlined.Download
import androidx.compose.material3.OutlinedTextField
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.foundation.text.KeyboardOptions
import java.net.URL
"""

HELPERS = f"""
{MARKER}

/** Root of all app files (visible via Android/data/com.webtoapp/files/Download/MyFiles). */
private fun appFilesRoot(context: Context): File =
    (context.getExternalFilesDir(android.os.Environment.DIRECTORY_DOWNLOADS)
        ?: context.filesDir)
        .resolve("MyFiles")
        .also {{ it.mkdirs() }}

/** One-shot migration of legacy storage into the new root. */
private fun migrateLegacyStorageIfNeeded(context: Context) {{
    val newRoot = appFilesRoot(context)
    val legacyExternal = context.getExternalFilesDir(null)
    val legacyInternal = context.filesDir

    if (legacyExternal != null && legacyExternal != newRoot.parentFile) {{
        FileSection.entries
            .filter {{ !it.isUserProjects && it.dirName.isNotBlank() }}
            .forEach {{ section ->
                val old = File(legacyExternal, section.dirName)
                val new = File(newRoot, section.dirName)
                if (old.exists() && !new.exists()) {{
                    new.parentFile?.mkdirs()
                    old.renameTo(new)
                }} else if (old.exists() && new.exists()) {{
                    old.listFiles()?.forEach {{ f -> f.renameTo(File(new, f.name)) }}
                    old.delete()
                }}
            }}
    }}

    USER_PROJECT_DIRS.forEach {{ dirName ->
        val old = File(legacyInternal, dirName)
        val new = File(newRoot, dirName)
        if (old.exists() && !new.exists()) {{
            new.parentFile?.mkdirs()
            old.renameTo(new)
        }} else if (old.exists() && new.exists()) {{
            old.listFiles()?.forEach {{ f -> f.renameTo(File(new, f.name)) }}
            old.delete()
        }}
    }}
}}
"""

ENUM_DOWNLOADS = '    DOWNLOADS("downloads"),\n'

DOWNLOAD_STATE = """
    var showDownloadDialog by remember { mutableStateOf(false) }
    var downloadUrl by remember { mutableStateOf("") }
    var downloading by remember { mutableStateOf(false) }
"""

DOWNLOAD_BUTTON = """                IconButton(onClick = { showDownloadDialog = true }) {
                    Icon(Icons.Outlined.Download, contentDescription = "Download")
                }
"""

DOWNLOAD_DIALOG = """
    if (showDownloadDialog) {
        WtaAlertDialog(
            onDismissRequest = { if (!downloading) showDownloadDialog = false },
            title = { Text("Скачать файл") },
            text = {
                Column {
                    OutlinedTextField(
                        value = downloadUrl,
                        onValueChange = { downloadUrl = it },
                        label = { Text("URL") },
                        singleLine = true,
                        enabled = !downloading,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
                        modifier = Modifier.fillMaxWidth()
                    )
                    if (downloading) {
                        Spacer(Modifier.height(12.dp))
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            CircularProgressIndicator(modifier = Modifier.size(18.dp))
                            Spacer(Modifier.width(8.dp))
                            Text("Загрузка…")
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(
                    enabled = !downloading && downloadUrl.isNotBlank(),
                    onClick = {
                        scope.launch {
                            downloading = true
                            val result = downloadFromUrl(
                                url = downloadUrl.trim(),
                                targetDir = File(appFilesRoot(context), FileSection.DOWNLOADS.dirName)
                                    .also { it.mkdirs() }
                            )
                            downloading = false
                            result.onSuccess { file ->
                                downloadUrl = ""
                                showDownloadDialog = false
                                snackbarHostState.showSnackbar("Скачано: ${file.name}")
                                rescan(keepSelection = true)
                            }.onFailure { e ->
                                snackbarHostState.showSnackbar("Ошибка: ${e.message}")
                            }
                        }
                    }
                ) { Text("Скачать") }
            },
            dismissButton = {
                TextButton(
                    enabled = !downloading,
                    onClick = { showDownloadDialog = false }
                ) { Text("Отмена") }
            }
        )
    }
"""

DOWNLOAD_FN = """
private suspend fun downloadFromUrl(url: String, targetDir: File): Result<File> =
    withContext(Dispatchers.IO) {
        runCatching {
            targetDir.mkdirs()
            val fileName = url.substringAfterLast('/')
                .substringBefore('?')
                .ifBlank { "download_${System.currentTimeMillis()}" }
            val target = File(targetDir, fileName)
            URL(url).openStream().use { input ->
                target.outputStream().use { output -> input.copyTo(output) }
            }
            target
        }
    }
"""

SCAN_ALL_NEW = """private fun scanAllSections(context: Context): List<SectionSnapshot> {
    val root = appFilesRoot(context)
    return FileSection.entries.map { section ->
        if (section.isUserProjects) {
            scanUserProjects(root)
        } else {
            val dir = File(root, section.dirName)
            val entries = if (!dir.exists()) {
                emptyList()
            } else {
                dir.listFiles { f -> f.isFile }
                    ?.map { f ->
                        FileEntry(
                            file = f,
                            name = f.name,
                            size = f.length(),
                            lastModified = f.lastModified(),
                            section = section
                        )
                    }
                    ?: emptyList()
            }
            SectionSnapshot(section, entries)
        }
    }
}
"""

SCAN_USER_NEW = """private fun scanUserProjects(root: File): SectionSnapshot {
    val entries = USER_PROJECT_DIRS.flatMap { projectDirName ->
        val projectDir = File(root, projectDirName)
        if (!projectDir.exists() || !projectDir.isDirectory) return@flatMap emptyList()
        projectDir.listFiles { f -> f.isDirectory }
            ?.map { sub ->
                FileEntry(
                    file = sub,
                    name = "$projectDirName/${sub.name}",
                    size = dirSize(sub),
                    lastModified = sub.lastModified(),
                    section = FileSection.USER_PROJECTS
                )
            }
            ?: emptyList()
    }
    return SectionSnapshot(FileSection.USER_PROJECTS, entries)
}
"""

CLEAR_NEW = """private fun clearSection(context: Context, section: FileSection) {
    val root = appFilesRoot(context)
    if (section.isUserProjects) {
        USER_PROJECT_DIRS.forEach { dirName ->
            File(root, dirName)
                .listFiles { f -> f.isDirectory }
                ?.forEach { sub -> sub.deleteRecursively() }
        }
    } else {
        File(root, section.dirName).listFiles()?.forEach { f ->
            if (f.isDirectory) f.deleteRecursively() else f.delete()
        }
    }
}
"""


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> int:
    if not TARGET.exists():
        print(f"ERROR: {TARGET} not found.")
        print(f"Make sure you're running this from the repo root: {REPO_ROOT}")
        return 1

    src = read(TARGET)

    backup = TARGET.with_suffix(".kt.bak")
    shutil.copy2(TARGET, backup)
    print(f"Backup: {backup}\n")

    print("1. Imports")
    src = insert_after(src, "import java.util.Locale\n", IMPORTS, "new imports")

    print("2. Helpers (appFilesRoot + migrate)")
    src = insert_before(src, "private const val MAX_LOG_CHARS", HELPERS, "storage helpers")

    print("3. DOWNLOADS enum entry")
    src = replace_once(
        src,
        '    USER_PROJECTS("");',
        ENUM_DOWNLOADS + '    USER_PROJECTS("");',
        "DOWNLOADS enum",
    )

    print("4. Download dialog state")
    src = insert_after(
        src,
        "    var showSortMenu by remember { mutableStateOf(false) }\n",
        DOWNLOAD_STATE,
        "state vars",
    )

    print("5. Download button in TopAppBar")
    src = insert_after(
        src,
        "            } else {\n",
        DOWNLOAD_BUTTON,
        "IconButton",
    )

    print("6. Download dialog")
    src = insert_before(
        src,
        "@Composable\nprivate fun SummaryCard(",
        DOWNLOAD_DIALOG,
        "AlertDialog",
    )

    print("7. Rewrite scan/clear functions")
    src = replace_fn(src, "private fun scanAllSections(context: Context): List<SectionSnapshot> {",
                     SCAN_ALL_NEW, "scanAllSections")
    src = replace_fn(src, "private fun scanUserProjects(filesBase: File): SectionSnapshot {",
                     SCAN_USER_NEW, "scanUserProjects")
    src = replace_fn(src, "private fun clearSection(context: Context, section: FileSection) {",
                     CLEAR_NEW, "clearSection")

    print("8. downloadFromUrl() helper")
    src = insert_before(src, "private fun shareFile(",
                        DOWNLOAD_FN + "\n", "downloadFromUrl")

    print("9. LaunchedEffect migration")
    src = replace_once(
        src,
        "LaunchedEffect(Unit) { rescan() }",
        "LaunchedEffect(Unit) {\n"
        "        withContext(Dispatchers.IO) { migrateLegacyStorageIfNeeded(context) }\n"
        "        rescan()\n"
        "    }",
        "migration hook",
    )

    write(TARGET, src)
    print(f"\nDone. Patched: {TARGET}")
    print(f"Backup:  {backup}")
    print("\nNext steps:")
    print("  ./gradlew :app:compileStandardDebugKotlin -PallowDebugSignedRelease=true")
    print("  git add app/src/main/java/com/webtoapp/ui/screens/FileManagerScreen.kt apply_patch.py")
    print('  git commit -m "Add download button, move My Files to Downloads/MyFiles"')
    print("  git push")
    return 0


if __name__ == "__main__":
    sys.exit(main())
    
