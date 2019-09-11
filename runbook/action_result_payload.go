package runbook

type ActionResultPayload struct {
	IsSuccessful   bool   `json:"isSuccessful,omitempty"`
	EntityId       string `json:"entityId,omitempty"`
	EntityType     string `json:"entityType,omitempty"`
	Action         string `json:"action,omitempty"`
	FailureMessage string `json:"failureMessage,omitempty"`
}
