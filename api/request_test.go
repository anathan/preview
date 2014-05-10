package api

import (
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
