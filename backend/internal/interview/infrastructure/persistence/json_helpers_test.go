package persistence

import "testing"

func TestDecodeJSONBRejectsCorruptData(t *testing.T) {
	var value struct {
		Name string `json:"name"`
	}
	if err := decodeJSONB([]byte("{"), &value); err == nil {
		t.Fatal("expected corrupt JSONB to return error")
	}
}

func TestDecodeJSONBAllowsNullAndEmptyData(t *testing.T) {
	for _, raw := range [][]byte{nil, {}, []byte("null")} {
		var value struct {
			Name string `json:"name"`
		}
		if err := decodeJSONB(raw, &value); err != nil {
			t.Fatalf("decode %q: %v", raw, err)
		}
	}
}
