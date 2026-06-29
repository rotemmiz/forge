package dev.opcode42.core.data

import dagger.Binds
import dagger.Module
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

/**
 * Binds repository interfaces to their default implementations. Mirrors the `@Binds` abstract-class
 * pattern in :feature:connections. The impls are `@Singleton` with `@Inject` constructors, so Hilt
 * needs only these bindings — no `@Provides`.
 */
@Module
@InstallIn(SingletonComponent::class)
abstract class DataModule {
    @Binds @Singleton
    abstract fun bindSessionRepository(impl: DefaultSessionRepository): SessionRepository

    @Binds @Singleton
    abstract fun bindChatRepository(impl: DefaultChatRepository): ChatRepository

    @Binds @Singleton
    abstract fun bindTerminalRepository(impl: DefaultTerminalRepository): TerminalRepository
}
