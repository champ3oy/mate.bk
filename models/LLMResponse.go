package models

type LLMResponse struct {
	Choices []struct {
		FinishReason string      `json:"finish_reason"`
		Index        int         `json:"index"`
		Logprobs     interface{} `json:"logprobs"`
		Message      struct {
			Content string `json:"content"`
			Role    string `json:"role"`
		} `json:"message"`
	} `json:"choices"`
	Created           int    `json:"created"`
	ID                string `json:"id"`
	Model             string `json:"model"`
	Object            string `json:"object"`
	SystemFingerprint string `json:"system_fingerprint"`
	Usage             struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
