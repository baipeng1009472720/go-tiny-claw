package slackchannel

import (
	"context"

	goslack "github.com/slack-go/slack"
)

type slackMessageSender struct {
	client *goslack.Client
}

func (s *slackMessageSender) SendMessage(ctx context.Context, channelID string, threadTimestamp string, text string) (string, error) {
	options := []goslack.MsgOption{goslack.MsgOptionText(text, false)}
	if threadTimestamp != "" {
		options = append(options, goslack.MsgOptionTS(threadTimestamp))
	}

	_, timestamp, err := s.client.PostMessageContext(ctx, channelID, options...)
	return timestamp, err
}

func (s *slackMessageSender) UpdateMessage(ctx context.Context, channelID string, timestamp string, text string) error {
	_, _, _, err := s.client.UpdateMessageContext(
		ctx,
		channelID,
		timestamp,
		goslack.MsgOptionText(text, false),
	)
	return err
}
