// Package android documents the Android cloud-build path (T03).
//
// Builds run on GitHub Actions (.github/workflows/build-android.yml) using the
// generic WebView template in android-shell/. The GUI submits via the shared
// github.RunCloudJob with platform=android; artifacts land in builds/android/.
package android

// WorkflowFile is the GitHub Actions workflow that builds the debug APK.
const WorkflowFile = "build-android.yml"
