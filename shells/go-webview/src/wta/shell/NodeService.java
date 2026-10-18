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

import com.webtoapp.core.nodejs.NodeBridge;

import java.io.File;

public class NodeService extends Service {
    private static final String TAG = "NodeService";
    private static final String CHANNEL_ID = "wta_node";
    private static final int NOTIF_ID = 0x574E;

    static {
        try {
            System.loadLibrary("node_bridge");
            Log.i(TAG, "libnode_bridge.so loaded");
        } catch (UnsatisfiedLinkError e) {
            Log.e(TAG, "loadLibrary node_bridge failed: " + e.getMessage());
        }
    }

    private volatile boolean running = false;

    @Override
    public IBinder onBind(Intent intent) { return null; }

    @Override
    public void onCreate() {
        super.onCreate();
        createChannel();
    }

    private void createChannel() {
        if (Build.VERSION.SDK_INT >= 26) {
            NotificationManager nm = getSystemService(NotificationManager.class);
            if (nm != null) {
                NotificationChannel ch = new NotificationChannel(
                    CHANNEL_ID, "Node.js", NotificationManager.IMPORTANCE_LOW);
                nm.createNotificationChannel(ch);
            }
        }
    }

    private Notification buildNotification() {
        Notification.Builder b;
        if (Build.VERSION.SDK_INT >= 26) {
            b = new Notification.Builder(this, CHANNEL_ID);
        } else {
            b = new Notification.Builder(this);
        }
        return b.setContentTitle("Node.js")
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
        final String entryFile = intent.getStringExtra("entryFile");
        final int port = intent.getIntExtra("port", 19000);

        if (projectDir == null) {
            Log.e(TAG, "projectDir is null");
            stopSelf();
            return START_NOT_STICKY;
        }

        new Thread(() -> runNode(projectDir, entryFile, port), "NodeJS-Runtime").start();
        return START_STICKY;
    }

    private void runNode(String projectDir, String entryFile, int port) {
        try {
            String nodeSo = getApplicationInfo().nativeLibraryDir + "/libnode.so";
            Log.e(TAG, "loading libnode: " + nodeSo);
            if (!NodeBridge.nativeLoadNode(nodeSo)) {
                Log.e(TAG, "nativeLoadNode failed");
                return;
            }
            Log.e(TAG, "libnode loaded: " + nodeSo);

            File bootstrap = new File(projectDir, ".wta_bootstrap.cjs");
            String entryPath = new File(projectDir, entryFile).getAbsolutePath();
            String entryEscaped = entryPath.replace("\\", "\\\\").replace("\"", "\\\"");

            String bootstrapCode =
                "'use strict';\n" +
                "process.on('uncaughtException', function(e) { console.error('[wta] uncaughtException:', e && (e.stack || e.message) || e); });\n" +
                "process.on('unhandledRejection', function(r) { console.error('[wta] unhandledRejection:', r && (r.stack || r.message) || r); });\n" +
                "process.exit = function(c) { console.error('[wta] process.exit(' + c + ') intercepted'); };\n" +
                "setInterval(function() {}, 60000);\n" +
                "try { require(\"" + entryEscaped + "\"); } catch (e) { console.error('[wta] entry failed:', e && (e.stack || e.message) || e); }\n";
            writeFile(bootstrap, bootstrapCode);
            Log.e(TAG, "bootstrap: " + bootstrap.getAbsolutePath());

            setEnv("HOME", getFilesDir().getAbsolutePath());
            setEnv("TMPDIR", getCacheDir().getAbsolutePath());
            setEnv("NODE_ENV", "production");
            setEnv("HOST", "127.0.0.1");
            setEnv("NODE_PATH", new File(projectDir, "node_modules").getAbsolutePath());
            setEnv("PORT", String.valueOf(port));
            Log.e(TAG, "env set: PORT=" + port);

            String[] args = {"node", bootstrap.getAbsolutePath()};
            NodeBridge.OutputCallback cb = new NodeBridge.OutputCallback() {
                @Override public void onOutput(String line, boolean isError) {
                    Log.e("NodeJS", (isError ? "ERR " : "") + line);
                }
            };

            running = true;
            Log.e(TAG, "calling nativeStartNode...");
            int rc = NodeBridge.nativeStartNode(args, cb);
            running = false;
            Log.e(TAG, "nativeStartNode returned: " + rc);
        } catch (Throwable t) {
            Log.e(TAG, "NodeService fatal", t);
        }
    }

    private void writeFile(File f, String content) throws Exception {
        try (java.io.FileWriter w = new java.io.FileWriter(f)) {
            w.write(content);
        }
    }

    private void setEnv(String k, String v) {
        try {
            Os.setenv(k, v, true);
        } catch (Throwable t) {
            Log.w(TAG, "setenv " + k + " failed: " + t.getMessage());
        }
    }

    @Override
    public void onDestroy() {
        super.onDestroy();
        Log.e(TAG, "onDestroy");
    }
}
