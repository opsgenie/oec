package queue

type payload struct {
	Entity       entity       `json:"entity"`
	Action       string       `json:"action"`
	MappedAction mappedAction `json:"mappedActionV2"`
}

type entity struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type mappedAction struct {
	Name       string `json:"name"`
	ExtraField string `json:"extraField"`
}
