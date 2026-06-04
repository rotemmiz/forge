package dev.forge.core.store

import java.util.UUID

internal actual fun currentTimeMillis(): Long = System.currentTimeMillis()

internal actual fun randomIdSuffix(): String = UUID.randomUUID().toString().take(8)
