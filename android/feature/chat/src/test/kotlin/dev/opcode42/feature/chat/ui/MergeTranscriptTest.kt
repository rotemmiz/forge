package dev.opcode42.feature.chat.ui

import org.junit.Assert.assertEquals
import org.junit.Test

class MergeTranscriptTest {
    @Test
    fun `empty base uses spoken verbatim`() {
        assertEquals("hello world", mergeTranscript("", "hello world"))
    }

    @Test
    fun `blank base uses spoken verbatim`() {
        assertEquals("hello", mergeTranscript("   ", "hello"))
    }

    @Test
    fun `non-empty base is separated from spoken by a single space`() {
        assertEquals("fix the bug now", mergeTranscript("fix the bug", "now"))
    }

    @Test
    fun `trailing whitespace on base is collapsed to one space`() {
        assertEquals("note this down", mergeTranscript("note this  ", "down"))
    }

    @Test
    fun `successive partials replace rather than stack`() {
        // Each partial is merged against the same captured base, so a longer
        // transcript fully supersedes the previous one.
        val base = "draft:"
        assertEquals("draft: open the", mergeTranscript(base, "open the"))
        assertEquals("draft: open the file", mergeTranscript(base, "open the file"))
    }
}
