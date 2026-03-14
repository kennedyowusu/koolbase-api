package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const resendAPI = "https://api.resend.com/emails"

type ResendProvider struct {
	apiKey string
	from   string
	client *http.Client
}

func NewResend(apiKey, from string) *ResendProvider {
	return &ResendProvider{apiKey: apiKey, from: from, client: &http.Client{}}
}

func (p *ResendProvider) Send(ctx context.Context, msg Message) error {
	payload := map[string]any{
		"from":    p.from,
		"to":      []string{msg.To},
		"subject": msg.Subject,
		"html":    msg.HTML,
	}
	if msg.Text != "" {
		payload["text"] = msg.Text
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal resend payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendAPI, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("send resend request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("resend API error %d: %v", resp.StatusCode, errBody)
	}

	return nil
}
