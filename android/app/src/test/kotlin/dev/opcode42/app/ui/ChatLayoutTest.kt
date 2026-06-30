package dev.opcode42.app.ui

import androidx.window.core.layout.WindowHeightSizeClass
import androidx.window.core.layout.WindowWidthSizeClass
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Truth table for [chatLayoutFor] — the three standard width tiers (Material
 * `WindowWidthSizeClass`, 600/840dp breakpoints):
 *  - Compact (<600): single pane · overlay menu · no right panel.
 *  - Medium (600–839): chat + right panel · inline-push rail closed by default.
 *  - Expanded (≥840): the full triptych — right panel + rail open & persistent,
 *    except on a short (Compact-height) window where the rail stays closed.
 */
class ChatLayoutTest {

    @Test fun paneTier_mapsTheThreeStandardWidthClasses() {
        assertEquals(ChatPaneTier.Compact, chatPaneTier(WindowWidthSizeClass.COMPACT))
        assertEquals(ChatPaneTier.Medium, chatPaneTier(WindowWidthSizeClass.MEDIUM))
        assertEquals(ChatPaneTier.Expanded, chatPaneTier(WindowWidthSizeClass.EXPANDED))
    }

    // ── Compact ────────────────────────────────────────────────────────────────
    @Test fun phonePortrait_singlePane_overlay_noRightPanel() {
        val l = chatLayoutFor(WindowWidthSizeClass.COMPACT, WindowHeightSizeClass.MEDIUM)
        assertTrue(l.singlePane)
        assertFalse(l.showRightPanel)
        assertEquals(LeftRailMode.Overlay, l.leftRailMode)
        assertFalse(l.railPersistent)
    }

    @Test fun foldedCover_compactWidth_behavesLikePhone() {
        val l = chatLayoutFor(WindowWidthSizeClass.COMPACT, WindowHeightSizeClass.EXPANDED)
        assertTrue(l.singlePane)
        assertEquals(LeftRailMode.Overlay, l.leftRailMode)
        assertFalse(l.railPersistent)
    }

    @Test fun tinySplitScreen_bothCompact_isSinglePane() {
        val l = chatLayoutFor(WindowWidthSizeClass.COMPACT, WindowHeightSizeClass.COMPACT)
        assertTrue(l.singlePane)
        assertFalse(l.showRightPanel)
        assertEquals(LeftRailMode.Overlay, l.leftRailMode)
        assertFalse(l.railPersistent)
    }

    // ── Medium: right panel + inline-push rail, closed by default ────────────────
    @Test fun foldablePortrait_rightPanel_inlinePush_railClosed() {
        val l = chatLayoutFor(WindowWidthSizeClass.MEDIUM, WindowHeightSizeClass.MEDIUM)
        assertFalse(l.singlePane)
        assertTrue(l.showRightPanel)
        assertEquals(LeftRailMode.InlinePush, l.leftRailMode)
        assertFalse(l.railPersistent)
    }

    @Test fun tabletPortrait_mediumWidth_railClosed() {
        val l = chatLayoutFor(WindowWidthSizeClass.MEDIUM, WindowHeightSizeClass.EXPANDED)
        assertFalse(l.singlePane)
        assertTrue(l.showRightPanel)
        assertEquals(LeftRailMode.InlinePush, l.leftRailMode)
        assertFalse(l.railPersistent)
    }

    // ── Expanded: the full triptych — rail open & persistent (when tall enough) ──
    @Test fun tabletLandscape_expandedWidth_persistentTriptych() {
        val l = chatLayoutFor(WindowWidthSizeClass.EXPANDED, WindowHeightSizeClass.MEDIUM)
        assertFalse(l.singlePane)
        assertTrue(l.showRightPanel)
        assertEquals(LeftRailMode.InlinePush, l.leftRailMode)
        assertTrue(l.railPersistent)
    }

    @Test fun foldableOpenLandscape_expandedWidth_persistentTriptych() {
        val l = chatLayoutFor(WindowWidthSizeClass.EXPANDED, WindowHeightSizeClass.EXPANDED)
        assertFalse(l.singlePane)
        assertTrue(l.showRightPanel)
        assertEquals(LeftRailMode.InlinePush, l.leftRailMode)
        assertTrue(l.railPersistent)
    }

    @Test fun largePhoneLandscape_expandedWidthShortHeight_railNotPersistent() {
        // Wide but short (large phone in landscape): Expanded width, Compact height —
        // still multi-pane, but the rail stays closed so three panes don't crowd it.
        val l = chatLayoutFor(WindowWidthSizeClass.EXPANDED, WindowHeightSizeClass.COMPACT)
        assertFalse(l.singlePane)
        assertTrue(l.showRightPanel)
        assertEquals(LeftRailMode.InlinePush, l.leftRailMode)
        assertFalse(l.railPersistent)
    }
}
