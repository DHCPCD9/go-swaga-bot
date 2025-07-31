package discord

import (
	"encoding/base64"
	"fmt"

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

	if configuration.Config.Discord.IndexAllChannels && count == 0 {
		for _, guild := range s.State.Guilds {

			log.Infof("Indexing all channels in guild: %s (%s)", guild.Name, guild.ID)
			if channels, err := s.GuildChannels(guild.ID); err != nil {
				log.Errorf("Failed to get channels for guild %s: %v", guild.Name, err)
				continue
			} else {
				for _, channel := range channels {
					if channel.Type == discordgo.ChannelTypeGuildText {
						log.Infof("Indexing channel: %s (%s)", channel.Name, channel.ID)

						beforeId := ""
						for {
							messages, err := s.ChannelMessages(channel.ID, 100, beforeId, "", "")
							if err != nil {
								log.Errorf("Failed to get messages for channel %s: %v", channel.Name, err)
								break
							}

							if len(messages) == 0 {
								log.Infof("No more messages to index in channel %s", channel.Name)
								break
							}

							indexedMessages := make([]database.IndexedMessages, 0, len(messages))
							for _, message := range messages {
								indexedMessage := database.IndexedMessages{
									MessageID:   message.ID,
									Content:     message.Content,
									ChannelID:   channel.ID,
									ChannelName: channel.Name,
									GuildID:     guild.ID,
									GuildName:   guild.Name,
									CreatedAt:   message.Timestamp.Unix(),
									AuthorID:    message.Author.ID,
									Username:    message.Author.Username,
								}

								indexedMessages = append(indexedMessages, indexedMessage)
							}

							if err := database.Pool.Create(&indexedMessages).Error; err != nil {
								log.Errorf("Failed to index %d messages in channel %s: %v", len(indexedMessages), channel.Name, err)
							} else {
								log.Infof("Indexed message %d messages in channel %s", len(indexedMessages), channel.Name)
							}
							beforeId = messages[len(messages)-1].ID
							log.Infof("Indexed %d messages in channel %s", len(messages), channel.Name)
							if len(messages) < 100 {
								log.Infof("No more messages to index in channel %s", channel.Name)
								break
							}
						}
					}
				}
			}

		}
	}

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

	if isMeMentioned && m.Author.ID != s.State.User.ID {
		s.ChannelTyping(m.ChannelID)

		parts := gemini.BuildParts()
		/**
		  ```
		  <@userId> Activities <count>:
		<activity1> - [activity1-state]
		<activity2>
		  <@userId> Asked: <text>
		  [with reference: (text)]
		  ```

		*/

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

		baseText := fmt.Sprintf("<@%s> Asked: %s", m.Author.ID, m.Content)

		presences, err := s.State.Presence(m.GuildID, m.Author.ID)

		log.Debugf("Presence for %s: %+v", m.Author.ID, presences)
		if err != nil {
			log.Errorf("Failed to get presence for user %s: %v", m.Author.ID, err)
		}

		if presences != nil {
			baseText += fmt.Sprintf("Activities %d:", len(presences.Activities))
			for _, activity := range presences.Activities {
				if activity.Name == "" {
					continue
				}
				baseText += fmt.Sprintf("\n- %s", activity.Name)
				if activity.State != "" {
					baseText += fmt.Sprintf(" - [%s]", activity.State)
				}

			}
		}

		//And for mentioned users
		for _, mention := range m.Mentions {
			presences, err := s.State.Presence(m.GuildID, mention.ID)
			log.Debugf("Presence for %s: %+v", mention.ID, presences)
			if err != nil {
				log.Errorf("Failed to get presence for user %s: %v", mention.ID, err)
			}
			if presences != nil {
				baseText += fmt.Sprintf("\n<@%s> Activities %d:", mention.ID, len(presences.Activities))
				for _, activity := range presences.Activities {
					if activity.Name == "" {
						continue
					}
					baseText += fmt.Sprintf("\n- %s", activity.Name)
					if activity.State != "" {
						baseText += fmt.Sprintf(" - [%s]", activity.State)
					}
				}
			}
		}

		if m.ReferencedMessage != nil {
			// parts.Parts = append(parts.Parts, gemini.Parts{Text: fmt.Sprintf("<@%s> Asked: %s [with reference: %s]", m.Author.ID, m.Content, m.ReferencedMessage.Content)})
			baseText += fmt.Sprintf("<@%s> Asked: %s [with reference: %s]", m.Author.ID, m.Content, m.ReferencedMessage.Content)
		} else {
			// parts.Parts = append(parts.Parts, gemini.Parts{Text: fmt.Sprintf("<@%s> Asked: %s", m.Author.ID, m.Content)})
		}

		baseText += "[Attachment from part above]"

		log.Debug("Base text for Gemini request: ", baseText)

		parts.Parts = append(parts.Parts, gemini.Parts{Text: baseText, InlineData: nil})

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

			if _, err := s.ChannelMessageSendReply(m.ChannelID, answer, m.Reference()); err != nil {
				log.Errorf("Failed to send message to channel %s: %v", m.ChannelID, err)
			} else {
				log.Infof("Sent response to channel %s: %s", m.ChannelID, answer)
			}
		}
	}
}
