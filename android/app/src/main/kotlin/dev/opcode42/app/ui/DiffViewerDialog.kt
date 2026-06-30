package dev.opcode42.app.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import dev.opcode42.core.design.brand.Spinner
import dev.opcode42.core.design.text.StartEllipsisText
import dev.opcode42.core.design.theme.Error
import dev.opcode42.core.design.theme.Hairline
import dev.opcode42.core.design.theme.OnSurfaceFaint
import dev.opcode42.core.design.theme.OnSurfaceVariant
import dev.opcode42.core.design.theme.Opcode42Mono
import dev.opcode42.core.design.theme.Opcode42Shapes
import dev.opcode42.core.design.theme.OutlineVariant
import dev.opcode42.core.design.theme.SurfaceContainer
import dev.opcode42.core.design.theme.Tertiary
import dev.opcode42.core.model.SnapshotFileDiff
import dev.opcode42.feature.chat.ui.UnifiedDiffView

/**
 * A centered popup that shows one CHANGES file's diff, rendered in the *exact* chat style — it
 * wraps the chat's public [UnifiedDiffView] verbatim (per-line tint, hunk headers, horizontal
 * scroll, 400-line cap) so a tapped row looks pixel-identical to the inline edit card. The window
 * is just chrome: a [Dialog] + a header (path · +adds −dels · close) over the diff body.
 *
 * The patch is fetched lazily via [loadDiff] on open (the lightweight `/vcs/status` row carries no
 * patch). The header counts come from [summary] so they show immediately, before the body loads.
 */
@Composable
internal fun DiffViewerDialog(
    summary: SnapshotFileDiff,
    loadDiff: suspend (String) -> SnapshotFileDiff?,
    onDismiss: () -> Unit,
) {
    val file = summary.file?.takeIf { it.isNotBlank() } ?: "unknown"
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false),
    ) {
        var state by remember(file) { mutableStateOf<DiffLoad>(DiffLoad.Loading) }
        LaunchedEffect(file) {
            // [loadDiff] is best-effort (no `/vcs`, or the file isn't in the patch set → null), so
            // a null result — not an exception — collapses to "Empty diff."; cancellation propagates.
            val diff = loadDiff(file)
            state = if (diff == null) DiffLoad.Empty else DiffLoad.Loaded(diff)
        }
        Surface(
            shape = Opcode42Shapes.sm,
            color = SurfaceContainer,
            border = BorderStroke(1.dp, OutlineVariant),
            modifier = Modifier
                .fillMaxWidth(0.92f)
                .fillMaxHeight(0.8f),
        ) {
            Column(Modifier.fillMaxSize()) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(start = 14.dp, end = 4.dp, top = 4.dp, bottom = 4.dp),
                ) {
                    // Start-ellipsize so the filename + extension (the useful tail) survives.
                    StartEllipsisText(
                        text = file,
                        style = TextStyle(fontFamily = Opcode42Mono, fontSize = 13.sp, color = Tertiary),
                        modifier = Modifier.weight(1f),
                    )
                    Spacer(Modifier.width(8.dp))
                    Text(
                        text = buildAnnotatedString {
                            withStyle(SpanStyle(color = Tertiary)) { append("+${summary.additions}") }
                            append(" ")
                            withStyle(SpanStyle(color = Error)) { append("−${summary.deletions}") }
                        },
                        fontFamily = Opcode42Mono,
                        fontSize = 12.5.sp,
                    )
                    IconButton(onClick = onDismiss) {
                        Icon(
                            Icons.Default.Close,
                            contentDescription = "Close",
                            tint = OnSurfaceVariant,
                            modifier = Modifier.size(18.dp),
                        )
                    }
                }
                HorizontalDivider(color = Hairline)
                Box(Modifier.fillMaxWidth().weight(1f)) {
                    when (val s = state) {
                        DiffLoad.Loading ->
                            Spinner(modifier = Modifier.align(Alignment.Center), visible = true)
                        DiffLoad.Empty ->
                            Text(
                                text = "Empty diff.",
                                fontFamily = Opcode42Mono,
                                fontSize = 13.sp,
                                color = OnSurfaceFaint,
                                modifier = Modifier.align(Alignment.Center).padding(24.dp),
                            )
                        is DiffLoad.Loaded ->
                            Column(
                                Modifier.fillMaxSize().verticalScroll(rememberScrollState()),
                            ) {
                                UnifiedDiffView(diffs = listOf(s.diff))
                            }
                    }
                }
            }
        }
    }
}

/** Load state for the lazily-fetched patch driving [DiffViewerDialog]. */
private sealed interface DiffLoad {
    data object Loading : DiffLoad
    data object Empty : DiffLoad
    data class Loaded(val diff: SnapshotFileDiff) : DiffLoad
}
