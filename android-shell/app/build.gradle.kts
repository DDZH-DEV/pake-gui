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

android {
    namespace = "com.pake.shell"
    compileSdk = 34

    defaultConfig {
        applicationId = pakeProp("PAKE_APP_ID", "com.pake.shell")
        minSdk = 24
        targetSdk = 34
        versionCode = 1
        versionName = pakeProp("PAKE_APP_VERSION", "1.0.0")

        val startUrl = pakeProp("PAKE_START_URL", "https://example.com")
        buildConfigField("String", "START_URL", "\"${escapeJava(startUrl)}\"")
        resValue("string", "app_name", pakeProp("PAKE_APP_NAME", "Pake App"))
        manifestPlaceholders["usesCleartextTraffic"] =
            pakeProp("PAKE_CLEARTEXT", "false")
    }

    buildFeatures {
        buildConfig = true
    }

    buildTypes {
        debug {
            isDebuggable = true
        }
        release {
            isMinifyEnabled = false
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}
