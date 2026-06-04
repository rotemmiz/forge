package dev.forge.core.network

/**
 * Lightweight contract so networking code (e.g. AuthInterceptor on Android)
 * doesn't depend on the full ServerConnectionManager.
 *
 * Pure multiplatform types (commonMain) — shareable with a future iOS client.
 */
interface ActiveConnectionProvider {
    val active: ServerConnectionConfig?
}

data class ServerConnectionConfig(
    val url: String,
    val http: HttpConfig,
)

data class HttpConfig(
    val url: String,
    val username: String? = null,
    val password: String? = null,
)
