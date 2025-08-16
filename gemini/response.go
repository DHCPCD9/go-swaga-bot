package gemini

type GeminiResponse struct {
	Candidates    []Candidates  `json:"candidates"`
	UsageMetadata UsageMetadata `json:"usageMetadata"`
	ModelVersion  string        `json:"modelVersion"`
	ResponseID    string        `json:"responseId"`
}

type Content struct {
	Parts []Parts `json:"parts"`
	Role  string  `json:"role"`
}
type Candidates struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason"`
	Index        int     `json:"index"`
}
type PromptTokensDetails struct {
	Modality   string `json:"modality"`
	TokenCount int    `json:"tokenCount"`
}
type UsageMetadata struct {
	PromptTokenCount     int                   `json:"promptTokenCount"`
	CandidatesTokenCount int                   `json:"candidatesTokenCount"`
	TotalTokenCount      int                   `json:"totalTokenCount"`
	PromptTokensDetails  []PromptTokensDetails `json:"promptTokensDetails"`
}

type PromptJson struct {
	UserID     string   `json:"user_id"`
	Username   string   `json:"username"`
	KnownNames []string `json:"known_names"`
	Activities []struct {
		Activity string `json:"activity"`
		State    string `json:"state"`
		Substate string `json:"substate"`
	} `json:"activities"`
	Facts      []string `json:"facts"`
	Text       string   `json:"text"`
	Reference  string   `json:"reference"`
	References []struct {
		ID   string `json:"id"`
		Text string `json:"text"`
		User string `json:"user"`
	} `json:"references"`
	ReferenceUsers []struct {
		ID         string   `json:"id"`
		Username   string   `json:"username"`
		KnownNames []string `json:"known_names"`
		Facts      []string `json:"facts"`
	} `json:"reference_users"`
}

type ResponseJson struct {
	Response string `json:"response"`
	Facts    []struct {
		Fact string `json:"fact"`
		User string `json:"user"`
		Type string `json:"type"`
	} `json:"facts"`
	Usernames []struct {
		Username string `json:"username"`
		User     string `json:"user"`
		Type     string `json:"type"`
	} `json:"usernames"`
}
