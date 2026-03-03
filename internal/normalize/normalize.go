package normalize

import (
	"encoding/json"
	"fmt"
)

// DiagnosticCode identifies the reason for a diagnostic entry.
type DiagnosticCode string

const (
	DiagnosticCodeNormalizationFallback DiagnosticCode = "NORMALIZATION_FALLBACK"
	DiagnosticCodeTimeout               DiagnosticCode = "TIMEOUT"
)

// Diagnostic carries a machine-readable code and human-readable message.
type Diagnostic struct {
	Code    DiagnosticCode `json:"code"`
	Message string         `json:"message"`
}

// Data is the normalized output payload returned inside the MCP response envelope.
type Data struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Normalizer converts raw CLI stdout into structured Data.
type Normalizer struct{}

// New creates a new Normalizer.
func New() *Normalizer {
	return &Normalizer{}
}

// Normalize attempts to parse stdout as JSON and wrap it in a Data struct.
// If stdout is valid JSON it is used as-is for the payload.
// If parsing fails, a NORMALIZATION_FALLBACK diagnostic is returned with the raw text.
func (n *Normalizer) Normalize(dataType string, stdout []byte) (Data, []Diagnostic) {
	if len(stdout) == 0 {
		return Data{Type: dataType, Payload: json.RawMessage(`{}`)}, nil
	}

	if json.Valid(stdout) {
		return Data{Type: dataType, Payload: json.RawMessage(stdout)}, nil
	}

	// Text fallback: wrap raw output in a JSON object.
	raw, err := textFallback(stdout)
	if err != nil {
		raw = json.RawMessage(`{"raw":""}`)
	}

	diag := Diagnostic{
		Code:    DiagnosticCodeNormalizationFallback,
		Message: fmt.Sprintf("normalize: stdout is not valid JSON, falling back to raw text for type %q", dataType),
	}
	return Data{Type: dataType, Payload: raw}, []Diagnostic{diag}
}

func textFallback(stdout []byte) (json.RawMessage, error) {
	type textPayload struct {
		Raw string `json:"raw"`
	}
	b, err := json.Marshal(textPayload{Raw: string(stdout)})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
