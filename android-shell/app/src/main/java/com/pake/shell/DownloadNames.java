package com.pake.shell;

import android.net.Uri;
import android.webkit.MimeTypeMap;
import android.webkit.URLUtil;

import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.util.Locale;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Resolves a user-friendly download filename.
 * Avoids saving PHP/ASP endpoints as {@code export.php} when a real name is available.
 */
final class DownloadNames {
    private static final Pattern CD_FILENAME_STAR = Pattern.compile(
            "filename\\*\\s*=\\s*([^']*)'[^']*'([^;]+)", Pattern.CASE_INSENSITIVE);
    private static final Pattern CD_FILENAME_QUOTED = Pattern.compile(
            "filename\\s*=\\s*\"([^\"]+)\"", Pattern.CASE_INSENSITIVE);
    private static final Pattern CD_FILENAME_PLAIN = Pattern.compile(
            "filename\\s*=\\s*([^;\\s]+)", Pattern.CASE_INSENSITIVE);

    private DownloadNames() {
    }

    static String resolve(String url, String contentDisposition, String mimeType) {
        String fromCd = fromContentDisposition(contentDisposition);
        if (isUsefulName(fromCd)) {
            return finalizeName(fromCd, mimeType, url);
        }

        String fromQuery = fromQueryParams(url);
        if (isUsefulName(fromQuery)) {
            return finalizeName(fromQuery, mimeType, url);
        }

        String guessed = null;
        try {
            guessed = URLUtil.guessFileName(url, contentDisposition, mimeType);
        } catch (Exception ignored) {
        }
        if (isUsefulName(guessed) && !isScriptLike(guessed)) {
            return finalizeName(guessed, mimeType, url);
        }

        String fromPath = lastPathSegment(url);
        if (isUsefulName(fromPath) && !isScriptLike(fromPath)) {
            return finalizeName(fromPath, mimeType, url);
        }

        return "download_" + System.currentTimeMillis() + extensionFor(mimeType, url, null);
    }

    static String uniqueIn(java.io.File dir, String name) {
        name = sanitize(name);
        if (dir == null || !dir.exists()) {
            return name;
        }
        FileProbe probe = splitName(name);
        FileCandidate first = new FileCandidate(dir, probe.base, probe.ext, 0);
        if (!first.exists()) {
            return first.name();
        }
        for (int i = 1; i < 1000; i++) {
            FileCandidate c = new FileCandidate(dir, probe.base, probe.ext, i);
            if (!c.exists()) {
                return c.name();
            }
        }
        return probe.base + "_" + System.currentTimeMillis() + probe.ext;
    }

    private static String fromContentDisposition(String cd) {
        if (cd == null || cd.trim().isEmpty()) {
            return null;
        }
        Matcher star = CD_FILENAME_STAR.matcher(cd);
        if (star.find()) {
            String charset = star.group(1);
            String encoded = star.group(2).trim().replace("\"", "");
            try {
                String cs = (charset == null || charset.isEmpty()) ? "UTF-8" : charset.trim();
                return URLDecoder.decode(encoded, cs);
            } catch (Exception e) {
                try {
                    return URLDecoder.decode(encoded, "UTF-8");
                } catch (Exception ignored) {
                    return encoded;
                }
            }
        }
        Matcher quoted = CD_FILENAME_QUOTED.matcher(cd);
        if (quoted.find()) {
            return quoted.group(1);
        }
        Matcher plain = CD_FILENAME_PLAIN.matcher(cd);
        if (plain.find()) {
            String v = plain.group(1).trim();
            if (v.regionMatches(true, 0, "filename*", 0, 9)) {
                return null;
            }
            return trimQuotes(v);
        }
        return null;
    }

    private static String fromQueryParams(String url) {
        if (url == null || url.isEmpty()) {
            return null;
        }
        try {
            Uri uri = Uri.parse(url);
            String[] keys = {"filename", "file", "name", "download", "fname", "fn", "attname"};
            for (String key : keys) {
                String v = uri.getQueryParameter(key);
                if (isUsefulName(v) && !isScriptLike(v)) {
                    return v;
                }
            }
        } catch (Exception ignored) {
        }
        return null;
    }

    private static String lastPathSegment(String url) {
        try {
            String path = Uri.parse(url).getLastPathSegment();
            if (path == null || path.isEmpty()) {
                return null;
            }
            return URLDecoder.decode(path, StandardCharsets.UTF_8.name());
        } catch (Exception e) {
            return null;
        }
    }

    private static String finalizeName(String raw, String mimeType, String url) {
        String name = sanitize(raw);
        if (isScriptLike(name) || !name.contains(".")) {
            String base = name;
            int dot = base.lastIndexOf('.');
            if (dot > 0 && isScriptLike(name)) {
                base = base.substring(0, dot);
            }
            base = sanitize(base);
            if (base.isEmpty()) {
                base = "download_" + System.currentTimeMillis();
            }
            name = base + extensionFor(mimeType, url, null);
        }
        if (name.isEmpty() || ".".equals(name)) {
            return "download_" + System.currentTimeMillis() + extensionFor(mimeType, url, null);
        }
        return name;
    }

    private static boolean isUsefulName(String name) {
        if (name == null) {
            return false;
        }
        String n = name.trim();
        if (n.isEmpty() || n.equals(".") || n.equals("..")) {
            return false;
        }
        // Bare query tokens without an extension are weak hints; still allow if not script-like.
        return !n.equalsIgnoreCase("null") && !n.equalsIgnoreCase("undefined");
    }

    static boolean isScriptLike(String name) {
        if (name == null) {
            return true;
        }
        String lower = name.trim().toLowerCase(Locale.US);
        int q = lower.indexOf('?');
        if (q >= 0) {
            lower = lower.substring(0, q);
        }
        int dot = lower.lastIndexOf('.');
        String ext = dot >= 0 ? lower.substring(dot + 1) : "";
        switch (ext) {
            case "php":
            case "phtml":
            case "asp":
            case "aspx":
            case "jsp":
            case "jspx":
            case "cgi":
            case "pl":
            case "py":
            case "rb":
            case "do":
            case "action":
            case "ashx":
            case "asmx":
            case "svc":
                return true;
            default:
                return false;
        }
    }

    private static String extensionFor(String mimeType, String url, String currentName) {
        if (currentName != null) {
            int dot = currentName.lastIndexOf('.');
            if (dot >= 0 && dot < currentName.length() - 1 && !isScriptLike(currentName)) {
                return "";
            }
        }
        String mime = mimeType == null ? "" : mimeType.trim().toLowerCase(Locale.US);
        int semi = mime.indexOf(';');
        if (semi >= 0) {
            mime = mime.substring(0, semi).trim();
        }
        switch (mime) {
            case "image/jpeg":
            case "image/jpg":
                return ".jpg";
            case "image/png":
                return ".png";
            case "image/gif":
                return ".gif";
            case "image/webp":
                return ".webp";
            case "image/svg+xml":
                return ".svg";
            case "application/pdf":
                return ".pdf";
            case "application/zip":
                return ".zip";
            case "application/json":
                return ".json";
            case "text/plain":
                return ".txt";
            case "text/csv":
                return ".csv";
            case "text/html":
                return ".html";
            case "audio/mpeg":
                return ".mp3";
            case "video/mp4":
                return ".mp4";
            case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
                return ".xlsx";
            case "application/vnd.ms-excel":
                return ".xls";
            case "application/msword":
                return ".doc";
            case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
                return ".docx";
            default:
                break;
        }
        if (!mime.isEmpty()) {
            String sub = MimeTypeMap.getSingleton().getExtensionFromMimeType(mime);
            if (sub != null && !sub.isEmpty()) {
                return "." + sub;
            }
        }
        try {
            String path = Uri.parse(url).getPath();
            if (path != null) {
                String ext = MimeTypeMap.getFileExtensionFromUrl(path);
                if (ext != null && !ext.isEmpty() && !isScriptLike("x." + ext)) {
                    return "." + ext.toLowerCase(Locale.US);
                }
            }
        } catch (Exception ignored) {
        }
        return ".bin";
    }

    private static String sanitize(String name) {
        if (name == null) {
            return "download";
        }
        String n = name.trim();
        int slash = Math.max(n.lastIndexOf('/'), n.lastIndexOf('\\'));
        if (slash >= 0 && slash < n.length() - 1) {
            n = n.substring(slash + 1);
        }
        n = n.replaceAll("[\\x00-\\x1f\\x7f<>:\"|?*]", "_");
        n = n.replaceAll("[\\s]+", " ").trim();
        if (n.length() > 180) {
            FileProbe p = splitName(n);
            String base = p.base;
            if (base.length() > 120) {
                base = base.substring(0, 120);
            }
            n = base + p.ext;
        }
        if (n.isEmpty()) {
            return "download";
        }
        return n;
    }

    private static String trimQuotes(String v) {
        if (v == null) {
            return null;
        }
        v = v.trim();
        if (v.length() >= 2) {
            char a = v.charAt(0);
            char b = v.charAt(v.length() - 1);
            if ((a == '"' && b == '"') || (a == '\'' && b == '\'')) {
                return v.substring(1, v.length() - 1);
            }
        }
        return v;
    }

    private static FileProbe splitName(String name) {
        int dot = name.lastIndexOf('.');
        if (dot > 0 && dot < name.length() - 1) {
            return new FileProbe(name.substring(0, dot), name.substring(dot));
        }
        return new FileProbe(name, "");
    }

    private static final class FileProbe {
        final String base;
        final String ext;

        FileProbe(String base, String ext) {
            this.base = base;
            this.ext = ext;
        }
    }

    private static final class FileCandidate {
        private final java.io.File dir;
        private final String base;
        private final String ext;
        private final int index;

        FileCandidate(java.io.File dir, String base, String ext, int index) {
            this.dir = dir;
            this.base = base;
            this.ext = ext;
            this.index = index;
        }

        String name() {
            if (index <= 0) {
                return base + ext;
            }
            return base + " (" + index + ")" + ext;
        }

        boolean exists() {
            return new java.io.File(dir, name()).exists();
        }
    }
}
