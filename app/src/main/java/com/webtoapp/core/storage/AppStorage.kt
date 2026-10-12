package com.webtoapp.core.storage

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
