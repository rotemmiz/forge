package dev.forge.app

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.core.content.ContextCompat
import dagger.hilt.android.AndroidEntryPoint
import dev.forge.app.navigation.ForgeNavGraph
import dev.forge.feature.chat.ui.ForgeTheme
import dev.forge.feature.notifications.PushController
import dev.forge.feature.notifications.PushDeepLink
import dev.forge.feature.settings.AppPreferences
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    @Inject
    lateinit var prefs: AppPreferences

    @Inject
    lateinit var pushController: PushController

    // Latest push deep-link target (session to open), updated on launch + onNewIntent.
    private val deepLink = MutableStateFlow<PushDeepLink.Target?>(null)

    private val requestNotificationPermission =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { /* result ignored */ }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        deepLink.value = PushDeepLink.fromIntent(intent)
        maybeRequestNotificationPermission()
        setContent {
            val darkTheme by prefs.darkTheme.collectAsState(initial = true)
            val scope = rememberCoroutineScope()
            val target by deepLink.collectAsState()
            val consumed = remember { mutableStateOf<String?>(null) }
            // Emit the session id once per distinct deep-link tap.
            val pendingSessionId = target?.sessionId?.takeIf { it != consumed.value }
            ForgeTheme(darkTheme = darkTheme) {
                ForgeNavGraph(
                    isDarkTheme = darkTheme,
                    onToggleTheme = { scope.launch { prefs.setDarkTheme(!darkTheme) } },
                    deepLinkSessionId = pendingSessionId,
                    onDeepLinkConsumed = { consumed.value = it },
                )
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        PushDeepLink.fromIntent(intent)?.let { deepLink.value = it }
    }

    private fun maybeRequestNotificationPermission() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return
        if (!pushController.isPushAvailable) return
        val granted = ContextCompat.checkSelfPermission(
            this, Manifest.permission.POST_NOTIFICATIONS,
        ) == PackageManager.PERMISSION_GRANTED
        if (!granted) requestNotificationPermission.launch(Manifest.permission.POST_NOTIFICATIONS)
    }
}
