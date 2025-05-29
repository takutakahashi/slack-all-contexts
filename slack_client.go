package main

import (
	"context"
	"time"

	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

type SlackClient struct {
	client  *slack.Client
	limiter *rate.Limiter
}

func NewSlackClient(token string) *SlackClient {
	client := slack.New(token)
	limiter := rate.NewLimiter(rate.Every(time.Second), 1)

	return &SlackClient{
		client:  client,
		limiter: limiter,
	}
}

func (sc *SlackClient) waitForRateLimit(ctx context.Context) error {
	return sc.limiter.Wait(ctx)
}

func (sc *SlackClient) GetChannelInfo(ctx context.Context, channelID string) (*slack.Channel, error) {
	if err := sc.waitForRateLimit(ctx); err != nil {
		return nil, err
	}

	channel, err := sc.client.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
		ChannelID: channelID,
	})
	return channel, err
}

func (sc *SlackClient) GetConversationHistory(ctx context.Context, channelID string, cursor string, limit int) (*slack.GetConversationHistoryResponse, error) {
	if err := sc.waitForRateLimit(ctx); err != nil {
		return nil, err
	}

	params := &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Cursor:    cursor,
		Limit:     limit,
	}

	return sc.client.GetConversationHistoryContext(ctx, params)
}

func (sc *SlackClient) GetConversationReplies(ctx context.Context, channelID, timestamp string, cursor string, limit int) ([]slack.Message, bool, string, error) {
	if err := sc.waitForRateLimit(ctx); err != nil {
		return nil, false, "", err
	}

	params := &slack.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: timestamp,
		Cursor:    cursor,
		Limit:     limit,
	}

	return sc.client.GetConversationRepliesContext(ctx, params)
}

func (sc *SlackClient) GetUserInfo(ctx context.Context, userID string) (*slack.User, error) {
	if err := sc.waitForRateLimit(ctx); err != nil {
		return nil, err
	}

	user, err := sc.client.GetUserInfoContext(ctx, userID)
	return user, err
}