package catalog

import _ "embed"

//go:embed testdata/models.json
var fixtureJSON []byte

// Fixture returns a small embedded catalog (OpenAI gpt-4o + a Groq model). It is
// the deterministic catalog the engine test harness injects, and doubles as a
// minimal built-in default when no models.dev cache is available.
func Fixture() Catalog {
	cat, err := Parse(fixtureJSON)
	if err != nil {
		panic("catalog: embedded fixture is invalid: " + err.Error())
	}
	return cat
}
