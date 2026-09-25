package models

import (
	"encoding/json"
	"testing"
)

func TestWrongPictureConfigShowTargetCountRoundTrip(t *testing.T) {
	// Frontend payload WITH the toggle on must survive JSON decode.
	raw := []byte(`{"targets":[],"tolerancePercent":5,"maxPoints":1000,"showTargetCount":true}`)
	var cfg WrongPictureConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !cfg.ShowTargetCount {
		t.Fatalf("showTargetCount=true was dropped on decode")
	}
	out, _ := json.Marshal(cfg)
	var back map[string]interface{}
	json.Unmarshal(out, &back)
	if back["showTargetCount"] != true {
		t.Fatalf("showTargetCount missing in re-encoded JSON: %s", out)
	}

	// Old documents WITHOUT the field must decode to false (zero value).
	var old WrongPictureConfig
	if err := json.Unmarshal([]byte(`{"targets":[],"tolerancePercent":5,"maxPoints":1000}`), &old); err != nil {
		t.Fatalf("unmarshal old doc: %v", err)
	}
	if old.ShowTargetCount {
		t.Fatalf("old doc without field should default to false")
	}
}
