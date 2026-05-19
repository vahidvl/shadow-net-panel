package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mhsanaei/3x-ui/v3/database"
	"github.com/mhsanaei/3x-ui/v3/database/model"
	"github.com/mhsanaei/3x-ui/v3/logger"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type ShadowNetService struct {
	settingService SettingService
	inboundService InboundService

	salesBot    *telego.Bot
	sentinelBot *telego.Bot
	adminBot    *telego.Bot

	mu sync.Mutex
}

func (s *ShadowNetService) initBot(token string) (*telego.Bot, error) {
	if token == "" {
		return nil, fmt.Errorf("bot token is empty")
	}
	return telego.NewBot(token)
}

func (s *ShadowNetService) GetSalesBot() (*telego.Bot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.salesBot != nil {
		return s.salesBot, nil
	}
	token, _ := s.settingService.getString("snBotTokenSales")
	bot, err := s.initBot(token)
	if err != nil {
		return nil, err
	}
	s.salesBot = bot
	return s.salesBot, nil
}

func (s *ShadowNetService) GetSentinelBot() (*telego.Bot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sentinelBot != nil {
		return s.sentinelBot, nil
	}
	token, _ := s.settingService.getString("snBotTokenSentinel")
	bot, err := s.initBot(token)
	if err != nil {
		return nil, err
	}
	s.sentinelBot = bot
	return s.sentinelBot, nil
}

func (s *ShadowNetService) GetAdminBot() (*telego.Bot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.adminBot != nil {
		return s.adminBot, nil
	}
	token, _ := s.settingService.getString("snBotTokenAdmin")
	bot, err := s.initBot(token)
	if err != nil {
		return nil, err
	}
	s.adminBot = bot
	return s.adminBot, nil
}

func (s *ShadowNetService) NotifyPenalty(clientEmail string, penaltyCount int, maxPenalty int, ip string) {
	adminChatId, _ := s.settingService.getString("snAdminChatId")
	
	// 1. Notify Admin via Admin Bot
	if adminBot, err := s.GetAdminBot(); err == nil && adminChatId != "" {
		var id int64
		fmt.Sscanf(adminChatId, "%d", &id)
		
		msg := fmt.Sprintf("⚠️ <b>Penalty Issued</b>\n\n<b>Client:</b> %s\n<b>IP:</b> %s\n<b>Strikes:</b> %d/%d", 
			clientEmail, ip, penaltyCount, maxPenalty)
		if penaltyCount >= maxPenalty {
			msg += "\n\n🚫 <b>Account Disabled Automatically</b>"
		}
		
		adminBot.SendMessage(context.Background(), tu.Message(tu.ID(id), msg).WithParseMode(telego.ModeHTML))
	}

	logger.Infof("Shadow-Net Penalty: %s (IP: %s) -> %d/%d", clientEmail, ip, penaltyCount, maxPenalty)
	
	if penaltyCount >= maxPenalty {
		logger.Warningf("Shadow-Net: Banning client %s due to max penalties", clientEmail)
	}
}

func (s *ShadowNetService) CheckAndApplyPenalty(clientEmail string, ip string) (int, error) {
	db := database.GetDB()
	
	// We need to find the inbound containing this client to update the settings JSON
	inbound := &model.Inbound{}
	err := db.Model(&model.Inbound{}).Where("settings LIKE ?", "%"+clientEmail+"%").First(inbound).Error
	if err != nil {
		return 0, err
	}

	settings := map[string][]model.Client{}
	json.Unmarshal([]byte(inbound.Settings), &settings)
	clients := settings["clients"]

	newPenalty := 0
	maxPenalty, _ := s.settingService.getInt("snMaxPenalty")
	if maxPenalty <= 0 {
		maxPenalty = 3
	}

	clientFound := false
	for i := range clients {
		if clients[i].Email == clientEmail {
			clients[i].Penalty++
			newPenalty = clients[i].Penalty
			if newPenalty >= maxPenalty {
				clients[i].Enable = false
			}
			clientFound = true
			break
		}
	}

	if !clientFound {
		return 0, fmt.Errorf("client not found in inbound settings")
	}

	// Save back to inbound
	newSettings, _ := json.Marshal(settings)
	inbound.Settings = string(newSettings)
	
	err = db.Save(inbound).Error
	if err != nil {
		return 0, err
	}

	// Trigger notification
	go s.NotifyPenalty(clientEmail, newPenalty, maxPenalty, ip)

	return newPenalty, nil
}
