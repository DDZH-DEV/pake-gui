package com.pake.shell;

import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.os.Build;

import androidx.core.app.NotificationCompat;
import androidx.core.app.NotificationManagerCompat;

/** Local notifications; FCM can post into the same channel later. */
final class AppNotifications {
    static final String CHANNEL_ID = "pake_general";

    private AppNotifications() {
    }

    static void ensureChannel(Context ctx) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
            return;
        }
        NotificationManager nm = ctx.getSystemService(NotificationManager.class);
        if (nm == null) {
            return;
        }
        NotificationChannel existing = nm.getNotificationChannel(CHANNEL_ID);
        if (existing != null) {
            return;
        }
        NotificationChannel ch = new NotificationChannel(
                CHANNEL_ID,
                "应用通知",
                NotificationManager.IMPORTANCE_DEFAULT);
        ch.setDescription("来自应用的消息通知");
        nm.createNotificationChannel(ch);
    }

    static void show(Context ctx, String title, String body) {
        ensureChannel(ctx);
        Intent open = new Intent(ctx, MainActivity.class);
        open.addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP | Intent.FLAG_ACTIVITY_CLEAR_TOP);
        int flags = PendingIntent.FLAG_UPDATE_CURRENT;
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            flags |= PendingIntent.FLAG_IMMUTABLE;
        }
        PendingIntent pi = PendingIntent.getActivity(ctx, 0, open, flags);
        String t = (title == null || title.trim().isEmpty()) ? "通知" : title.trim();
        String b = body == null ? "" : body.trim();
        NotificationCompat.Builder nb = new NotificationCompat.Builder(ctx, CHANNEL_ID)
                .setSmallIcon(android.R.drawable.stat_notify_chat)
                .setContentTitle(t)
                .setContentText(b)
                .setStyle(new NotificationCompat.BigTextStyle().bigText(b))
                .setAutoCancel(true)
                .setContentIntent(pi)
                .setPriority(NotificationCompat.PRIORITY_DEFAULT);
        try {
            NotificationManagerCompat.from(ctx).notify((int) (System.currentTimeMillis() & 0xffff), nb.build());
        } catch (SecurityException ignored) {
            // missing POST_NOTIFICATIONS on API 33+
        }
    }
}
