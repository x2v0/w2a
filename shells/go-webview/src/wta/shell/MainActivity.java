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
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.FrameLayout;

import org.json.JSONObject;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;

public class MainActivity extends Activity {
    private WebView webView;
    private JSONObject cfg;
    private View splash;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
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
        webView.setWebChromeClient(new WebChromeClient());
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

        String url = cfg.optString("url", "about:blank");
        webView.loadUrl(url);
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

    private void createWebViewAndLoad(String url) {
        FrameLayout root = new FrameLayout(this);
        webView = new WebView(this);
        WebSettings ws = webView.getSettings();
        ws.setJavaScriptEnabled(true);
        ws.setDomStorageEnabled(true);
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
