package render

// ./soffice --headless --nologo --nofirststartwizard --convert-to pdf ~/Downloads/ChefConf2014schedule.docx --outdir ~/Desktop/

import (
	"bytes"
	"github.com/ngerakines/preview/common"
	"log"
	"os/exec"
	"time"
)

type libreOfficeConverter struct {
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

func NewLibreOfficeConverter(
	sasm common.SourceAssetStorageManager,
	gasm common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
	temporaryFileManager common.TemporaryFileManager,
	downloader common.Downloader,
	uploader common.Uploader,
	workChannel RenderAgentWorkChannel) Renderer {

	renderer := new(libreOfficeConverter)
	renderer.sasm = sasm
	renderer.gasm = gasm
	renderer.templateManager = templateManager
	renderer.temporaryFileManager = temporaryFileManager
	renderer.downloader = downloader
	renderer.uploader = uploader
	renderer.workChannel = workChannel
	renderer.statusListeners = make([]RenderStatusChannel, 0, 0)
	renderer.stop = make(chan (chan bool))

	go renderer.start()

	return renderer
}

func (renderer *libreOfficeConverter) start() {
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
		case id, ok := <-renderer.workChannel:
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

func (renderer *libreOfficeConverter) Stop() {
	callback := make(chan bool)
	renderer.stop <- callback
	select {
	case <-callback:
	case <-time.After(5 * time.Second):
	}
	close(renderer.stop)
}

func (renderer *libreOfficeConverter) AddStatusListener(listener RenderStatusChannel) {
	renderer.statusListeners = append(renderer.statusListeners, listener)
}

func (renderer *libreOfficeConverter) Dispatch() RenderAgentWorkChannel {
	return renderer.workChannel
}

func (renderer *libreOfficeConverter) renderGeneratedAsset(id string) {

	generatedAsset, err := renderer.gasm.FindById(id)
	if err != nil {
		log.Fatal("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}

	statusCallback := renderer.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = common.GeneratedAssetStatusProcessing
	renderer.gasm.Update(generatedAsset)

	sourceAssets, err := renderer.sasm.FindBySourceAssetId(generatedAsset.SourceAssetId)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}
	if len(sourceAssets) == 0 {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoSourceAssetsFoundForId), nil}
		return
	}
	sourceAsset := sourceAssets[0]

	templates, err := renderer.templateManager.FindByIds([]string{generatedAsset.TemplateId})
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindTemplatesById), nil}
		return
	}
	if len(sourceAssets) == 0 {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoTemplatesFoundForId), nil}
		return
	}
	template := templates[0]

	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	sourceFile, err := renderer.tryDownload(urls)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoDownloadUrlsWork), nil}
		return
	}
	defer sourceFile.Release()

	destination := sourceFile.Path() + "-" + template.Id + ".pdf"
	destinationTemporaryFile := renderer.temporaryFileManager.Create(destination)
	defer destinationTemporaryFile.Release()

	size, err := renderer.getSize(template)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineRenderSize), nil}
		return
	}

	err = renderer.createPdf(sourceFile.Path(), destination, size)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotResizeImage), nil}
		return
	}

	err = renderer.upload(generatedAsset.Location, destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotUploadAsset), nil}
		return
	}

	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusComplete, nil}
}

func ok() {
}

func (renderer *libreOfficeConverter) createPdf(source, destination string, size int) error {
	_, err := exec.LookPath("soffice")
	if err != nil {
		log.Println("convert command not found")
		return err
	}

	cmd := exec.Command("/Applications/LibreOffice.app/Contents/program/soffice", "--headless", "--nologo", "--nofirststartwizard", "--convert-to", "pdf", source, "--outdir", "~/Desktop/")
	cmd.Dir = "/Applications/LibreOffice.app/Contents/program"
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

func (renderer *libreOfficeConverter) tryDownload(urls []string) (common.TemporaryFile, error) {
	return nil, common.ErrorNotImplemented
}

func (renderer *libreOfficeConverter) getSize(template *common.Template) (int, error) {
	return 0, common.ErrorNotImplemented
}

func (renderer *libreOfficeConverter) upload(uploadDestination, renderedFilePath string) error {
	return common.ErrorNotImplemented
}

func (renderer *libreOfficeConverter) commitStatus(id string, existingAttributes []common.Attribute) chan generatedAssetUpdate {
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
						for _, listener := range renderer.statusListeners {
							listener <- RenderStatus{id, status, common.RenderAgentImageMagick}
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
