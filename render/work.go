package render

import (
	"github.com/ngerakines/preview/common"
	"log"
	"strings"
	"sync"
	"time"
)

type RendererManager struct {
	generatedAssetStorageManager common.GeneratedAssetStorageManager
	temporaryFileManager         common.TemporaryFileManager
	workStatus                   RenderStatusChannel
	workChannels                 map[string]RenderAgentWorkChannel
	renderAgents                 map[string][]RenderAgent
	activeWork                   map[string][]string
	maxWork                      map[string]int

	stop chan (chan bool)
	mu   sync.Mutex
}

func NewRendererManager(
	generatedAssetStorageManager common.GeneratedAssetStorageManager,
	temporaryFileManager common.TemporaryFileManager) *RendererManager {
	rm := new(RendererManager)
	rm.generatedAssetStorageManager = generatedAssetStorageManager
	rm.temporaryFileManager = temporaryFileManager
	rm.workStatus = make(RenderStatusChannel, 100)
	rm.workChannels = make(map[string]RenderAgentWorkChannel)
	for _, renderAgent := range common.RenderAgents {
		rm.workChannels[renderAgent] = make(RenderAgentWorkChannel, 200)
	}
	rm.renderAgents = make(map[string][]RenderAgent)
	rm.activeWork = make(map[string][]string)
	rm.maxWork = make(map[string]int)

	rm.stop = make(chan (chan bool))
	go rm.run()

	return rm
}

func (rm *RendererManager) AddListener(listener RenderStatusChannel) {
	for _, renderAgents := range rm.renderAgents {
		for _, renderAgent := range renderAgents {
			renderAgent.AddStatusListener(listener)
		}
	}
}

func (rm *RendererManager) Stop() {
	for _, renderAgents := range rm.renderAgents {
		for _, renderAgent := range renderAgents {
			renderAgent.Stop()
		}
	}
	for _, workChannel := range rm.workChannels {
		close(workChannel)
	}

	callback := make(chan bool)
	rm.stop <- callback
	select {
	case <-callback:
	case <-time.After(5 * time.Second):
	}
	close(rm.stop)
}

func (rm *RendererManager) AddImageMagickRenderAgent(sasm common.SourceAssetStorageManager, tm common.TemplateManager, downloader common.Downloader, uploader common.Uploader, maxWorkIncrease int) RenderAgent {
	renderAgent := newImageMagickRenderAgent(sasm, rm.generatedAssetStorageManager, tm, rm.temporaryFileManager, downloader, uploader, rm.workChannels[common.RenderAgentImageMagick])
	renderAgent.AddStatusListener(rm.workStatus)
	rm.AddRenderAgent(common.RenderAgentImageMagick, renderAgent, maxWorkIncrease)
	return renderAgent
}

func (rm *RendererManager) AddDocumentRenderAgent(sasm common.SourceAssetStorageManager, tm common.TemplateManager, downloader common.Downloader, uploader common.Uploader, maxWorkIncrease int) RenderAgent {
	renderAgent := newDocumentRenderAgent(sasm, rm.generatedAssetStorageManager, tm, rm.temporaryFileManager, downloader, uploader, "todo-base-path", rm.workChannels[common.RenderAgentDocument])
	renderAgent.AddStatusListener(rm.workStatus)
	rm.AddRenderAgent(common.RenderAgentDocument, renderAgent, maxWorkIncrease)
	return renderAgent
}

func (rm *RendererManager) AddRenderAgent(name string, renderAgent RenderAgent, maxWorkIncrease int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	renderAgents, hasRenderAgents := rm.renderAgents[name]
	if !hasRenderAgents {
		renderAgents = make([]RenderAgent, 0, 0)
		renderAgents = append(renderAgents, renderAgent)
		rm.renderAgents[name] = renderAgents
		rm.maxWork[name] = maxWorkIncrease
		rm.activeWork[name] = make([]string, 0, 0)
		return
	}

	renderAgents = append(renderAgents, renderAgent)
	rm.renderAgents[name] = renderAgents

	maxWork := rm.maxWork[name]
	rm.maxWork[name] = maxWork + maxWorkIncrease
}

func (rm *RendererManager) run() {
	for {
		select {
		case ch, ok := <-rm.stop:
			{
				log.Println("Stopping")
				if !ok {
					return
				}
				ch <- true
				return
			}
		case statusUpdate, ok := <-rm.workStatus:
			{
				if !ok {
					return
				}
				log.Println("received status update", statusUpdate)
				rm.handleStatus(statusUpdate)
			}
		case <-time.After(5 * time.Second):
			{
				rm.dispatchMoreWork()
			}
		}
	}
}

func (rm *RendererManager) dispatchMoreWork() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for name, renderAgents := range rm.renderAgents {
		workCount := rm.workToDispatchCount(name)
		rendererCount := len(renderAgents)
		log.Println("Looking for work for", name, "and found", workCount, "slots for", rendererCount, "render agents.")
		if workCount > 0 && rendererCount > 0 {
			renderAgent := renderAgents[0]
			generatedAssets, err := rm.generatedAssetStorageManager.FindWorkForService(name, workCount)
			if err != nil {
				log.Println("generatedAssetStorageManager.FindWorkForService error", err)
			} else {
				for _, generatedAsset := range generatedAssets {
					generatedAsset.Status = common.GeneratedAssetStatusScheduled
					err := rm.generatedAssetStorageManager.Update(generatedAsset)
					if err == nil {
						log.Println("Dispatching", generatedAsset.Id)
						rm.activeWork[name] = uniqueListWith(rm.activeWork[name], generatedAsset.Id)
						renderAgent.Dispatch() <- generatedAsset.Id
					}
				}
			}
		} else {
			log.Println("work-count", workCount, "renderer-count", rendererCount)
		}
	}
}

func (rm *RendererManager) handleStatus(renderStatus RenderStatus) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if renderStatus.Status == common.GeneratedAssetStatusComplete || strings.HasPrefix(renderStatus.Status, common.GeneratedAssetStatusFailed) {
		activeWork, hasActiveWork := rm.activeWork[renderStatus.Service]
		if hasActiveWork {
			rm.activeWork[renderStatus.Service] = listWithout(activeWork, renderStatus.GeneratedAssetId)
		}
	}
	log.Println("active work for", renderStatus.Service, "is:", rm.activeWork[renderStatus.Service])
}

func (rm *RendererManager) workToDispatchCount(name string) int {
	activework, hasActiveWork := rm.activeWork[name]
	maxWork, hasMaxWork := rm.maxWork[name]
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
