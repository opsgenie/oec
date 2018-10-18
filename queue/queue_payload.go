package queue

type QueuePayload struct {
	Params Params `json:"params,omitempty"`
}

type Params struct {
	Type            string         `json:"type,omitempty"`
	AlertId         string         `json:"alertId,omitempty"`
	CustomerId      string         `json:"customerId,omitempty"`
	Action          string         `json:"action,omitempty"`
	MappedActionV2  MappedActionV2 `json:"mappedActionV2,omitempty"`
	IntegrationId   string         `json:"integrationId,omitempty"`
	IntegrationName string         `json:"integrationName,omitempty"`
	IntegrationType string         `json:"integrationType,omitempty"`
	SendViaMarid    bool           `json:"sendViaMarid,omitempty"`
	QueueUrls       []string       `json:"queueUrls,omitempty"`
	QueueName       string         `json:"queueName,omitempty"`
}

type MappedActionV2 struct {
	Name       string `json:"name,omitempty"`
	ExtraField string `json:"type,omitempty"`
}
