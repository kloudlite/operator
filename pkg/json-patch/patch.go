package json_patch

import (
	"encoding/json"
	jsonpatch "github.com/evanphx/json-patch/v5"
)

// +kubebuilder:object:generate=true

type PatchOperation struct {
	Op   string `json:"op"`
	Path string `json:"path"`
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Value json.RawMessage `json:"value,omitempty"`
}

func ApplyPatch(jsonDoc any, patch []PatchOperation) ([]byte, error) {
	doc, err := json.Marshal(jsonDoc)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(patch)
	if err != nil {
		return nil, err
	}

	p, err := jsonpatch.DecodePatch(b)
	if err != nil {
		return nil, err
	}

	return p.Apply(doc)
}
