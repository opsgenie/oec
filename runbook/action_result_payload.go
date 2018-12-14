package runbook

type ActionResultPayload struct {
	IsSuccessful   bool   `json:"isSuccessful,omitempty"`
	AlertId        string `json:"alertId,omitempty"`
	Action         string `json:"action,omitempty"`
	FailureMessage string `json:"failureMessage,omitempty"`
}
