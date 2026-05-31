package dev.forge.core.sdk

import android.util.Log
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import okio.ByteString

/**
 * Wraps an OkHttp WebSocket for PTY I/O.
 *
 * Incoming binary frames:
 *   - Starting with 0x00: cursor control JSON — skipped (not rendered yet)
 *   - Otherwise: raw terminal output bytes — emitted on [output]
 *
 * Outgoing: raw keystroke bytes sent as binary frames.
 */
class PtyClient(
    private val webSocket: WebSocket,
    private val _output: MutableSharedFlow<ByteArray> = MutableSharedFlow(extraBufferCapacity = 256),
) {
    val output: SharedFlow<ByteArray> = _output

    fun send(bytes: ByteArray) {
        webSocket.send(ByteString.of(*bytes))
    }

    fun close() {
        webSocket.close(1000, null)
    }

    companion object {
        /**
         * Creates a WebSocketListener that emits to the given flow.
         * Create the listener, build the WebSocket, then construct PtyClient(ws, flow).
         */
        fun createListener(output: MutableSharedFlow<ByteArray>): WebSocketListener =
            object : WebSocketListener() {
                override fun onMessage(webSocket: WebSocket, bytes: ByteString) {
                    val raw = bytes.toByteArray()
                    if (raw.isEmpty()) return
                    // 0x00 prefix = cursor control frame — skip for now
                    if (raw[0] == 0x00.toByte()) return
                    output.tryEmit(raw)
                }

                override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                    Log.e("PtyClient", "WebSocket failure", t)
                    // Emit empty bytes as EOF sentinel
                    output.tryEmit(ByteArray(0))
                }

                override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                    output.tryEmit(ByteArray(0))
                }
            }
    }
}
