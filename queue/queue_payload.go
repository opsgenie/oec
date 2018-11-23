package queue

type QueuePayload struct {
	Source          Source `json:"source,omitempty"`
	Alert           Alert  `json:"alert,omitempty"`
	Action          string `json:"action,omitempty"`
	IntegrationId   string `json:"integrationName,omitempty"`
	IntegrationName string `json:"integrationId,omitempty"`
}

type Source struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

type Alert struct {
	UpdatedAt  int64    `json:"updatedAt,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Teams      []string `json:"teams,omitempty"`
	Recipients []string `json:"recipients,omitempty"`
	Message    string   `json:"message,omitempty"`
	Username   string   `json:"username,omitempty"`
	AlertId    string   `json:"alertId,omitempty"`
	Source     string   `json:"source,omitempty"`
	Alias      string   `json:"alias,omitempty"`
	TinyId     string   `json:"tinyId,omitempty"`
	CreatedAt  int64    `json:"createdAt,omitempty"`
	UserId     string   `json:"userId,omitempty"`
	Entity     string   `json:"entity,omitempty"`
}
