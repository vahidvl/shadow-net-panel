package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mhsanaei/3x-ui/v3/database"
	"github.com/mhsanaei/3x-ui/v3/database/model"
	"github.com/mhsanaei/3x-ui/v3/web/service"
	"github.com/mhsanaei/3x-ui/v3/web/session"
	"github.com/mhsanaei/3x-ui/v3/xray"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

type BotController struct {
	inboundService service.InboundService
	settingService service.SettingService
	xrayService    service.XrayService
}

func NewBotController(g *gin.RouterGroup) *BotController {
	a := &BotController{}
	a.initRouter(g)
	return a
}

func (a *BotController) checkBotAPIAuth(c *gin.Context) {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "ShadowNet_Secret_2026" {
		c.Next()
		return
	}
	if session.IsLogin(c) {
		c.Next()
		return
	}
	c.AbortWithStatus(http.StatusForbidden)
}

func (a *BotController) initRouter(g *gin.RouterGroup) {
	g.GET("/violations", a.checkBotAPIAuth, a.getViolations)
	g.POST("/clients/reset_penalty", a.checkBotAPIAuth, a.resetPenalty)
	g.GET("/clients/info", a.checkBotAPIAuth, a.getClientInfo)
	g.POST("/clients/add", a.checkBotAPIAuth, a.addClient)
	g.GET("/qrcode", a.getQRCode)
	g.GET("/bot/config", a.getBotConfig)
	g.POST("/bot/settings/update", a.checkBotAPIAuth, a.updateBotSetting)
	g.POST("/bot/plans/manage", a.checkBotAPIAuth, a.managePlans)
	g.GET("/bot/session", a.getBotSession)
	g.POST("/bot/session", a.checkBotAPIAuth, a.updateBotSession)
	g.GET("/clients/details", a.checkBotAPIAuth, a.getClientDetails)
	g.POST("/clients/add_trial", a.checkBotAPIAuth, a.addTrialClient)
	g.POST("/transactions/create", a.checkBotAPIAuth, a.createTransaction)
	g.GET("/transactions/pending", a.checkBotAPIAuth, a.getPendingTransactions)
	g.POST("/transactions/action", a.checkBotAPIAuth, a.transactionAction)
}

func (a *BotController) getSubBaseURL(c *gin.Context) string {
	db := database.GetDB()
	var setting model.Setting
	if err := db.Where("key = ?", "subURI").First(&setting).Error; err == nil && setting.Value != "" {
		return strings.TrimSuffix(setting.Value, "/") + "/"
	}
	subDomain, _ := a.settingService.GetSubDomain()
	if subDomain == "" {
		subDomain, _ = a.settingService.GetWebDomain()
	}
	if subDomain == "" {
		subDomain = c.Request.Host
		if idx := strings.Index(subDomain, ":"); idx >= 0 {
			subDomain = subDomain[:idx]
		}
	}
	subPort, _ := a.settingService.GetSubPort()
	subPath, _ := a.settingService.GetSubPath()
	subKeyFile, _ := a.settingService.GetSubKeyFile()
	subCertFile, _ := a.settingService.GetSubCertFile()

	proto := "http"
	if subKeyFile != "" && subCertFile != "" {
		proto = "https"
	}

	baseURL := ""
	if (subPort == 443 && proto == "https") || (subPort == 80 && proto == "http") || subPort == 0 {
		baseURL = fmt.Sprintf("%s://%s", proto, subDomain)
	} else {
		baseURL = fmt.Sprintf("%s://%s:%d", proto, subDomain, subPort)
	}
	if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(subPath, "/") {
		baseURL += "/"
	}
	baseURL += subPath
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return baseURL
}

func formatBoolAsIntString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func (a *BotController) getViolations(c *gin.Context) {
	var violations []model.Violation
	db := database.GetDB()
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}
	if err := db.Order("timestamp desc").Limit(limit).Find(&violations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, violations)
}

func (a *BotController) resetPenalty(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}
	db := database.GetDB()
	var inbounds []model.Inbound
	if err := db.Find(&inbounds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	targetInboundID := 0
	var targetSettings map[string]any

	for _, inbound := range inbounds {
		var settings map[string]any
		if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
			continue
		}
		clients, ok := settings["clients"].([]any)
		if !ok {
			continue
		}
		found := false
		for i, client := range clients {
			clientMap, ok := client.(map[string]any)
			if !ok {
				continue
			}
			if clientMap["email"] == email {
				clientMap["penalty"] = 0
				clientMap["enable"] = true
				clients[i] = clientMap
				found = true
				break
			}
		}
		if found {
			settings["clients"] = clients
			targetInboundID = inbound.Id
			targetSettings = settings
			break
		}
	}

	if targetInboundID > 0 {
		settingsBytes, err := json.MarshalIndent(targetSettings, "", "  ")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := db.Model(&model.Inbound{}).Where("id = ?", targetInboundID).Update("settings", string(settingsBytes)).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		db.Model(&xray.ClientTraffic{}).Where("email = ?", email).Updates(map[string]any{
			"enable": true,
		})

		a.xrayService.SetToNeedRestart()
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": fmt.Sprintf("Penalty reset for %s", email)})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
}

func (a *BotController) getClientInfo(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}
	db := database.GetDB()
	var inbounds []model.Inbound
	if err := db.Find(&inbounds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, inbound := range inbounds {
		var settings map[string]any
		if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
			continue
		}
		clients, ok := settings["clients"].([]any)
		if !ok {
			continue
		}
		for _, client := range clients {
			clientMap, ok := client.(map[string]any)
			if !ok {
				continue
			}
			if clientMap["email"] == email {
				limitIp := 0
				if v, ok := clientMap["limitIp"].(float64); ok {
					limitIp = int(v)
				}
				totalGB := int64(0)
				if v, ok := clientMap["totalGB"].(float64); ok {
					totalGB = int64(v)
				}
				expiryTime := int64(0)
				if v, ok := clientMap["expiryTime"].(float64); ok {
					expiryTime = int64(v)
				}
				c.JSON(http.StatusOK, gin.H{
					"email":      email,
					"limitIp":    limitIp,
					"totalGB":    totalGB,
					"expiryTime": expiryTime,
				})
				return
			}
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
}

func (a *BotController) addClient(c *gin.Context) {
	email := c.Query("email")
	chat_id := c.Query("chat_id")
	inboundIdStr := c.DefaultQuery("inbound_id", "1")
	gbLimitStr := c.DefaultQuery("gb_limit", "50")
	daysLimitStr := c.DefaultQuery("days_limit", "30")

	inboundId, err := strconv.Atoi(inboundIdStr)
	if err != nil {
		inboundId = 1
	}
	gbLimit, err := strconv.Atoi(gbLimitStr)
	if err != nil {
		gbLimit = 50
	}
	daysLimit, err := strconv.Atoi(daysLimitStr)
	if err != nil {
		daysLimit = 30
	}

	db := database.GetDB()
	var inbounds []model.Inbound
	if err := db.Find(&inbounds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, inbound := range inbounds {
		var settings map[string]any
		if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
			continue
		}
		clients, ok := settings["clients"].([]any)
		if !ok {
			continue
		}
		for _, client := range clients {
			clientMap, ok := client.(map[string]any)
			if !ok {
				continue
			}
			if clientMap["email"] == email {
				uuidVal, _ := clientMap["id"].(string)
				subBase := a.getSubBaseURL(c)
				subURI := fmt.Sprintf("%s%s", subBase, uuidVal)

				c.JSON(http.StatusOK, gin.H{
					"status":   "success",
					"message":  "Client already exists.",
					"uuid":     uuidVal,
					"exists":   true,
					"sub_link": subURI,
				})
				return
			}
		}
	}

	clientUUID := uuid.NewString()
	expiryMs := int64(0)
	if daysLimit > 0 {
		expiryMs = int64((time.Now().Unix() + int64(daysLimit*86400)) * 1000)
	}
	totalBytes := int64(0)
	if gbLimit > 0 {
		totalBytes = int64(gbLimit) * 1024 * 1024 * 1024
	}

	newClient := model.Client{
		ID:         clientUUID,
		Email:      email,
		Enable:     true,
		ExpiryTime: expiryMs,
		TotalGB:    totalBytes,
		Flow:       "xtls-rprx-vision",
		LimitIP:    2,
		SubID:      uuid.NewString(),
	}

	clientsList := []model.Client{newClient}
	settingsMap := map[string]any{
		"clients": clientsList,
	}
	settingsBytes, err := json.Marshal(settingsMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	inboundPayload := &model.Inbound{
		Id:       inboundId,
		Settings: string(settingsBytes),
	}

	needRestart, err := a.inboundService.AddInboundClient(inboundPayload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if needRestart {
		a.xrayService.SetToNeedRestart()
	}

	subBase := a.getSubBaseURL(c)
	subURI := fmt.Sprintf("%s%s", subBase, clientUUID)

	if chat_id != "" {
		go func() {
			cfURL := "https://shadow-net-bot.prem-ir.workers.dev/deliver"
			payload := map[string]string{
				"chat_id": chat_id,
				"uuid":    clientUUID,
			}
			bodyBytes, _ := json.Marshal(payload)
			resp, err := http.Post(cfURL, "application/json", bytes.NewBuffer(bodyBytes))
			if err == nil {
				resp.Body.Close()
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"message":  fmt.Sprintf("Client %s created.", email),
		"uuid":     clientUUID,
		"sub_link": subURI,
	})
}

func (a *BotController) getQRCode(c *gin.Context) {
	data := c.Query("data")
	if data == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "data is required"})
		return
	}
	png, err := qrcode.Encode(data, qrcode.Medium, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "image/png", png)
}

func (a *BotController) getBotConfig(c *gin.Context) {
	enableTrial, _ := a.settingService.GetSnBotEnableTrial()
	trialVolumeMb, _ := a.settingService.GetSnBotTrialVolumeMb()
	trialExpiryHours, _ := a.settingService.GetSnBotTrialExpiryHours()
	trialIpLimit, _ := a.settingService.GetSnBotTrialIpLimit()
	enableZibal, _ := a.settingService.GetSnBotEnableZibal()
	enableCardToCard, _ := a.settingService.GetSnBotEnableCardToCard()
	supportEnabled, _ := a.settingService.GetSnBotSupportEnabled()
	cardNumber, _ := a.settingService.GetSnBotCardNumber()
	cardOwner, _ := a.settingService.GetSnBotCardOwner()

	var plans []model.BotPlan
	db := database.GetDB()
	if err := db.Where("enabled = 1").Find(&plans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	settings := map[string]string{
		"enable_trial":        formatBoolAsIntString(enableTrial),
		"trial_volume_mb":    strconv.Itoa(trialVolumeMb),
		"trial_expiry_hours": strconv.Itoa(trialExpiryHours),
		"trial_ip_limit":     strconv.Itoa(trialIpLimit),
		"enable_zibal":      formatBoolAsIntString(enableZibal),
		"enable_card_to_card": formatBoolAsIntString(enableCardToCard),
		"support_enabled":   formatBoolAsIntString(supportEnabled),
		"card_number":       cardNumber,
		"card_owner":        cardOwner,
	}

	type PlanResp struct {
		Id         int    `json:"id"`
		Name       string `json:"name"`
		VolumeGb   int    `json:"volume_gb"`
		Days       int    `json:"days"`
		PriceToman int    `json:"price_toman"`
	}
	plansResp := make([]PlanResp, 0, len(plans))
	for _, p := range plans {
		plansResp = append(plansResp, PlanResp{
			Id:         p.Id,
			Name:       p.Name,
			VolumeGb:   p.VolumeGb,
			Days:       p.Days,
			PriceToman: p.PriceToman,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"settings": settings,
		"plans":    plansResp,
	})
}

func (a *BotController) updateBotSetting(c *gin.Context) {
	type Payload struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	var req Payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	var setting model.Setting
	err := db.Where("key = ?", req.Key).First(&setting).Error
	switch err {
	case gorm.ErrRecordNotFound:
		setting = model.Setting{Key: req.Key, Value: req.Value}
		if err := db.Create(&setting).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case nil:
		setting.Value = req.Value
		if err := db.Save(&setting).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": fmt.Sprintf("Setting '%s' updated.", req.Key)})
}

func (a *BotController) managePlans(c *gin.Context) {
	type Payload struct {
		Id         int    `json:"id"`
		Name       string `json:"name"`
		VolumeGb   int    `json:"volume_gb"`
		Days       int    `json:"days"`
		PriceToman int    `json:"price_toman"`
		Enabled    int    `json:"enabled"`
		Action     string `json:"action"`
	}
	var req Payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	switch req.Action {
	case "add":
		plan := model.BotPlan{
			Name:       req.Name,
			VolumeGb:   req.VolumeGb,
			Days:       req.Days,
			PriceToman: req.PriceToman,
			Enabled:    1,
		}
		if err := db.Create(&plan).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "edit":
		var plan model.BotPlan
		if err := db.First(&plan, req.Id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
			return
		}
		plan.Name = req.Name
		plan.VolumeGb = req.VolumeGb
		plan.Days = req.Days
		plan.PriceToman = req.PriceToman
		plan.Enabled = req.Enabled
		if err := db.Save(&plan).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "delete":
		if err := db.Delete(&model.BotPlan{}, req.Id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "toggle":
		var plan model.BotPlan
		if err := db.First(&plan, req.Id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
			return
		}
		if plan.Enabled == 1 {
			plan.Enabled = 0
		} else {
			plan.Enabled = 1
		}
		if err := db.Save(&plan).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": fmt.Sprintf("Plan action '%s' executed.", req.Action)})
}

func (a *BotController) getBotSession(c *gin.Context) {
	chatIdStr := c.Query("chat_id")
	if chatIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat_id is required"})
		return
	}
	chatId, err := strconv.ParseInt(chatIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat_id"})
		return
	}

	db := database.GetDB()
	var session model.BotSession
	err = db.Where("chat_id = ?", chatId).First(&session).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"chat_id": chatId,
			"state":   "start",
			"data":    gin.H{},
		})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var sessionData map[string]any
	if session.Data != "" {
		json.Unmarshal([]byte(session.Data), &sessionData)
	}
	if sessionData == nil {
		sessionData = map[string]any{}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"chat_id": chatId,
		"state":   session.State,
		"data":    sessionData,
	})
}

func (a *BotController) updateBotSession(c *gin.Context) {
	type Payload struct {
		ChatID int64          `json:"chat_id"`
		State  string         `json:"state"`
		Data   map[string]any `json:"data"`
	}
	var req Payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dataBytes, _ := json.Marshal(req.Data)
	db := database.GetDB()

	var session model.BotSession
	err := db.Where("chat_id = ?", req.ChatID).First(&session).Error
	switch err {
	case gorm.ErrRecordNotFound:
		session = model.BotSession{
			ChatId: req.ChatID,
			State:  req.State,
			Data:   string(dataBytes),
		}
		if err := db.Create(&session).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case nil:
		session.State = req.State
		session.Data = string(dataBytes)
		if err := db.Save(&session).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Session updated."})
}

func (a *BotController) getClientDetails(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}

	db := database.GetDB()

	var traffic xray.ClientTraffic
	hasTraffic := true
	if err := db.Where("email = ?", email).First(&traffic).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			hasTraffic = false
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	var inbounds []model.Inbound
	if err := db.Find(&inbounds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	uuidVal := ""
	limitIp := 0
	foundClient := false

	for _, inbound := range inbounds {
		var settings map[string]any
		if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
			continue
		}
		clients, ok := settings["clients"].([]any)
		if !ok {
			continue
		}
		for _, client := range clients {
			clientMap, ok := client.(map[string]any)
			if !ok {
				continue
			}
			if clientMap["email"] == email {
				uuidVal, _ = clientMap["id"].(string)
				if v, ok := clientMap["limitIp"].(float64); ok {
					limitIp = int(v)
				}
				foundClient = true
				break
			}
		}
		if foundClient {
			break
		}
	}

	if !hasTraffic && !foundClient {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	}

	enable := true
	up := int64(0)
	down := int64(0)
	total := int64(0)
	expiryTime := int64(0)

	if hasTraffic {
		enable = traffic.Enable
		up = traffic.Up
		down = traffic.Down
		total = traffic.Total
		expiryTime = traffic.ExpiryTime
	}

	subURI := ""
	if uuidVal != "" {
		subBase := a.getSubBaseURL(c)
		subURI = fmt.Sprintf("%s%s", subBase, uuidVal)
	}

	c.JSON(http.StatusOK, gin.H{
		"email":       email,
		"uuid":        uuidVal,
		"enable":      enable,
		"up":          up,
		"down":        down,
		"total":       total,
		"expiry_time": expiryTime,
		"limit_ip":    limitIp,
		"sub_link":    subURI,
	})
}

func (a *BotController) addTrialClient(c *gin.Context) {
	type Payload struct {
		ChatID   int64  `json:"chat_id"`
		Username string `json:"username"`
	}
	var req Payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := fmt.Sprintf("shawn_user_%d", req.ChatID)
	db := database.GetDB()

	var existsTrial model.TrialLog
	if err := db.Where("chat_id = ?", req.ChatID).First(&existsTrial).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "denied",
			"message": "شما قبلاً اکانت تست دریافت کرده‌اید و مجاز به دریافت مجدد نیستید.",
		})
		return
	}

	var count int64
	db.Model(&xray.ClientTraffic{}).Where("email = ?", email).Count(&count)
	if count > 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":  "denied",
			"message": "شما در حال حاضر دارای یک اشتراک فعال یا قبلی هستید.",
		})
		return
	}

	enableTrial, _ := a.settingService.GetSnBotEnableTrial()
	if !enableTrial {
		c.JSON(http.StatusOK, gin.H{
			"status":  "disabled",
			"message": "در حال حاضر دریافت اکانت تست رایگان از طرف مدیریت غیرفعال شده است.",
		})
		return
	}

	volMb, _ := a.settingService.GetSnBotTrialVolumeMb()
	expiryHours, _ := a.settingService.GetSnBotTrialExpiryHours()
	ipLimit, _ := a.settingService.GetSnBotTrialIpLimit()

	if volMb == 0 {
		volMb = 1024
	}
	if expiryHours == 0 {
		expiryHours = 24
	}
	if ipLimit == 0 {
		ipLimit = 2
	}

	trialLog := &model.TrialLog{
		ChatId:   req.ChatID,
		Username: req.Username,
	}
	if err := db.Create(trialLog).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	clientUUID := uuid.NewString()
	expiryMs := int64((time.Now().Unix() + int64(expiryHours*3600)) * 1000)
	totalBytes := int64(volMb) * 1024 * 1024

	client := model.Client{
		ID:         clientUUID,
		Email:      email,
		Enable:     true,
		ExpiryTime: expiryMs,
		TotalGB:    totalBytes,
		Flow:       "xtls-rprx-vision",
		LimitIP:    ipLimit,
		SubID:      uuid.NewString(),
	}

	clientsList := []model.Client{client}
	settingsMap := map[string]any{
		"clients": clientsList,
	}
	settingsBytes, err := json.Marshal(settingsMap)
	if err != nil {
		db.Delete(&model.TrialLog{}, req.ChatID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	inboundPayload := &model.Inbound{
		Id:       2,
		Settings: string(settingsBytes),
	}

	needRestart, err := a.inboundService.AddInboundClient(inboundPayload)
	if err != nil {
		db.Delete(&model.TrialLog{}, req.ChatID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if needRestart {
		a.xrayService.SetToNeedRestart()
	}

	subBase := a.getSubBaseURL(c)
	subURI := fmt.Sprintf("%s%s", subBase, clientUUID)

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"message":  "اکانت تست ۱ روزه با موفقیت صادر شد.",
		"uuid":     clientUUID,
		"sub_link": subURI,
	})
}

func (a *BotController) createTransaction(c *gin.Context) {
	type Payload struct {
		ChatID      int64  `json:"chat_id"`
		Username    string `json:"username"`
		PlanID      int    `json:"plan_id"`
		Amount      int    `json:"amount"`
		CardDetails string `json:"card_details"`
	}
	var req Payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	tx := model.PendingTransaction{
		ChatId:      req.ChatID,
		Username:    req.Username,
		PlanId:      req.PlanID,
		Amount:      req.Amount,
		CardDetails: req.CardDetails,
		Status:      "pending",
	}

	if err := db.Create(&tx).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "transaction_id": tx.Id})
}

func (a *BotController) getPendingTransactions(c *gin.Context) {
	var txs []model.PendingTransaction
	db := database.GetDB()
	if err := db.Where("status = ?", "pending").Find(&txs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type TxResp struct {
		Id          int    `json:"id"`
		ChatId      int64  `json:"chat_id"`
		Username    string `json:"username"`
		PlanId      int    `json:"plan_id"`
		Amount      int    `json:"amount"`
		CardDetails string `json:"card_details"`
		Status      string `json:"status"`
		CreatedAt   int64  `json:"created_at"`
	}
	resps := make([]TxResp, 0, len(txs))
	for _, t := range txs {
		resps = append(resps, TxResp{
			Id:          t.Id,
			ChatId:      t.ChatId,
			Username:    t.Username,
			PlanId:      t.PlanId,
			Amount:      t.Amount,
			CardDetails: t.CardDetails,
			Status:      t.Status,
			CreatedAt:   t.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, resps)
}

func (a *BotController) transactionAction(c *gin.Context) {
	type Payload struct {
		TransactionID int    `json:"transaction_id"`
		Action        string `json:"action"`
	}
	var req Payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	var tx model.PendingTransaction
	if err := db.First(&tx, req.TransactionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	if tx.Status != "pending" {
		c.JSON(http.StatusOK, gin.H{"status": "error", "message": "Transaction is already processed."})
		return
	}

	if req.Action == "reject" {
		tx.Status = "rejected"
		if err := db.Save(&tx).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Transaction rejected",
			"chat_id": tx.ChatId,
			"action":  "reject",
		})
		return
	}

	var plan model.BotPlan
	if err := db.First(&plan, tx.PlanId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	tx.Status = "approved"
	if err := db.Save(&tx).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	email := fmt.Sprintf("shawn_user_%d", tx.ChatId)
	clientUUID := uuid.NewString()
	expiryMs := int64(0)
	if plan.Days > 0 {
		expiryMs = int64((time.Now().Unix() + int64(plan.Days*86400)) * 1000)
	}
	totalBytes := int64(0)
	if plan.VolumeGb > 0 {
		totalBytes = int64(plan.VolumeGb) * 1024 * 1024 * 1024
	}

	client := model.Client{
		ID:         clientUUID,
		Email:      email,
		Enable:     true,
		ExpiryTime: expiryMs,
		TotalGB:    totalBytes,
		Flow:       "xtls-rprx-vision",
		LimitIP:    2,
		SubID:      uuid.NewString(),
	}

	clientsList := []model.Client{client}
	settingsMap := map[string]any{
		"clients": clientsList,
	}
	settingsBytes, err := json.Marshal(settingsMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	inboundPayload := &model.Inbound{
		Id:       2,
		Settings: string(settingsBytes),
	}

	needRestart, err := a.inboundService.AddInboundClient(inboundPayload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if needRestart {
		a.xrayService.SetToNeedRestart()
	}

	subBase := a.getSubBaseURL(c)
	subURI := fmt.Sprintf("%s%s", subBase, clientUUID)

	if tx.ChatId > 0 {
		go func() {
			cfURL := "https://shadow-net-bot.prem-ir.workers.dev/deliver"
			payload := map[string]string{
				"chat_id": strconv.FormatInt(tx.ChatId, 10),
				"uuid":    clientUUID,
			}
			bodyBytes, _ := json.Marshal(payload)
			resp, err := http.Post(cfURL, "application/json", bytes.NewBuffer(bodyBytes))
			if err == nil {
				resp.Body.Close()
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"message":  "Transaction approved and client created.",
		"chat_id":  tx.ChatId,
		"uuid":     clientUUID,
		"sub_link": subURI,
		"action":   "approve",
	})
}
