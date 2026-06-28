package dev.forge.feature.sessions

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import dev.forge.core.model.AppEvent
import dev.forge.core.model.PermissionRequest
import dev.forge.core.model.QuestionRequest
import dev.forge.core.model.Session
import dev.forge.core.network.ActiveConnectionProvider
import dev.forge.core.sdk.ForgeClient
import dev.forge.core.store.AppState
import dev.forge.core.store.AppStore
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.filterNotNull
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import javax.inject.Inject

/** True when the session has been archived (opencode sets `time.archived` to a number). */
val Session.isArchived: Boolean
    get() = time?.archived != null

data class SessionListUiState(
    /** Sessions to display: active sessions, or archived ones when [showArchived] is set. */
    val sessions: List<Session> = emptyList(),
    /** Count of archived sessions, for the "Archived (n)" affordance. */
    val archivedCount: Int = 0,
    /** When true the list shows archived sessions instead of active ones. */
    val showArchived: Boolean = false,
    /** sessionID → status string ("busy" while a turn is in flight, else "idle"). */
    val statuses: Map<String, String> = emptyMap(),
    /** sessionID → first pending permission request, for inline Approve/Deny in the menu. */
    val pendingPermissions: Map<String, PermissionRequest> = emptyMap(),
    /** sessionID → first pending question, for an inline reply field in the menu. */
    val pendingQuestions: Map<String, QuestionRequest> = emptyMap(),
    val isLoading: Boolean = false,
    val error: String? = null,
)

/**
 * Pure projection from the global store to the list UI state. Kept side-effect-free and
 * top-level so the active/archived split, recency ordering, and the per-session
 * status/permission/question maps can be unit-tested without a ViewModel or coroutines.
 */
internal fun projectSessionList(appState: AppState, showArchived: Boolean): SessionListUiState {
    val (archived, active) = appState.sessions.partition { it.isArchived }
    // Most-recently-active first (opencode bumps time.updated on each new message),
    // falling back to creation time. The store keeps sessions in ID order for the
    // binary-search upsert; recency ordering is a display-layer concern only.
    val visible = (if (showArchived) archived else active)
        .sortedByDescending { it.time?.updated ?: it.time?.created ?: 0L }
    return SessionListUiState(
        sessions = visible,
        archivedCount = archived.size,
        showArchived = showArchived,
        statuses = appState.sessionStatus,
        // Surface only the first pending request per session — the menu shows one
        // actionable affordance per row; the rest queue behind it as each is answered.
        pendingPermissions = appState.permissions
            .mapNotNull { (id, list) -> list.firstOrNull()?.let { id to it } }
            .toMap(),
        pendingQuestions = appState.questions
            .mapNotNull { (id, list) -> list.firstOrNull()?.let { id to it } }
            .toMap(),
    )
}

@HiltViewModel
class SessionListViewModel @Inject constructor(
    private val client: ForgeClient,
    private val store: AppStore,
    private val connectionProvider: ActiveConnectionProvider,
) : ViewModel() {

    // Local view toggle: active list vs. archived list. opencode's session list returns
    // both; filtering is client-side (the daemon does not drop archived sessions).
    private val _showArchived = MutableStateFlow(false)

    val uiState: StateFlow<SessionListUiState> =
        combine(store.state, _showArchived) { appState, showArchived ->
            projectSessionList(appState, showArchived)
        }.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), SessionListUiState())

    private val _isCreating = MutableStateFlow(false)
    val isCreating: StateFlow<Boolean> = _isCreating.asStateFlow()

    init {
        loadSessions()
        // Re-fetch when a connection becomes active (e.g., first-time server add)
        viewModelScope.launch {
            connectionProvider.activeFlow
                .filterNotNull()
                .distinctUntilChanged()
                .collect { loadSessions() }
        }
    }

    fun toggleShowArchived() {
        _showArchived.value = !_showArchived.value
    }

    fun loadSessions() {
        viewModelScope.launch {
            try {
                val directory = connectionProvider.active?.directory
                val sessions = client.listSessions(directory)
                sessions.forEach { session ->
                    store.dispatch(AppEvent.SessionUpdated(session))
                }
            } catch (e: Exception) {
                // Sessions will load from SSE events once connected
            }
        }
    }

    fun forkSession(sessionId: String, onForked: (Session) -> Unit) {
        viewModelScope.launch {
            try {
                val newSession = client.forkSession(sessionId)
                store.dispatch(AppEvent.SessionUpdated(newSession))
                onForked(newSession)
            } catch (e: Exception) {
                android.util.Log.e("SessionListVM", "forkSession failed", e)
            }
        }
    }

    /** PATCH /session/{id} title; the returned session updates the store (and SSE echoes it). */
    fun renameSession(sessionId: String, title: String) {
        val trimmed = title.trim()
        if (trimmed.isEmpty()) return
        viewModelScope.launch {
            try {
                store.dispatch(AppEvent.SessionUpdated(client.renameSession(sessionId, trimmed)))
            } catch (e: Exception) {
                android.util.Log.e("SessionListVM", "renameSession failed", e)
            }
        }
    }

    /**
     * Archive the session via PATCH /session/{id} `time.archived`. The returned session
     * (now archived) updates the store, so it drops out of the active list. There is no
     * un-archive path — opencode treats archived as set-only.
     */
    fun archiveSession(sessionId: String) {
        viewModelScope.launch {
            try {
                store.dispatch(AppEvent.SessionUpdated(client.archiveSession(sessionId)))
            } catch (e: Exception) {
                android.util.Log.e("SessionListVM", "archiveSession failed", e)
            }
        }
    }

    fun deleteSession(sessionId: String) {
        viewModelScope.launch {
            try {
                client.deleteSession(sessionId)
                store.dispatch(AppEvent.SessionRemoved(sessionId))
            } catch (e: Exception) {
                android.util.Log.e("SessionListVM", "deleteSession failed", e)
            }
        }
    }

    // ── In-menu pending-action replies ────────────────────────────────────────
    // Mirrors ChatViewModel.replyPermission/replyQuestion/rejectQuestion so a session
    // that needs the user can be answered straight from the sessions menu, without
    // opening it. The store maps drop the request on the *Replied/*Rejected event.

    /** Approve or deny a session's pending permission from the menu. */
    fun replyPermission(requestId: String, allow: Boolean) {
        viewModelScope.launch {
            try {
                client.replyPermission(requestId, allow)
                store.dispatch(AppEvent.PermissionReplied(requestId))
            } catch (_: Exception) { }
        }
    }

    /** Answer a session's pending question from the menu. */
    fun replyQuestion(requestId: String, answer: String) {
        viewModelScope.launch {
            try {
                client.replyQuestion(requestId, answer)
                store.dispatch(AppEvent.QuestionReplied(requestId))
            } catch (_: Exception) { }
        }
    }

    /** Skip a session's pending question from the menu. */
    fun rejectQuestion(requestId: String) {
        viewModelScope.launch {
            try {
                client.rejectQuestion(requestId)
                store.dispatch(AppEvent.QuestionRejected(requestId))
            } catch (_: Exception) { }
        }
    }

    fun createSession(directory: String? = null, onCreated: (Session) -> Unit) {
        viewModelScope.launch {
            _isCreating.value = true
            try {
                val session = client.createSession(directory)
                store.dispatch(AppEvent.SessionUpdated(session))
                onCreated(session)
            } catch (_: Exception) {
            } finally {
                _isCreating.value = false
            }
        }
    }
}
