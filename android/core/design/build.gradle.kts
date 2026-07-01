plugins {
    id("opcode42.android.library")
    alias(libs.plugins.compose.compiler)
}

android {
    namespace = "dev.opcode42.core.design"
    buildFeatures { compose = true }
}

dependencies {
    // The module's public API returns/accepts Compose types (Color tokens,
    // FontFamily, RoundedCornerShape, Modifier/Dp composables), so the deps that
    // surface in that ABI are `api` — a consumer that isn't already a Compose
    // module can then use the tokens/brand without re-declaring them. Because of
    // this bespoke `api` visibility the shared compose convention is intentionally
    // not applied here.
    api(platform(libs.compose.bom))
    api(libs.compose.ui)
    api(libs.compose.ui.graphics)
    api(libs.compose.material3)
    implementation(libs.compose.ui.tooling.preview)
    debugImplementation(libs.compose.ui.tooling)
    testImplementation(libs.junit)
}
