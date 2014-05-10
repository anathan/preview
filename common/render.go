package common

import (
	"bytes"
	"image"
	"image/jpeg"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"
)

var (
	RenderAgentImageMagick = "renderAgentImageMagick"
	RenderAgents           = []string{RenderAgentImageMagick}
)

type ImageMagickRendererWorkChannel chan string
type RenderStatusChannel chan RenderStatus

type Renderer interface {
	Stop()
	Dispatch() ImageMagickRendererWorkChannel
	AddStatusListener(listener RenderStatusChannel)
}

type RenderStatus struct {
	GeneratedAssetId string
	Status           string
	Service          string
}

type imageMagickRenderer struct {
	sasm                 SourceAssetStorageManager
	gasm                 GeneratedAssetStorageManager
	templateManager      TemplateManager
	downloader           Downloader
	uploader             Uploader
	incomingWork         ImageMagickRendererWorkChannel
	statusListeners      []RenderStatusChannel
	temporaryFileManager TemporaryFileManager
	stop                 chan (chan bool)
}

func NewImageMagickRenderer(
	sasm SourceAssetStorageManager,
	gasm GeneratedAssetStorageManager,
	templateManager TemplateManager,
	temporaryFileManager TemporaryFileManager,
	downloader Downloader,
	uploader Uploader,
	incomingWork ImageMagickRendererWorkChannel) Renderer {

	renderer := new(imageMagickRenderer)
	renderer.sasm = sasm
	renderer.gasm = gasm
	renderer.templateManager = templateManager
	renderer.temporaryFileManager = temporaryFileManager
	renderer.downloader = downloader
	renderer.uploader = uploader
	renderer.incomingWork = incomingWork
	renderer.statusListeners = make([]RenderStatusChannel, 0, 0)
	renderer.stop = make(chan (chan bool))

	go renderer.start()

	return renderer
}

func (renderer *imageMagickRenderer) start() {
	for {
		select {
		case ch, ok := <-renderer.stop:
			{
				log.Println("Stopping")
				if !ok {
					return
				}
				ch <- true
				return
			}
		case id, ok := <-renderer.incomingWork:
			{
				if !ok {
					return
				}
				log.Println("Received dispatch message", id)
				renderer.renderGeneratedAsset(id)
			}
		}
	}
}

func (renderer *imageMagickRenderer) Stop() {
	callback := make(chan bool)
	renderer.stop <- callback
	select {
	case <-callback:
	case <-time.After(5 * time.Second):
	}
	close(renderer.stop)
}

func (renderer *imageMagickRenderer) AddStatusListener(listener RenderStatusChannel) {
	renderer.statusListeners = append(renderer.statusListeners, listener)
}

func (renderer *imageMagickRenderer) Dispatch() ImageMagickRendererWorkChannel {
	return renderer.incomingWork
}

func (renderer *imageMagickRenderer) renderGeneratedAsset(id string) {

	generatedAsset, err := renderer.gasm.FindById(id)
	if err != nil {
		log.Fatal("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}

	statusCallback := renderer.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = GeneratedAssetStatusProcessing
	renderer.gasm.Update(generatedAsset)

	sourceAssets, err := renderer.sasm.FindBySourceAssetId(generatedAsset.SourceAssetId)
	if err != nil {
		statusCallback <- generatedAssetUpdate{NewGeneratedAssetError(ErrorUnableToFindSourceAssetsById), nil}
		return
	}
	if len(sourceAssets) == 0 {
		statusCallback <- generatedAssetUpdate{NewGeneratedAssetError(ErrorNoSourceAssetsFoundForId), nil}
		return
	}
	sourceAsset := sourceAssets[0]

	templates, err := renderer.templateManager.FindByIds([]string{generatedAsset.TemplateId})
	if err != nil {
		statusCallback <- generatedAssetUpdate{NewGeneratedAssetError(ErrorUnableToFindTemplatesById), nil}
		return
	}
	if len(sourceAssets) == 0 {
		statusCallback <- generatedAssetUpdate{NewGeneratedAssetError(ErrorNoTemplatesFoundForId), nil}
		return
	}
	template := templates[0]

	urls := sourceAsset.GetAttribute(SourceAssetAttributeSource)
	sourceFile, err := renderer.tryDownload(urls)
	if err != nil {
		statusCallback <- generatedAssetUpdate{NewGeneratedAssetError(ErrorNoDownloadUrlsWork), nil}
		return
	}
	defer sourceFile.Release()

	destination := sourceFile.Path() + "-" + template.Id + ".jpg"
	destinationTemporaryFile := renderer.temporaryFileManager.Create(destination)
	defer destinationTemporaryFile.Release()

	size, err := renderer.getSize(template)
	if err != nil {
		statusCallback <- generatedAssetUpdate{NewGeneratedAssetError(ErrorCouldNotDetermineRenderSize), nil}
		return
	}

	err = renderer.resize(sourceFile.Path(), destination, size)
	if err != nil {
		statusCallback <- generatedAssetUpdate{NewGeneratedAssetError(ErrorCouldNotResizeImage), nil}
		return
	}

	err = renderer.upload(generatedAsset.Location, destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{NewGeneratedAssetError(ErrorCouldNotUploadAsset), nil}
		return
	}

	bounds, err := renderer.getBounds(destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{NewGeneratedAssetError(ErrorCouldNotDetermineRenderSize), nil}
		return
	}

	fi, err := os.Stat(destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{NewGeneratedAssetError(ErrorCouldNotDetermineFileSize), nil}
		return
	}

	newAttributes := []Attribute{
		generatedAsset.AddAttribute("imageHeight", []string{strconv.Itoa(bounds.Max.X)}),
		generatedAsset.AddAttribute("imageWidth", []string{strconv.Itoa(bounds.Max.Y)}),
		// NKG: I'm sure this is going to break something.
		generatedAsset.AddAttribute("fileSize", []string{strconv.FormatInt(fi.Size(), 10)}),
	}

	statusCallback <- generatedAssetUpdate{GeneratedAssetStatusComplete, newAttributes}
}

func (renderer *imageMagickRenderer) tryDownload(urls []string) (TemporaryFile, error) {
	for _, url := range urls {
		tempFile, err := renderer.downloader.Download(url)
		if err == nil {
			return tempFile, nil
		}
	}
	return nil, ErrorNoDownloadUrlsWork
}

func (renderer *imageMagickRenderer) getBounds(path string) (*image.Rectangle, error) {
	reader, err := os.Open(path)
	if err != nil {
		log.Println("os.Open error", err)
		return nil, err
	}
	defer reader.Close()
	image, err := jpeg.Decode(reader)
	if err != nil {
		log.Println("jpeg.Decode error", err)
		return nil, err
	}
	bounds := image.Bounds()
	return &bounds, nil
}

func (renderer *imageMagickRenderer) resize(source, destination string, size int) error {
	_, err := exec.LookPath("convert")
	if err != nil {
		log.Println("convert command not found")
		return err
	}

	cmd := exec.Command("convert", source, "-resize", strconv.Itoa(size), destination)
	log.Println(cmd)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Println(buf.String())

	return nil
}

func (renderer *imageMagickRenderer) getSize(template *Template) (int, error) {
	rawSize, err := GetFirstAttribute(template, TemplateAttributeHeight)
	if err == nil {
		sizeValue, err := strconv.Atoi(rawSize)
		if err == nil {
			return sizeValue, nil
		}
		return 0, err
	}
	return 0, err
}

func (renderer *imageMagickRenderer) upload(uploadDestination, renderedFilePath string) error {
	return renderer.uploader.Upload(uploadDestination, renderedFilePath)
}

type generatedAssetUpdate struct {
	status     string
	attributes []Attribute
}

func (renderer *imageMagickRenderer) commitStatus(id string, existingAttributes []Attribute) chan generatedAssetUpdate {
	commitChannel := make(chan generatedAssetUpdate, 10)

	go func() {
		status := NewGeneratedAssetError(ErrorUnknownError)
		attributes := make([]Attribute, 0, 0)
		for _, attribute := range existingAttributes {
			attributes = append(attributes, attribute)
		}
		for {
			select {
			case message, ok := <-commitChannel:
				{
					if !ok {
						for _, listener := range renderer.statusListeners {
							listener <- RenderStatus{id, status, RenderAgentImageMagick}
						}
						generatedAsset, err := renderer.gasm.FindById(id)
						if err != nil {
							log.Fatal("This is not good:", err)
							return
						}
						generatedAsset.Status = status
						generatedAsset.Attributes = attributes
						renderer.gasm.Update(generatedAsset)
						return
					}
					status = message.status
					if message.attributes != nil {
						for _, attribute := range message.attributes {
							attributes = append(attributes, attribute)
						}
					}
				}
			}
		}
	}()
	return commitChannel
}
