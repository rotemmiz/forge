package dev.opcode42.core.data

import dev.opcode42.core.sdk.Opcode42Client
import dev.opcode42.core.sdk.PtyClient
import dev.opcode42.core.sdk.PtyInfo
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Data-layer wrapper over the daemon's PTY endpoints so [dev.opcode42.feature.terminal] does not
 * depend on the REST client directly. The websocket lifecycle (cursor tracking, emulator feed)
 * stays in the ViewModel — it is terminal-specific UI state, not a data concern.
 */
interface TerminalRepository {
    suspend fun createPty(directory: String): Result<PtyInfo>
    fun connectPty(ptyId: String, authToken: String, cursor: Long?): PtyClient
    suspend fun resizePty(ptyId: String, rows: Int, cols: Int): Result<Unit>
}

@Singleton
class DefaultTerminalRepository @Inject constructor(
    private val client: Opcode42Client,
) : TerminalRepository {

    override suspend fun createPty(directory: String): Result<PtyInfo> =
        resultOf { client.createPty(directory) }

    override fun connectPty(ptyId: String, authToken: String, cursor: Long?): PtyClient =
        client.connectPty(ptyId, authToken, cursor)

    override suspend fun resizePty(ptyId: String, rows: Int, cols: Int): Result<Unit> =
        resultOf { client.resizePty(ptyId, rows, cols) }
}
