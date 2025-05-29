package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	var (
		token     = flag.String("token", "", "Slack Bot Token (required for fetch mode)")
		channelID = flag.String("channel", "", "Channel ID to process")
		dbPath    = flag.String("db", "slack_data.db", "SQLite database path")
		mode      = flag.String("mode", "fetch", "Mode: fetch (default) or export")
		output    = flag.String("output", "", "Output file path for export mode")
		outputDir = flag.String("output-dir", "", "Output directory for exporting all channels")
	)
	flag.Parse()

	db, err := NewDatabase(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	switch *mode {
	case "fetch":
		if err := runFetchMode(*token, *channelID, db); err != nil {
			log.Fatalf("Fetch mode failed: %v", err)
		}
	case "export":
		if err := runExportMode(*channelID, *output, *outputDir, db); err != nil {
			log.Fatalf("Export mode failed: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Error: Invalid mode '%s'. Use 'fetch' or 'export'\n", *mode)
		flag.Usage()
		os.Exit(1)
	}
}

func runFetchMode(token, channelID string, db *Database) error {
	if token == "" {
		if envToken := os.Getenv("SLACK_BOT_TOKEN"); envToken != "" {
			token = envToken
		} else {
			fmt.Fprintf(os.Stderr, "Error: Slack token is required. Use -token flag or SLACK_BOT_TOKEN environment variable\n")
			flag.Usage()
			os.Exit(1)
		}
	}

	if channelID == "" {
		fmt.Fprintf(os.Stderr, "Error: Channel ID is required. Use -channel flag\n")
		flag.Usage()
		os.Exit(1)
	}

	channelID = strings.TrimPrefix(channelID, "#")

	ctx := context.Background()
	slackClient := NewSlackClient(token)
	processor := NewMessageProcessor(slackClient, db)

	log.Printf("Starting to process channel: %s", channelID)
	log.Printf("Database: %s", db)

	if err := processor.ProcessChannel(ctx, channelID); err != nil {
		return fmt.Errorf("failed to process channel: %w", err)
	}

	log.Println("Processing completed successfully")
	return nil
}

func runExportMode(channelID, output, outputDir string, db *Database) error {
	exporter := NewExporter(db)

	if outputDir != "" {
		log.Printf("Exporting all channels to directory: %s", outputDir)
		return exporter.ExportAllChannels(outputDir)
	}

	if channelID == "" {
		fmt.Fprintf(os.Stderr, "Error: Channel ID is required for single channel export. Use -channel flag\n")
		flag.Usage()
		os.Exit(1)
	}

	if output == "" {
		channelID = strings.TrimPrefix(channelID, "#")
		output = fmt.Sprintf("channel_%s.txt", channelID)
	}

	log.Printf("Exporting channel %s to %s", channelID, output)
	return exporter.ExportToText(channelID, output)
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  Fetch messages:\n")
		fmt.Fprintf(os.Stderr, "    %s -token xoxb-your-token -channel C1234567890\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "    %s -channel general  # using SLACK_BOT_TOKEN env var\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n  Export to text:\n")
		fmt.Fprintf(os.Stderr, "    %s -mode export -channel C1234567890 -output channel.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "    %s -mode export -output-dir ./exports  # export all channels\n", os.Args[0])
	}
}