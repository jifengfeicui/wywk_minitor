package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"gorm.io/gorm"

	"wywk/api"
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

	fmt.Println(stats)

	notification.SendBarkNotifications(barkTokens, stats, shopName)
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

func main() {
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
