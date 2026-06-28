package dev.forge.feature.sessions

import dev.forge.core.model.PermissionRequest
import dev.forge.core.model.QuestionRequest
import dev.forge.core.model.Session
import dev.forge.core.model.SessionTime
import dev.forge.core.store.AppState
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Unit coverage for [projectSessionList] — the pure store→UI projection that feeds the
 * sessions menu, including the per-session status / permission / question maps that drive
 * the in-menu spinner and inline actions.
 */
class SessionListProjectionTest {

    private fun session(id: String, archived: Long? = null, updated: Long? = null) =
        Session(id = id, time = SessionTime(created = 0, updated = updated, archived = archived))

    @Test fun activeSessions_sortedByRecency_archivedExcludedButCounted() {
        val state = AppState(
            sessions = listOf(
                session("a", updated = 100),
                session("b", updated = 300),
                session("c", archived = 5),
            ),
        )
        val ui = projectSessionList(state, showArchived = false)
        assertEquals(listOf("b", "a"), ui.sessions.map { it.id })
        assertEquals(1, ui.archivedCount)
    }

    @Test fun showArchived_returnsOnlyArchived() {
        val state = AppState(sessions = listOf(session("a"), session("c", archived = 5)))
        val ui = projectSessionList(state, showArchived = true)
        assertEquals(listOf("c"), ui.sessions.map { it.id })
        assertTrue(ui.showArchived)
    }

    @Test fun statusAndPendingMaps_projectedPerSession() {
        val perm = PermissionRequest(id = "p1", sessionID = "a", title = "Run rm?")
        val q = QuestionRequest(id = "q1", sessionID = "b", message = "Which env?")
        val state = AppState(
            sessions = listOf(session("a"), session("b")),
            sessionStatus = mapOf("a" to "busy", "b" to "idle"),
            permissions = mapOf("a" to listOf(perm)),
            questions = mapOf("b" to listOf(q)),
        )
        val ui = projectSessionList(state, showArchived = false)
        assertEquals("busy", ui.statuses["a"])
        assertEquals("p1", ui.pendingPermissions["a"]?.id)
        assertNull(ui.pendingPermissions["b"])
        assertEquals("q1", ui.pendingQuestions["b"]?.id)
        assertNull(ui.pendingQuestions["a"])
    }

    @Test fun firstPendingRequestPerSession_isSurfaced() {
        val state = AppState(
            sessions = listOf(session("a")),
            permissions = mapOf(
                "a" to listOf(
                    PermissionRequest(id = "p1", sessionID = "a"),
                    PermissionRequest(id = "p2", sessionID = "a"),
                ),
            ),
        )
        val ui = projectSessionList(state, showArchived = false)
        assertEquals("p1", ui.pendingPermissions["a"]?.id)
    }
}
