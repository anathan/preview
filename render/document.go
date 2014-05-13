package render

import (
	"bytes"
	"fmt"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type documentRenderAgent struct {
	sasm                 common.SourceAssetStorageManager
	gasm                 common.GeneratedAssetStorageManager
	templateManager      common.TemplateManager
	downloader           common.Downloader
	uploader             common.Uploader
	workChannel          RenderAgentWorkChannel
	statusListeners      []RenderStatusChannel
	temporaryFileManager common.TemporaryFileManager
	tempFileBasePath     string
	stop                 chan (chan bool)
}

func newDocumentRenderAgent(
	sasm common.SourceAssetStorageManager,
	gasm common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
	temporaryFileManager common.TemporaryFileManager,
	downloader common.Downloader,
	uploader common.Uploader,
	tempFileBasePath string,
	workChannel RenderAgentWorkChannel) RenderAgent {

	renderAgent := new(documentRenderAgent)
	renderAgent.sasm = sasm
	renderAgent.gasm = gasm
	renderAgent.templateManager = templateManager
	renderAgent.temporaryFileManager = temporaryFileManager
	renderAgent.downloader = downloader
	renderAgent.uploader = uploader
	renderAgent.workChannel = workChannel
	renderAgent.tempFileBasePath = tempFileBasePath
	renderAgent.statusListeners = make([]RenderStatusChannel, 0, 0)
	renderAgent.stop = make(chan (chan bool))

	go renderAgent.start()

	return renderAgent
}

func (renderAgent *documentRenderAgent) start() {
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

func (renderAgent *documentRenderAgent) Stop() {
	callback := make(chan bool)
	renderAgent.stop <- callback
	select {
	case <-callback:
	case <-time.After(5 * time.Second):
	}
	close(renderAgent.stop)
}

func (renderAgent *documentRenderAgent) AddStatusListener(listener RenderStatusChannel) {
	renderAgent.statusListeners = append(renderAgent.statusListeners, listener)
}

func (renderAgent *documentRenderAgent) Dispatch() RenderAgentWorkChannel {
	return renderAgent.workChannel
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
func (renderAgent *documentRenderAgent) renderGeneratedAsset(id string) {

	// 1. Get the generated asset
	generatedAsset, err := renderAgent.gasm.FindById(id)
	if err != nil {
		log.Fatal("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}

	statusCallback := renderAgent.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = common.GeneratedAssetStatusProcessing
	renderAgent.gasm.Update(generatedAsset)

	// 2. Get the source asset
	sourceAsset, err := renderAgent.getSourceAsset(generatedAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}

	// 3. Get the template... not needed yet

	// 4. Fetch the source asset file
	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	sourceFile, err := renderAgent.tryDownload(urls)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoDownloadUrlsWork), nil}
		return
	}
	defer sourceFile.Release()

	// 	// 5. Create a temporary destination directory.
	destination, err := renderAgent.createTemporaryDestinationDirectory()
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}
	destinationTemporaryFile := renderAgent.temporaryFileManager.Create(destination)
	defer destinationTemporaryFile.Release()

	err = renderAgent.createPdf(sourceFile.Path(), destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotResizeImage), nil}
		return
	}

	files, err := renderAgent.getRenderedFiles(destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}
	if len(files) != 1 {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	pages, err := renderAgent.getPdfPageCount(files[0])
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	err = renderAgent.uploader.Upload(generatedAsset.Location, files[0])
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotUploadAsset), nil}
		return
	}

	fi, err := os.Stat(destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineFileSize), nil}
		return
	}
	pdfFileSize := fi.Size()

	pdfSourceAsset, err := common.NewSourceAsset(sourceAsset.Id, common.SourceAssetTypePdf)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	pdfSourceAsset.AddAttribute(common.SourceAssetAttributeSize, []string{strconv.FormatInt(pdfFileSize, 10)})
	pdfSourceAsset.AddAttribute(common.SourceAssetAttributePages, []string{strconv.Itoa(pages)})
	pdfSourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{generatedAsset.Location})
	pdfSourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{"pdf"})
	// TODO: Add support for the expiration attribute.

	log.Println("pdfSourceAsset", pdfSourceAsset)
	renderAgent.sasm.Store(pdfSourceAsset)
	legacyDefaultTemplates, err := renderAgent.templateManager.FindByIds(common.LegacyDefaultTemplates)
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
			pdfGeneratedAsset, err := common.NewGeneratedAssetFromSourceAsset(pdfSourceAsset, legacyTemplate, location)
			if err != nil {
				statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
				return
			}
			pdfGeneratedAsset.AddAttribute(common.GeneratedAssetAttributePage, []string{strconv.Itoa(page)})
			log.Println("pdfGeneratedAsset", pdfGeneratedAsset)
			renderAgent.gasm.Store(pdfGeneratedAsset)
		}
	}

	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusComplete, nil}
}

func (renderAgent *documentRenderAgent) getSourceAsset(generatedAsset *common.GeneratedAsset) (*common.SourceAsset, error) {
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

func (renderAgent *documentRenderAgent) createPdf(source, destination string) error {
	_, err := exec.LookPath("/Applications/LibreOffice.app/Contents/MacOS/soffice")
	if err != nil {
		log.Println("soffice command not found")
		return err
	}

	// TODO: Make this path configurable.
	cmd := exec.Command("/Applications/LibreOffice.app/Contents/MacOS/soffice", "--headless", "--nologo", "--nofirststartwizard", "--convert-to", "pdf", source, "--outdir", destination)
	log.Println(cmd)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	log.Println(buf.String())
	if err != nil {
		log.Println("error running command", err)
		return err
	}

	return nil
}

var pdfPageCount = regexp.MustCompile(`Pages:\s+(\d+)`)

// pdfinfo ~/Desktop/ChefConf2014schedule.pdf
func (renderAgent *documentRenderAgent) getPdfPageCount(file string) (int, error) {
	_, err := exec.LookPath("pdfinfo")
	if err != nil {
		log.Println("pdfinfo command not found")
		return 0, err
	}
	out, err := exec.Command("pdfinfo", file).Output()
	if err != nil {
		log.Fatal(err)
		return 0, err
	}
	matches := pdfPageCount.FindStringSubmatch(string(out))
	if len(matches) == 2 {
		return strconv.Atoi(matches[1])
	}
	return 0, nil
}

func (renderAgent *documentRenderAgent) tryDownload(urls []string) (common.TemporaryFile, error) {
	for _, url := range urls {
		tempFile, err := renderAgent.downloader.Download(url)
		if err == nil {
			return tempFile, nil
		}
	}
	return nil, common.ErrorNoDownloadUrlsWork
}

func (renderAgent *documentRenderAgent) commitStatus(id string, existingAttributes []common.Attribute) chan generatedAssetUpdate {
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
							listener <- RenderStatus{id, status, common.RenderAgentDocument}
						}
						generatedAsset, err := renderAgent.gasm.FindById(id)
						if err != nil {
							panic(err)
							return
						}
						generatedAsset.Status = status
						generatedAsset.Attributes = attributes
						log.Println("Updating", generatedAsset)
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

func (renderAgent *documentRenderAgent) createTemporaryDestinationDirectory() (string, error) {
	uuid, err := util.NewUuid()
	if err != nil {
		return "", err
	}
	tmpPath := filepath.Join(renderAgent.tempFileBasePath, uuid)
	err = os.MkdirAll(tmpPath, 0777)
	if err != nil {
		log.Println("error creating tmp dir", err)
		return "", err
	}
	return tmpPath, nil
}

func (renderAgent *documentRenderAgent) getRenderedFiles(path string) ([]string, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Println("Error reading files in placeholder base directory:", err)
		return nil, err
	}
	paths := make([]string, 0, 0)
	for _, file := range files {
		if !file.IsDir() {
			// NKG: The convert command will create files of the same name but with the ".pdf" extension.
			if strings.HasSuffix(file.Name(), ".pdf") {
				paths = append(paths, filepath.Join(path, file.Name()))
			}
		}
	}
	return paths, nil
}
