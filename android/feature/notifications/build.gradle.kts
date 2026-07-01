plugins {
    id("opcode42.android.library")
    id("opcode42.android.hilt")
}

android {
    namespace = "dev.opcode42.feature.notifications"
    testOptions {
        // PushRegistrar logs via android.util.Log; return defaults so JVM unit
        // tests don't hit the un-mocked-method RuntimeException.
        unitTests.isReturnDefaultValues = true
    }
}

dependencies {
    implementation(project(":core:sdk"))

    implementation(libs.android.core.ktx)
    implementation(libs.datastore.preferences)
    implementation(libs.kotlinx.coroutines.android)

    // Firebase Cloud Messaging. The google-services Gradle plugin is intentionally
    // NOT applied (it would require a checked-in google-services.json at build
    // time); Firebase is initialized at runtime, gated on config presence.
    implementation(platform(libs.firebase.bom))
    implementation(libs.firebase.messaging)

    testImplementation(libs.junit)
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(libs.kotlinx.serialization.json)
    testImplementation(libs.okhttp.mockwebserver)
    testImplementation(libs.okhttp)
}
