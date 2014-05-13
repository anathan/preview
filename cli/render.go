package cli

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/ngerakines/preview/util"
	_ "io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type PreviewCliCommand interface {
	Execute()
}

type RenderCommand struct {
	host    string
	files   []string
	verbose int
}

type generateRequest struct {
	id       string
	fileType string
	url      string
	size     int64
}

func NewRenderCommand(arguments map[string]interface{}) PreviewCliCommand {
	command := new(RenderCommand)
	command.host = getConfigString(arguments, "<host>")
	if len(command.host) == 0 {
		command.host = "localhost:8080"
	}
	command.files = getConfigStringArray(arguments, "<file>")
	command.verbose = getConfigInt(arguments, "--verbose")
	return command
}

func (command *RenderCommand) String() string {
	return fmt.Sprintf("RenderCommand<host=%s files=%q verbose=%d>", command.host, command.files, command.verbose)
}

func (command *RenderCommand) Execute() {
	for _, file := range command.files {
		fileUrl := command.urlForFile(file)
		if command.verbose > 0 {
			log.Println("Peparing to send file", fileUrl)
		}
		request := newGenerateRequestFromFile(fileUrl)
		if command.verbose > 1 {
			log.Println(request.ToLegacyRequestPayload())
		}
		command.submitGenerateRequest(request)
		if command.verbose > 0 {
			log.Printf("http://%s/asset/%s/jumbo/0", command.host, request.id)
		}
	}
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
	url := command.buildSubmitGenerateRequestUrl(request)
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

func (command *RenderCommand) buildSubmitGenerateRequestUrl(request *generateRequest) string {
	return fmt.Sprintf("http://%s/api/v1/preview/%s", command.host, request.id)
}
