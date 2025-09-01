package daily

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"wywk/models"
	"wywk/notification"
)

// DailyStats holds the result of the overall aggregation query.
type DailyStats struct {
	AvgUsageRate   float64
	MaxUsageRate   float64
	AvgUsedDevices float64
	MaxUsedDevices float64 // Use float64 for easier scanning from AVG
	RecordCount    int64
}

// HourlyStat holds the result of the hourly aggregation query.
type HourlyStat struct {
	Hour           string // Comes as a string like "08", "14" from strftime
	AvgRate        float64
	AvgUsedDevices float64
}

// buildBar creates a simple text-based bar for a percentage.
func buildBar(percentage float64, barLength int) string {
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}
	filledLength := int(percentage / 100 * float64(barLength))
	return strings.Repeat("█", filledLength) + strings.Repeat("░", barLength-filledLength)
}

// GenerateAndSendDailyReport queries the database for yesterday's statistics and sends a report.
func GenerateAndSendDailyReport(db *gorm.DB, commonCode string, barkTokens []string) {
	log.Printf("Generating daily report for %s", commonCode)

	var shop models.Shop
	if err := db.Where("common_code = ?", commonCode).First(&shop).Error; err != nil {
		log.Printf("Could not find shop with common_code %s: %v", commonCode, err)
		return
	}

	// --- Calculate yesterday's time range ---
	now := time.Now()
	year, month, day := now.Date()
	todayStart := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
	yesterdayStart := todayStart.AddDate(0, 0, -1)

	// --- Query 1: Overall Daily Stats ---
	var stats DailyStats
	result := db.Model(&models.Snapshot{}).
		Select("COUNT(*) as record_count, AVG(usage_rate) as avg_usage_rate, MAX(usage_rate) as max_usage_rate, AVG(used_devices) as avg_used_devices, MAX(used_devices) as max_used_devices").
		Where("shop_id = ? AND timestamp >= ? AND timestamp < ?", shop.ID, yesterdayStart, todayStart).
		Group("shop_id").
		Scan(&stats)

	if result.Error != nil {
		log.Printf("Error querying daily stats for shop %s: %v", shop.Name, result.Error)
		return
	}

	if stats.RecordCount == 0 {
		log.Printf("No snapshots found for shop %s for yesterday.", shop.Name)
		return
	}

	// --- Query 2: Get TotalDevices from the last snapshot ---
	var lastSnapshot models.Snapshot
	db.Model(&models.Snapshot{}).
		Where("shop_id = ? AND timestamp >= ? AND timestamp < ?", shop.ID, yesterdayStart, todayStart).
		Order("timestamp DESC").
		First(&lastSnapshot)

	// --- Query 3: Hourly Breakdown ---
	var hourlyStats []HourlyStat
	db.Model(&models.Snapshot{}).
		Select("strftime('%H', timestamp) as hour, AVG(usage_rate) as avg_rate, AVG(used_devices) as avg_used_devices").
		Where("shop_id = ? AND timestamp >= ? AND timestamp < ?", shop.ID, yesterdayStart, todayStart).
		Group("hour").
		Order("hour").
		Scan(&hourlyStats)

	// --- Format Report ---
	var report strings.Builder
	report.WriteString(fmt.Sprintf(
		"【%s】昨日数据报告\n设备总数: %d\n记录数: %d\n平均使用率: %.2f%%\n峰值使用率: %.2f%%\n平均在用: %.1f台\n峰值在用: %.0f台\n",
		shop.Name,
		lastSnapshot.TotalDevices,
		stats.RecordCount,
		stats.AvgUsageRate,
		stats.MaxUsageRate,
		stats.AvgUsedDevices,
		stats.MaxUsedDevices,
	))

	if len(hourlyStats) > 0 {
		report.WriteString("\n--- 分时段使用率 ---\n")
		// Create a map for easy lookup
		hourlyMap := make(map[int]HourlyStat)
		for _, hs := range hourlyStats {
			hour, _ := strconv.Atoi(hs.Hour)
			hourlyMap[hour] = hs
		}

		//report.WriteString("╔══════╦══════╦══════╗\n")
		report.WriteString("║ 时段 ║ 使用率 ║ 在用台数 ║\n")
		//report.WriteString("╠══════╬══════╬══════╣\n")
		for h := 0; h < 24; h++ {
			if hs, ok := hourlyMap[h]; ok {
				// 使用整数，更紧凑
				report.WriteString(fmt.Sprintf("║ %02d:00 ║  %3.0f%% ║    %2.0f    ║\n", h, hs.AvgRate, hs.AvgUsedDevices))
			}
		}
	}

	//fmt.Println(report.String())
	notification.SendBarkNotifications(barkTokens, report.String(), shop.Name)
}
