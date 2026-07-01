plugins {
    id("opcode42.android.library")
    id("opcode42.android.compose")
    id("opcode42.android.hilt")
}

android {
    namespace = "dev.opcode42.feature.sessions"
}

dependencies {
    api(project(":core:model"))
    api(project(":core:store"))
    implementation(project(":core:data"))
    implementation(project(":core:design"))
    implementation(project(":feature:connections"))
    implementation(libs.android.lifecycle.viewmodel.compose)
    implementation(libs.hilt.navigation.compose)
    implementation(libs.compose.material.icons.extended)
    testImplementation(libs.junit)
}
