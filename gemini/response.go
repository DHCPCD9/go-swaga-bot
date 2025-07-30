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
