package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/slack-go/slack"
)

type MessageProcessor struct {
	slackClient *SlackClient
	db          *Database
	userCache   map[string]bool
}

func NewMessageProcessor(slackClient *SlackClient, db *Database) *MessageProcessor {
	return &MessageProcessor{
		slackClient: slackClient,
		db:          db,
		userCache:   make(map[string]bool),
	}
}

func (mp *MessageProcessor) ProcessChannel(ctx context.Context, channelID string) error {
	channel, err := mp.slackClient.GetChannelInfo(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to get channel info: %w", err)
	}

	if err := mp.db.SaveChannel(channelID, channel.Name); err != nil {
		return fmt.Errorf("failed to save channel: %w", err)
	}

	log.Printf("Processing channel: %s (%s)", channel.Name, channelID)

	lastTimestamp, err := mp.db.GetLastMessageTimestamp(channelID)
	if err != nil {
		return fmt.Errorf("failed to get last timestamp: %w", err)
	}

	return mp.fetchAllMessages(ctx, channelID, lastTimestamp)
}

func (mp *MessageProcessor) fetchAllMessages(ctx context.Context, channelID, oldest string) error {
	var cursor string
	messageCount := 0

	for {
		resp, err := mp.slackClient.GetConversationHistory(ctx, channelID, cursor, 200)
		if err != nil {
			return fmt.Errorf("failed to get conversation history: %w", err)
		}

		for _, message := range resp.Messages {
			if oldest != "" && message.Timestamp <= oldest {
				continue
			}

			if err := mp.saveMessage(ctx, channelID, message); err != nil {
				log.Printf("Failed to save message %s: %v", message.Timestamp, err)
				continue
			}

			messageCount++

			if message.ThreadTimestamp != "" && message.ReplyCount > 0 {
				if err := mp.fetchThreadReplies(ctx, channelID, message.ThreadTimestamp); err != nil {
					log.Printf("Failed to fetch replies for thread %s: %v", message.ThreadTimestamp, err)
				}
			}
		}

		if !resp.HasMore {
			break
		}
		cursor = resp.ResponseMetaData.NextCursor
	}

	log.Printf("Processed %d messages for channel %s", messageCount, channelID)
	return nil
}

func (mp *MessageProcessor) fetchThreadReplies(ctx context.Context, channelID, threadTS string) error {
	var cursor string
	replyCount := 0

	for {
		replies, hasMore, nextCursor, err := mp.slackClient.GetConversationReplies(ctx, channelID, threadTS, cursor, 200)
		if err != nil {
			return fmt.Errorf("failed to get conversation replies: %w", err)
		}

		for i, reply := range replies {
			if i == 0 {
				continue
			}

			if err := mp.saveReply(ctx, channelID, threadTS, reply); err != nil {
				log.Printf("Failed to save reply %s: %v", reply.Timestamp, err)
				continue
			}
			replyCount++
		}

		if !hasMore {
			break
		}
		cursor = nextCursor
	}

	if replyCount > 0 {
		log.Printf("Fetched %d replies for thread %s", replyCount, threadTS)
	}
	return nil
}

func (mp *MessageProcessor) saveMessage(ctx context.Context, channelID string, message slack.Message) error {
	if message.User != "" {
		if err := mp.ensureUserInfo(ctx, message.User); err != nil {
			log.Printf("Failed to fetch user info for %s: %v", message.User, err)
		}
	}

	text := message.Text
	if message.SubType == "bot_message" && len(message.Attachments) > 0 {
		var attachmentTexts []string
		for _, attachment := range message.Attachments {
			if attachment.Text != "" {
				attachmentTexts = append(attachmentTexts, attachment.Text)
			}
		}
		if len(attachmentTexts) > 0 {
			text = text + "\n" + strings.Join(attachmentTexts, "\n")
		}
	}

	return mp.db.SaveMessage(
		message.Timestamp,
		channelID,
		message.User,
		text,
		message.ThreadTimestamp,
		message.ReplyCount,
	)
}

func (mp *MessageProcessor) saveReply(ctx context.Context, channelID, threadTS string, reply slack.Message) error {
	if reply.User != "" {
		if err := mp.ensureUserInfo(ctx, reply.User); err != nil {
			log.Printf("Failed to fetch user info for %s: %v", reply.User, err)
		}
	}

	text := reply.Text
	if reply.SubType == "bot_message" && len(reply.Attachments) > 0 {
		var attachmentTexts []string
		for _, attachment := range reply.Attachments {
			if attachment.Text != "" {
				attachmentTexts = append(attachmentTexts, attachment.Text)
			}
		}
		if len(attachmentTexts) > 0 {
			text = text + "\n" + strings.Join(attachmentTexts, "\n")
		}
	}

	return mp.db.SaveReply(
		reply.Timestamp,
		threadTS,
		channelID,
		reply.User,
		text,
	)
}

func (mp *MessageProcessor) ensureUserInfo(ctx context.Context, userID string) error {
	if mp.userCache[userID] {
		return nil
	}

	user, err := mp.slackClient.GetUserInfo(ctx, userID)
	if err != nil {
		return err
	}

	var email, profileImage string
	if user.Profile.Email != "" {
		email = user.Profile.Email
	}
	if user.Profile.Image512 != "" {
		profileImage = user.Profile.Image512
	}

	if err := mp.db.SaveUser(
		user.ID,
		user.Name,
		user.RealName,
		user.Profile.DisplayName,
		email,
		profileImage,
	); err != nil {
		return err
	}

	mp.userCache[userID] = true
	return nil
}