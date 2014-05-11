package render

// ./soffice --headless --nologo --nofirststartwizard --convert-to pdf ~/Downloads/ChefConf2014schedule.docx --outdir ~/Desktop/

import (
	"bytes"
	"fmt"
	"github.com/ngerakines/preview/common"
	"log"
	"os/exec"
	"strconv"
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
	basePath string,
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

/*
1. Get the generated asset
2. Get the source asset
3. Get the template
4. Fetch the source asset file
5. Create a temporary destination directory.
6. Convert the source asset file into a pdf using the temporary destination directory.
7. Given a file in that directory exists, determine how many pages it contains.
8. Create a new source asset record for the pdf.
9. Upload the new source asset pdf file.
10. For each page in the pdf, create a generated asset record for each of the default templates.
11. Update the status of the generated asset as complete.
*/
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

	destination, err := renderer.createTemporaryDestinationDirectory()
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}
	destinationTemporaryFile := renderer.temporaryFileManager.Create(destination)
	defer destinationTemporaryFile.Release()

	err = renderer.createPdf(sourceFile.Path(), destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotResizeImage), nil}
		return
	}

	files, err := renderer.getRenderedFiles(destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}
	if len(files) != 1 {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	pages := 1

	storedFile := "protocol://path/to/new/file"
	err = renderer.upload(storedFile, files[0])
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotUploadAsset), nil}
		return
	}

	// TODO: write this code
	var pdfFileSize int64 = 1

	pdfSourceAsset := common.NewSourceAsset(sourceAsset.Id, common.SourceAssetTypePdf)
	pdfSourceAsset.AddAttribute(common.SourceAssetAttributeSize, []string{strconv.FormatInt(pdfFileSize, 10)})
	pdfSourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{storedFile})
	pdfSourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{"pdf"})
	// TODO: Add support for the expiration attribute.

	renderer.sasm.Store(pdfSourceAsset)
	legacyDefaultTemplates, err := renderer.templateManager.FindByIds(common.LegacyDefaultTemplates)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	// TODO: Have the new source asset and generated assets be created in batch in the storage managers.
	for page := 0; page < pages; page++ {
		for _, legacyTemplate := range legacyDefaultTemplates {
			// TODO: This can be put into a small lookup table create/set at the time of structure init.
			placeholderSize, err := common.GetFirstAttribute(legacyTemplate, common.TemplateAttributePlaceholderSize)
			if err != nil {
				statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
				return
			}
			// TODO: Update simple blueprint and image magick render agent to use this url structure.
			location := fmt.Sprintf("local:///%s/%s/%d", sourceAsset.Id, placeholderSize, page)
			pdfGeneratedAsset := common.NewGeneratedAssetFromSourceAsset(pdfSourceAsset, template, location)
			pdfGeneratedAsset.AddAttribute(common.GeneratedAssetAttributePage, []string{strconv.Itoa(page)})
			renderer.gasm.Store(pdfGeneratedAsset)
		}
	}

	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusComplete, nil}
}

func (renderer *libreOfficeConverter) createPdf(source, destination string) error {
	_, err := exec.LookPath("soffice")
	if err != nil {
		log.Println("convert command not found")
		return err
	}

	// TODO: Make this path configurable.
	cmd := exec.Command("/Applications/LibreOffice.app/Contents/program/soffice", "--headless", "--nologo", "--nofirststartwizard", "--convert-to", "pdf", source, "--outdir", destination)
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

func (renderer *libreOfficeConverter) createTemporaryDestinationDirectory() (string, error) {
	return "", common.ErrorNotImplemented
}

func (renderer *libreOfficeConverter) getRenderedFiles(path string) ([]string, error) {
	return []string{}, common.ErrorNotImplemented
}
