package dev.opcode42.feature.chat.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class NormalizeRmsTest {
    @Test
    fun `floor and below clamp to zero`() {
        assertEquals(0f, normalizeRms(-2f), 0f)
        assertEquals(0f, normalizeRms(-20f), 0f)
    }

    @Test
    fun `ceiling and above clamp to one`() {
        assertEquals(1f, normalizeRms(10f), 0f)
        assertEquals(1f, normalizeRms(50f), 0f)
    }

    @Test
    fun `midpoint maps to one half`() {
        // floor -2, ceil 10 → midpoint 4 dB
        assertEquals(0.5f, normalizeRms(4f), 1e-4f)
    }

    @Test
    fun `output is monotonic in input`() {
        assertTrue(normalizeRms(0f) < normalizeRms(5f))
        assertTrue(normalizeRms(5f) < normalizeRms(9f))
    }
}
