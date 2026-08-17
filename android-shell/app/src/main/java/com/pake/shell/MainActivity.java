package com.pake.shell;

import android.Manifest;
import android.annotation.SuppressLint;
import android.app.Activity;
import android.app.DownloadManager;
import android.content.ActivityNotFoundException;
import android.content.ContentValues;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.os.Environment;
import android.provider.MediaStore;
import android.util.Base64;
import android.view.View;
import android.view.ViewGroup;
import android.view.WindowManager;
import android.webkit.CookieManager;
import android.webkit.DownloadListener;
import android.webkit.JavascriptInterface;
import android.webkit.ValueCallback;
import android.webkit.WebChromeClient;
import android.webkit.WebResourceRequest;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.FrameLayout;
import android.widget.ProgressBar;
import android.widget.Toast;

import androidx.core.content.ContextCompat;
import androidx.core.content.FileProvider;
import androidx.core.graphics.Insets;
import androidx.core.view.ViewCompat;
import androidx.core.view.WindowCompat;
import androidx.core.view.WindowInsetsCompat;
import androidx.core.view.WindowInsetsControllerCompat;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;

/**
 * Generic site WebView shell. Options come from BuildConfig (injected by CI).
 */
public class MainActivity extends Activity {
    private static final int REQ_FILE = 1001;
    private static final int REQ_NOTIF = 1002;

    private WebView webView;
    private ProgressBar progressBar;
    private SwipeRefreshLayout swipe;
    private ValueCallback<Uri[]> filePathCallback;
    private Uri cameraOutputUri;
    private int safeTopPx;
    private int safeBottomPx;
    private int safeLeftPx;
    private int safeRightPx;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
        webView = findViewById(R.id.webview);
        progressBar = findViewById(R.id.progress);
        swipe = findViewById(R.id.swipe);
        applyDisplayCutoutAndInsets();
        if (!BuildConfig.PULL_REFRESH) {
            swipe.setEnabled(false);
        } else {
            swipe.setOnRefreshListener(() -> webView.reload());
        }
        if (!BuildConfig.PROGRESS_BAR) {
            progressBar.setVisibility(View.GONE);
        }
        if (BuildConfig.PUSH_PLACEHOLDER) {
            AppNotifications.ensureChannel(this);
        }
        configureWebView();
        webView.loadUrl(BuildConfig.START_URL);
    }

    private void applyDisplayCutoutAndInsets() {
        View root = findViewById(R.id.root);
        WindowCompat.setDecorFitsSystemWindows(getWindow(), false);

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            WindowManager.LayoutParams lp = getWindow().getAttributes();
            lp.layoutInDisplayCutoutMode =
                    WindowManager.LayoutParams.LAYOUT_IN_DISPLAY_CUTOUT_MODE_SHORT_EDGES;
            getWindow().setAttributes(lp);
        }

        if (BuildConfig.FULLSCREEN) {
            getWindow().addFlags(WindowManager.LayoutParams.FLAG_FULLSCREEN);
            WindowInsetsControllerCompat controller =
                    WindowCompat.getInsetsController(getWindow(), root);
            if (controller != null) {
                controller.hide(WindowInsetsCompat.Type.statusBars()
                        | WindowInsetsCompat.Type.navigationBars());
                controller.setSystemBarsBehavior(
                        WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE);
            }
        }

        ViewCompat.setOnApplyWindowInsetsListener(root, (v, insets) -> {
            Insets bars = insets.getInsets(
                    WindowInsetsCompat.Type.systemBars() | WindowInsetsCompat.Type.displayCutout());
            safeTopPx = bars.top;
            safeBottomPx = bars.bottom;
            safeLeftPx = bars.left;
            safeRightPx = bars.right;
            if (BuildConfig.FULLSCREEN) {
                // Edge-to-edge: H5 uses CSS vars; keep a small top offset for the progress bar.
                v.setPadding(0, 0, 0, 0);
                updateProgressOffset(safeTopPx);
            } else {
                v.setPadding(bars.left, bars.top, bars.right, bars.bottom);
                updateProgressOffset(0);
            }
            return insets;
        });
        ViewCompat.requestApplyInsets(root);
    }

    private void updateProgressOffset(int topPx) {
        if (progressBar == null) {
            return;
        }
        ViewGroup.LayoutParams lp = progressBar.getLayoutParams();
        if (lp instanceof FrameLayout.LayoutParams) {
            FrameLayout.LayoutParams flp = (FrameLayout.LayoutParams) lp;
            flp.topMargin = Math.max(0, topPx);
            progressBar.setLayoutParams(flp);
        }
    }

    @SuppressLint({"SetJavaScriptEnabled", "AddJavascriptInterface"})
    private void configureWebView() {
        if (BuildConfig.DEBUG) {
            WebView.setWebContentsDebuggingEnabled(true);
        }

        WebSettings s = webView.getSettings();
        s.setJavaScriptEnabled(true);
        s.setDomStorageEnabled(true);
        s.setDatabaseEnabled(true);
        s.setLoadWithOverviewMode(true);
        s.setUseWideViewPort(true);
        s.setSupportZoom(true);
        s.setBuiltInZoomControls(true);
        s.setDisplayZoomControls(false);
        s.setMediaPlaybackRequiresUserGesture(false);
        s.setAllowFileAccess(true);
        s.setAllowContentAccess(true);
        s.setMixedContentMode(WebSettings.MIXED_CONTENT_COMPATIBILITY_MODE);
        if (BuildConfig.USER_AGENT != null && !BuildConfig.USER_AGENT.trim().isEmpty()) {
            s.setUserAgentString(BuildConfig.USER_AGENT.trim());
        }

        CookieManager cookies = CookieManager.getInstance();
        cookies.setAcceptCookie(true);
        cookies.setAcceptThirdPartyCookies(webView, true);

        // Always expose native helpers (share / safe-area). Push APIs honor BuildConfig.
        webView.addJavascriptInterface(new PakeBridge(), "PakeAndroid");
        if (BuildConfig.ENABLE_DOWNLOAD) {
            webView.addJavascriptInterface(new DownloadBridge(), "PakeDownload");
        }

        webView.setWebViewClient(new WebViewClient() {
            @Override
            public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest request) {
                return handleUri(request.getUrl());
            }

            @Override
            @SuppressWarnings("deprecation")
            public boolean shouldOverrideUrlLoading(WebView view, String url) {
                return handleUri(Uri.parse(url));
            }

            @Override
            public void onPageFinished(WebView view, String url) {
                if (swipe != null) {
                    swipe.setRefreshing(false);
                }
                injectSafeAreaCss();
                injectAssets();
            }
        });

        webView.setWebChromeClient(new WebChromeClient() {
            @Override
            public void onProgressChanged(WebView view, int newProgress) {
                if (!BuildConfig.PROGRESS_BAR || progressBar == null) {
                    return;
                }
                if (newProgress >= 100) {
                    progressBar.setVisibility(View.GONE);
                } else {
                    progressBar.setVisibility(View.VISIBLE);
                    progressBar.setProgress(newProgress);
                }
            }

            @Override
            public boolean onShowFileChooser(WebView webView, ValueCallback<Uri[]> callback,
                                             FileChooserParams params) {
                if (!BuildConfig.ENABLE_FILE_UPLOAD) {
                    if (callback != null) {
                        callback.onReceiveValue(null);
                    }
                    return false;
                }
                if (filePathCallback != null) {
                    filePathCallback.onReceiveValue(null);
                }
                filePathCallback = callback;
                return startFileChooser(params);
            }
        });

        webView.setDownloadListener(new DownloadListener() {
            @Override
            public void onDownloadStart(String url, String userAgent, String contentDisposition,
                                        String mimeType, long contentLength) {
                if (BuildConfig.ENABLE_DOWNLOAD) {
                    enqueueDownload(url, userAgent, contentDisposition, mimeType);
                } else {
                    openExternal(Uri.parse(url));
                }
            }
        });
    }

    private boolean startFileChooser(WebChromeClient.FileChooserParams params) {
        Intent content = new Intent(Intent.ACTION_GET_CONTENT);
        content.addCategory(Intent.CATEGORY_OPENABLE);
        content.setType("*/*");
        content.putExtra(Intent.EXTRA_ALLOW_MULTIPLE, params != null && params.getMode()
                == WebChromeClient.FileChooserParams.MODE_OPEN_MULTIPLE);

        Intent chooser = Intent.createChooser(content, "选择文件");
        if (BuildConfig.ENABLE_CAMERA) {
            try {
                File photo = new File(getCacheDir(), "capture_" + System.currentTimeMillis() + ".jpg");
                cameraOutputUri = FileProvider.getUriForFile(this, getPackageName() + ".fileprovider", photo);
                Intent capture = new Intent(android.provider.MediaStore.ACTION_IMAGE_CAPTURE);
                capture.putExtra(android.provider.MediaStore.EXTRA_OUTPUT, cameraOutputUri);
                capture.addFlags(Intent.FLAG_GRANT_WRITE_URI_PERMISSION | Intent.FLAG_GRANT_READ_URI_PERMISSION);
                chooser.putExtra(Intent.EXTRA_INITIAL_INTENTS, new Intent[]{capture});
            } catch (Exception ignored) {
                cameraOutputUri = null;
            }
        }
        try {
            startActivityForResult(chooser, REQ_FILE);
            return true;
        } catch (ActivityNotFoundException e) {
            if (filePathCallback != null) {
                filePathCallback.onReceiveValue(null);
                filePathCallback = null;
            }
            return false;
        }
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);
        if (requestCode != REQ_FILE || filePathCallback == null) {
            return;
        }
        Uri[] result = null;
        if (resultCode == RESULT_OK) {
            if (data != null && data.getClipData() != null) {
                int n = data.getClipData().getItemCount();
                result = new Uri[n];
                for (int i = 0; i < n; i++) {
                    result[i] = data.getClipData().getItemAt(i).getUri();
                }
            } else if (data != null && data.getData() != null) {
                result = new Uri[]{data.getData()};
            } else if (cameraOutputUri != null) {
                result = new Uri[]{cameraOutputUri};
            }
        }
        filePathCallback.onReceiveValue(result);
        filePathCallback = null;
        cameraOutputUri = null;
    }

    private void enqueueDownload(String url, String userAgent, String contentDisposition, String mimeType) {
        if (url == null || url.isEmpty()) {
            return;
        }
        if (url.startsWith("blob:")) {
            saveBlobViaJs(url, mimeType, contentDisposition);
            return;
        }
        if (url.startsWith("data:")) {
            saveDataUrl(url, mimeType, contentDisposition);
            return;
        }
        try {
            String name = DownloadNames.resolve(url, contentDisposition, mimeType);
            File downloads = Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_DOWNLOADS);
            name = DownloadNames.uniqueIn(downloads, name);

            DownloadManager.Request req = new DownloadManager.Request(Uri.parse(url));
            if (mimeType != null && !mimeType.trim().isEmpty()) {
                req.setMimeType(mimeType);
            }
            String cookie = CookieManager.getInstance().getCookie(url);
            if (cookie != null) {
                req.addRequestHeader("Cookie", cookie);
            }
            if (userAgent != null && !userAgent.isEmpty()) {
                req.addRequestHeader("User-Agent", userAgent);
            }
            req.addRequestHeader("Referer", webView != null ? webView.getUrl() : url);
            req.setTitle(name);
            req.setDescription("正在保存到「下载」");
            req.setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED);
            req.setDestinationInExternalPublicDir(Environment.DIRECTORY_DOWNLOADS, name);
            DownloadManager dm = (DownloadManager) getSystemService(DOWNLOAD_SERVICE);
            if (dm == null) {
                openExternal(Uri.parse(url));
                return;
            }
            dm.enqueue(req);
            Toast.makeText(this, "已开始下载\n" + name, Toast.LENGTH_LONG).show();
        } catch (Exception e) {
            Toast.makeText(this, "下载失败，尝试用系统浏览器打开", Toast.LENGTH_SHORT).show();
            openExternal(Uri.parse(url));
        }
    }

    private void saveBlobViaJs(String blobUrl, String mimeType, String contentDisposition) {
        String safeUrl = jsonString(blobUrl);
        String safeMime = jsonString(mimeType == null ? "" : mimeType);
        String safeCd = jsonString(contentDisposition == null ? "" : contentDisposition);
        String js = "(function(){var u=" + safeUrl + ",m=" + safeMime + ",c=" + safeCd + ";"
                + "fetch(u).then(function(r){return r.blob()}).then(function(b){"
                + "var reader=new FileReader();"
                + "reader.onloadend=function(){if(window.PakeDownload){"
                + "PakeDownload.saveDataUrl(String(reader.result||''), m||b.type||'', c);"
                + "}};"
                + "reader.onerror=function(){if(window.PakeDownload){PakeDownload.fail('读取文件失败');}};"
                + "reader.readAsDataURL(b);"
                + "}).catch(function(e){if(window.PakeDownload){PakeDownload.fail(String(e&&e.message||e));}});"
                + "})();";
        webView.evaluateJavascript(js, null);
        Toast.makeText(this, "正在准备下载…", Toast.LENGTH_SHORT).show();
    }

    private void saveDataUrl(String dataUrl, String mimeHint, String contentDisposition) {
        try {
            int comma = dataUrl.indexOf(',');
            if (comma < 0) {
                throw new IllegalArgumentException("bad data url");
            }
            String meta = dataUrl.substring(5, comma); // after "data:"
            String payload = dataUrl.substring(comma + 1);
            String mime = mimeHint;
            if (mime == null || mime.trim().isEmpty()) {
                int semi = meta.indexOf(';');
                mime = semi >= 0 ? meta.substring(0, semi) : meta;
            }
            if (mime == null || mime.trim().isEmpty()) {
                mime = "application/octet-stream";
            }
            byte[] bytes;
            if (meta.toLowerCase(Locale.US).contains(";base64")) {
                bytes = Base64.decode(payload, Base64.DEFAULT);
            } else {
                bytes = URLDecoderCompat.decode(payload).getBytes(StandardCharsets.UTF_8);
            }
            String name = DownloadNames.resolve("download", contentDisposition, mime);

            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
                ContentValues values = new ContentValues();
                values.put(MediaStore.Downloads.DISPLAY_NAME, name);
                values.put(MediaStore.Downloads.MIME_TYPE, mime);
                values.put(MediaStore.Downloads.IS_PENDING, 1);
                Uri collection = MediaStore.Downloads.EXTERNAL_CONTENT_URI;
                Uri item = getContentResolver().insert(collection, values);
                if (item == null) {
                    throw new IllegalStateException("MediaStore insert failed");
                }
                try (java.io.OutputStream os = getContentResolver().openOutputStream(item)) {
                    if (os == null) {
                        throw new IllegalStateException("openOutputStream failed");
                    }
                    os.write(bytes);
                }
                values.clear();
                values.put(MediaStore.Downloads.IS_PENDING, 0);
                getContentResolver().update(item, values, null, null);
                Toast.makeText(this, "已保存到「下载」\n" + name, Toast.LENGTH_LONG).show();
                return;
            }

            File downloads = Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_DOWNLOADS);
            File destDir = downloads;
            if (destDir == null || (!destDir.exists() && !destDir.mkdirs())) {
                destDir = getExternalFilesDir(Environment.DIRECTORY_DOWNLOADS);
            }
            if (destDir == null) {
                destDir = getFilesDir();
            }
            name = DownloadNames.uniqueIn(destDir, name);
            File out = new File(destDir, name);
            try (FileOutputStream fos = new FileOutputStream(out)) {
                fos.write(bytes);
            }
            try {
                Intent scan = new Intent(Intent.ACTION_MEDIA_SCANNER_SCAN_FILE);
                scan.setData(Uri.fromFile(out));
                sendBroadcast(scan);
            } catch (Exception ignored) {
            }
            Toast.makeText(this, "已保存到\n" + out.getAbsolutePath(), Toast.LENGTH_LONG).show();
        } catch (Exception e) {
            Toast.makeText(this, "保存失败: " + e.getMessage(), Toast.LENGTH_SHORT).show();
        }
    }

    private static String jsonString(String s) {
        if (s == null) {
            return "\"\"";
        }
        return "\"" + s.replace("\\", "\\\\").replace("\"", "\\\"")
                .replace("\n", "\\n").replace("\r", "\\r") + "\"";
    }

    /** Minimal decode for data: URLs that are not base64. */
    private static final class URLDecoderCompat {
        static String decode(String s) {
            try {
                return java.net.URLDecoder.decode(s, "UTF-8");
            } catch (Exception e) {
                return s;
            }
        }
    }

    private void injectSafeAreaCss() {
        if (webView == null) {
            return;
        }
        String js = "(function(){"
                + "var d=document.documentElement;"
                + "d.style.setProperty('--pake-safe-top','" + safeTopPx + "px');"
                + "d.style.setProperty('--pake-safe-bottom','" + safeBottomPx + "px');"
                + "d.style.setProperty('--pake-safe-left','" + safeLeftPx + "px');"
                + "d.style.setProperty('--pake-safe-right','" + safeRightPx + "px');"
                + "var m=document.querySelector('meta[name=viewport]');"
                + "if(!m){m=document.createElement('meta');m.name='viewport';"
                + "m.content='width=device-width,initial-scale=1,viewport-fit=cover';"
                + "document.head&&document.head.appendChild(m);}"
                + "else if((m.content||'').indexOf('viewport-fit')<0){"
                + "m.content=(m.content?m.content+',':'')+'viewport-fit=cover';}"
                + "})();";
        webView.evaluateJavascript(js, null);
    }

    private void injectAssets() {
        try {
            String[] names = getAssets().list("inject");
            if (names == null) {
                return;
            }
            for (String name : names) {
                if (name == null) {
                    continue;
                }
                String lower = name.toLowerCase(Locale.US);
                if (lower.endsWith(".js")) {
                    String js = readAsset("inject/" + name);
                    if (!js.isEmpty()) {
                        webView.evaluateJavascript(js, null);
                    }
                } else if (lower.endsWith(".css")) {
                    String css = readAsset("inject/" + name)
                            .replace("\\", "\\\\")
                            .replace("'", "\\'")
                            .replace("\n", "\\n")
                            .replace("\r", "");
                    String js = "(function(){var s=document.createElement('style');s.type='text/css';"
                            + "s.appendChild(document.createTextNode('" + css + "'));"
                            + "document.head.appendChild(s);})();";
                    webView.evaluateJavascript(js, null);
                }
            }
        } catch (Exception ignored) {
        }
    }

    private String readAsset(String path) {
        try (InputStream in = getAssets().open(path);
             BufferedReader br = new BufferedReader(new InputStreamReader(in, StandardCharsets.UTF_8))) {
            StringBuilder sb = new StringBuilder();
            String line;
            while ((line = br.readLine()) != null) {
                sb.append(line).append('\n');
            }
            return sb.toString();
        } catch (Exception e) {
            return "";
        }
    }

    private boolean handleUri(Uri uri) {
        if (uri == null) {
            return false;
        }
        String scheme = uri.getScheme() == null ? "" : uri.getScheme().toLowerCase(Locale.US);
        if ("mailto".equals(scheme) || "tel".equals(scheme) || "sms".equals(scheme)
                || "intent".equals(scheme) || "market".equals(scheme)) {
            openExternal(uri);
            return true;
        }
        if ("http".equals(scheme) || "https".equals(scheme)) {
            String policy = BuildConfig.LINK_POLICY == null ? "allowlist"
                    : BuildConfig.LINK_POLICY.trim().toLowerCase(Locale.US);
            if ("internal".equals(policy)) {
                return false;
            }
            if ("external".equals(policy)) {
                openExternal(uri);
                return true;
            }
            // allowlist (default)
            if (isAllowedHost(uri.getHost())) {
                return false;
            }
            openExternal(uri);
            return true;
        }
        openExternal(uri);
        return true;
    }

    private boolean isAllowedHost(String host) {
        if (host == null || host.isEmpty()) {
            return false;
        }
        String h = host.toLowerCase(Locale.US);
        for (String allow : allowedHosts()) {
            if (allow.isEmpty()) {
                continue;
            }
            if (h.equals(allow) || h.endsWith("." + allow)) {
                return true;
            }
        }
        return false;
    }

    private List<String> allowedHosts() {
        List<String> out = new ArrayList<>();
        try {
            String host = Uri.parse(BuildConfig.START_URL).getHost();
            if (host != null && !host.isEmpty()) {
                out.add(host.toLowerCase(Locale.US));
            }
        } catch (Exception ignored) {
        }
        if (BuildConfig.SAFE_DOMAINS != null && !BuildConfig.SAFE_DOMAINS.trim().isEmpty()) {
            for (String part : BuildConfig.SAFE_DOMAINS.split("[,;\\s]+")) {
                String d = part.trim().toLowerCase(Locale.US);
                if (!d.isEmpty()) {
                    out.add(d);
                }
            }
        }
        return out;
    }

    private void openExternal(Uri uri) {
        try {
            startActivity(new Intent(Intent.ACTION_VIEW, uri));
        } catch (Exception ignored) {
        }
    }

    @Override
    public void onBackPressed() {
        if (webView != null && webView.canGoBack()) {
            webView.goBack();
            return;
        }
        super.onBackPressed();
    }

    @Override
    protected void onDestroy() {
        if (webView != null) {
            webView.loadUrl("about:blank");
            webView.destroy();
            webView = null;
        }
        super.onDestroy();
    }

    @Override
    public void onRequestPermissionsResult(int requestCode, String[] permissions, int[] grantResults) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults);
        if (requestCode != REQ_NOTIF) {
            return;
        }
        boolean ok = grantResults.length > 0 && grantResults[0] == PackageManager.PERMISSION_GRANTED;
        Toast.makeText(this, ok ? "已允许通知" : "未授予通知权限", Toast.LENGTH_SHORT).show();
    }

    /** Native helpers for H5: share, safe-area, optional local notify / FCM hooks. */
    public class PakeBridge {
        @JavascriptInterface
        public void share(String title, String text, String url) {
            runOnUiThread(() -> {
                try {
                    Intent send = new Intent(Intent.ACTION_SEND);
                    send.setType("text/plain");
                    StringBuilder body = new StringBuilder();
                    if (text != null && !text.trim().isEmpty()) {
                        body.append(text.trim());
                    }
                    if (url != null && !url.trim().isEmpty()) {
                        if (body.length() > 0) {
                            body.append('\n');
                        }
                        body.append(url.trim());
                    }
                    if (body.length() == 0 && webView != null && webView.getUrl() != null) {
                        body.append(webView.getUrl());
                    }
                    send.putExtra(Intent.EXTRA_TEXT, body.toString());
                    if (title != null && !title.trim().isEmpty()) {
                        send.putExtra(Intent.EXTRA_SUBJECT, title.trim());
                    }
                    String chooserTitle = (title != null && !title.trim().isEmpty()) ? title.trim() : "分享";
                    startActivity(Intent.createChooser(send, chooserTitle));
                } catch (Exception e) {
                    Toast.makeText(MainActivity.this, "无法打开分享", Toast.LENGTH_SHORT).show();
                }
            });
        }

        @JavascriptInterface
        public String getSafeAreaInsets() {
            return "{\"top\":" + safeTopPx
                    + ",\"bottom\":" + safeBottomPx
                    + ",\"left\":" + safeLeftPx
                    + ",\"right\":" + safeRightPx + "}";
        }

        @JavascriptInterface
        public boolean isPushConfigured() {
            // FCM token wiring is opt-in later; local notify channel is ready when enabled in GUI.
            return BuildConfig.PUSH_PLACEHOLDER;
        }

        @JavascriptInterface
        public String getPushToken() {
            // Placeholder until google-services.json + FCM are wired in CI.
            return "";
        }

        @JavascriptInterface
        public void requestPushPermission() {
            runOnUiThread(() -> {
                if (!BuildConfig.PUSH_PLACEHOLDER) {
                    Toast.makeText(MainActivity.this, "未开启推送能力", Toast.LENGTH_SHORT).show();
                    return;
                }
                if (Build.VERSION.SDK_INT >= 33) {
                    if (ContextCompat.checkSelfPermission(MainActivity.this,
                            Manifest.permission.POST_NOTIFICATIONS)
                            == PackageManager.PERMISSION_GRANTED) {
                        Toast.makeText(MainActivity.this, "通知权限已授予", Toast.LENGTH_SHORT).show();
                        return;
                    }
                    requestPermissions(new String[]{Manifest.permission.POST_NOTIFICATIONS}, REQ_NOTIF);
                } else {
                    Toast.makeText(MainActivity.this, "当前系统无需额外通知权限", Toast.LENGTH_SHORT).show();
                }
            });
        }

        @JavascriptInterface
        public void showNotification(String title, String body) {
            runOnUiThread(() -> {
                if (!BuildConfig.PUSH_PLACEHOLDER) {
                    Toast.makeText(MainActivity.this, "未开启推送能力", Toast.LENGTH_SHORT).show();
                    return;
                }
                AppNotifications.show(MainActivity.this, title, body);
            });
        }
    }

    /** Handles blob:/data: downloads from the page context. */
    public class DownloadBridge {
        @JavascriptInterface
        public void saveDataUrl(String dataUrl, String mimeType, String contentDisposition) {
            runOnUiThread(() -> MainActivity.this.saveDataUrl(dataUrl, mimeType, contentDisposition));
        }

        @JavascriptInterface
        public void fail(String message) {
            runOnUiThread(() ->
                    Toast.makeText(MainActivity.this,
                            "下载失败: " + (message == null ? "" : message),
                            Toast.LENGTH_SHORT).show());
        }
    }
}
