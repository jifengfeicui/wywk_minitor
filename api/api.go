package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	. "wywk/models"
)

func GetShopStats(db *gorm.DB, commonCode string) (string, string, error) {
	shopInfo, err := getShopInfo("https://vip-gateway.wywk.cn", commonCode)
	if err != nil {
		return "", "", err
	}

	shop, err := createOrUpdateShop(db, commonCode, shopInfo)
	if err != nil {
		return "", "", err
	}

	if shopInfo.Data.ShopStatus != "营业中" {
		handleNonOperatingStatus(db, shop.ID, shopInfo.Data.ShopStatus)
		return fmt.Sprintf(`店名: %s
地址: %s
状态: %s`, shop.Name, shop.Address, shopInfo.Data.ShopStatus), shop.Name, nil
	}

	detailResponse, err := getShopDetails(commonCode)
	if err != nil {
		return "", shop.Name, err
	}

	totalDevices, usedDevices, roomStats, roomCodeToName, physicalRoomProperties := processShopData(detailResponse)

	err = saveShopData(db, shop, totalDevices, usedDevices, roomStats, roomCodeToName, physicalRoomProperties)
	if err != nil {
		return "", shop.Name, err
	}

	notification := formatNotification(shop, totalDevices, usedDevices, roomStats, roomCodeToName)
	return notification, shop.Name, nil
}

func getShopInfo(baseURL, commonCode string) (*ShopInfoResponse, error) {
	shopInfoURL := fmt.Sprintf("%s/asset-svc/shop/store/portal/baseMessage?commonCode=%s", baseURL, commonCode)
	resp, err := http.Get(shopInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var shopInfoResponse ShopInfoResponse
	if err := json.Unmarshal(body, &shopInfoResponse); err != nil {
		return nil, fmt.Errorf("failed to parse shop info JSON: %w", err)
	}
	return &shopInfoResponse, nil
}

func createOrUpdateShop(db *gorm.DB, commonCode string, shopInfo *ShopInfoResponse) (*Shop, error) {
	shop := Shop{
		CommonCode: commonCode,
		Name:       shopInfo.Data.StoreName,
		Address:    shopInfo.Data.StoreAddress,
	}
	if err := db.Where(Shop{CommonCode: commonCode}).Assign(shop).FirstOrCreate(&shop).Error; err != nil {
		return nil, fmt.Errorf("failed to save shop to DB: %w", err)
	}
	return &shop, nil
}

func handleNonOperatingStatus(db *gorm.DB, shopID uint, status string) {
	snapshot := Snapshot{
		ShopID:     shopID,
		Timestamp:  time.Now(),
		ShopStatus: status,
	}
	if err := db.Create(&snapshot).Error; err != nil {
		log.Printf("Failed to save non-operating snapshot: %v", err)
	}
}

func getShopDetails(commonCode string) (*DetailResponse, error) {
	apiURL := "https://vip-gateway.wywk.cn/surf-internet/shop/v3/get"
	payload := map[string]string{"commonCode": commonCode}
	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var detailResponse DetailResponse
	if err := json.Unmarshal(body, &detailResponse); err != nil {
		return nil, fmt.Errorf("failed to parse detailed shop data JSON: %w", err)
	}

	if detailResponse.Code != 0 {
		return nil, fmt.Errorf("API returned error code %d: %v", detailResponse.Code, detailResponse.Message)
	}
	return &detailResponse, nil
}

func processShopData(detailResponse *DetailResponse) (int, int, map[string]map[string]int, map[string]string, map[int]RoomProperties) {
	totalDevices := 0
	usedDevices := 0
	roomStats := make(map[string]map[string]int)
	physicalRoomProperties := make(map[int]RoomProperties)
	relations := make(map[int][]int)
	roomCodeToName := make(map[string]string)

	for _, areaData := range detailResponse.Data.Areas {
		for _, element := range areaData.Elements {
			if element.ElementCode == "PRIVATE_ROOM" {
				physicalRoomProperties[element.ID] = RoomProperties{
					NoSmoking: element.NoSmokingFlag,
					Width:     element.Width,
					Height:    element.Height,
				}
			}
		}
		for _, relation := range areaData.Relations {
			relations[relation.ParentID] = relation.ChildID
		}
	}

	seatToRoomID := make(map[int]int)
	for roomID, seatIDs := range relations {
		for _, seatID := range seatIDs {
			seatToRoomID[seatID] = roomID
		}
	}

	for _, areaData := range detailResponse.Data.Areas {
		for _, element := range areaData.Elements {
			if element.ElementCode == "SEAT" && element.ClientInfo != nil {
				roomCode := element.ClientInfo.RoomCode
				roomName := element.ClientInfo.RoomName
				if _, ok := roomCodeToName[roomCode]; !ok {
					roomCodeToName[roomCode] = roomName
				}

				totalDevices++
				if _, ok := roomStats[roomCode]; !ok {
					roomStats[roomCode] = make(map[string]int)
					roomStats[roomCode]["total"] = 0
					roomStats[roomCode]["used"] = 0
					roomStats[roomCode]["roomID"] = seatToRoomID[element.ID]
				}
				roomStats[roomCode]["total"]++
				if element.ClientInfo.Status == 1 {
					usedDevices++
					roomStats[roomCode]["used"]++
				}
			}
		}
	}
	return totalDevices, usedDevices, roomStats, roomCodeToName, physicalRoomProperties
}

func saveShopData(db *gorm.DB, shop *Shop, totalDevices int, usedDevices int, roomStats map[string]map[string]int, roomCodeToName map[string]string, physicalRoomProperties map[int]RoomProperties) error {
	mainSnapshot := Snapshot{
		ShopID:     shop.ID,
		Timestamp:  time.Now(),
		ShopStatus: "营业中",
	}

	return db.Transaction(func(tx *gorm.DB) error {
		mainSnapshot.TotalDevices = totalDevices
		mainSnapshot.UsedDevices = usedDevices
		mainSnapshot.UsageRate = float64(usedDevices) / float64(totalDevices) * 100
		if err := tx.Create(&mainSnapshot).Error; err != nil {
			return fmt.Errorf("failed to save main snapshot: %w", err)
		}

		for roomCode, stats := range roomStats {
			roomName := roomCodeToName[roomCode]
			roomID := stats["roomID"]
			roomDetails, detailsFound := physicalRoomProperties[roomID]

			var existingRoom Room
			if err := tx.Where(Room{Code: roomCode}).FirstOrInit(&existingRoom).Error; err != nil {
				return fmt.Errorf("failed to find or init room %s: %w", roomName, err)
			}

			if detailsFound {
				existingRoom.NoSmoking = roomDetails.NoSmoking
				existingRoom.Width = roomDetails.Width
				existingRoom.Height = roomDetails.Height
			}
			existingRoom.ShopID = shop.ID
			existingRoom.Name = roomName
			existingRoom.TotalDevices = stats["total"]

			if err := tx.Save(&existingRoom).Error; err != nil {
				return fmt.Errorf("failed to save room %s to DB: %w", roomName, err)
			}

			roomSnapshot := RoomSnapshot{
				SnapshotID:   mainSnapshot.ID,
				RoomID:       existingRoom.ID,
				TotalDevices: stats["total"],
				UsedDevices:  stats["used"],
				UsageRate:    float64(stats["used"]) / float64(stats["total"]) * 100,
			}
			if err := tx.Create(&roomSnapshot).Error; err != nil {
				return fmt.Errorf("failed to save room snapshot for %s: %w", roomName, err)
			}
		}

		return nil
	})
}

func formatNotification(shop *Shop, totalDevices int, usedDevices int, roomStats map[string]map[string]int, roomCodeToName map[string]string) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf(`店名: %s
地址: %s
`, shop.Name, shop.Address))
	result.WriteString(fmt.Sprintf(`总设备: %d, 在用: %d
`, totalDevices, usedDevices))
	if totalDevices > 0 {
		usageRate := float64(usedDevices) / float64(totalDevices) * 100
		result.WriteString(fmt.Sprintf(`总使用率: %.2f%%

`, usageRate))
	}

	result.WriteString(`各房间使用率:
`)
	for roomCode, stats := range roomStats {
		if stats["total"] > 0 {
			roomName := roomCodeToName[roomCode]
			roomUsageRate := float64(stats["used"]) / float64(stats["total"]) * 100
			result.WriteString(fmt.Sprintf(`%s: %.2f%% (%d/%d)
`, roomName, roomUsageRate, stats["used"], stats["total"]))
		}
	}

	return result.String()
}
