package wta.shell;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.Service;
import android.content.Intent;
import android.os.Build;
import android.os.IBinder;
import android.system.Os;
import android.util.Log;

import java.io.BufferedReader;
import java.io.File;
import java.io.InputStreamReader;

/**
 * Runs a Go binary built with -buildmode=pie from assets/go_app/<binaryName>.
 *
 * Architecture (matches WebToApp GoRuntime.kt for targetSdk < 29):
 *   MainActivity extracts assets/go_app → filesDir/go_app
 *   → startForegroundService(GoService) on :goapp process
 *   → GoService.runGo:
 *       1. chmod 0755 (assets don't preserve exec bit)
 *       2. Os.setenv PORT/HOST/ADDR/HOME/TMPDIR/GIN_MODE
 *       3. ProcessBuilder(binaryPath) — plain exec (targetSdk 28: W^X off)
 *       4. Pipe stdout+stderr → Log.e("GoApp", ...)
 *   ← MainActivity polls http://127.0.0.1:PORT → WebView.loadUrl
 *
 * On targetSdk 29+, execve would be blocked and libgo_exec_loader.so
 * (memfd+execveat) would be needed. We use targetSdk 28 → plain exec works.
 */
public class GoService extends Service {
    private static final String TAG = "GoService";
    private static final String CHANNEL_ID = "wta_go";
    private static final int NOTIF_ID = 0x5747; // 'WG'

    private volatile boolean running = false;

    @Override public IBinder onBind(Intent intent) { return null; }

    @Override
    public void onCreate() {
        super.onCreate();
        createChannel();
    }

    private void createChannel() {
        if (Build.VERSION.SDK_INT >= 26) {
            NotificationManager nm = getSystemService(NotificationManager.class);
            if (nm != null) {
                nm.createNotificationChannel(new NotificationChannel(
                    CHANNEL_ID, "Go server", NotificationManager.IMPORTANCE_LOW));
            }
        }
    }

    private Notification buildNotification() {
        Notification.Builder b = Build.VERSION.SDK_INT >= 26
            ? new Notification.Builder(this, CHANNEL_ID)
            : new Notification.Builder(this);
        return b.setContentTitle("Go server")
                .setContentText("Running")
                .setSmallIcon(android.R.drawable.stat_notify_sync)
                .setOngoing(true)
                .build();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        try {
            if (Build.VERSION.SDK_INT >= 29) {
                startForeground(NOTIF_ID, buildNotification(),
                    android.content.pm.ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE);
            } else {
                startForeground(NOTIF_ID, buildNotification());
            }
        } catch (Throwable t) {
            Log.e(TAG, "startForeground failed", t);
        }

        final String projectDir = intent.getStringExtra("projectDir");
        final String binaryName = intent.getStringExtra("binaryName");
        final int port = intent.getIntExtra("port", 19000);

        if (projectDir == null || binaryName == null) {
            Log.e(TAG, "missing extras: projectDir=" + projectDir + " binaryName=" + binaryName);
            stopSelf();
            return START_NOT_STICKY;
        }

        new Thread(() -> runGo(projectDir, binaryName, port), "GoApp-Runtime").start();
        return START_STICKY;
    }

    private void runGo(String projectDir, String binaryName, int port) {
        try {
            String binaryPath = new File(projectDir, binaryName).getAbsolutePath();
            File binary = new File(binaryPath);
            if (!binary.exists()) {
                Log.e(TAG, "binary not found: " + binaryPath);
                return;
            }
            Log.e(TAG, "binary: " + binaryPath + " (" + binary.length() + " b)");

            // chmod 0755 — AssetManager doesn't preserve exec bit
            try {
                Os.chmod(binaryPath, 0755);
                Log.e(TAG, "chmod 0755 ok");
            } catch (Throwable t) {
                Log.w(TAG, "chmod failed: " + t.getMessage());
            }

            // Env (like Kotlin GoRuntime.startServer)
            try {
                Os.setenv("PORT", String.valueOf(port), true);
                Os.setenv("HOST", "127.0.0.1", true);
                Os.setenv("ADDR", "127.0.0.1:" + port, true);
                Os.setenv("HOME", getFilesDir().getAbsolutePath(), true);
                Os.setenv("TMPDIR", getCacheDir().getAbsolutePath(), true);
                Os.setenv("GIN_MODE", "release", true);
                Log.e(TAG, "env: PORT=" + port);
            } catch (Throwable t) {
                Log.w(TAG, "setenv failed: " + t.getMessage());
            }

            // Spawn: plain ProcessBuilder (targetSdk 28 → W^X off → works)
            ProcessBuilder pb = new ProcessBuilder(binaryPath);
            pb.directory(new File(projectDir));
            pb.redirectErrorStream(true);
            Process proc = pb.start();
            running = true;
            Log.e(TAG, "spawned: " + binaryPath);

            // Pipe stdout+stderr → logcat
            try (BufferedReader reader = new BufferedReader(
                    new InputStreamReader(proc.getInputStream()))) {
                String line;
                while ((line = reader.readLine()) != null) {
                    Log.e("GoApp", line);
                }
            }

            int exit = proc.waitFor();
            running = false;
            Log.e(TAG, "go exited: " + exit);
        } catch (Throwable t) {
            Log.e(TAG, "GoService fatal", t);
        }
    }

    @Override
    public void onDestroy() {
        super.onDestroy();
        Log.e(TAG, "onDestroy");
    }
}
