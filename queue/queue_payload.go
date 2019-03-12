package queue

type QueuePayload struct {
	Alert        Alert        `json:"alert"`
	Action       string       `json:"action"`
	MappedAction MappedAction `json:"mappedActionV2"`
}

type Alert struct {
	AlertId string `json:"alertId"`
}

type MappedAction struct {
	Name       string `json:"name"`
	ExtraField string `json:"extraField"`
}

/* Unmarshaling of full payload is not necessary

type QueuePayload struct {
	Source           Source           `json:"source,omitempty"`
	Alert            Alert            `json:"alert,omitempty"`
	Action           string           `json:"action,omitempty"`
	MappedAction     MappedAction     `json:"mappedAction,omitempty"`
	IntegrationId    string           `json:"integrationName,omitempty"`
	IntegrationName  string           `json:"integrationId,omitempty"`
	EscalationId     string           `json:"escalationId,omitempty"`
	EscalationName   string           `json:"escalationName,omitempty"`
	EscalationNotify EscalationNotify `json:"escalationNotify,omitempty"`
	EscalationTime   int64            `json:"escalationTime,omitempty"`
	RepeatCount      int              `json:"repeatCount,omitempty"`
}

type Source struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

type Alert struct {
	AlertId       string   `json:"alertId,omitempty"`
	Message       string   `json:"message,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	TinyId        string   `json:"tinyId,omitempty"`
	Entity        string   `json:"entity,omitempty"`
	Alias         string   `json:"alias,omitempty"`
	CreatedAt     int64    `json:"createdAt,omitempty"`
	UpdatedAt     int64    `json:"updatedAt,omitempty"`
	Username      string   `json:"username,omitempty"`
	UserId        string   `json:"userId,omitempty"`
	Recipient     string   `json:"recipient,omitempty"`
	Team          string   `json:"team,omitempty"`
	Owner         string   `json:"owner,omitempty"`
	Recipients    []string `json:"recipients,omitempty"`
	Teams         []string `json:"teams,omitempty"`
	Actions       []string `json:"actions,omitempty"`
	SnoozeEndDate string   `json:"snoozeEndDate,omitempty"`
	SnoozedUntil  int64    `json:"snoozedUntil,omitempty"`
	AddedTags     string   `json:"addedTags,omitempty"`
	RemovedTags   string   `json:"removedTags,omitempty"`
	Priority      string   `json:"priority,omitempty"`
	OldPriority   string   `json:"oldPriority,omitempty"`
	Source        string   `json:"source,omitempty"`
}

type MappedAction struct {
	Name 		string `json:"name,omitempty"`
	ExtraField	string `json:"extraField,omitempty"`
}

type EscalationNotify struct {
	Entity string `json:"entity,omitempty"`
	Id     string `json:"id,omitempty"`
	Type   string `json:"type,omitempty"`
	Name   string `json:"name,omitempty"`
}
*/
