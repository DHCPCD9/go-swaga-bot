package gemini

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"

	"github.com/DHCPCD9/go-swaga-bot/configuration"
	"github.com/DHCPCD9/go-swaga-bot/database"
	log "github.com/sirupsen/logrus"
)

type GeminiBody struct {
	Contents         []Contents       `json:"contents"`
	GenerationConfig GenerationConfig `json:"generationConfig"`
}
type Parts struct {
	Text       string `json:"text,omitempty"`
	InlineData *struct {
		MimeType string `json:"mime_type"`
		Data     string `json:"data"`
	} `json:"inline_data,omitempty"`
}
type Contents struct {
	Parts []Parts `json:"parts"`
}
type ThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}
type GenerationConfig struct {
	ThinkingConfig ThinkingConfig `json:"thinkingConfig"`
}

//go:embed base-prompt.txt
var PROMPT string

func BuildParts(userid string, serverid string) *Contents {

	var messages []database.IndexedMessages

	// Retrieve Last 1000 messages from the database
	if err := database.Pool.Order("created_at DESC").Where("user_id  = ? AND guild_id = ?", userid, serverid).Limit(100).Find(&messages).Error; err != nil {
		log.Errorf("Failed to retrieve messages from database: %v", err)
		return nil
	}

	var contents []Parts

	contents = []Parts{}
	contents = append(contents, Parts{Text: PROMPT})

	//Do it in format that is described above, and it can take up to 1M tokens, but better limit it to 100k tokens and split it into parts
	// Split messages into parts
	parts := SplitSlicesIntoParts(messages, 1000000)
	for _, part := range parts {
		var text string
		for _, message := range part {
			text += message.GuildID + "/" + message.GuildName + "/" + message.ChannelID + "/" + message.ChannelName + "/" + message.Username + "/" + message.AuthorID + "/" + message.MessageID + ": " + message.Content + "\n"
			if message.Content == "" {
				log.Warnf("Message with ID %s in channel %s has empty content", message.MessageID, message.ChannelName)
			}
		}
		contents = append(contents, Parts{Text: text})
	}

	return &Contents{
		Parts: contents,
	}

}

func SplitSlicesIntoParts(messages []database.IndexedMessages, maxSize int) [][]database.IndexedMessages {
	var parts [][]database.IndexedMessages
	var currentPart []database.IndexedMessages
	currentSize := 0

	for _, message := range messages {
		messageSize := len(message.Content)
		if currentSize+messageSize > maxSize {
			parts = append(parts, currentPart)
			currentPart = []database.IndexedMessages{message}
			currentSize = messageSize
		} else {
			currentPart = append(currentPart, message)
			currentSize += messageSize
		}
	}

	if len(currentPart) > 0 {
		parts = append(parts, currentPart)
	}

	return parts
}

func BuildBody(contents []Contents) *GeminiBody {
	body := &GeminiBody{
		Contents: contents,
		GenerationConfig: GenerationConfig{
			ThinkingConfig: ThinkingConfig{
				ThinkingBudget: 0, // Set a reasonable thinking budget
			},
		},
	}

	return body
}

func SendRequest(body *GeminiBody) (*GeminiResponse, error) {
	request, err := http.NewRequest("POST", "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent", nil)

	if err != nil {
		log.Errorf("Failed to create request: %v", err)
		return nil, err
	}
	request.Header.Set("x-goog-api-key", configuration.Config.Gemini.Token)

	request.Header.Set("Content-Type", "application/json")
	bodyData, err := json.Marshal(body)
	bodyDataToSend := bytes.NewBuffer(bodyData)
	request.Body = io.NopCloser(bodyDataToSend)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Errorf("Failed to send request: %v", err)
		return nil, err
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Failed to read response body: %v", err)
		return nil, err
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		log.Errorf("Request failed with status code %d: %s", response.StatusCode, responseBody)
		return nil, err
	}

	var geminiResponse GeminiResponse
	if err := json.Unmarshal(responseBody, &geminiResponse); err != nil {
		log.Errorf("Failed to unmarshal response: %v", err)
		return nil, err
	}

	log.Infof("Received response: %s", geminiResponse.Candidates[0].Content.Parts[0].Text)
	return &geminiResponse, nil
}
