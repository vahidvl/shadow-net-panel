package service

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/logger"
)

func getHttpClient(timeout time.Duration) *http.Client {
	settingService := SettingService{}
	enable, _ := settingService.GetSnPanelProxyEnable()
	proxyURL, _ := settingService.GetSnPanelProxyURL()

	logger.Infof("getHttpClient: proxy enabled=%v, url=%s", enable, proxyURL)

	transport := &http.Transport{}
	if enable && proxyURL != "" {
		isURI := strings.HasPrefix(proxyURL, "vless://") ||
			strings.HasPrefix(proxyURL, "vmess://") ||
			strings.HasPrefix(proxyURL, "trojan://")

		if isURI {
			logger.Infof("getHttpClient: using URI bridge for %s", proxyURL)
			bridge := GetProxyBridgeService()
			err := bridge.Start(proxyURL)
			if err != nil {
				logger.Warning("start proxy bridge failed:", err)
				transport.Proxy = http.ProxyFromEnvironment
			} else {
				u, _ := url.Parse(fmt.Sprintf("socks5://127.0.0.1:%d", bridge.GetBridgePort()))
				transport.Proxy = http.ProxyURL(u)
				logger.Infof("getHttpClient: proxy set to %s", u.String())
			}
		} else if u, err := url.Parse(proxyURL); err == nil {
			logger.Infof("getHttpClient: using direct proxy %s", proxyURL)
			transport.Proxy = http.ProxyURL(u)
		} else {
			logger.Warning("invalid panel proxy URL:", err)
			transport.Proxy = http.ProxyFromEnvironment
		}
	} else {
		logger.Infof("getHttpClient: proxy disabled, using environment proxy")
		// Stop bridge if disabled
		GetProxyBridgeService().Stop()
		transport.Proxy = http.ProxyFromEnvironment
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
