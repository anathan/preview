package cli

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"fmt"
	"github.com/ngerakines/preview/util"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PreviewCliCommand interface {
	Execute()
}

type RenderCommand struct {
	host    string
	files   []string
	verbose int
	verify  bool
}

type generateRequest struct {
	id       string
	fileType string
	url      string
	size     int64
}

type generateResponse struct {
	body string
}

type imageInfo struct {
	Url           string  `json:"url"`
	Width         float64 `json:"width"`
	Height        float64 `json:"height"`
	Expires       float64 `json:"expires"`
	IsFinal       bool    `json:"isFinal"`
	IsPlaceholder bool    `json:"isPlaceholder"`
}

type previewInfoResponse struct {
	Version string `json:"version"`
	Files   []struct {
		FileId string    `json:"file_id"`
		Jumbo  imageInfo `json:"jumbo"`
		Large  imageInfo `json:"large"`
		Medium imageInfo `json:"medium"`
		Small  imageInfo `json:"small"`
	} `json:"files"`
}

func NewRenderCommand(arguments map[string]interface{}) PreviewCliCommand {
	command := new(RenderCommand)
	command.host = getConfigString(arguments, "<host>")
	if len(command.host) == 0 {
		command.host = "localhost:8080"
	}
	command.files = getConfigStringArray(arguments, "<file>")
	command.verbose = getConfigInt(arguments, "--verbose")
	command.verify = getConfigBool(arguments, "--verify")
	return command
}

func (command *RenderCommand) String() string {
	return fmt.Sprintf("RenderCommand<host=%s files=%q verbose=%d>", command.host, command.files, command.verbose)
}

func (command *RenderCommand) Execute() {
	pendingIds := make(map[string]bool)
	for _, file := range command.filesToSubmit() {
		fileUrl := command.urlForFile(file)
		if command.verbose > 0 {
			log.Println("Peparing to send file", fileUrl)
		}
		request := newGenerateRequestFromFile(fileUrl)
		pendingIds[request.id] = false
		if command.verbose > 1 {
			log.Println(request.ToLegacyRequestPayload())
		}
		command.submitGenerateRequest(request)
		if command.verbose > 0 {
			log.Printf("http://%s/asset/%s/jumbo/0", command.host, request.id)
		}
	}
	if command.verify {
		for len(pendingIds) > 0 {
			for id := range pendingIds {
				previewInfoResponse, previewInfoResponseErr := command.submitPreviewInfoRequest(id)
				if previewInfoResponseErr != nil {
					log.Println("Error checking", id, ":", previewInfoResponseErr)
					delete(pendingIds, id)
				} else {
					if command.isComplete(previewInfoResponse) {
						delete(pendingIds, id)
					}
				}
			}
			if len(pendingIds) > 0 {
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (command *RenderCommand) filesToSubmit() []string {
	files := make([]string, 0, 0)
	for _, file := range command.files {
		shouldTry, path := command.absFilePath(file)
		if shouldTry {
			f, err := os.Open(path)
			if err == nil {
				defer f.Close()
				fi, err := f.Stat()
				if err == nil {
					switch mode := fi.Mode(); {
					case mode.IsDir():
						subdirFiles, err := ioutil.ReadDir(path)
						if err == nil {
							for _, subdirFile := range subdirFiles {
								if !subdirFile.IsDir() {
									files = append(files, filepath.Join(path, subdirFile.Name()))
								}
							}
						}
					case mode.IsRegular():
						files = append(files, path)
					}
				}
			}
		}
	}
	return files
}

func (command *RenderCommand) absFilePath(file string) (bool, string) {
	if strings.HasPrefix(file, "http://") || strings.HasPrefix(file, "https://") {
		return false, ""
	}
	if strings.HasPrefix(file, "file://") {
		return true, file[7:]
	}
	if strings.HasPrefix(file, "/") {
		return true, file
	}
	return true, filepath.Join(util.Cwd(), file)
}

func (command *RenderCommand) urlForFile(file string) string {
	if strings.HasPrefix(file, "http://") || strings.HasPrefix(file, "https://") {
		panic("Remote files are not supported.")
	}
	if strings.HasPrefix(file, "file://") {
		return file
	}
	if strings.HasPrefix(file, "/") {
		return "file://" + file
	}
	return "file://" + filepath.Join(util.Cwd(), file)
}

func (request *generateRequest) ToLegacyRequestPayload() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("type: %s\n", request.fileType))
	buffer.WriteString(fmt.Sprintf("url: %s\n", request.url))
	buffer.WriteString(fmt.Sprintf("size: %d\n", request.size))
	return buffer.String()
}

func newGenerateRequestFromFile(file string) *generateRequest {
	localFilePath := file[5:]
	log.Println("Creating new request for file", localFilePath)
	fi, err := os.Stat(localFilePath)
	if err != nil {
		panic("Could not read file size")
	}
	return newGenerateRequest(uuid.New(), filepath.Ext(localFilePath)[1:], file, fi.Size())
}

func newGenerateRequest(id, fileType, url string, size int64) *generateRequest {
	request := new(generateRequest)
	request.id = id
	request.fileType = fileType
	request.url = url
	request.size = size
	return request
}

func (command *RenderCommand) submitGenerateRequest(request *generateRequest) error {
	url := command.buildSubmitGenerateRequestUrl(request.id)
	if command.verbose > 0 {
		log.Println("Submitting request to", url)
	}
	req, err := http.NewRequest("PUT", url, strings.NewReader(request.ToLegacyRequestPayload()))
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}

func (command *RenderCommand) submitPreviewInfoRequest(id string) (*previewInfoResponse, error) {
	url := command.buildSubmitPreviewInfoRequest(id)
	if command.verbose > 0 {
		log.Println("Submitting request to", url)
	}
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	return newPreviewInfoResponse(body)
}

func (command *RenderCommand) buildSubmitGenerateRequestUrl(id string) string {
	return fmt.Sprintf("http://%s/api/v1/preview/%s", command.host, id)
}

func (command *RenderCommand) buildSubmitPreviewInfoRequest(id string) string {
	return fmt.Sprintf("http://%s/api/v1/preview/%s", command.host, id)
}

func (command *RenderCommand) isComplete(response *previewInfoResponse) bool {
	complete := true
	for _, file := range response.Files {
		if file.Jumbo.IsFinal == false {
			if command.verbose > 1 {
				log.Println("File", file.FileId, "incomplete:", file.Jumbo.Url)
			}
			complete = false
		}
		if file.Large.IsFinal == false {
			if command.verbose > 1 {
				log.Println("File", file.FileId, "incomplete:", file.Large.Url)
			}
			complete = false
		}
		if file.Medium.IsFinal == false {
			if command.verbose > 1 {
				log.Println("File", file.FileId, "incomplete:", file.Medium.Url)
			}
			complete = false
		}
		if file.Small.IsFinal == false {
			if command.verbose > 1 {
				log.Println("File", file.FileId, "incomplete:", file.Small.Url)
			}
			complete = false
		}
	}
	if complete && command.verbose > 0 {
		for _, file := range response.Files {
			log.Println("File", file.FileId, "complete:", file.Jumbo.Url)
			log.Println("File", file.FileId, "complete:", file.Large.Url)
			log.Println("File", file.FileId, "complete:", file.Medium.Url)
			log.Println("File", file.FileId, "complete:", file.Small.Url)
		}
	}
	return complete
}

func newPreviewInfoResponse(body []byte) (*previewInfoResponse, error) {
	var response previewInfoResponse
	e := json.Unmarshal(body, &response)
	if e != nil {
		return nil, e
	}
	return &response, nil
}

func formatPhpDate(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
