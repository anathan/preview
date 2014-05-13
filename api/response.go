package api

type previewInfoCollection struct {
	FileId string     `json:"file_id"`
	Page   int32      `json:"page"`
	Jumbo  *imageInfo `json:"jumbo"`
	Large  *imageInfo `json:"large"`
	Medium *imageInfo `json:"medium"`
	Small  *imageInfo `json:"small"`
}

type imageInfo struct {
	Url           string `json:"url"`
	Width         int32  `json:"width"`
	Height        int32  `json:"height"`
	Expires       int64  `json:"expires"`
	IsFinal       bool   `json:"isFinal"`
	IsPlaceholder bool   `json:"isPlaceholder"`
	Page          int32  `json:"page"`
}

type previewInfoResponse struct {
	Version string                   `json:"version"`
	Files   []*previewInfoCollection `json:"files"`
}
