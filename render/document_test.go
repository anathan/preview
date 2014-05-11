package render

import (
	_ "github.com/ngerakines/testutils"
	"testing"
)

func TestConvertDocxToPdf(t *testing.T) {
	if testing.Short() {
		t.Skip("Short Tests Only: TestRenderJpegPreview")
		return
	}

}
