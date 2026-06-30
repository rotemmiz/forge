package dev.opcode42.feature.sessions.ui

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.CallSplit
import androidx.compose.material.icons.filled.Archive
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.util.lerp
import dev.opcode42.core.design.brand.Spinner
import dev.opcode42.core.design.theme.Error
import dev.opcode42.core.design.theme.Hairline
import dev.opcode42.core.design.theme.OnSurface
import dev.opcode42.core.design.theme.OnSurfaceFaint
import dev.opcode42.core.design.theme.OnSurfaceGhost
import dev.opcode42.core.design.theme.OnSurfaceVariant
import dev.opcode42.core.design.theme.Opcode42Mono
import dev.opcode42.core.design.theme.Primary
import dev.opcode42.core.design.theme.Secondary
import dev.opcode42.core.design.theme.SecondaryContainer
import dev.opcode42.core.design.theme.SurfaceContainerLow
import dev.opcode42.core.model.PermissionRequest
import dev.opcode42.core.model.QuestionRequest
import dev.opcode42.core.model.Session
import dev.opcode42.feature.sessions.relativeTime
import kotlin.math.roundToInt

// ─── Rail-morph geometry (compact) ─────────────────────────────────────────────
private val RailRowBand = 46.dp // constant row height across the open⇄collapsed morph
private val RailLeftInset = 11.dp // == (60 − 38)/2, so the avatar lands centered in the 60dp band
private val AvatarSize = 38.dp
private val SpinnerBaseDp = 16.dp // open loader size (item: increased from 13)
private val SpinnerBadge = 18.dp // the collapsed loader's bordered backing disc
// Busy-loader center, in row-start-relative dp, lerped by progress (1=open … 0=collapsed):
private val SpinX1 = 200.dp // open: trailing-right of the 220dp row
private val SpinX0 = 47.dp // collapsed: the avatar's top-right
private val SpinY1 = 23.dp // open: vertically centered (band/2)
private val SpinY0 = 6.dp // collapsed: up at the avatar's top edge
private const val SpinScale1 = 1.0f
private const val SpinScale0 = 0.8f

/**
 * One rich session row shared by the full-screen list and the in-chat left rail.
 *
 * The full list (`compact=false`) leads with a status dot and a `dir · time` meta line and is
 * unaffected by [progress]. The narrow [compact] rail MORPHS between open (220dp) and collapsed
 * (60dp) driven by [progress] (`1f`=open … `0f`=collapsed): the title + status/`time · workdir`
 * meta fade out while a single-letter avatar fades in at the same vertical position, the active
 * amber tint crossfades, and the busy loader translates up + scales onto the avatar.
 *
 * [progress] is a provider read ONLY inside draw/layout lambdas, so the per-frame float never
 * recomposes the row. Both layouts keep the long-press menu (Rename/Fork/Archive/Delete) and the
 * inline permission/question affordances ([SessionPendingActions]).
 */
@OptIn(ExperimentalFoundationApi::class)
@Composable
internal fun SessionRow(
    session: Session,
    isActive: Boolean,
    status: String?,
    pendingPermission: PermissionRequest?,
    pendingQuestion: QuestionRequest?,
    showArchived: Boolean,
    onClick: () -> Unit,
    onRename: () -> Unit,
    onArchive: () -> Unit,
    onFork: () -> Unit,
    onDelete: () -> Unit,
    onApprove: () -> Unit,
    onDeny: () -> Unit,
    onReply: (String) -> Unit,
    onSkip: () -> Unit,
    modifier: Modifier = Modifier,
    compact: Boolean = false,
    progress: () -> Float = { 1f },
) {
    var showMenu by remember { mutableStateOf(false) }
    val needsInput = pendingPermission != null || pendingQuestion != null
    val busy = isSessionBusy(status)
    val accent = Secondary // hoisted: a @Composable token can't be read inside a draw/offset lambda

    Box(modifier.fillMaxWidth()) {
        if (compact) {
            CompactRailRow(
                session = session,
                isActive = isActive,
                busy = busy,
                needsInput = needsInput,
                accent = accent,
                progress = progress,
                onClick = onClick,
                onLongPress = { showMenu = true },
                pendingPermission = pendingPermission,
                pendingQuestion = pendingQuestion,
                onApprove = onApprove,
                onDeny = onDeny,
                onReply = onReply,
                onSkip = onSkip,
            )
        } else {
            FullListRow(
                session = session,
                isActive = isActive,
                busy = busy,
                needsInput = needsInput,
                accent = accent,
                onClick = onClick,
                onLongPress = { showMenu = true },
                pendingPermission = pendingPermission,
                pendingQuestion = pendingQuestion,
                onApprove = onApprove,
                onDeny = onDeny,
                onReply = onReply,
                onSkip = onSkip,
            )
        }
        DropdownMenu(expanded = showMenu, onDismissRequest = { showMenu = false }) {
            DropdownMenuItem(
                text = { Text("Rename session") },
                leadingIcon = { Icon(Icons.Default.Edit, contentDescription = null) },
                onClick = { showMenu = false; onRename() },
            )
            DropdownMenuItem(
                text = { Text("Fork session") },
                leadingIcon = { Icon(Icons.AutoMirrored.Filled.CallSplit, contentDescription = null) },
                onClick = { showMenu = false; onFork() },
            )
            // opencode has no un-archive path, so archive is offered only on active rows.
            if (!showArchived) {
                DropdownMenuItem(
                    text = { Text("Archive session") },
                    leadingIcon = { Icon(Icons.Default.Archive, contentDescription = null) },
                    onClick = { showMenu = false; onArchive() },
                )
            }
            DropdownMenuItem(
                text = { Text("Delete session") },
                leadingIcon = { Icon(Icons.Default.Delete, contentDescription = null) },
                onClick = { showMenu = false; onDelete() },
            )
        }
    }
}

// ─── Compact rail row (the open⇄collapsed morph) ───────────────────────────────

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun CompactRailRow(
    session: Session,
    isActive: Boolean,
    busy: Boolean,
    needsInput: Boolean,
    accent: Color,
    progress: () -> Float,
    onClick: () -> Unit,
    onLongPress: () -> Unit,
    pendingPermission: PermissionRequest?,
    pendingQuestion: QuestionRequest?,
    onApprove: () -> Unit,
    onDeny: () -> Unit,
    onReply: (String) -> Unit,
    onSkip: () -> Unit,
) {
    // Flip the structural bits (drop the inline actions) once, at the midpoint — not per frame.
    val open by remember { derivedStateOf { progress() > 0.5f } }

    Column(Modifier.fillMaxWidth()) {
        Box(
            Modifier
                .fillMaxWidth()
                .heightIn(min = RailRowBand)
                .combinedClickable(onClick = onClick, onLongClick = onLongPress),
        ) {
            // (1) Active pill — open state, full-row amber tint + left accent, fades OUT on collapse.
            if (isActive) {
                Box(
                    Modifier
                        .matchParentSize()
                        .padding(horizontal = 6.dp, vertical = 1.dp)
                        .graphicsLayer { alpha = progress() }
                        .clip(RoundedCornerShape(6.dp))
                        .background(SecondaryContainer)
                        .drawBehind { drawRect(accent, size = Size(2.5.dp.toPx(), size.height)) },
                )
            }
            // (2) Avatar — collapsed state, left-pinned (ends centered in 60dp), fades IN on collapse.
            Box(
                Modifier
                    .align(Alignment.CenterStart)
                    .padding(start = RailLeftInset)
                    .size(AvatarSize)
                    .graphicsLayer { alpha = 1f - progress() }
                    .clip(RoundedCornerShape(8.dp))
                    .then(
                        if (isActive) {
                            Modifier
                                .background(SecondaryContainer)
                                .drawBehind { drawRect(accent, size = Size(2.dp.toPx(), size.height)) }
                        } else {
                            Modifier
                        },
                    ),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = sessionInitial(session.title),
                    fontFamily = Opcode42Mono,
                    fontSize = 13.sp,
                    fontWeight = FontWeight.Bold,
                    color = if (isActive) OnSurface else OnSurfaceVariant,
                )
            }
            // (3) Open content — title + status/`time · workdir` meta, shares the avatar's left
            //     inset and fades OUT on collapse.
            Column(
                Modifier
                    .align(Alignment.CenterStart)
                    .fillMaxWidth()
                    .padding(start = RailLeftInset, end = if (busy) 30.dp else 12.dp)
                    .graphicsLayer { alpha = progress() },
            ) {
                Text(
                    text = session.title?.takeIf { it.isNotBlank() } ?: "New session",
                    fontSize = 13.5.sp,
                    fontWeight = if (isActive) FontWeight.Medium else FontWeight.Normal,
                    color = if (isActive) OnSurface else OnSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                val dir = session.directory?.substringAfterLast('/')?.takeIf { it.isNotBlank() }
                val rel = relativeTime(session.time?.updated ?: session.time?.created ?: 0L)
                    .takeIf { it.isNotEmpty() }
                // Meta = live status (amber) when busy / needs-input, else the relative time
                // (faint); the workdir basename trails the time with a middle dot.
                val meta = when {
                    needsInput -> "needs input"
                    busy && isActive -> listOfNotNull("running", dir).joinToString(" · ")
                    busy -> listOfNotNull("background", dir).joinToString(" · ")
                    else -> listOfNotNull(rel, dir).joinToString(" · ").takeIf { it.isNotEmpty() }
                }
                val metaColor = when {
                    needsInput -> Error
                    busy -> Secondary
                    else -> OnSurfaceFaint
                }
                if (meta != null) {
                    Text(
                        text = meta,
                        fontSize = 12.sp,
                        color = metaColor,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            // (4) Busy loader — a single spinner that translates up + scales from the open
            //     trailing-right onto the collapsed avatar, seating into a bordered badge.
            if (busy) {
                Box(
                    Modifier
                        .offset {
                            val p = progress()
                            val s = SpinnerBadge.toPx()
                            IntOffset(
                                (lerp(SpinX0.toPx(), SpinX1.toPx(), p) - s / 2f).roundToInt(),
                                (lerp(SpinY0.toPx(), SpinY1.toPx(), p) - s / 2f).roundToInt(),
                            )
                        }
                        .size(SpinnerBadge)
                        .graphicsLayer { alpha = 1f - progress() }
                        .clip(CircleShape)
                        .background(SurfaceContainerLow)
                        .border(1.dp, Hairline, CircleShape),
                )
                Box(
                    Modifier
                        .offset {
                            val p = progress()
                            val s = SpinnerBaseDp.toPx()
                            IntOffset(
                                (lerp(SpinX0.toPx(), SpinX1.toPx(), p) - s / 2f).roundToInt(),
                                (lerp(SpinY0.toPx(), SpinY1.toPx(), p) - s / 2f).roundToInt(),
                            )
                        }
                        .graphicsLayer {
                            val sc = lerp(SpinScale0, SpinScale1, progress())
                            scaleX = sc
                            scaleY = sc
                        },
                ) {
                    Spinner(size = SpinnerBaseDp, color = accent)
                }
            }
        }
        // Inline permission/question actions: fade with the rail, then drop once collapsed.
        if (needsInput && open) {
            SessionPendingActions(
                permission = pendingPermission,
                question = pendingQuestion,
                onApprove = onApprove,
                onDeny = onDeny,
                onReply = onReply,
                onSkip = onSkip,
                modifier = Modifier
                    .graphicsLayer { alpha = progress() }
                    .padding(start = 12.dp, end = 12.dp, bottom = 10.dp),
            )
        }
    }
}

// ─── Full-screen list row (status dot + dir · time; unaffected by the rail morph) ──

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun FullListRow(
    session: Session,
    isActive: Boolean,
    busy: Boolean,
    needsInput: Boolean,
    accent: Color,
    onClick: () -> Unit,
    onLongPress: () -> Unit,
    pendingPermission: PermissionRequest?,
    pendingQuestion: QuestionRequest?,
    onApprove: () -> Unit,
    onDeny: () -> Unit,
    onReply: (String) -> Unit,
    onSkip: () -> Unit,
) {
    Column(
        Modifier
            .fillMaxWidth()
            // Active row: amber selection tint + a 2.5dp amber accent rail down the left.
            .then(
                if (isActive) {
                    Modifier
                        .background(SecondaryContainer)
                        .drawBehind { drawRect(accent, size = Size(2.5.dp.toPx(), size.height)) }
                } else {
                    Modifier
                },
            ),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .combinedClickable(onClick = onClick, onLongClick = onLongPress)
                .padding(horizontal = 16.dp, vertical = 10.dp),
        ) {
            StatusLeading(busy = busy, needsInput = needsInput, isActive = isActive)
            Spacer(Modifier.width(10.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    text = session.title?.takeIf { it.isNotBlank() } ?: "New session",
                    fontSize = 15.sp,
                    fontWeight = if (isActive) FontWeight.Medium else FontWeight.Normal,
                    color = OnSurface,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                val dir = session.directory?.substringAfterLast('/')?.takeIf { it.isNotBlank() }
                val rel = relativeTime(session.time?.updated ?: session.time?.created ?: 0L)
                    .takeIf { it.isNotEmpty() }
                val meta = listOfNotNull(dir, rel).joinToString("  ·  ").takeIf { it.isNotEmpty() }
                if (meta != null) {
                    Text(
                        text = meta,
                        fontSize = 12.sp,
                        color = OnSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
        if (needsInput) {
            SessionPendingActions(
                permission = pendingPermission,
                question = pendingQuestion,
                onApprove = onApprove,
                onDeny = onDeny,
                onReply = onReply,
                onSkip = onSkip,
                modifier = Modifier.padding(start = 16.dp, end = 16.dp, bottom = 10.dp),
            )
        }
    }
}

/** First letter of a session title, for the collapsed-rail avatar. */
private fun sessionInitial(title: String?): String =
    title?.trim()?.firstOrNull()?.uppercaseChar()?.toString() ?: "?"

/** Leading status indicator (full list): spinner while busy, else a dot (needs-input / active / idle). */
@Composable
private fun StatusLeading(busy: Boolean, needsInput: Boolean, isActive: Boolean) {
    if (busy) {
        SessionStatusSpinner("busy", Modifier)
        return
    }
    val color = when {
        needsInput -> Error
        isActive -> Primary
        else -> OnSurfaceGhost
    }
    Box(Modifier.size(12.dp), contentAlignment = Alignment.Center) {
        Box(Modifier.size(7.dp).clip(CircleShape).background(color))
    }
}
