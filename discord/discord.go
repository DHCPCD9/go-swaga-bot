package discord

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/DHCPCD9/go-swaga-bot/configuration"
	"github.com/DHCPCD9/go-swaga-bot/database"
	"github.com/DHCPCD9/go-swaga-bot/gemini"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

var discord *discordgo.Session

func Init() error {

	if configuration.Config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	if database.Pool == nil {
		return fmt.Errorf("database not initialized")
	}

	discord, err := discordgo.New("Bot " + configuration.Config.Discord.Token)

	if err != nil {
		log.Errorf("error creating Discord session: %v", err)
		return fmt.Errorf("error creating Discord session: %w", err)
	}

	discord.Identify.Intents = discordgo.IntentsAll

	discord.AddHandler(handleReady)
	discord.AddHandler(handleMessage)
	err = discord.Open()

	if err != nil {
		log.Errorf("error opening Discord session: %v", err)
		return fmt.Errorf("error opening Discord session: %w", err)
	}

	return nil
}

func handleReady(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateCustomStatus("Listening to you~")
	log.Infof("Logged in as %s#%s", event.User.Username, event.User.Discriminator)

	var count int64
	database.Pool.Model(&database.IndexedMessages{}).Count(&count)
	log.Infof("Indexed %d messages in the database", count)
}

func handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {

	log.Infof("Received message from %s: %s", m.Author.Username, m.Content)

	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		log.Errorf("Failed to get channel %s: %v", m.ChannelID, err)
		return
	}

	guild, err := s.State.Guild(channel.GuildID)
	if err != nil {
		log.Errorf("Failed to get guild %s: %v", channel.GuildID, err)
		return
	}

	indexedMessage := database.IndexedMessages{
		MessageID:   m.ID,
		Content:     m.Content,
		ChannelID:   m.ChannelID,
		ChannelName: channel.Name,
		GuildID:     m.GuildID,
		GuildName:   guild.Name,
		CreatedAt:   m.Timestamp.Unix(),
		AuthorID:    m.Author.ID,
		Username:    m.Author.Username,
	}

	if err := database.Pool.Create(&indexedMessage).Error; err != nil {
		log.Errorf("Failed to index message %s in channel %s: %v", m.ID, channel.Name, err)
	} else {
		log.Infof("Indexed message %s in channel %s", m.ID, channel.Name)
	}

	isMeMentioned := false
	for _, mention := range m.Mentions {
		if mention.ID == s.State.User.ID {
			isMeMentioned = true
			break
		}
	}

	// 	{
	//     "user_id": <userId>,
	//     "username": <username>,
	//     "known_names": ["name1", "name2"],
	//     "activities": [
	//         {
	//             "activity": "<activity_name>",
	//             "state": "<activity_state>",
	//             "substate": "<activity_substate>"
	//         }
	//     ],
	//     "facts": ["fact1", "fact2"],
	//     "text": "<text>",
	//     "reference": <reference_id>,
	//     "references": [{"id": <id>, "text": "text", "user": <user_id>}],
	//     "reference_users": [{"id": <user_id>, "username": "<username>", "known_names": ["name1", "name2"], "facts": ["fact1", "fact2"]}],
	// }
	if isMeMentioned && m.Author.ID != s.State.User.ID {
		s.ChannelTyping(m.ChannelID)

		parts := gemini.BuildParts(m.Author.ID, m.GuildID)

		for _, attachment := range m.Attachments {
			// if attachment.ContentType != "" && strings.HasPrefix(attachment.ContentType, "image/") {
			//Downloading from discord and converting image to base64
			data, err := s.Request("GET", attachment.URL, nil)
			if err != nil {
				log.Errorf("Failed to download image %s: %v", attachment.URL, err)
				continue
			}
			b64Image := base64.StdEncoding.EncodeToString(data)
			parts.Parts = append(parts.Parts, gemini.Parts{
				InlineData: &struct {
					MimeType string "json:\"mime_type\""
					Data     string "json:\"data\""
				}{
					MimeType: attachment.ContentType,
					Data:     b64Image,
				},
			})
			// }
		}

		// baseText := fmt.Sprintf("<@%s> Asked: %s", m.Author.ID, m.Content)
		var dbUser database.KnownUsers
		parsedID, _ := strconv.Atoi(m.Author.ID)

		var facts []database.UserFact
		var names []database.UserName

		database.Pool.First(&dbUser, "id = ?", parsedID)
		database.Pool.Find(&facts, "user_id = ?", parsedID)
		database.Pool.Find(&names, "user_id = ?", parsedID)
		basePrompt := gemini.PromptJson{
			UserID:     m.Author.ID,
			Username:   m.Author.Username,
			Text:       m.Content,
			KnownNames: database.NamestToStrings(names),
			Activities: make([]struct {
				Activity string "json:\"activity\""
				State    string "json:\"state\""
				Substate string "json:\"substate\""
			}, 0),
			Facts:     database.FactsToStrings(facts),
			Reference: m.Reference().MessageID,
			References: make([]struct {
				ID   string "json:\"id\""
				Text string "json:\"text\""
				User string "json:\"user\""
			}, 0),
			ReferenceUsers: make([]struct {
				ID         string   "json:\"id\""
				Username   string   "json:\"username\""
				KnownNames []string "json:\"known_names\""
				Facts      []string "json:\"facts\""
			}, 0),
		}

		presences, err := s.State.Presence(m.GuildID, m.Author.ID)

		log.Debugf("Presence for %s: %+v", m.Author.ID, presences)
		if err != nil {
			log.Errorf("Failed to get presence for user %s: %v", m.Author.ID, err)
		}

		if presences != nil {
			basePrompt.Activities = make([]struct {
				Activity string "json:\"activity\""
				State    string "json:\"state\""
				Substate string "json:\"substate\""
			}, 0)
			for _, activity := range presences.Activities {
				if activity.Name == "" {
					continue
				}
				basePrompt.Activities = append(basePrompt.Activities, struct {
					Activity string "json:\"activity\""
					State    string "json:\"state\""
					Substate string "json:\"substate\""
				}{
					Activity: activity.Name,
					State:    activity.State,
					Substate: activity.Details,
				})
			}
		}

		//And for mentioned users
		for _, mention := range m.Mentions {
			presences, err := s.State.Presence(m.GuildID, mention.ID)
			log.Debugf("Presence for %s: %+v", mention.ID, presences)
			if err != nil {
				log.Errorf("Failed to get presence for user %s: %v", mention.ID, err)
			}

			var dbUser database.KnownUsers
			database.Pool.First(&dbUser, "id = ?", mention.ID)

			var facts []database.UserFact
			var names []database.UserName

			database.Pool.Find(&facts, "user_id = ?", mention.ID)
			database.Pool.Find(&names, "user_id = ?", mention.ID)
			if presences != nil {
				basePrompt.ReferenceUsers = append(basePrompt.ReferenceUsers, struct {
					ID         string   "json:\"id\""
					Username   string   "json:\"username\""
					KnownNames []string "json:\"known_names\""
					Facts      []string "json:\"facts\""
				}{
					ID:         mention.ID,
					Username:   mention.Username,
					KnownNames: database.NamestToStrings(names),
					Facts:      database.FactsToStrings(facts),
				})
			} else {
				basePrompt.ReferenceUsers = append(basePrompt.ReferenceUsers, struct {
					ID         string   "json:\"id\""
					Username   string   "json:\"username\""
					KnownNames []string "json:\"known_names\""
					Facts      []string "json:\"facts\""
				}{
					ID:         mention.ID,
					Username:   mention.Username,
					KnownNames: make([]string, 0),
					Facts:      make([]string, 0),
				})
			}
		}

		// baseText += "[Attachment from part above]"

		var marshaledBasePrompt []byte
		marshaledBasePrompt, err = json.Marshal(basePrompt)
		log.Debug("Base text for Gemini request: ", string(marshaledBasePrompt))

		parts.Parts = append(parts.Parts, gemini.Parts{Text: string(marshaledBasePrompt), InlineData: nil})

		body := gemini.BuildBody([]gemini.Contents{*parts})

		if err != nil {
			log.Errorf("Failed to send Gemini request: %v", err)
			return
		}

		response, err := gemini.SendRequest(body)

		// jsonToSave, _ := json.Marshal(response)

		// os.WriteFile("gemini_request.json", jsonToSave, 0644)

		if err != nil {
			log.Errorf("Failed to send Gemini request: %v", err)
			return
		}

		if len(response.Candidates) > 0 {
			//Answering the first candidate
			answer := response.Candidates[0].Content.Parts[0].Text

			answer = strings.TrimLeft(answer, "```json")
			answer = strings.TrimRight(answer, "```")
			var parsedAnswer gemini.ResponseJson
			if err = json.Unmarshal([]byte(answer), &parsedAnswer); err != nil {
				if _, err := s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Failed to process message: %s", err.Error()), m.Reference()); err != nil {
					log.Errorf("Failed to send message to channel %s: %v", m.ChannelID, err)
				} else {
					log.Infof("Sent response to channel %s: %s", m.ChannelID, answer)
				}
				return
			}

			for _, username := range parsedAnswer.Usernames {
				if username.Type == "add" {
					parsed, _ := strconv.Atoi(username.User)
					database.AddUsername(uint64(parsed), username.Username)
				}
			}

			for _, fact := range parsedAnswer.Facts {
				parsed, _ := strconv.Atoi(fact.User)

				if fact.Type == "add" {
					database.AddFact(uint64(parsed), fact.Fact)
				} else {
					database.RemoveFact(uint64(parsed), fact.Fact)
				}
			}

			if _, err := s.ChannelMessageSendReply(m.ChannelID, parsedAnswer.Response, m.Reference()); err != nil {
				log.Errorf("Failed to send message to channel %s: %v", m.ChannelID, err)
			} else {
				log.Infof("Sent response to channel %s: %s", m.ChannelID, answer)
			}
		}
	}
}
