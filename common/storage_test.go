package common

import (
	_ "github.com/ngerakines/testutils"
	"testing"
)

func TestInMemorySourceAssetStorage(t *testing.T) {
	sasm := NewSourceAssetStorageManager()

	sourceAsset, err := NewSourceAsset("4AE594A7-A48E-45E4-A5E1-4533E50BBDA3", SourceAssetTypeOrigin)
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}
	err = sasm.Store(sourceAsset)
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
		t.Error("One result expected:", len(results))
		return
	}
	if results[0].Id != "4AE594A7-A48E-45E4-A5E1-4533E50BBDA3" {
		t.Errorf("Unexpected result returned: (%+v)", results[0])
		return
	}
}
