package persistence

import "testing"

func TestUnmarshalJSONBRejectsCorruptData(t *testing.T) {
	var values []string
	if err := unmarshalJSONB(&values, []byte("{")); err == nil {
		t.Fatal("expected corrupt JSONB to return error")
	}
}

func TestUnmarshalJSONBAllowsNullAndEmptyData(t *testing.T) {
	for _, raw := range [][]byte{nil, {}, []byte("null")} {
		var values []string
		if err := unmarshalJSONB(&values, raw); err != nil {
			t.Fatalf("unmarshal %q: %v", raw, err)
		}
	}
}
