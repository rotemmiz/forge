package spec

import (
	"encoding/json"
	"reflect"
	"testing"
)

func jsonEqual(t *testing.T, a, b json.RawMessage) bool {
	t.Helper()
	var av, bv any
	if err := json.Unmarshal(a, &av); err != nil {
		t.Fatalf("unmarshal a: %v", err)
	}
	if err := json.Unmarshal(b, &bv); err != nil {
		t.Fatalf("unmarshal b: %v", err)
	}
	return reflect.DeepEqual(av, bv)
}

// TestEmitMatchesReferenceWhenFullyWired asserts that emitting from the full
// reference operation set reproduces a spec whose operation set is identical to
// the frozen contract — the happy path the daemon hits once every reference
// operation is registered (real handler or 501 stub).
func TestEmitMatchesReferenceWhenFullyWired(t *testing.T) {
	refOps, err := Operations()
	if err != nil {
		t.Fatalf("Operations: %v", err)
	}
	doc, err := Emit(refOps)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}

	emitted, err := EmittedOperations(doc)
	if err != nil {
		t.Fatalf("EmittedOperations: %v", err)
	}
	baseline, err := EmittedOperations(reference)
	if err != nil {
		t.Fatalf("EmittedOperations(reference): %v", err)
	}

	drift := CompareOps(baseline, emitted, nil)
	if drift.Breaking() {
		t.Fatalf("emitted-from-reference must not drift; got:\n%v", drift.Report())
	}
	if len(emitted) != len(baseline) {
		t.Errorf("emitted op count = %d, want %d (reference)", len(emitted), len(baseline))
	}
}

// TestEmitIsDeterministic asserts the emitted bytes are stable across calls, so
// the gate and the served /openapi.json are reproducible.
func TestEmitIsDeterministic(t *testing.T) {
	refOps, err := Operations()
	if err != nil {
		t.Fatalf("Operations: %v", err)
	}
	a, err := Emit(refOps)
	if err != nil {
		t.Fatalf("Emit a: %v", err)
	}
	b, err := Emit(refOps)
	if err != nil {
		t.Fatalf("Emit b: %v", err)
	}
	if string(a) != string(b) {
		t.Fatal("Emit is not deterministic across calls")
	}
}

// TestEmitPreservesReferenceSchemas asserts the emitted doc reuses the frozen
// reference's top-level fields (components, info) verbatim, so request/response
// schemas stay byte-identical to the contract.
func TestEmitPreservesReferenceSchemas(t *testing.T) {
	refOps, err := Operations()
	if err != nil {
		t.Fatalf("Operations: %v", err)
	}
	doc, err := Emit(refOps)
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}

	var emittedDoc, refDoc map[string]json.RawMessage
	if err := json.Unmarshal(doc, &emittedDoc); err != nil {
		t.Fatalf("unmarshal emitted: %v", err)
	}
	if err := json.Unmarshal(reference, &refDoc); err != nil {
		t.Fatalf("unmarshal reference: %v", err)
	}
	// MarshalIndent re-indents nested JSON, so compare semantically (not byte-wise):
	// the schema content must be unchanged from the frozen contract.
	for _, field := range []string{"openapi", "info", "components"} {
		if !jsonEqual(t, emittedDoc[field], refDoc[field]) {
			t.Errorf("emitted %q differs semantically from reference", field)
		}
	}
}

// TestEmitTagsAdditions asserts an operation not in the reference is emitted as a
// tagged Forge addition (so the diff gate classifies it as additive).
func TestEmitTagsAdditions(t *testing.T) {
	extra := Operation{Method: "GET", Path: "/openapi.json"} // a real known-addition
	doc, err := Emit([]Operation{extra})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	var parsed struct {
		Paths map[string]map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(doc, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	op := parsed.Paths["/openapi.json"]["get"]
	if string(op["x-forge-addition"]) != "true" {
		t.Errorf("addition not tagged: got x-forge-addition=%s", op["x-forge-addition"])
	}
}
