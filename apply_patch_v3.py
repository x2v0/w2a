#!/usr/bin/env python3
"""
FileManagerScreen.kt patch v3 — safe, idempotent.

Usage:  python3 apply_patch_v3.py

Does nothing if any anchor is missing. Backup at *.kt.prepatch.
"""

import re
import shutil
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent
TARGET = REPO / "app/src/main/java/com/webtoapp/ui/screens/FileManagerScreen.kt"
MARKER = "// WTA_PATCH_FM_V3"


# ---- pieces ---------------------------------------------------------------

IMPORTS = """import androidx.compose.material.icons.outlined.Download
import androidx.compose.material3.OutlinedTextField
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.foundation.text.KeyboardOptions
import java.net.URL
"""

HELPERS = """
// WTA_PATCH_FM_V3_HELPERS

/** Root of all app files (visible via Android/data/com.webtoapp/files/Download/MyFiles). */
private fun appFilesRoot(context: Context): File =
    (context.getExternalFilesDir(android.os.Environment.DIRECTORY_DOWNLOADS)
        ?: context.filesDir)
        .resolve("MyFiles")
        .also { it.mkdirs() }

/** One-shot migration of legacy storage into the new root. */
private fun migrateLegacyStorageIfNeeded(context: Context) {
    val newRoot = appFilesRoot(context)
    val legacyExternal = context.getExternalFilesDir(null)
    val legacyInternal = context.filesDir

    if (legacyExternal != null && legacyExternal != newRoot.parentFile) {
        FileSection.entries
            .filter { !it.isUserProjects && it.dirName.isNotBlank() }
            .forEach { section ->
                val old = File(legacyExternal, section.dirName)
                val new = File(newRoot, section.dirName)
                if (old.exists() && !new.exists()) {
                    new.parentFile?.mkdirs()
                    old.renameTo(new)
                } else if (old.exists() && new.exists()) {
                    old.listFiles()?.forEach { f -> f.renameTo(File(new, f.name)) }
                    old.delete()
                }
            }
    }

    USER_PROJECT_DIRS.forEach { dirName ->
        val old = File(legacyInternal, dirName)
        val new = File(newRoot, dirName)
        if (old.exists() && !new.exists()) {
            new.parentFile?.mkdirs()
            old.renameTo(new)
        } else if (old.exists() && new.exists()) {
            old.listFiles()?.forEach { f -> f.renameTo(File(new, f.name)) }
            old.delete()
        }
    }
}
"""

DOWNLOAD_BUTTON = """                IconButton(onClick = { showDownloadDialog = true }) {
                    Icon(Icons.Outlined.Download, contentDescription = "Download")
                }
"""

DOWNLOAD_DIALOG = """    if (showDownloadDialog) {
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

DOWNLOAD_FN = """private suspend fun downloadFromUrl(url: String, targetDir: File): Result<File> =
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

SCAN_ALL = """private fun scanAllSections(context: Context): List<SectionSnapshot> {
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

SCAN_USER = """private fun scanUserProjects(root: File): SectionSnapshot {
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

CLEAR = """private fun clearSection(context: Context, section: FileSection) {
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


# ---- helpers --------------------------------------------------------------

def insert_after(src, anchor, addition, desc):
    idx = src.find(anchor)
    if idx < 0:
        raise RuntimeError(f"anchor not found ({desc}): {anchor[:70]!r}")
    print(f"  [ok] {desc}")
    return src[:idx + len(anchor)] + addition + src[idx + len(anchor):]


def insert_before(src, anchor, addition, desc):
    idx = src.find(anchor)
    if idx < 0:
        raise RuntimeError(f"anchor not found ({desc}): {anchor[:70]!r}")
    print(f"  [ok] {desc}")
    return src[:idx] + addition + src[idx:]


def replace_first(src, old, new, desc):
    if old not in src:
        raise RuntimeError(f"pattern not found ({desc}): {old[:70]!r}")
    print(f"  [ok] {desc}")
    return src.replace(old, new, 1)


def replace_fn(src, fn_start, new_body, desc):
    pat = re.compile(r"^" + re.escape(fn_start) + r".*?^\}", re.DOTALL | re.MULTILINE)
    new_src, n = pat.subn(new_body.rstrip() + "\n", src, count=1)
    if n != 1:
        raise RuntimeError(f"function not found ({desc}): {fn_start[:70]!r}")
    print(f"  [ok] {desc}")
    return new_src


# ---- main -----------------------------------------------------------------

def main() -> int:
    if not TARGET.exists():
        print(f"ERROR: {TARGET} not found")
        return 1

    src = TARGET.read_text(encoding="utf-8")
    if MARKER in src:
        print("Already patched (v3). Exiting.")
        return 0

    # Verify exact button anchor first (this is what killed v1/v2)
    button_anchor = (
        "            } else {\n"
        "                IconButton(\n"
        "                    onClick = { scope.launch { rescan() } },\n"
        "                    enabled = !loading\n"
        "                ) {\n"
    )
    if button_anchor not in src:
        print("ABORT: could not find the exact actions/else/rescan button anchor.")
        print("Send the output of:  sed -n '280,310p' FileManagerScreen.kt")
        return 2

    print("All anchors verified. Applying patches…")

    backup = TARGET.with_suffix(".kt.prepatch")
    shutil.copy2(TARGET, backup)
    print(f"Backup: {backup}")

    # 1. imports
    src = insert_after(src, "import java.util.Locale\n", IMPORTS, "imports")

    # 2. helpers before MAX_LOG_CHARS
    src = insert_before(src, "private const val MAX_LOG_CHARS", HELPERS + "\n",
                        "helpers (appFilesRoot + migrate)")

    # 3. DOWNLOADS enum entry
    src = replace_first(src, '    USER_PROJECTS("");',
                        '    DOWNLOADS("downloads"),\n    USER_PROJECTS("");',
                        "DOWNLOADS enum")

    # 4. state vars
    src = insert_after(
        src,
        "    var showSortMenu by remember { mutableStateOf(false) }\n",
        "    var showDownloadDialog by remember { mutableStateOf(false) }\n"
        "    var downloadUrl by remember { mutableStateOf(\"\") }\n"
        "    var downloading by remember { mutableStateOf(false) }\n",
        "state vars",
    )

    # 5. download button in TopAppBar actions
    src = insert_before(src, button_anchor, DOWNLOAD_BUTTON, "download button")

    # 6. download dialog before SummaryCard
    src = insert_before(src, "@Composable\nprivate fun SummaryCard(",
                        DOWNLOAD_DIALOG + "\n", "download dialog")

    # 7. replace scan/clear functions
    src = replace_fn(src,
        "private fun scanAllSections(context: Context): List<SectionSnapshot> {",
        SCAN_ALL, "scanAllSections")
    src = replace_fn(src,
        "private fun scanUserProjects(filesBase: File): SectionSnapshot {",
        SCAN_USER, "scanUserProjects")
    src = replace_fn(src,
        "private fun clearSection(context: Context, section: FileSection) {",
        CLEAR, "clearSection")

    # 8. downloadFromUrl helper before shareFile
    src = insert_before(src, "private fun shareFile(", DOWNLOAD_FN + "\n",
                        "downloadFromUrl helper")

    # 9. migration hook
    src = replace_first(
        src,
        "LaunchedEffect(Unit) { rescan() }",
        "LaunchedEffect(Unit) {\n"
        "        withContext(Dispatchers.IO) { migrateLegacyStorageIfNeeded(context) }\n"
        "        rescan()\n"
        "    }",
        "migration hook",
    )

    # 10. sectionLabel — add DOWNLOADS branch
    src = insert_after(
        src,
        "private fun sectionLabel(section: FileSection): String = when (section) {\n",
        '    FileSection.DOWNLOADS -> "Downloads"\n',
        "sectionLabel branch",
    )

    # marker at top
    src = MARKER + "\n" + src

    TARGET.write_text(src, encoding="utf-8")
    print(f"\nWrote: {TARGET}")
    print(f"Backup: {backup}")
    print("\nNext:")
    print("  ./gradlew :app:compileStandardDebugKotlin -PallowDebugSignedRelease=true")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except RuntimeError as e:
        print(f"\nABORT: {e}")
        print("Nothing was written. File is unchanged.")
        sys.exit(3)
        
