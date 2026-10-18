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

import com.webtoapp.core.nodejs.NodeBridge;
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
        cfg = readConfig();

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

        String appType = cfg.optString("appType", "WEB");
        if ("NODEJS_APP".equals(appType)) {
            startNodeServer();
        } else {
            String url = cfg.optString("url", "about:blank");
            webView.loadUrl(url);
        }
    }

    private void startNodeServer() {
        String nodeSo = getApplicationInfo().nativeLibraryDir + "/libnode.so";
        java.io.File nodeFile = new java.io.File(nodeSo);
        if (!nodeFile.exists()) {
            android.util.Log.e("MainActivity", "libnode.so not found at " + nodeSo);
            webView.loadData("<h1>libnode.so missing</h1>", "text/html", "utf-8");
            return;
        }
        if (!NodeBridge.nativeLoadNode(nodeSo)) {
            android.util.Log.e("MainActivity", "nativeLoadNode failed");
            webView.loadData("<h1>Node load failed</h1>", "text/html", "utf-8");
            return;
        }

        // Read port + entryFile from nodejsConfig
        int port = 3000;
        String entryFile = "index.js";
        JSONObject nc = cfg.optJSONObject("nodejsConfig");
        if (nc != null) {
            port = nc.optInt("port", port);
            entryFile = nc.optString("entryFile", entryFile);
        }

        // Extract assets/nodejs_app → filesDir/nodejs_app
        String appDir;
        try {
            appDir = new java.io.File(getFilesDir(), "nodejs_app").getAbsolutePath();
            extractAssetsDir("nodejs_app", appDir);
        } catch (Exception e) {
            android.util.Log.e("MainActivity", "extract failed", e);
            webView.loadData("<h1>Extract failed</h1>", "text/html", "utf-8");
            return;
        }

        final String scriptPath = appDir + "/" + entryFile;
        final int finalPort = port;
        android.util.Log.i("MainActivity", "node: " + scriptPath + " port=" + finalPort);

        new Thread(() -> {
            try {
                int rc = NodeBridge.nativeStartNode(
                    new String[]{"node", scriptPath},
                    new NodeBridge.OutputCallback() {
                        @Override public void onOutput(String line, boolean isError) {
                            android.util.Log.i("NodeJS", (isError ? "ERR " : "") + line);
                        }
                    });
                android.util.Log.i("MainActivity", "nativeStartNode rc=" + rc);
            } catch (Throwable t) {
                android.util.Log.e("MainActivity", "nativeStartNode threw", t);
            }
        }, "node-launcher").start();

        // Poll until server responds, then WebView
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
                        runOnUiThread(() -> webView.loadUrl(url));
                        return;
                    }
                } catch (Exception ignored) {}
                try { Thread.sleep(500); } catch (InterruptedException ie) { return; }
            }
            runOnUiThread(() -> webView.loadData(
                "<h1>Node timeout</h1>", "text/html", "utf-8"));
        }, "node-readiness").start();
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
