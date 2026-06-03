package oauth

import "context"

// Provider is a built-in OAuth-capable provider. It mirrors the per-provider
// `auth` plugin hook opencode loads (plugin/index.ts:88-163), reduced to the
// loopback authorization-code (PKCE) shape Forge supports today. A provider that
// also offers an API-key method advertises it via Methods (Type "api"), but the
// key entry itself flows through PUT /auth/{providerID}; only OAuth methods are
// driven by this package.
type Provider interface {
	// ID is the models.dev provider id (auth.json key), e.g. "xai".
	ID() string
	// Methods lists this provider's auth methods (the GET /provider/auth shape).
	Methods() []Method
	// CallbackPort is the fixed loopback port this provider's OAuth client
	// registered its redirect_uri against, or 0 for "any free port". Some auth
	// servers (e.g. xAI) reject a redirect_uri whose host:port does not match the
	// registered client, so the callback server must bind this exact port
	// (xai.ts:36-43).
	CallbackPort() int
	// Authorize starts the OAuth flow for method index `methodIndex` with the
	// given prompt inputs. redirectURI is the loopback (or proxied) callback URL
	// the daemon will receive the browser redirect on; the provider embeds it in
	// the authorize URL. It returns the Authorization to hand to the client plus
	// a Handshake that completes the flow once the code/redirect arrives.
	Authorize(ctx context.Context, methodIndex int, inputs map[string]string, redirectURI string) (Authorization, *Handshake, error)
}

// Handshake carries the per-attempt secrets a provider needs to finish an OAuth
// flow: the PKCE pair, the CSRF state, and the exchange function. It is held in
// the Service's pending map between authorize() and callback() (the Go analogue
// of provider/auth.ts `pending` Map).
type Handshake struct {
	// State is the CSRF state echoed in the redirect; "" for non-loopback flows.
	State string
	// Exchange turns the authorization code (loopback/code methods) into a Token.
	// For "auto"/device flows that ignore the code, it is invoked with "".
	Exchange func(ctx context.Context, code string) (Token, error)
}
