package api

import (
	"encoding/json"
	"github.com/ngerakines/preview/common"
	"strconv"
)

type generatePreviewRequest struct {
	id          string
	requestType string
	url         string
	size        int64
}

func newGeneratePreviewRequestFromText(id, body string) ([]*generatePreviewRequest, error) {
	if len(id) == 0 {
		return nil, common.ErrorInvalidFileId
	}
	vals := splitText(body)
	gpr := new(generatePreviewRequest)
	gpr.id = id

	requestType, hasRequestType := vals["type"]
	if !hasRequestType {
		return nil, common.ErrorMissingFieldType
	}
	gpr.requestType = requestType

	url, hasUrl := vals["url"]
	if !hasUrl {
		return nil, common.ErrorMissingFieldUrl
	}
	gpr.url = url

	size, hasSize := vals["size"]
	if !hasSize {
		return nil, common.ErrorMissingFieldSize
	}
	sizeValue, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		// TODO: This should return a different error.
		return nil, common.ErrorMissingFieldSize
	}
	gpr.size = sizeValue

	gprs := make([]*generatePreviewRequest, 0, 0)
	gprs = append(gprs, gpr)
	return gprs, nil
}

func newGeneratePreviewRequestFromJson(body string) ([]*generatePreviewRequest, error) {
	var data struct {
		Version int `json:"version"`
		Files   []struct {
			Id          string `json:"file_id"`
			RequestType string `json:"type"`
			Url         string `json:"url"`
			Size        string `json:"size"`
		} `json:"files"`
	}
	err := json.Unmarshal([]byte(body), &data)
	if err != nil {
		return nil, err
	}

	gprs := make([]*generatePreviewRequest, 0, 0)
	for _, file := range data.Files {
		gpr := new(generatePreviewRequest)
		gpr.id = file.Id
		gpr.requestType = file.RequestType
		sizeValue, err := strconv.ParseInt(file.Size, 10, 64)
		if err != nil {
			return nil, common.ErrorMissingFieldSize
		}
		gpr.size = sizeValue
		gpr.url = file.Url
		gprs = append(gprs, gpr)
	}
	return gprs, nil
}
