package common

import (
	"testing"
)

func TestInMemorySourceAssetStorage(t *testing.T) {
	sasm := NewSourceAssetStorageManager()

	err := sasm.Store(NewSourceAsset("4AE594A7-A48E-45E4-A5E1-4533E50BBDA3", SourceAssetTypeOrigin))
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}

	results, err := sasm.FindBySourceAssetId("4AE594A7-A48E-45E4-A5E1-4533E50BBDA3")
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}
	if len(results) != 1 {
		t.Error("One result expected")
		return
	}
	if results[0].Id != "4AE594A7-A48E-45E4-A5E1-4533E50BBDA3" {
		t.Errorf("Unexpected result returned: (%+v)", results[0])
		return
	}
}