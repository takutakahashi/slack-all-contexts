package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Exporter struct {
	db *Database
}

func NewExporter(db *Database) *Exporter {
	return &Exporter{db: db}
}

func (e *Exporter) ExportToText(channelID, outputPath string) error {
	messages, err := e.db.GetAllMessagesWithReplies(channelID)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	if len(messages) == 0 {
		return fmt.Errorf("no messages found for channel %s", channelID)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	channelName := messages[0].ChannelName
	
	fmt.Fprintf(file, "# Slack Channel Export: #%s\n", channelName)
	fmt.Fprintf(file, "Channel ID: %s\n", channelID)
	fmt.Fprintf(file, "Export Date: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "Total Messages: %d\n\n", len(messages))
	fmt.Fprintf(file, "="+ strings.Repeat("=", 70) + "\n\n")

	for _, msg := range messages {
		timestamp := e.formatTimestamp(msg.Timestamp)
		userDisplay := e.formatUserDisplay(msg.UserID, msg.UserName, msg.UserRealName, msg.UserDisplayName)

		fmt.Fprintf(file, "[%s] %s:\n%s\n", timestamp, userDisplay, msg.Text)

		if len(msg.Replies) > 0 {
			fmt.Fprintf(file, "\n  Thread Replies (%d):\n", len(msg.Replies))
			for _, reply := range msg.Replies {
				replyTime := e.formatTimestamp(reply.Timestamp)
				replyUserDisplay := e.formatUserDisplay(reply.UserID, reply.UserName, reply.UserRealName, reply.UserDisplayName)
				fmt.Fprintf(file, "  [%s] %s: %s\n", replyTime, replyUserDisplay, reply.Text)
			}
		}

		fmt.Fprintf(file, "\n" + strings.Repeat("-", 80) + "\n\n")
	}

	return nil
}

func (e *Exporter) ExportAllChannels(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	channels, err := e.db.GetChannels()
	if err != nil {
		return fmt.Errorf("failed to get channels: %w", err)
	}

	if len(channels) == 0 {
		return fmt.Errorf("no channels found in database")
	}

	for channelID, channelName := range channels {
		safeChannelName := strings.ReplaceAll(channelName, "/", "_")
		safeChannelName = strings.ReplaceAll(safeChannelName, " ", "_")
		
		outputPath := fmt.Sprintf("%s/%s_%s.txt", outputDir, safeChannelName, channelID)
		
		if err := e.ExportToText(channelID, outputPath); err != nil {
			fmt.Printf("Warning: Failed to export channel %s (%s): %v\n", channelName, channelID, err)
			continue
		}
		
		fmt.Printf("Exported channel #%s to %s\n", channelName, outputPath)
	}

	return nil
}

func (e *Exporter) formatTimestamp(ts string) string {
	timestamp, err := strconv.ParseFloat(ts, 64)
	if err != nil {
		return ts
	}
	
	t := time.Unix(int64(timestamp), 0)
	return t.Format("2006-01-02 15:04:05")
}

func (e *Exporter) formatUserDisplay(userID, userName, realName, displayName string) string {
	if userID == "" {
		return "Unknown"
	}

	var parts []string
	
	if displayName != "" {
		parts = append(parts, displayName)
	} else if realName != "" {
		parts = append(parts, realName)
	}
	
	if userName != "" {
		parts = append(parts, fmt.Sprintf("(@%s)", userName))
	}
	
	if len(parts) == 0 {
		return fmt.Sprintf("User<%s>", userID)
	}
	
	return strings.Join(parts, " ")
}