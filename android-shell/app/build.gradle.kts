plugins {
    id("com.android.application")
}

fun pakeProp(key: String, def: String): String {
    val fromEnv = System.getenv(key)?.trim().orEmpty()
    if (fromEnv.isNotEmpty()) return fromEnv
    val fromP = (findProperty(key) as? String)?.trim().orEmpty()
    if (fromP.isNotEmpty()) return fromP
    return def
}

fun escapeJava(s: String): String =
    s.replace("\\", "\\\\").replace("\"", "\\\"")

fun pakeBool(key: String, def: Boolean): Boolean {
    val raw = pakeProp(key, if (def) "true" else "false").lowercase()
    return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}

android {
    namespace = "com.pake.shell"
    compileSdk = 34

    defaultConfig {
        applicationId = pakeProp("PAKE_APP_ID", "com.pake.shell")
        minSdk = 24
        targetSdk = 34
        versionCode = pakeProp("PAKE_VERSION_CODE", "1").toIntOrNull() ?: 1
        versionName = pakeProp("PAKE_APP_VERSION", "1.0.0")

        val startUrl = pakeProp("PAKE_START_URL", "https://example.com")
        buildConfigField("String", "START_URL", "\"${escapeJava(startUrl)}\"")
        buildConfigField("String", "USER_AGENT", "\"${escapeJava(pakeProp("PAKE_USER_AGENT", ""))}\"")
        buildConfigField("String", "SAFE_DOMAINS", "\"${escapeJava(pakeProp("PAKE_SAFE_DOMAINS", ""))}\"")
        buildConfigField("String", "LINK_POLICY", "\"${escapeJava(pakeProp("PAKE_LINK_POLICY", "allowlist"))}\"")
        buildConfigField("boolean", "FULLSCREEN", pakeBool("PAKE_FULLSCREEN", false).toString())
        buildConfigField("boolean", "PULL_REFRESH", pakeBool("PAKE_PULL_REFRESH", true).toString())
        buildConfigField("boolean", "PROGRESS_BAR", pakeBool("PAKE_PROGRESS_BAR", true).toString())
        buildConfigField("boolean", "ENABLE_FILE_UPLOAD", pakeBool("PAKE_FILE_UPLOAD", true).toString())
        buildConfigField("boolean", "ENABLE_CAMERA", pakeBool("PAKE_CAMERA", false).toString())
        buildConfigField("boolean", "ENABLE_DOWNLOAD", pakeBool("PAKE_DOWNLOAD", true).toString())
        buildConfigField("boolean", "PUSH_PLACEHOLDER", pakeBool("PAKE_PUSH_PLACEHOLDER", false).toString())

        resValue("string", "app_name", pakeProp("PAKE_APP_NAME", "Pake App"))
        manifestPlaceholders["usesCleartextTraffic"] = pakeProp("PAKE_CLEARTEXT", "false")
        manifestPlaceholders["screenOrientation"] = pakeProp("PAKE_ORIENTATION", "unspecified")
    }

    buildFeatures {
        buildConfig = true
    }

    signingConfigs {
        val storeFilePath = pakeProp("PAKE_STORE_FILE", "")
        if (storeFilePath.isNotEmpty()) {
            create("release") {
                storeFile = file(storeFilePath)
                storePassword = pakeProp("PAKE_STORE_PASSWORD", "")
                keyAlias = pakeProp("PAKE_KEY_ALIAS", "")
                keyPassword = pakeProp("PAKE_KEY_PASSWORD", "")
            }
        }
    }

    buildTypes {
        debug {
            isDebuggable = true
        }
        release {
            isMinifyEnabled = false
            val releaseSigning = signingConfigs.findByName("release")
            if (releaseSigning != null) {
                signingConfig = releaseSigning
            }
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    implementation("androidx.swiperefreshlayout:swiperefreshlayout:1.1.0")
    implementation("androidx.core:core:1.13.1")
}
