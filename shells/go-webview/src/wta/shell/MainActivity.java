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

        String url = cfg.optString("url", "about:blank");
        webView.loadUrl(url);
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
