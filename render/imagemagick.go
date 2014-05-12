package render

import (
	"bytes"
	"fmt"
	"github.com/ngerakines/preview/common"
	"image"
	"image/jpeg"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type imageMagickRenderAgent struct {
	sasm                 common.SourceAssetStorageManager
	gasm                 common.GeneratedAssetStorageManager
	templateManager      common.TemplateManager
	downloader           common.Downloader
	uploader             common.Uploader
	workChannel          RenderAgentWorkChannel
	statusListeners      []RenderStatusChannel
	temporaryFileManager common.TemporaryFileManager
	stop                 chan (chan bool)
}

func newImageMagickRenderAgent(
	sasm common.SourceAssetStorageManager,
	gasm common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
	temporaryFileManager common.TemporaryFileManager,
	downloader common.Downloader,
	uploader common.Uploader,
	workChannel RenderAgentWorkChannel) RenderAgent {

	renderAgent := new(imageMagickRenderAgent)
	renderAgent.sasm = sasm
	renderAgent.gasm = gasm
	renderAgent.templateManager = templateManager
	renderAgent.temporaryFileManager = temporaryFileManager
	renderAgent.downloader = downloader
	renderAgent.uploader = uploader
	renderAgent.workChannel = workChannel
	renderAgent.statusListeners = make([]RenderStatusChannel, 0, 0)
	renderAgent.stop = make(chan (chan bool))

	go renderAgent.start()

	return renderAgent
}

func (renderAgent *imageMagickRenderAgent) start() {
	for {
		select {
		case ch, ok := <-renderAgent.stop:
			{
				log.Println("Stopping")
				if !ok {
					return
				}
				ch <- true
				return
			}
		case id, ok := <-renderAgent.workChannel:
			{
				if !ok {
					return
				}
				log.Println("Received dispatch message", id)
				renderAgent.renderGeneratedAsset(id)
			}
		}
	}
}

func (renderAgent *imageMagickRenderAgent) Stop() {
	callback := make(chan bool)
	renderAgent.stop <- callback
	select {
	case <-callback:
	case <-time.After(5 * time.Second):
	}
	close(renderAgent.stop)
}

func (renderAgent *imageMagickRenderAgent) AddStatusListener(listener RenderStatusChannel) {
	renderAgent.statusListeners = append(renderAgent.statusListeners, listener)
}

func (renderAgent *imageMagickRenderAgent) Dispatch() RenderAgentWorkChannel {
	return renderAgent.workChannel
}

func (renderAgent *imageMagickRenderAgent) renderGeneratedAsset(id string) {

	generatedAsset, err := renderAgent.gasm.FindById(id)
	if err != nil {
		log.Fatal("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}

	statusCallback := renderAgent.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = common.GeneratedAssetStatusProcessing
	renderAgent.gasm.Update(generatedAsset)

	sourceAsset, err := renderAgent.getSourceAsset(generatedAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}

	fileType, err := renderAgent.getSourceAssetFileType(sourceAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineFileType), nil}
		return
	}

	templates, err := renderAgent.templateManager.FindByIds([]string{generatedAsset.TemplateId})
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindTemplatesById), nil}
		return
	}
	if len(templates) == 0 {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoTemplatesFoundForId), nil}
		return
	}
	template := templates[0]

	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	sourceFile, err := renderAgent.tryDownload(urls)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoDownloadUrlsWork), nil}
		return
	}
	defer sourceFile.Release()

	destination := sourceFile.Path() + "-" + template.Id + ".jpg"
	destinationTemporaryFile := renderAgent.temporaryFileManager.Create(destination)
	defer destinationTemporaryFile.Release()

	size, err := renderAgent.getSize(template)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineRenderSize), nil}
		return
	}

	if fileType == "pdf" {
		page, _ := renderAgent.getGeneratedAssetPage(generatedAsset)
		err = renderAgent.imageFromPdf(sourceFile.Path(), destination, size, page)
	} else if fileType == "gif" {
		err = renderAgent.firstGifFrame(sourceFile.Path(), destination, size)
	} else {
		err = renderAgent.resize(sourceFile.Path(), destination, size)
	}
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotResizeImage), nil}
		return
	}

	err = renderAgent.uploader.Upload(generatedAsset.Location, destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotUploadAsset), nil}
		return
	}

	bounds, err := renderAgent.getBounds(destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineRenderSize), nil}
		return
	}

	fi, err := os.Stat(destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineFileSize), nil}
		return
	}

	newAttributes := []common.Attribute{
		generatedAsset.AddAttribute("imageHeight", []string{strconv.Itoa(bounds.Max.X)}),
		generatedAsset.AddAttribute("imageWidth", []string{strconv.Itoa(bounds.Max.Y)}),
		// NKG: I'm sure this is going to break something.
		generatedAsset.AddAttribute("fileSize", []string{strconv.FormatInt(fi.Size(), 10)}),
	}

	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusComplete, newAttributes}
}

func (renderAgent *imageMagickRenderAgent) getSourceAsset(generatedAsset *common.GeneratedAsset) (*common.SourceAsset, error) {
	sourceAssets, err := renderAgent.sasm.FindBySourceAssetId(generatedAsset.SourceAssetId)
	if err != nil {
		return nil, err
	}
	for _, sourceAsset := range sourceAssets {
		if sourceAsset.IdType == generatedAsset.SourceAssetType {
			return sourceAsset, nil
		}
	}
	return nil, common.ErrorNoSourceAssetsFoundForId
}

func (renderAgent *imageMagickRenderAgent) tryDownload(urls []string) (common.TemporaryFile, error) {
	for _, url := range urls {
		tempFile, err := renderAgent.downloader.Download(url)
		if err == nil {
			return tempFile, nil
		}
	}
	return nil, common.ErrorNoDownloadUrlsWork
}

func (renderAgent *imageMagickRenderAgent) getBounds(path string) (*image.Rectangle, error) {
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

func (renderAgent *imageMagickRenderAgent) resize(source, destination string, size int) error {
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

func (renderAgent *imageMagickRenderAgent) imageFromPdf(source, destination string, size, page int) error {
	_, err := exec.LookPath("convert")
	if err != nil {
		log.Println("convert command not found")
		return err
	}

	cmd := exec.Command("convert", "-colorspace", "RGB", fmt.Sprintf("%s[%d]", source, page), "-resize", strconv.Itoa(size), "+adjoin", destination)
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

func (renderAgent *imageMagickRenderAgent) firstGifFrame(source, destination string, size int) error {
	_, err := exec.LookPath("convert")
	if err != nil {
		log.Println("convert command not found")
		return err
	}

	cmd := exec.Command("convert", fmt.Sprintf("%s[0]", source), "-resize", strconv.Itoa(size), destination)
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

func (renderAgent *imageMagickRenderAgent) getSize(template *common.Template) (int, error) {
	rawSize, err := common.GetFirstAttribute(template, common.TemplateAttributeHeight)
	if err == nil {
		sizeValue, err := strconv.Atoi(rawSize)
		if err == nil {
			return sizeValue, nil
		}
		return 0, err
	}
	return 0, err
}

func (renderAgent *imageMagickRenderAgent) getGeneratedAssetPage(generatedAsset *common.GeneratedAsset) (int, error) {
	rawPage, err := common.GetFirstAttribute(generatedAsset, common.GeneratedAssetAttributePage)
	if err == nil {
		pageValue, err := strconv.Atoi(rawPage)
		if err == nil {
			return pageValue, nil
		}
		return 0, err
	}
	return 0, err
}

func (renderAgent *imageMagickRenderAgent) getSourceAssetFileType(sourceAsset *common.SourceAsset) (string, error) {
	fileType, err := common.GetFirstAttribute(sourceAsset, common.SourceAssetAttributeType)
	if err == nil {
		return fileType, nil
	}
	return "unknown", err
}

type generatedAssetUpdate struct {
	status     string
	attributes []common.Attribute
}

func (renderAgent *imageMagickRenderAgent) commitStatus(id string, existingAttributes []common.Attribute) chan generatedAssetUpdate {
	commitChannel := make(chan generatedAssetUpdate, 10)

	go func() {
		status := common.NewGeneratedAssetError(common.ErrorUnknownError)
		attributes := make([]common.Attribute, 0, 0)
		for _, attribute := range existingAttributes {
			attributes = append(attributes, attribute)
		}
		for {
			select {
			case message, ok := <-commitChannel:
				{
					if !ok {
						for _, listener := range renderAgent.statusListeners {
							listener <- RenderStatus{id, status, common.RenderAgentImageMagick}
						}
						generatedAsset, err := renderAgent.gasm.FindById(id)
						if err != nil {
							log.Fatal(err.Error())
							return
						}
						generatedAsset.Status = status
						generatedAsset.Attributes = attributes
						renderAgent.gasm.Update(generatedAsset)
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
