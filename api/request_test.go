package api

import (
	_ "github.com/ngerakines/testutils"
	"testing"
)

func TestNewGeneratePreviewRequestFromTextInvalidId(t *testing.T) {
	_, err := newGeneratePreviewRequestFromText("", "")
	if err == nil {
		t.Error("No error was returned, but expected 'PRVCOM2'.")
		return
	}
	if err.Error() != "PRVCOM2" {
		t.Errorf("Invalid error returned, should be 'PRVCOM2': %s", err.Error())
	}
}

func TestNewGeneratePreviewRequestFromTextMissingType(t *testing.T) {
	_, err := newGeneratePreviewRequestFromText("1234", "")
	if err == nil {
		t.Error("No error was returned, but expected 'PRVCOM3'.")
		return
	}
	if err.Error() != "PRVCOM3" {
		t.Errorf("Invalid error returned, should be 'PRVCOM3': %s", err.Error())
	}
}

func TestNewGeneratePreviewRequestFromTextMissingUrl(t *testing.T) {
	_, err := newGeneratePreviewRequestFromText("1234", "type: jpg\n")
	if err == nil {
		t.Error("No error was returned, but expected 'PRVCOM4'.")
		return
	}
	if err.Error() != "PRVCOM4" {
		t.Errorf("Invalid error returned, should be 'PRVCOM4': %q", err.Error())
	}
}

func TestNewGeneratePreviewRequestFromTextMissingSize(t *testing.T) {
	_, err := newGeneratePreviewRequestFromText("1234", "type: jpg\nurl: http://www.hightail.com/\n")
	if err == nil {
		t.Error("No error was returned, but expected 'PRVCOM5'.")
		return
	}
	if err.Error() != "PRVCOM5" {
		t.Errorf("Invalid error returned, should be 'PRVCOM5': %q", err.Error())
	}
}

func TestJsonParsing(t *testing.T) {
	example := `{
    "version": 1,
    "files": [
        {
            "file_id": "abcd1234",
            "url": "http://ngerakines.me/resume.pdf",
            "size": "12345",
            "type": "pdf"
        },
        {
            "file_id": "abcd1235",
            "url": "http://ngerakines.me/ngerakines.png",
            "size": "12346",
            "type": "png"
        }
    ]
}`
	gprs, err := newGeneratePreviewRequestFromJson(example)
	if err != nil {
		t.Error("Unexpected error parsing json:", err)
		return
	}
	if len(gprs) != 2 {
		t.Error("Expected 2 generate preview requests but got", len(gprs))
	}
}

func TestJsonParsingExampleA(t *testing.T) {
	example := `{"version":1,"files":[{"file_id":"a6270b69-10b7-4649-ac25-505df3524194","batch_id":"113930","url":"https://pat/to/file","size":"879394","type":"jpg","expiration_time":null}]}`
	gprs, err := newGeneratePreviewRequestFromJson(example)
	if err != nil {
		t.Error("Unexpected error parsing json:", err)
		return
	}
	if len(gprs) != 1 {
		t.Error("Expected one generate preview request but got", len(gprs))
	}
}
