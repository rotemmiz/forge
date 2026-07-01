package dev.opcode42.core.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class Session(
    val id: String,
    val title: String? = null,
    val slug: String? = null,
    val projectID: String? = null,
    val workspaceID: String? = null,
    val directory: String? = null,
    val path: String? = null,
    val parentID: String? = null,
    val cost: Double? = null,
    val tokens: TokenUsage? = null,
    val summary: SessionSummary? = null,
    val share: SessionShare? = null,
    val time: SessionTime? = null,
    val version: String? = null,
)

@Serializable
data class TokenUsage(
    val input: Double = 0.0,
    val output: Double = 0.0,
    val reasoning: Double = 0.0,
    val cache: CacheUsage = CacheUsage(),
    /** Provider-reported total when present; the daemon's overflow check prefers it. */
    val total: Double? = null,
) {
    /**
     * Context-window occupancy for THIS turn — mirrors opencode's context gauge
     * (tui sidebar/context.tsx:28-31, prompt/index.tsx:266-268): the sum of
     * input + output + reasoning + cache. The wire's session-level [total] is
     * deliberately ignored here — it tracks lifetime usage, not this turn's
     * footprint. (The daemon's compaction check is what prefers [total]; that's a
     * separate numerator, server-side in overflow.go.) Drives the live context gauge.
     */
    val contextFootprint: Long
        get() = (input + output + reasoning + cache.read + cache.write).toLong()
}

@Serializable
data class CacheUsage(
    val read: Double = 0.0,
    val write: Double = 0.0,
)

@Serializable
data class SessionSummary(
    val additions: Double = 0.0,
    val deletions: Double = 0.0,
    val files: Double = 0.0,
)

@Serializable
data class SessionShare(val url: String)

@Serializable
data class SessionTime(
    val created: Long = 0,
    val updated: Long? = null,
    // Unix-ms timestamp set when the session is archived. opencode types this as a
    // finite number; null/absent means "not archived" and there is no un-archive path
    // (session.ts:304, groups/session.ts:51). The session list filters on this.
    val archived: Long? = null,
)
