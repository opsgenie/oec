package queue

type QueuePayload struct {
	Entity       Entity       `json:"entity"`
	Action       string       `json:"action"`
	MappedAction MappedAction `json:"mappedActionV2"`
}

type Entity struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type MappedAction struct {
	Name       string `json:"name"`
	ExtraField string `json:"extraField"`
}
