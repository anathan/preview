package render

type Renderer interface {
	Stop()
	AddStatusListener(listener RenderStatusChannel)
	Dispatch() RenderAgentWorkChannel
}

type RenderAgentWorkChannel chan string

type RenderStatusChannel chan RenderStatus

type RenderStatus struct {
	GeneratedAssetId string
	Status           string
	Service          string
}
