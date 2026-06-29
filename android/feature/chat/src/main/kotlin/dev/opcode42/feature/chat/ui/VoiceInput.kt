package dev.opcode42.feature.chat.ui

import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.speech.RecognitionListener
import android.speech.RecognizerIntent
import android.speech.SpeechRecognizer
import android.util.Log
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import java.util.Locale

private const val VOICE_TAG = "VoiceInput"

/**
 * Drives a single-utterance [SpeechRecognizer] session for the composer. The
 * recognizer is created lazily on first [start] and must only be touched from
 * the main thread — all `RecognitionListener` callbacks arrive there too, so the
 * Compose-observable [isListening] flips are safe to read during composition.
 *
 * Partial transcripts stream out via `onPartial` as the user speaks; the
 * finalized utterance is delivered once via `onFinal`. The caller decides how to
 * merge these into the field (see [PromptInput], which anchors them to the text
 * present when listening began).
 */
@Stable
class VoiceInputController internal constructor(
    private val context: Context,
    private val onPartial: (String) -> Unit,
    private val onFinal: (String) -> Unit,
) {
    /** Whether an on-device/cloud recognition provider exists. Cached at construction. */
    val isAvailable: Boolean = SpeechRecognizer.isRecognitionAvailable(context)

    var isListening by mutableStateOf(false)
        private set

    private var recognizer: SpeechRecognizer? = null

    private val listener = object : RecognitionListener {
        override fun onPartialResults(partialResults: Bundle?) {
            partialResults?.firstText()?.let(onPartial)
        }

        override fun onResults(results: Bundle?) {
            results?.firstText()?.let(onFinal)
            isListening = false
        }

        override fun onError(error: Int) {
            Log.d(VOICE_TAG, "recognition error: $error")
            isListening = false
        }

        override fun onReadyForSpeech(params: Bundle?) {}
        override fun onBeginningOfSpeech() {}
        override fun onRmsChanged(rmsdB: Float) {}
        override fun onBufferReceived(buffer: ByteArray?) {}
        override fun onEndOfSpeech() {}
        override fun onEvent(eventType: Int, params: Bundle?) {}
    }

    /** Begin a recognition session. No-op if already listening or unavailable. */
    fun start() {
        if (isListening || !isAvailable) return
        val r = recognizer ?: SpeechRecognizer.createSpeechRecognizer(context).also {
            it.setRecognitionListener(listener)
            recognizer = it
        }
        val intent = Intent(RecognizerIntent.ACTION_RECOGNIZE_SPEECH).apply {
            putExtra(RecognizerIntent.EXTRA_LANGUAGE_MODEL, RecognizerIntent.LANGUAGE_MODEL_FREE_FORM)
            putExtra(RecognizerIntent.EXTRA_PARTIAL_RESULTS, true)
            putExtra(RecognizerIntent.EXTRA_LANGUAGE, Locale.getDefault())
            putExtra(RecognizerIntent.EXTRA_CALLING_PACKAGE, context.packageName)
        }
        isListening = true
        r.startListening(intent)
    }

    /** End the current session, delivering whatever was captured via `onFinal`. */
    fun stop() {
        if (!isListening) return
        // stopListening() flushes a final result through onResults; that callback
        // clears isListening. Guard here in case the provider stays silent.
        recognizer?.stopListening()
        isListening = false
    }

    fun toggle() = if (isListening) stop() else start()

    /** Abort the session, discarding any pending result. Used when the field is sent. */
    fun cancel() {
        if (!isListening) return
        recognizer?.cancel()
        isListening = false
    }

    /** Release the underlying recognizer. Call once when the composable leaves. */
    fun destroy() {
        recognizer?.destroy()
        recognizer = null
        isListening = false
    }
}

private fun Bundle.firstText(): String? =
    getStringArrayList(SpeechRecognizer.RESULTS_RECOGNITION)
        ?.firstOrNull()
        ?.takeIf { it.isNotBlank() }

/**
 * Remembers a [VoiceInputController] tied to the current context, forwarding the
 * latest `onPartial`/`onFinal` lambdas without recreating the recognizer, and
 * destroying it on dispose.
 */
@Composable
fun rememberVoiceInput(
    onPartial: (String) -> Unit,
    onFinal: (String) -> Unit,
): VoiceInputController {
    val context = LocalContext.current
    val currentPartial by rememberUpdatedState(onPartial)
    val currentFinal by rememberUpdatedState(onFinal)
    val controller = remember(context) {
        VoiceInputController(
            context = context,
            onPartial = { currentPartial(it) },
            onFinal = { currentFinal(it) },
        )
    }
    DisposableEffect(controller) {
        onDispose { controller.destroy() }
    }
    return controller
}
