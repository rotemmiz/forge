package dev.opcode42.core.data

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.IOException

/**
 * Pins the [resultOf] contract the repository cancellation-safety relies on: it must wrap ordinary
 * failures as `Result.failure` but **re-throw** [CancellationException] so structured-concurrency
 * cancellation is never swallowed (see the loadDiff/send fixes in [DefaultChatRepository]).
 */
class DataResultTest {

    @Test fun rethrowsCancellation() {
        assertThrows(CancellationException::class.java) {
            runBlocking { resultOf { throw CancellationException("cancelled") } }
        }
    }

    @Test fun wrapsNonCancellationExceptionsAsFailure() = runTest {
        val result = resultOf { throw IOException("boom") }
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull() is IOException)
    }

    @Test fun returnsSuccessValue() = runTest {
        assertEquals(42, resultOf { 42 }.getOrNull())
    }
}
