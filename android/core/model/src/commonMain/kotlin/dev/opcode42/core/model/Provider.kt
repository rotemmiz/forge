package dev.opcode42.core.model

import kotlinx.serialization.Serializable

/** A provider/model pair — the shape POST /session/{id}/message accepts as `model`. */
@Serializable
data class ModelRef(
    val providerID: String,
    val modelID: String,
)

/**
 * A model's token limits (GET /provider → all[].models[id].limit; opencode
 * `Model.limit`). `context` is the real context-window size the gauge divides by;
 * 0 means the daemon didn't report it (e.g. model missing from the models.dev catalog).
 * Typed `Double` to match the wire contract (openapi `Model.limit` fields are `number`,
 * like [TokenUsage]) — an integer-typed field would throw on a fractional literal and,
 * because the `/provider` decode swallows any exception, take the whole picker with it.
 */
@Serializable
data class ModelLimit(
    val context: Double = 0.0,
    val input: Double = 0.0,
    val output: Double = 0.0,
)

/** One model offered by a provider (GET /provider → all[].models is keyed by id). */
@Serializable
data class ModelInfo(
    val id: String,
    val name: String? = null,
    val limit: ModelLimit = ModelLimit(),
) {
    /** Human label, falling back to the raw id. */
    val label: String get() = name?.takeIf { it.isNotBlank() } ?: id
}

/** A provider surfaced by GET /provider (only the fields the model picker needs). */
@Serializable
data class ProviderInfo(
    val id: String,
    val name: String? = null,
    val models: Map<String, ModelInfo> = emptyMap(),
) {
    val label: String get() = name?.takeIf { it.isNotBlank() } ?: id
}

/** GET /provider → `{ all, default: providerID→modelID, connected: providerID[] }`. */
@Serializable
data class ProvidersResponse(
    val all: List<ProviderInfo> = emptyList(),
    val default: Map<String, String> = emptyMap(),
    val connected: List<String> = emptyList(),
)
