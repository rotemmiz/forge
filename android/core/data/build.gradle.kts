plugins {
    id("opcode42.android.library")
    id("opcode42.android.hilt")
}

android {
    namespace = "dev.opcode42.core.data"
}

dependencies {
    api(project(":core:model"))
    api(project(":core:store"))
    // api (not implementation): TerminalRepository exposes PtyInfo/PtyClient in its public surface.
    api(project(":core:sdk"))
    implementation(project(":core:network"))
    implementation(libs.kotlinx.coroutines.android)

    testImplementation(libs.junit)
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(libs.okhttp)
    testImplementation(libs.okhttp.mockwebserver)
}
