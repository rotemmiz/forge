package dev.forge.feature.chat

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import dev.forge.core.model.*
import dev.forge.core.network.SseManager
import dev.forge.core.sdk.ForgeClient
import dev.forge.core.store.AppStore
import dev.forge.core.store.OptimisticMessage
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import javax.inject.Inject

data class ChatUiState(
    val session: Session? = null,
    val messages: List<Message> = emptyList(),
    val parts: Map<String, List<Part>> = emptyMap(),
    val optimisticMessages: List<OptimisticMessage> = emptyList(),
    val pendingPermissions: List<PermissionRequest> = emptyList(),
    val pendingQuestions: List<QuestionRequest> = emptyList(),
    val isLoading: Boolean = false,
    val isSending: Boolean = false,
)

@HiltViewModel
class ChatViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val client: ForgeClient,
    private val store: AppStore,
    private val sseManager: SseManager,
) : ViewModel() {

    private val sessionId: String = checkNotNull(savedStateHandle["sessionId"])

    val uiState: StateFlow<ChatUiState> = store.state
        .map { appState ->
            val session = appState.sessions.firstOrNull { it.id == sessionId }
            ChatUiState(
                session = session,
                messages = appState.messages[sessionId] ?: emptyList(),
                parts = appState.parts,
                optimisticMessages = appState.optimisticMessages[sessionId] ?: emptyList(),
                pendingPermissions = appState.permissions[sessionId] ?: emptyList(),
                pendingQuestions = appState.questions[sessionId] ?: emptyList(),
            )
        }
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), ChatUiState())

    private val directory: String? get() = uiState.value.session?.directory

    init {
        loadMessages()
        // Subscribe to per-directory SSE once the session's directory is resolved
        viewModelScope.launch {
            uiState.collect { state ->
                val dir = state.session?.directory
                if (dir != null) {
                    sseManager.subscribeDirectory(dir)
                    return@collect  // only need to do this once
                }
            }
        }
    }

    private fun loadMessages() {
        viewModelScope.launch {
            try {
                val messages = client.getMessages(sessionId, directory)
                android.util.Log.d("ChatVM", "loadMessages: got ${messages.size} messages for $sessionId")
                messages.forEach { msg ->
                    android.util.Log.d("ChatVM", "  dispatch msg ${msg.id} role=${msg.role} parts=${msg.parts.size}")
                    store.dispatch(AppEvent.MessageUpdated(msg))
                }
            } catch (e: Exception) {
                android.util.Log.e("ChatVM", "loadMessages failed", e)
            }
        }
    }

    /** A7 — Optimistic prompt submit */
    fun sendPrompt(text: String) {
        viewModelScope.launch {
            val optimisticId = store.addOptimistic(sessionId, text)
            try {
                client.sendPrompt(sessionId, text, directory)
            } catch (e: Exception) {
                store.removeOptimistic(sessionId, optimisticId)
            }
        }
    }

    /** A8 — Permission reply */
    fun replyPermission(requestId: String, allow: Boolean) {
        viewModelScope.launch {
            try {
                client.replyPermission(requestId, allow)
                store.dispatch(AppEvent.PermissionReplied(requestId))
            } catch (_: Exception) { }
        }
    }

    /** A8 — Question reply */
    fun replyQuestion(requestId: String, answer: String) {
        viewModelScope.launch {
            try {
                client.replyQuestion(requestId, answer)
                store.dispatch(AppEvent.QuestionReplied(requestId))
            } catch (_: Exception) { }
        }
    }

    fun rejectQuestion(requestId: String) {
        viewModelScope.launch {
            try {
                client.rejectQuestion(requestId)
                store.dispatch(AppEvent.QuestionRejected(requestId))
            } catch (_: Exception) { }
        }
    }
}
