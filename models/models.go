package models

import (
	"time"
)

// region GORM Models
type Shop struct {
	ID         uint   `gorm:"primaryKey"`
	CommonCode string `gorm:"uniqueIndex"`
	Name       string
	Address    string
	Snapshots  []Snapshot `gorm:"foreignKey:ShopID"`
	Rooms      []Room     `gorm:"foreignKey:ShopID"`
}

type Room struct {
	ID           uint `gorm:"primaryKey"`
	ShopID       uint
	Code         string `gorm:"uniqueIndex"`
	Name         string
	TotalDevices int
	NoSmoking    int
	Width        float64        // Changed to float64 to match JSON
	Height       float64        // Changed to float64 to match JSON
	Snapshots    []RoomSnapshot `gorm:"foreignKey:RoomID"`
}

type Snapshot struct {
	ID            uint      `gorm:"primaryKey"`
	ShopID        uint      `gorm:"index"`
	Timestamp     time.Time `gorm:"index"`
	ShopStatus    string
	TotalDevices  int
	UsedDevices   int
	UsageRate     float64        // New field for overall usage rate
	RoomSnapshots []RoomSnapshot `gorm:"foreignKey:SnapshotID"`
}

type RoomSnapshot struct {
	ID           uint `gorm:"primaryKey"`
	SnapshotID   uint `gorm:"index"`
	RoomID       uint `gorm:"index"`
	TotalDevices int
	UsedDevices  int
	UsageRate    float64 // New field for room usage rate
}

// endregion

// region API Response Structs
// Detailed shop data from the surf-internet API
type DetailResponse struct {
	Code    int         `json:"code"`
	Message interface{} `json:"message"`
	Data    DetailData  `json:"data"`
}

type DetailData struct {
	CommonCode   string      `json:"commonCode"`
	ShopName     interface{} `json:"shopName"`
	CanvasStatus int         `json:"canvasStatus"`
	CloseWords   interface{} `json:"closeWords"`
	CloseTips    interface{} `json:"closeTips"`
	Areas        []Area      `json:"areas"`
}

type Area struct {
	ID        int        `json:"id"`
	AreaCode  string     `json:"areaCode"`
	AreaName  string     `json:"areaName"`
	Direction string     `json:"direction"`
	Elements  []Element  `json:"elements"`
	Relations []Relation `json:"relations"`
}

type Relation struct {
	ParentID int   `json:"roomId"`
	ChildID  []int `json:"seatIds"`
}

type Element struct {
	ID                   int         `json:"id"`
	ElementCode          string      `json:"elementCode"` // "SEAT", "PRIVATE_ROOM", etc.
	ElementType          interface{} `json:"elementType"`
	DisplayName          string      `json:"displayName"`
	PointX               float64     `json:"pointX"`
	PointY               float64     `json:"pointY"`
	Width                float64     `json:"width"`
	Height               float64     `json:"height"`
	Rotate               float64     `json:"rotate"`
	BorderTop            int         `json:"borderTop"`
	BorderRight          int         `json:"borderRight"`
	BorderBottom         int         `json:"borderBottom"`
	BorderLeft           int         `json:"borderLeft"`
	RefEntityNo          interface{} `json:"refEntityNo"`
	NoSmokingFlag        int         `json:"noSmokingFlag"`
	BrokenFlag           interface{} `json:"brokenFlag"`
	BrokenReason         interface{} `json:"brokenReason"`
	BorderTopSite        int         `json:"borderTopSite"`
	BorderRightSite      int         `json:"borderRightSite"`
	BorderBottomSite     int         `json:"borderBottomSite"`
	BorderLeftSite       int         `json:"borderLeftSite"`
	ClientInfo           *ClientInfo `json:"clientInfo"`
	RoomFlag             int         `json:"roomFlag"`
	I18nLanguageMetaList interface{} `json:"i18nLanguageMetaList"`
}

type ClientInfo struct {
	RoomCode    string `json:"roomCode"`
	RoomName    string `json:"roomName"`
	ClientIp    string `json:"clientIp"`
	ClientNo    string `json:"clientNo"`
	DisplayName string `json:"displayName"`
	Status      int    `json:"status"` // 1 for used, 0 for available
}

// Basic shop info from the asset-svc API
type ShopInfoResponse struct {
	Data ShopInfoData `json:"data"`
}

type ShopInfoData struct {
	StoreName    string `json:"storeName"`
	StoreAddress string `json:"storeAddress"`
	ShopStatus   string `json:"shopStatus"`
}

//endregion
