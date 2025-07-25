package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type ClientInfo struct {
	RoomCode    string `json:"roomCode"`
	RoomName    string `json:"roomName"`
	ClientIp    string `json:"clientIp"`
	ClientNo    string `json:"clientNo"`
	DisplayName string `json:"displayName"`
	Status      int    `json:"status"`
}

type Element struct {
	ID          int         `json:"id"`
	ElementCode string      `json:"elementCode"`
	DisplayName string      `json:"displayName"`
	ClientInfo  *ClientInfo `json:"clientInfo"`
}

type Area struct {
	ID       int       `json:"id"`
	AreaName string    `json:"areaName"`
	Elements []Element `json:"elements"`
}

type Data struct {
	Areas []Area `json:"areas"`
}

type Response struct {
	Data Data `json:"data"`
}

func getShopStats(commonCode string) (string, error) {
	apiURL := "https://vip-gateway.wywk.cn/surf-internet/shop/v3/get"
	payload := map[string]string{"commonCode": commonCode}
	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response Response
	json.Unmarshal(body, &response)

	totalDevices := 0
	usedDevices := 0
	roomStats := make(map[string]map[string]int)

	for _, area := range response.Data.Areas {
		for _, element := range area.Elements {
			if element.ClientInfo != nil {
				totalDevices++
				if _, ok := roomStats[element.ClientInfo.RoomName]; !ok {
					roomStats[element.ClientInfo.RoomName] = make(map[string]int)
					roomStats[element.ClientInfo.RoomName]["total"] = 0
					roomStats[element.ClientInfo.RoomName]["used"] = 0
				}
				roomStats[element.ClientInfo.RoomName]["total"]++
				if element.ClientInfo.Status == 1 {
					usedDevices++
					roomStats[element.ClientInfo.RoomName]["used"]++
				}
			}
		}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("总设备: %d, 在用: %d\n", totalDevices, usedDevices))
	if totalDevices > 0 {
		usageRate := float64(usedDevices) / float64(totalDevices) * 100
		result.WriteString(fmt.Sprintf("总使用率: %.2f%%\n\n", usageRate))
	}

	result.WriteString("各房间使用率:\n")
	for roomName, stats := range roomStats {
		if stats["total"] > 0 {
			roomUsageRate := float64(stats["used"]) / float64(stats["total"]) * 100
			result.WriteString(fmt.Sprintf("%s: %.2f%% (%d/%d)\n", roomName, roomUsageRate, stats["used"], stats["total"]))
		}
	}

	return result.String(), nil
}

func sendBarkNotification(barkBaseURL, message string) error {
	// 确保 baseURL 没有结尾的 /
	barkBaseURL = strings.TrimRight(barkBaseURL, "/")

	// message 做 URL 编码（只编码一次）
	escapedMessage := url.PathEscape(message)

	// 构建最终 URL（只手动拼接路径部分）
	finalURL := fmt.Sprintf("%s/%s?group=wy", barkBaseURL, escapedMessage)

	resp, err := http.Get(finalURL)
	if err != nil {
		return fmt.Errorf("failed to send bark notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bark notification failed with status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Println("Bark notification sent successfully!")
	return nil
}

func sendBarkNotificationPhone(message string) error {
	var err error
	cyhbarkBaseURL := "https://api.day.app/5dDEJShrGCTCfoWo6V9FyG"
	err = sendBarkNotification(cyhbarkBaseURL, message)
	//lthbarkBaseURL := "https://api.day.app/5dDEJShrGCTCfoWo6V9FyG"
	//err=sendBarkNotification(lthbarkBaseURL, message)
	if err != nil {
		return err
	}
	return nil
}

func req(commonCode string) {
	stats, err := getShopStats(commonCode)
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
		// Also send error notification
		_ = sendBarkNotificationPhone(fmt.Sprintf("获取状态失败: %v", err))
		return
	}

	fmt.Println(stats)

	err = sendBarkNotificationPhone(stats)
	if err != nil {
		fmt.Printf("Error sending bark notification: %v\n", err)
	}
}

func main() {
	req("0437")
}
