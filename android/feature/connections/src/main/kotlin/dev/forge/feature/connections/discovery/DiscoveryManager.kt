package dev.forge.feature.connections.discovery

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Browses the LAN for opencode-compatible daemons and exposes the resolved set as a flow.
 *
 * Orchestration only — the Android plumbing lives in [NsdPlatform]. Responsibilities:
 *  - **Serial resolve.** `NsdManager.resolveService` can throw if called concurrently pre-API-34,
 *    so found services are queued and resolved one at a time (plan 07).
 *  - **De-dupe** by `host:port` so two TXT records or a re-announce don't double-list a daemon.
 *  - **Lifecycle.** [start]/[stop] acquire/release the multicast lock via the platform; [stop]
 *    clears the list and drops any in-flight resolve.
 *
 * Single-threaded: relies on [NsdPlatform] delivering every callback on one thread.
 */
@Singleton
class DiscoveryManager @Inject constructor(
    private val platform: NsdPlatform,
) {
    private val _servers = MutableStateFlow<List<DiscoveredServer>>(emptyList())
    val servers: StateFlow<List<DiscoveredServer>> = _servers.asStateFlow()

    private val _scanning = MutableStateFlow(false)
    val scanning: StateFlow<Boolean> = _scanning.asStateFlow()

    private var started = false

    // Serial-resolve state (touched only on the platform's callback thread).
    private val pending = ArrayDeque<RawService>()
    private var resolving = false

    private val callbacks = object : NsdPlatform.Callbacks {
        override fun onServiceFound(service: RawService) = enqueueResolve(service)
        override fun onServiceLost(service: RawService) = removeByName(service.name)
    }

    fun start(serviceTypes: List<String> = DEFAULT_SERVICE_TYPES) {
        if (started) return
        started = true
        _scanning.value = true
        platform.start(serviceTypes, callbacks)
    }

    fun stop() {
        if (!started) return
        started = false
        _scanning.value = false
        pending.clear()
        resolving = false
        platform.stop()
        _servers.value = emptyList()
    }

    private fun enqueueResolve(service: RawService) {
        if (!started) return
        if (!isInteresting(service)) return
        if (pending.any { it.name == service.name }) return
        pending.addLast(service)
        pumpResolve()
    }

    /**
     * `_opencode._tcp` records are always ours. `_http._tcp` is a firehose of every HTTP service on
     * the LAN (printers, casts, …), so accept those only when named like opencode's native mDNS
     * (`opencode-<port>`, see opencode `server/mdns.ts`) — avoids resolving the whole neighbourhood.
     */
    private fun isInteresting(service: RawService): Boolean =
        service.type.contains("_opencode", ignoreCase = true) ||
            service.name.startsWith("opencode", ignoreCase = true)

    private fun pumpResolve() {
        if (resolving) return
        val next = pending.removeFirstOrNull() ?: return
        resolving = true
        platform.resolve(next) { resolved ->
            resolving = false
            if (started && resolved != null) addOrUpdate(resolved)
            if (started) pumpResolve()
        }
    }

    private fun addOrUpdate(server: DiscoveredServer) {
        val hostPort = "${server.host}:${server.port}"
        val others = _servers.value.filterNot { "${it.host}:${it.port}" == hostPort }
        _servers.value = (others + server).sortedBy { it.serviceName.lowercase() }
    }

    private fun removeByName(name: String) {
        _servers.value = _servers.value.filterNot { it.serviceName == name }
    }

    companion object {
        /** Our preferred type — only real daemons answer (a Forge daemon should publish this). */
        const val SERVICE_TYPE_OPENCODE = "_opencode._tcp."

        /** opencode's native `--mdns` publishes a generic HTTP record; filtered by name. */
        const val SERVICE_TYPE_HTTP = "_http._tcp."

        val DEFAULT_SERVICE_TYPES = listOf(SERVICE_TYPE_OPENCODE, SERVICE_TYPE_HTTP)
    }
}
