package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gorm.io/gorm"

	"wywk/api"
	"wywk/daily"
	"wywk/db"
	"wywk/notification"
)

func processShop(db *gorm.DB, commonCode string, barkTokens []string) {
	log.Printf("Processing shop with common code: %s", commonCode)
	stats, shopName, err := api.GetShopStats(db, commonCode)
	if err != nil {
		log.Printf("Error getting stats for %s: %v", commonCode, err)
		notificationMessage := fmt.Sprintf("获取 %s 状态失败: %v", commonCode, err)
		notification.SendBarkNotifications(barkTokens, notificationMessage, "") // No shopName if error getting stats
		return
	}

	//fmt.Println(stats)
	//notification.SendBarkNotifications(barkTokens, stats, shopName)
	_, _ = stats, shopName
}

type Config struct {
	CommonCodes []string `json:"commonCodes"`
	BarkTokens  []string `json:"barkTokens"`
}

func ChangeWorkingDir() {
	var err error
	executable, err := os.Executable()
	_ = executable
	if err != nil {
		return
	}

	// 只有在linux的系统上，才修改工作目录为可执行文件的目录
	if runtime.GOOS == "linux" {
		_ = os.Chdir(filepath.Dir(executable))
	}
}

func crawlData() {
	ChangeWorkingDir()
	// --- Config Loading ---
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("Error opening config file: %v", err)
	}
	defer configFile.Close()

	var config Config
	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&config); err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}

	// --- Database Setup ---
	db := db.InitDB()

	// --- Main Logic ---
	for _, commonCode := range config.CommonCodes {
		processShop(db, commonCode, config.BarkTokens)
	}
}

func main() {
	// 1. Crawl live data and save it.
	crawlData()

	// 2. If it's between 00:00 and 01:00, generate and send a report from DB.
	now := time.Now()
	if now.Hour() == 0 {
		log.Println("Running daily report job...")
		ChangeWorkingDir()

		// --- Config Loading ---
		configFile, err := os.Open("config.json")
		if err != nil {
			log.Fatalf("Error opening config file for reporting: %v", err)
		}
		defer configFile.Close()

		var config Config
		jsonParser := json.NewDecoder(configFile)
		if err = jsonParser.Decode(&config); err != nil {
			log.Fatalf("Error parsing config file for reporting: %v", err)
		}

		// --- Database Setup ---
		db := db.InitDB()

		// --- Reporting Logic ---
		for _, commonCode := range config.CommonCodes {
			daily.GenerateAndSendDailyReport(db, commonCode, config.BarkTokens)
		}
		log.Println("Daily report job finished.")
	}
}
