package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func sendBarkNotification_get(barkBaseURL, message, shopName string) error {
	// Ensure barkBaseURL has a scheme
	if !strings.Contains(barkBaseURL, "://") {
		barkBaseURL = "https://api.day.app/" + barkBaseURL
	}
	barkBaseURL = strings.TrimRight(barkBaseURL, "/")
	escapedMessage := url.PathEscape(message)
	// Use shopName for the group parameter
	finalURL := fmt.Sprintf("%s/%s?group=%s", barkBaseURL, escapedMessage, url.QueryEscape(shopName))

	resp, err := http.Get(finalURL)
	if err != nil {
		return fmt.Errorf("failed to send bark notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bark notification failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Bark notification sent successfully for %s!", shopName)
	return nil
}

func sendBarkNotification(barkBaseURL, message, shopName string) error {
	// Ensure barkBaseURL has a scheme
	if !strings.Contains(barkBaseURL, "://") {
		barkBaseURL = "https://api.day.app/" + barkBaseURL
	}
	barkBaseURL = strings.TrimRight(barkBaseURL, "/")

	// Bark POST JSON 格式
	payload := map[string]string{
		"title": shopName,
		"body":  message,
		"group": shopName, // 用 shopName 分组
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := http.Post(barkBaseURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send bark notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bark notification failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Bark notification sent successfully for %s!", shopName)
	return nil
}

func SendBarkNotifications(barkTokens []string, message, shopName string) {
	if len(barkTokens) == 0 {
		log.Println("No Bark tokens configured. Skipping notification.")
		return
	}
	for _, token := range barkTokens {
		if err := sendBarkNotification(token, message, shopName); err != nil {
			log.Printf("Failed to send Bark notification to token ending in ...%s for shop %s: %v", getLast4Chars(token), shopName, err)
		}
	}
}

func getLast4Chars(s string) string {
	if len(s) > 4 {
		return s[len(s)-4:]
	}
	return s
}
