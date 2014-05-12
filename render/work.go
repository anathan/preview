package render

import (
	"github.com/ngerakines/preview/common"
	"log"
	"strings"
	"sync"
	"time"
)

type RenderAgentManager struct {
	sourceAssetStorageManager    common.SourceAssetStorageManager
	generatedAssetStorageManager common.GeneratedAssetStorageManager
	templateManager              common.TemplateManager
	temporaryFileManager         common.TemporaryFileManager
	workStatus                   RenderStatusChannel
	workChannels                 map[string]RenderAgentWorkChannel
	renderAgents                 map[string][]RenderAgent
	activeWork                   map[string][]string
	maxWork                      map[string]int

	stop chan (chan bool)
	mu   sync.Mutex
}

func NewRenderAgentManager(
	sourceAssetStorageManager common.SourceAssetStorageManager,
	generatedAssetStorageManager common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
	temporaryFileManager common.TemporaryFileManager) *RenderAgentManager {

	agentManager := new(RenderAgentManager)
	agentManager.sourceAssetStorageManager = sourceAssetStorageManager
	agentManager.generatedAssetStorageManager = generatedAssetStorageManager
	agentManager.templateManager = templateManager

	agentManager.temporaryFileManager = temporaryFileManager
	agentManager.workStatus = make(RenderStatusChannel, 100)
	agentManager.workChannels = make(map[string]RenderAgentWorkChannel)
	for _, renderAgent := range common.RenderAgents {
		agentManager.workChannels[renderAgent] = make(RenderAgentWorkChannel, 200)
	}
	agentManager.renderAgents = make(map[string][]RenderAgent)
	agentManager.activeWork = make(map[string][]string)
	agentManager.maxWork = make(map[string]int)

	agentManager.stop = make(chan (chan bool))
	go agentManager.run()

	return agentManager
}

func (agentManager *RenderAgentManager) AddListener(listener RenderStatusChannel) {
	for _, renderAgents := range agentManager.renderAgents {
		for _, renderAgent := range renderAgents {
			renderAgent.AddStatusListener(listener)
		}
	}
}

func (agentManager *RenderAgentManager) Stop() {
	for _, renderAgents := range agentManager.renderAgents {
		for _, renderAgent := range renderAgents {
			renderAgent.Stop()
		}
	}
	for _, workChannel := range agentManager.workChannels {
		close(workChannel)
	}

	callback := make(chan bool)
	agentManager.stop <- callback
	select {
	case <-callback:
	case <-time.After(5 * time.Second):
	}
	close(agentManager.stop)
}

func (agentManager *RenderAgentManager) AddImageMagickRenderAgent(downloader common.Downloader, uploader common.Uploader, maxWorkIncrease int) RenderAgent {
	renderAgent := newImageMagickRenderAgent(agentManager.sourceAssetStorageManager, agentManager.generatedAssetStorageManager, agentManager.templateManager, agentManager.temporaryFileManager, downloader, uploader, agentManager.workChannels[common.RenderAgentImageMagick])
	renderAgent.AddStatusListener(agentManager.workStatus)
	agentManager.AddRenderAgent(common.RenderAgentImageMagick, renderAgent, maxWorkIncrease)
	return renderAgent
}

func (agentManager *RenderAgentManager) AddDocumentRenderAgent(downloader common.Downloader, uploader common.Uploader, docCachePath string, maxWorkIncrease int) RenderAgent {
	renderAgent := newDocumentRenderAgent(agentManager.sourceAssetStorageManager, agentManager.generatedAssetStorageManager, agentManager.templateManager, agentManager.temporaryFileManager, downloader, uploader, docCachePath, agentManager.workChannels[common.RenderAgentDocument])
	renderAgent.AddStatusListener(agentManager.workStatus)
	agentManager.AddRenderAgent(common.RenderAgentDocument, renderAgent, maxWorkIncrease)
	return renderAgent
}

func (agentManager *RenderAgentManager) AddRenderAgent(name string, renderAgent RenderAgent, maxWorkIncrease int) {
	agentManager.mu.Lock()
	defer agentManager.mu.Unlock()

	renderAgents, hasRenderAgents := agentManager.renderAgents[name]
	if !hasRenderAgents {
		renderAgents = make([]RenderAgent, 0, 0)
		renderAgents = append(renderAgents, renderAgent)
		agentManager.renderAgents[name] = renderAgents
		agentManager.maxWork[name] = maxWorkIncrease
		agentManager.activeWork[name] = make([]string, 0, 0)
		return
	}

	renderAgents = append(renderAgents, renderAgent)
	agentManager.renderAgents[name] = renderAgents

	maxWork := agentManager.maxWork[name]
	agentManager.maxWork[name] = maxWork + maxWorkIncrease
}

func (agentManager *RenderAgentManager) run() {
	for {
		select {
		case ch, ok := <-agentManager.stop:
			{
				log.Println("Stopping")
				if !ok {
					return
				}
				ch <- true
				return
			}
		case statusUpdate, ok := <-agentManager.workStatus:
			{
				if !ok {
					return
				}
				log.Println("received status update", statusUpdate)
				agentManager.handleStatus(statusUpdate)
			}
		case <-time.After(5 * time.Second):
			{
				agentManager.dispatchMoreWork()
			}
		}
	}
}

func (agentManager *RenderAgentManager) dispatchMoreWork() {
	agentManager.mu.Lock()
	defer agentManager.mu.Unlock()

	for name, renderAgents := range agentManager.renderAgents {
		workCount := agentManager.workToDispatchCount(name)
		rendererCount := len(renderAgents)
		log.Println("Looking for work for", name, "and found", workCount, "slots for", rendererCount, "render agents.")
		if workCount > 0 && rendererCount > 0 {
			renderAgent := renderAgents[0]
			generatedAssets, err := agentManager.generatedAssetStorageManager.FindWorkForService(name, workCount)
			if err != nil {
				log.Println("generatedAssetStorageManager.FindWorkForService error", err)
			} else {
				for _, generatedAsset := range generatedAssets {
					generatedAsset.Status = common.GeneratedAssetStatusScheduled
					err := agentManager.generatedAssetStorageManager.Update(generatedAsset)
					if err == nil {
						log.Println("Dispatching", generatedAsset.Id, "to work for", name)
						agentManager.activeWork[name] = uniqueListWith(agentManager.activeWork[name], generatedAsset.Id)
						renderAgent.Dispatch() <- generatedAsset.Id
					}
				}
			}
		} else {
			log.Println("work-count", workCount, "renderer-count", rendererCount)
		}
	}
}

func (agentManager *RenderAgentManager) handleStatus(renderStatus RenderStatus) {
	agentManager.mu.Lock()
	defer agentManager.mu.Unlock()

	if renderStatus.Status == common.GeneratedAssetStatusComplete || strings.HasPrefix(renderStatus.Status, common.GeneratedAssetStatusFailed) {
		activeWork, hasActiveWork := agentManager.activeWork[renderStatus.Service]
		if hasActiveWork {
			agentManager.activeWork[renderStatus.Service] = listWithout(activeWork, renderStatus.GeneratedAssetId)
		}
	}
	log.Println("active work for", renderStatus.Service, "is:", agentManager.activeWork[renderStatus.Service])
}

func (agentManager *RenderAgentManager) workToDispatchCount(name string) int {
	activework, hasActiveWork := agentManager.activeWork[name]
	maxWork, hasMaxWork := agentManager.maxWork[name]
	if hasActiveWork && hasMaxWork {
		activeWorkCount := len(activework)
		if activeWorkCount < maxWork {
			return maxWork - activeWorkCount
		}
	}
	return 0
}

func listWithout(values []string, value string) []string {
	results := make([]string, 0, 0)
	for _, listValue := range values {
		if listValue != value {
			results = append(results, listValue)
		}
	}
	return results
}

func uniqueListWith(values []string, value string) []string {
	if values == nil {
		results := make([]string, 0, 1)
		results[0] = value
		return results
	}
	for _, ele := range values {
		if ele == value {
			return values
		}
	}
	return append(values, value)
}
