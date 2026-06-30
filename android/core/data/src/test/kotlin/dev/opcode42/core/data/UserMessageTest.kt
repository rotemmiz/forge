package dev.opcode42.core.data

import dev.opcode42.core.sdk.HttpException
import dev.opcode42.core.sdk.NotConfiguredException
import org.junit.Assert.assertEquals
import org.junit.Test
import java.io.IOException

/** Pins the failure → user-facing message mapping the chat/sessions snackbars rely on. */
class UserMessageTest {

    @Test fun mapsAuthErrors() {
        assertEquals("Not authorized", HttpException(401, null).toUserMessage())
        assertEquals("Not authorized", HttpException(403, null).toUserMessage())
    }

    @Test fun mapsNotFound() {
        assertEquals("Not found", HttpException(404, "x").toUserMessage())
    }

    @Test fun mapsServerErrorsWithCode() {
        assertEquals("Server error (500)", HttpException(500, null).toUserMessage())
        assertEquals("Server error (503)", HttpException(503, null).toUserMessage())
    }

    @Test fun mapsOtherHttpCodesWithCode() {
        assertEquals("Request failed (418)", HttpException(418, null).toUserMessage())
    }

    @Test fun mapsNoServerConfigured() {
        assertEquals("No server configured", NotConfiguredException().toUserMessage())
    }

    @Test fun mapsConnectivityFailure() {
        assertEquals("Can't reach the server", IOException("socket").toUserMessage())
    }

    @Test fun mapsUnknownToGenericMessage() {
        assertEquals("Something went wrong", IllegalArgumentException("boom").toUserMessage())
    }
}
