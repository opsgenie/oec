package queue

type payload struct {
	RequestId             string       `json:"requestId"`
	Entity                entity       `json:"entity"`
	Action                string       `json:"action"`
	MappedAction          mappedAction `json:"mappedActionV2"`
	ActionType            string       `json:"actionType"`
	DiscardScriptResponse bool         `json:"discardScriptResponse"`
}

type entity struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type mappedAction struct {
	Name       string `json:"name"`
	ExtraField string `json:"extraField"`
}

const (
	CustomActionType = "custom"
	HttpActionType   = "http"
)
