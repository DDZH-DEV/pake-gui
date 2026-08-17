package com.pake.shell;

import android.annotation.SuppressLint;
import android.app.Activity;
import android.app.DownloadManager;
import android.content.ActivityNotFoundException;
import android.content.Intent;
import android.net.Uri;
import android.os.Bundle;
import android.os.Environment;
import android.view.View;
import android.view.WindowManager;
import android.webkit.CookieManager;
import android.webkit.DownloadListener;
import android.webkit.JavascriptInterface;
import android.webkit.URLUtil;
import android.webkit.ValueCallback;
import android.webkit.WebChromeClient;
import android.webkit.WebResourceRequest;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.ProgressBar;
import android.widget.Toast;

import androidx.core.content.FileProvider;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;

import java.io.BufferedReader;
import java.io.File;
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

    private WebView webView;
    private ProgressBar progressBar;
    private SwipeRefreshLayout swipe;
    private ValueCallback<Uri[]> filePathCallback;
    private Uri cameraOutputUri;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        if (BuildConfig.FULLSCREEN) {
            getWindow().addFlags(WindowManager.LayoutParams.FLAG_FULLSCREEN);
        }
        setContentView(R.layout.activity_main);
        webView = findViewById(R.id.webview);
        progressBar = findViewById(R.id.progress);
        swipe = findViewById(R.id.swipe);
        if (!BuildConfig.PULL_REFRESH) {
            swipe.setEnabled(false);
        } else {
            swipe.setOnRefreshListener(() -> webView.reload());
        }
        if (!BuildConfig.PROGRESS_BAR) {
            progressBar.setVisibility(View.GONE);
        }
        configureWebView();
        webView.loadUrl(BuildConfig.START_URL);
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

        if (BuildConfig.PUSH_PLACEHOLDER) {
            webView.addJavascriptInterface(new PushBridge(), "PakeAndroid");
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
        try {
            String name = URLUtil.guessFileName(url, contentDisposition, mimeType);
            DownloadManager.Request req = new DownloadManager.Request(Uri.parse(url));
            req.setMimeType(mimeType);
            String cookie = CookieManager.getInstance().getCookie(url);
            if (cookie != null) {
                req.addRequestHeader("cookie", cookie);
            }
            if (userAgent != null && !userAgent.isEmpty()) {
                req.addRequestHeader("User-Agent", userAgent);
            }
            req.setTitle(name);
            req.setDescription("下载中");
            req.setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED);
            req.setDestinationInExternalPublicDir(Environment.DIRECTORY_DOWNLOADS, name);
            DownloadManager dm = (DownloadManager) getSystemService(DOWNLOAD_SERVICE);
            dm.enqueue(req);
            Toast.makeText(this, "开始下载: " + name, Toast.LENGTH_SHORT).show();
        } catch (Exception e) {
            openExternal(Uri.parse(url));
        }
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

    /** Optional JS bridge stub for future FCM wiring. */
    public class PushBridge {
        @JavascriptInterface
        public boolean isPushConfigured() {
            return false;
        }

        @JavascriptInterface
        public void requestPushPermission() {
            runOnUiThread(() ->
                    Toast.makeText(MainActivity.this, "推送占位：尚未配置 FCM", Toast.LENGTH_SHORT).show());
        }
    }
}
