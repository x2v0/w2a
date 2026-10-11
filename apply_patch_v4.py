#!/usr/bin/env python3
"""
Patch v4 — precise. Dialog goes INSIDE FileManagerScreen, right after LaunchedEffect.

Usage: python3 apply_patch_v4.py
"""
import re, shutil, sys
from pathlib import Path

REPO = Path(__file__).resolve().parent
TARGET = REPO / "app/src/main/java/com/webtoapp/ui/screens/FileManagerScreen.kt"
MARKER = "// WTA_PATCH_FM_V4"

IMPORTS = """import androidx.compose.material.icons.outlined.Download
import androidx.compose.material3.OutlinedTextField
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.foundation.text.KeyboardOptions
import java.net.URL
"""

HELPERS = """
// WTA_PATCH_FM_V4_HELPERS

private fun appFilesRoot(context: Context): File =
    (context.getExternalFilesDir(android.os.Environment.DIRECTORY_DOWNLOADS)
        ?: context.filesDir)
        .resolve("MyFiles")
        .also { it.mkdirs() }

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

BTN = """                IconButton(onClick = { showDownloadDialog = true }) {
                    Icon(Icons.Outlined.Download, contentDescription = "Download")
                }
"""

DIALOG = """    if (showDownloadDialog) {
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

FN = """private suspend fun downloadFromUrl(url: String, targetDir: File): Result<File> =
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
            val entries = if (!dir.exists()) emptyList()
            else dir.listFiles { f -> f.isFile }?.map { f ->
                FileEntry(f, f.name, f.length(), f.lastModified(), section)
            } ?: emptyList()
            SectionSnapshot(section, entries)
        }
    }
}
"""

SCAN_USER = """private fun scanUserProjects(root: File): SectionSnapshot {
    val entries = USER_PROJECT_DIRS.flatMap { projectDirName ->
        val projectDir = File(root, projectDirName)
        if (!projectDir.exists() || !projectDir.isDirectory) return@flatMap emptyList()
        projectDir.listFiles { f -> f.isDirectory }?.map { sub ->
            FileEntry(sub, "$projectDirName/${sub.name}", dirSize(sub),
                      sub.lastModified(), FileSection.USER_PROJECTS)
        } ?: emptyList()
    }
    return SectionSnapshot(FileSection.USER_PROJECTS, entries)
}
"""

CLEAR = """private fun clearSection(context: Context, section: FileSection) {
    val root = appFilesRoot(context)
    if (section.isUserProjects) {
        USER_PROJECT_DIRS.forEach { dirName ->
            File(root, dirName).listFiles { f -> f.isDirectory }
                ?.forEach { sub -> sub.deleteRecursively() }
        }
    } else {
        File(root, section.dirName).listFiles()?.forEach { f ->
            if (f.isDirectory) f.deleteRecursively() else f.delete()
        }
    }
}
"""

def main() -> int:
    if not TARGET.exists():
        print(f"ERROR: {TARGET} not found"); return 1
    src = TARGET.read_text(encoding="utf-8")
    if MARKER in src:
        print("Already patched (v4)."); return 0
    if "WTA_PATCH_FM" in src:
        print("ERROR: file has older patch marker. Revert first:"); 
        print("  git checkout 1caafa5 -- " + str(TARGET.relative_to(REPO)))
        return 1

    # Critical anchors
    a_btn = ("            } else {\n"
             "                IconButton(\n"
             "                    onClick = { scope.launch { rescan() } },\n"
             "                    enabled = !loading\n"
             "                ) {\n")
    a_le_old = "    LaunchedEffect(Unit) { rescan() }"
    a_le_new_hook = ("    LaunchedEffect(Unit) {\n"
                     "        withContext(Dispatchers.IO) { migrateLegacyStorageIfNeeded(context) }\n"
                     "        rescan()\n"
                     "    }")

    for name, needle in [
        ("TopAppBar rescan button", a_btn),
        ("LaunchedEffect anchor", a_le_old),
        ("USER_PROJECTS enum", '    USER_PROJECTS("");'),
        ("sectionLabel", "private fun sectionLabel(section: FileSection): String = when (section) {\n"),
        ("shareFile", "private fun shareFile("),
        ("SummaryCard", "@Composable\nprivate fun SummaryCard("),
        ("MAX_LOG_CHARS", "private const val MAX_LOG_CHARS"),
        ("scanAllSections", "private fun scanAllSections(context: Context): List<SectionSnapshot> {"),
        ("scanUserProjects", "private fun scanUserProjects(filesBase: File): SectionSnapshot {"),
        ("clearSection", "private fun clearSection(context: Context, section: FileSection) {"),
        ("Locale import", "import java.util.Locale\n"),
        ("showSortMenu state", "    var showSortMenu by remember { mutableStateOf(false) }\n"),
    ]:
        if needle not in src:
            print(f"ABORT: anchor not found: {name}"); return 2
    print("All anchors OK.")

    backup = TARGET.with_suffix(".kt.v4bak")
    shutil.copy2(TARGET, backup); print(f"Backup: {backup}")

    # 1. imports
    src = src.replace("import java.util.Locale\n",
                      "import java.util.Locale\n" + IMPORTS, 1)

    # 2. helpers before MAX_LOG_CHARS
    src = src.replace("private const val MAX_LOG_CHARS",
                      HELPERS + "\nprivate const val MAX_LOG_CHARS", 1)

    # 3. DOWNLOADS enum
    src = src.replace('    USER_PROJECTS("");',
                      '    DOWNLOADS("downloads"),\n    USER_PROJECTS("");', 1)

    # 4. state vars
    src = src.replace(
        "    var showSortMenu by remember { mutableStateOf(false) }\n",
        "    var showSortMenu by remember { mutableStateOf(false) }\n"
        "    var showDownloadDialog by remember { mutableStateOf(false) }\n"
        "    var downloadUrl by remember { mutableStateOf(\"\") }\n"
        "    var downloading by remember { mutableStateOf(false) }\n", 1)

    # 5. button in TopAppBar (before Refresh IconButton)
    src = src.replace(a_btn, BTN + a_btn, 1)

    # 6. migration hook + DIALOG right after it (guaranteed inside function body)
    new_le = a_le_new_hook + "\n\n" + DIALOG
    src = src.replace(a_le_old, new_le, 1)

    # 7. scan/clear
    for pat, body, name in [
        (r"^private fun scanAllSections\(context: Context\): List<SectionSnapshot> \{.*?^\}",
         SCAN_ALL, "scanAllSections"),
        (r"^private fun scanUserProjects\(filesBase: File\): SectionSnapshot \{.*?^\}",
         SCAN_USER, "scanUserProjects"),
        (r"^private fun clearSection\(context: Context, section: FileSection\) \{.*?^\}",
         CLEAR, "clearSection"),
    ]:
        src, n = re.subn(pat, body.rstrip() + "\n", src, count=1, flags=re.DOTALL | re.MULTILINE)
        if n != 1:
            print(f"ABORT: {name} replace failed"); return 3

    # 8. downloadFromUrl before shareFile
    src = src.replace("private fun shareFile(", FN + "\nprivate fun shareFile(", 1)

    # 9. sectionLabel branch
    src = src.replace(
        "private fun sectionLabel(section: FileSection): String = when (section) {\n",
        "private fun sectionLabel(section: FileSection): String = when (section) {\n"
        '    FileSection.DOWNLOADS -> "Downloads"\n', 1)

    # marker
    src = MARKER + "\n" + src

    TARGET.write_text(src, encoding="utf-8")
    print(f"\nWrote: {TARGET}")
    print(f"Backup: {backup}")
    return 0

if __name__ == "__main__":
    sys.exit(main())
    
