package wta.shell;

import android.app.Activity;
import android.content.Intent;
import android.content.pm.ActivityInfo;
import android.graphics.Color;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.view.View;
import android.view.WindowManager;
import android.webkit.WebChromeClient;
import android.webkit.WebSettings;
import android.print.PrintManager;
import android.print.PrintDocumentAdapter;
import android.print.PrintAttributes;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.FrameLayout;

import org.json.JSONObject;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;

public class MainActivity extends Activity {
    private static final int REQ_STORAGE = 0x5331;
    private static final int REQ_MANAGE_STORAGE = 0x5332;

    private WebView webView;
    private JSONObject cfg;
    private View splash;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        requestStoragePermissions();
        android.util.Log.e("MainActivity", "onCreate started");
        cfg = readConfig();
        android.util.Log.e("MainActivity", "config loaded: appType=" + cfg.optString("appType", "WEB"));

        String orient = cfg.optString("orientation", "auto");
        if ("portrait".equals(orient)) {
            setRequestedOrientation(ActivityInfo.SCREEN_ORIENTATION_PORTRAIT);
        } else if ("landscape".equals(orient)) {
            setRequestedOrientation(ActivityInfo.SCREEN_ORIENTATION_LANDSCAPE);
        }

        if (cfg.optBoolean("hideStatusBar", false)) {
            getWindow().setFlags(
                WindowManager.LayoutParams.FLAG_FULLSCREEN,
                WindowManager.LayoutParams.FLAG_FULLSCREEN);
        }

        // Load Node.js FIRST (before WebView / Chromium init), to avoid
        // libc++ / SIGSEGV handler conflicts between V8 and Chromium.
        String appType = cfg.optString("appType", "WEB");
        if ("NODEJS_APP".equals(appType)) {
            // MUST setContentView BEFORE returning — Activity without content view
            // crashes HWUI render thread (SIGABRT pthread_mutex_lock).
            android.widget.TextView loading = new android.widget.TextView(this);
            loading.setText("Starting Node.js...");
            loading.setTextColor(0xFFFFFFFF);
            loading.setGravity(android.view.Gravity.CENTER);
            loading.setBackgroundColor(0xFF1A1A1A);
            setContentView(loading);

            startNodeServer();
            return;
        }

        if ("GO_APP".equals(appType)) {
            android.widget.TextView loading = new android.widget.TextView(this);
            loading.setText("Starting Go server...");
            loading.setTextColor(0xFFFFFFFF);
            loading.setGravity(android.view.Gravity.CENTER);
            loading.setBackgroundColor(0xFF1A1A1A);
            setContentView(loading);

            startGoServer();
            return;
        }

        FrameLayout root = new FrameLayout(this);
        webView = new WebView(this);
        WebSettings s = webView.getSettings();
        s.setJavaScriptEnabled(cfg.optBoolean("javaScriptEnabled", true));
        s.setDomStorageEnabled(cfg.optBoolean("domStorageEnabled", true));
        s.setBuiltInZoomControls(cfg.optBoolean("zoomEnabled", false));
        s.setDisplayZoomControls(false);
        s.setLoadWithOverviewMode(true);
        s.setUseWideViewPort(true);
        s.setJavaScriptCanOpenWindowsAutomatically(true);
        s.setMediaPlaybackRequiresUserGesture(false);
        s.setSupportMultipleWindows(false);
        // WTA_HTML_FIX: allow file access from file URLs (assets + fetch)
        try {
            s.setAllowFileAccessFromFileURLs(true);
            s.setAllowUniversalAccessFromFileURLs(true);
        } catch (Throwable ignored) {}
        // wta HTML fix: file:// access for <script src> + fetch from assets
        s.setAllowFileAccess(true);
        s.setAllowContentAccess(true);
        try {
            s.setAllowFileAccessFromFileURLs(true);
            s.setAllowUniversalAccessFromFileURLs(true);
        } catch (Throwable ignored) {}
        String ua = cfg.optString("userAgent", "");
        if (!ua.isEmpty()) s.setUserAgentString(ua);
        if (cfg.optBoolean("clearCacheOnLaunch", false)) {
            webView.clearCache(true);
            webView.clearHistory();
        }
        if (Build.VERSION.SDK_INT >= 29) {
            String dm = cfg.optString("darkMode", "system");
            if ("dark".equals(dm)) s.setForceDark(WebSettings.FORCE_DARK_ON);
            else if ("light".equals(dm)) s.setForceDark(WebSettings.FORCE_DARK_OFF);
        }

        webView.setWebViewClient(new WebViewClient() {
            @Override
            public boolean shouldOverrideUrlLoading(WebView v, String url) {
                return handleUrl(url);
            }
            @Override
            public void onPageFinished(WebView v, String url) {
                String css = cfg.optString("injectCss", "");
                String js = cfg.optString("injectJs", "");
                if (!css.isEmpty()) {
                    try {
                        String safe = JSONObject.quote(css);
                        v.evaluateJavascript(
                            "(function(){var st=document.createElement('style');" +
                            "st.textContent=" + safe + ";" +
                            "(document.head||document.documentElement).appendChild(st);})();",
                            null);
                    } catch (Exception ignored) {}
                }
                if (!js.isEmpty()) v.evaluateJavascript(js, null);
                hideSplash();
            }
        });
        webView.setWebChromeClient(new WebChromeClient() {
            @Override
            public boolean onConsoleMessage(android.webkit.ConsoleMessage cm) {
                android.util.Log.i("WebConsole", cm.message()
                    + " @ " + cm.sourceId() + ":" + cm.lineNumber());
                return true;
            }
        });
        
        // WTA_PRINT_BRIDGE: нативная печать
        webView.addJavascriptInterface(new Object() {
            @android.webkit.JavascriptInterface
            public void print() {
                runOnUiThread(new Runnable() {
                    @Override
                    public void run() {
                        PrintManager printManager = (PrintManager) getSystemService(android.content.Context.PRINT_SERVICE);
                        String jobName = getString(android.R.string.untitled) + " Document";
                        PrintDocumentAdapter printAdapter = webView.createPrintDocumentAdapter(jobName);
                        printManager.print(jobName, printAdapter, new PrintAttributes.Builder().build());
                    }
                });
            }
        }, "AndroidPrint");

        root.addView(webView, new FrameLayout.LayoutParams(-1, -1));

        JSONObject splashCfg = cfg.optJSONObject("splash");
        if (splashCfg != null && splashCfg.optBoolean("enabled", false)) {
            FrameLayout sl = new FrameLayout(this);
            try {
                sl.setBackgroundColor(Color.parseColor(
                    splashCfg.optString("color", "#1a1a1a")));
            } catch (Exception e) {
                sl.setBackgroundColor(0xFF1A1A1A);
            }
            root.addView(sl, new FrameLayout.LayoutParams(-1, -1));
            splash = sl;
            long dur = splashCfg.optLong("duration", 1000);
            if (dur > 0) sl.postDelayed(this::hideSplash, dur);
        }

        setContentView(root);

        // WTA_LOGCAT_BRIDGE: expose logcat (own process) to JS
        webView.addJavascriptInterface(new Object() {
            @android.webkit.JavascriptInterface
            public String getLogcat(int lines) {
                try {
                    Process p = Runtime.getRuntime().exec(
                        new String[]{"logcat", "-d", "-t", String.valueOf(lines),
                                     "--pid=" + android.os.Process.myPid()});
                    java.io.BufferedReader r = new java.io.BufferedReader(
                        new java.io.InputStreamReader(p.getInputStream()));
                    StringBuilder sb = new StringBuilder();
                    String ln;
                    int count = 0;
                    while ((ln = r.readLine()) != null && count < 500) {
                        sb.append(ln).append('\n');
                        count++;
                    }
                    r.close();
                    return sb.toString();
                } catch (Exception e) {
                    return "[logcat error] " + e.getMessage();
                }
            }
        }, "AndroidLogcat");

        String url = cfg.optString("url", "about:blank");
        // wta HTML fix: for appType=HTML/FRONTEND use htmlConfig.entryFile from assets/html/
        if ("about:blank".equals(url) && ("HTML".equals(appType) || "FRONTEND".equals(appType))) {
            String entryFile = "index.html";
            org.json.JSONObject hc = cfg.optJSONObject("htmlConfig");
            if (hc != null) entryFile = hc.optString("entryFile", entryFile);
            url = "file:///android_asset/html/" + entryFile;
            android.util.Log.i("MainActivity", "HTML mode: loading " + url);
        }
        webView.loadUrl(url);
    }

    private void requestStoragePermissions() {
        if (Build.VERSION.SDK_INT < 23) return; // API < 23 — permissions at install

        // targetSdk < 30: READ/WRITE_EXTERNAL_STORAGE работают через legacy storage
        // targetSdk >= 30: нужен MANAGE_EXTERNAL_STORAGE
        if (Build.VERSION.SDK_INT >= 30) {
            // Все файлы (Android 11+)
            if (!android.os.Environment.isExternalStorageManager()) {
                try {
                    android.content.Intent intent = new android.content.Intent(
                        android.provider.Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION);
                    intent.setData(android.net.Uri.parse("package:" + getPackageName()));
                    startActivityForResult(intent, REQ_MANAGE_STORAGE);
                    android.util.Log.e("MainActivity", "requested MANAGE_EXTERNAL_STORAGE");
                } catch (Throwable t) {
                    android.util.Log.e("MainActivity", "manage storage intent failed", t);
                    // fallback на обычные runtime permissions
                    requestRuntimeStorage();
                }
            } else {
                android.util.Log.e("MainActivity", "MANAGE_EXTERNAL_STORAGE already granted");
            }
        } else {
            // API 23-29: обычные runtime permissions
            requestRuntimeStorage();
        }
    }

    private void requestRuntimeStorage() {
        java.util.List<String> perms = new java.util.ArrayList<>();
        if (checkSelfPermission(android.Manifest.permission.READ_EXTERNAL_STORAGE)
                != android.content.pm.PackageManager.PERMISSION_GRANTED) {
            perms.add(android.Manifest.permission.READ_EXTERNAL_STORAGE);
        }
        if (checkSelfPermission(android.Manifest.permission.WRITE_EXTERNAL_STORAGE)
                != android.content.pm.PackageManager.PERMISSION_GRANTED) {
            perms.add(android.Manifest.permission.WRITE_EXTERNAL_STORAGE);
        }
        if (!perms.isEmpty()) {
            android.util.Log.e("MainActivity", "requesting runtime: " + perms);
            requestPermissions(perms.toArray(new String[0]), REQ_STORAGE);
        } else {
            android.util.Log.e("MainActivity", "runtime storage already granted");
        }
    }

    @Override
    public void onRequestPermissionsResult(int req, String[] perms, int[] results) {
        super.onRequestPermissionsResult(req, perms, results);
        if (req == REQ_STORAGE) {
            boolean allGranted = true;
            for (int r : results) {
                if (r != android.content.pm.PackageManager.PERMISSION_GRANTED) {
                    allGranted = false;
                    break;
                }
            }
            android.util.Log.e("MainActivity", "storage perms: granted=" + allGranted + " results=" + java.util.Arrays.toString(results));
        }
    }

    @Override
    protected void onActivityResult(int req, int res, android.content.Intent data) {
        super.onActivityResult(req, res, data);
        if (req == REQ_MANAGE_STORAGE) {
            boolean granted = android.os.Environment.isExternalStorageManager();
            android.util.Log.e("MainActivity", "MANAGE_EXTERNAL_STORAGE result: " + granted);
            if (!granted) {
                // fallback — попробовать обычные runtime
                requestRuntimeStorage();
            }
        }
    }

    private void startNodeServer() {
        // Android Service on :nodejs process (fork from Zygote, W^X not blocked).
        // NodeBridge.nativeStartNode runs JNI in-process of :nodejs.
        // HWUI lives in the parent process, no libc++ conflict.
        android.util.Log.e("MainActivity", "startNodeServer: launching NodeService");

        // 1) Extract assets → filesDir/nodejs_app
        final String appDir;
        try {
            appDir = new java.io.File(getFilesDir(), "nodejs_app").getAbsolutePath();
            android.util.Log.e("MainActivity", "extract start: " + appDir);
            extractAssetsDir("nodejs_app", appDir);
            android.util.Log.e("MainActivity", "extract done");
        } catch (Exception e) {
            android.util.Log.e("MainActivity", "extract failed", e);
            showText("Extract failed");
            return;
        }

        // 2) Read config
        int port = 19000;
        String entryFile = "index.js";
        JSONObject nc = cfg.optJSONObject("nodejsConfig");
        if (nc != null) {
            port = nc.optInt("port", port);
            entryFile = nc.optString("entryFile", entryFile);
        }
        if (port <= 0) port = 19000;
        final int finalPort = port;
        final String finalEntryFile = entryFile;
        android.util.Log.e("MainActivity", "config: port=" + finalPort + " entry=" + finalEntryFile);

        // 3) Start NodeService (:nodejs process)
        Intent svc = new Intent(this, NodeService.class);
        svc.putExtra("projectDir", appDir);
        svc.putExtra("entryFile", finalEntryFile);
        svc.putExtra("port", finalPort);
        if (Build.VERSION.SDK_INT >= 26) {
            startForegroundService(svc);
        } else {
            startService(svc);
        }
        android.util.Log.e("MainActivity", "NodeService started");

        // 4) Poll HTTP → WebView
        new Thread(() -> {
            String url = "http://127.0.0.1:" + finalPort + "/";
            for (int i = 0; i < 120; i++) {
                try {
                    java.net.HttpURLConnection c = (java.net.HttpURLConnection)
                        new java.net.URL(url).openConnection();
                    c.setConnectTimeout(200);
                    c.setReadTimeout(200);
                    c.connect();
                    int code = c.getResponseCode();
                    c.disconnect();
                    if (code > 0) {
                        android.util.Log.e("MainActivity", "server ready on " + finalPort);
                        runOnUiThread(() -> createWebViewAndLoad(url));
                        return;
                    }
                } catch (Exception ignored) {}
                try { Thread.sleep(500); } catch (InterruptedException ie) { return; }
            }
            runOnUiThread(() -> showText("Node timeout"));
        }, "node-readiness").start();
    }

    private void startGoServer() {
        // Android Service on :goapp process — plain exec, targetSdk 28.
        // (No JNI: ProcessBuilder(binaryPath) → execve → works because W^X off.)
        android.util.Log.e("MainActivity", "startGoServer: launching GoService");

        JSONObject gc = cfg.optJSONObject("goAppConfig");
        if (gc == null) {
            android.util.Log.e("MainActivity", "goAppConfig missing in app_config.json");
            showText("goAppConfig missing");
            return;
        }

        String binaryName = gc.optString("binaryName", "");
        int port = gc.optInt("port", 0);
        if (port <= 0) port = 19000;
        if (binaryName.isEmpty()) {
            android.util.Log.e("MainActivity", "binaryName empty");
            showText("binaryName empty");
            return;
        }
        final int finalPort = port;
        final String finalBinaryName = binaryName;
        android.util.Log.e("MainActivity", "go: binary=" + finalBinaryName + " port=" + finalPort);

        final String goDir = new java.io.File(getFilesDir(), "go_app").getAbsolutePath();

        new Thread(() -> {
            try {
                // 1) Extract assets/go_app → filesDir/go_app
                android.util.Log.e("MainActivity", "extract go_app: " + goDir);
                extractAssetsDir("go_app", goDir);
                android.util.Log.e("MainActivity", "extract done");

                // 2) Start GoService (:goapp process)
                Intent svc = new Intent(this, GoService.class);
                svc.putExtra("projectDir", goDir);
                svc.putExtra("binaryName", finalBinaryName);
                svc.putExtra("port", finalPort);
                if (Build.VERSION.SDK_INT >= 26) {
                    startForegroundService(svc);
                } else {
                    startService(svc);
                }
                android.util.Log.e("MainActivity", "GoService started");

                // 3) Poll HTTP → WebView
                String url = "http://127.0.0.1:" + finalPort + "/";
                for (int i = 0; i < 120; i++) {
                    try {
                        java.net.HttpURLConnection c = (java.net.HttpURLConnection)
                            new java.net.URL(url).openConnection();
                        c.setConnectTimeout(200);
                        c.setReadTimeout(200);
                        c.connect();
                        int code = c.getResponseCode();
                        c.disconnect();
                        if (code > 0) {
                            android.util.Log.e("MainActivity", "go server ready on " + finalPort);
                            runOnUiThread(() -> createWebViewAndLoad(url));
                            return;
                        }
                    } catch (Exception ignored) {}
                    try { Thread.sleep(500); } catch (InterruptedException ie) { return; }
                }
                runOnUiThread(() -> showText("Go timeout"));
            } catch (Throwable t) {
                android.util.Log.e("MainActivity", "startGoServer error", t);
            }
        }, "go-extract").start();
    }

    private void createWebViewAndLoad(String url) {
        FrameLayout root = new FrameLayout(this);
        webView = new WebView(this);
        WebSettings ws = webView.getSettings();
        ws.setJavaScriptEnabled(true);
        ws.setDomStorageEnabled(true);
        
        // WTA_PRINT_BRIDGE: нативная печать
        webView.addJavascriptInterface(new Object() {
            @android.webkit.JavascriptInterface
            public void print() {
                runOnUiThread(new Runnable() {
                    @Override
                    public void run() {
                        PrintManager printManager = (PrintManager) getSystemService(android.content.Context.PRINT_SERVICE);
                        String jobName = getString(android.R.string.untitled) + " Document";
                        PrintDocumentAdapter printAdapter = webView.createPrintDocumentAdapter(jobName);
                        printManager.print(jobName, printAdapter, new PrintAttributes.Builder().build());
                    }
                });
            }
        }, "AndroidPrint");

        root.addView(webView, new FrameLayout.LayoutParams(-1, -1));
        setContentView(root);
        webView.loadUrl(url);
    }

    private void showText(String msg) {
        android.widget.TextView tv = new android.widget.TextView(this);
        tv.setText(msg);
        tv.setTextColor(0xFFFFFFFF);
        tv.setGravity(android.view.Gravity.CENTER);
        tv.setBackgroundColor(0xFF1A1A1A);
        setContentView(tv);
    }

    private void extractAssetsDir(String assetDir, String targetDir) throws java.io.IOException {
        java.io.File root = new java.io.File(targetDir);
        root.mkdirs();
        String[] items = getAssets().list(assetDir);
        if (items == null || items.length == 0) return;
        for (String item : items) {
            String sub = assetDir + "/" + item;
            String[] child = getAssets().list(sub);
            if (child != null && child.length > 0) {
                extractAssetsDir(sub, targetDir + "/" + item);
            } else {
                java.io.File out = new java.io.File(root, item);
                try (java.io.InputStream is = getAssets().open(sub);
                     java.io.OutputStream os = new java.io.FileOutputStream(out)) {
                    byte[] buf = new byte[8192];
                    int n;
                    while ((n = is.read(buf)) > 0) os.write(buf, 0, n);
                }
            }
        }
    }

    private void hideSplash() {
        if (splash == null) return;
        splash.animate().alpha(0f).setDuration(200).withEndAction(() -> {
            if (splash != null && splash.getParent() instanceof FrameLayout) {
                ((FrameLayout) splash.getParent()).removeView(splash);
            }
            splash = null;
        });
    }

    private boolean handleUrl(String url) {
        String mode = cfg.optString("externalLinks", "browser");
        String base = cfg.optString("url", "");
        String baseHost = Uri.parse(base).getHost();
        String newHost = Uri.parse(url).getHost();

        if (baseHost != null && baseHost.equals(newHost)) return false;
        if ("webview".equals(mode)) return false;
        if ("browser".equals(mode) || "ask".equals(mode)) {
            try {
                startActivity(new Intent(Intent.ACTION_VIEW, Uri.parse(url)));
            } catch (Exception ignored) {}
            return true;
        }
        return false;
    }

    private JSONObject readConfig() {
        try {
            InputStream is = getAssets().open("app_config.json");
            ByteArrayOutputStream bos = new ByteArrayOutputStream();
            byte[] buf = new byte[4096];
            int n;
            while ((n = is.read(buf)) > 0) bos.write(buf, 0, n);
            is.close();
            return new JSONObject(bos.toString("UTF-8"));
        } catch (Exception e) {
            try {
                JSONObject fb = new JSONObject();
                fb.put("url", "about:blank");
                return fb;
            } catch (Exception ignored) {
                return new JSONObject();
            }
        }
    }

    @Override
    public void onBackPressed() {
        if (webView != null && webView.canGoBack()) {
            webView.goBack();
        } else {
            super.onBackPressed();
        }
    }
}
