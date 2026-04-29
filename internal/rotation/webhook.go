package rotation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WebhookPayload struct {
	SecretName string    `json:"secret_name"`
	Timestamp  time.Time `json:"timestamp"`
	Message    string    `json:"message"`
}

func SendWebhook(url, secretName, message string) error {
	if url == "" {
		return nil
	}

	payload := WebhookPayload{
		SecretName: secretName,
		Timestamp:  time.Now().UTC(),
		Message:    message,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status: %s", resp.Status)
	}

	return nil
}
