package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v3/config"
	"github.com/mhsanaei/3x-ui/v3/logger"
	"github.com/mhsanaei/3x-ui/v3/util/json_util"
	"github.com/mhsanaei/3x-ui/v3/xray"
)

type ProxyBridgeService struct {
	process    *xray.Process
	mu         sync.Mutex
	lastURI    string
	configName string
	port       int // SOCKS5 listen port; defaults to 10811 for global bridge
}

var globalProxyBridge = &ProxyBridgeService{port: 10811}

func GetProxyBridgeService() *ProxyBridgeService {
	return globalProxyBridge
}

// NewTestBridge returns an isolated, ephemeral bridge on port 10812.
// It is used exclusively by TestProxy and never affects global status.
func NewTestBridge() *ProxyBridgeService {
	return &ProxyBridgeService{port: 10812}
}

func (s *ProxyBridgeService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.process != nil && s.process.IsRunning() {
		logger.Info("Stopping proxy bridge...")
		s.process.Stop()
	}
	s.process = nil
	s.lastURI = ""
	s.configName = "" // clear so GetStatus doesn't report stale name
}

func (s *ProxyBridgeService) GetBridgePort() int {
	if s.port == 0 {
		return 10811
	}
	return s.port
}

func (s *ProxyBridgeService) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	running := s.process != nil && s.process.IsRunning()
	status := "Stopped"
	errStr := ""
	configName := ""
	if running {
		status = "Running"
		configName = s.configName // only expose name when truly running
	} else if s.process != nil {
		status = "Error"
		if err := s.process.GetErr(); err != nil {
			errStr = err.Error()
		}
	}

	return map[string]interface{}{
		"running":    running,
		"status":     status,
		"configName": configName,
		"error":      errStr,
	}
}

func (s *ProxyBridgeService) Start(uri string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if uri == "" {
		if s.process != nil {
			s.process.Stop()
			s.process = nil
		}
		s.lastURI = ""
		return nil
	}

	if s.lastURI == uri && s.process != nil && s.process.IsRunning() {
		return nil
	}

	if s.process != nil {
		logger.Info("Restarting proxy bridge due to URI change...")
		s.process.Stop()
	}

	logger.Infof("Starting proxy bridge for URI: %s", uri)

	outbound, err := s.parseURI(uri)
	if err != nil {
		return fmt.Errorf("parse URI failed: %w", err)
	}

	// Extract config name from fragment
	s.configName = ""
	if u, err := url.Parse(uri); err == nil {
		s.configName = u.Fragment
		if s.configName == "" {
			s.configName = u.Hostname()
		}
	}

	outboundJSON, _ := json.Marshal([]interface{}{
		outbound,
		map[string]interface{}{
			"protocol": "freedom",
			"tag":      "direct",
		},
	})
	
	logger.Debugf("Generated outbound config: %s", string(outboundJSON))

	inbound := xray.InboundConfig{
		Listen:   json_util.RawMessage(`"127.0.0.1"`),
		Port:     s.GetBridgePort(),
		Protocol: "socks",
		Settings: json_util.RawMessage(`{"auth": "noauth", "udp": true}`),
	}

	cfg := &xray.Config{
		LogConfig:       json_util.RawMessage(`{"loglevel": "debug"}`),
		InboundConfigs:  []xray.InboundConfig{inbound},
		OutboundConfigs: json_util.RawMessage(outboundJSON),
	}

	tempConfigPath := filepath.Join(config.GetBinFolderPath(), fmt.Sprintf("panel_bridge_%d.json", s.GetBridgePort()))
	s.process = xray.NewTestProcess(cfg, tempConfigPath)
	
	err = s.process.Start()
	if err != nil {
		return fmt.Errorf("start bridge xray failed: %w", err)
	}

	// Wait a bit to ensure it's actually running and listening
	time.Sleep(1000 * time.Millisecond)
	if !s.process.IsRunning() {
		exitErr := s.process.GetErr()
		result := s.process.GetResult()
		return fmt.Errorf("proxy bridge exited immediately. Error: %v, Logs: %s", exitErr, result)
	}

	logger.Infof("Proxy bridge started successfully. Last log: %s", s.process.GetResult())
	s.lastURI = uri
	return nil
}

func (s *ProxyBridgeService) parseURI(uri string) (map[string]interface{}, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	protocol := u.Scheme
	switch protocol {
	case "vless":
		return s.parseVless(u)
	case "vmess":
		return s.parseVmess(u)
	case "trojan":
		return s.parseTrojan(u)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

func (s *ProxyBridgeService) parseVless(u *url.URL) (map[string]interface{}, error) {
	uuid := u.User.Username()
	host := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		portStr = "443"
	}
	port, _ := strconv.Atoi(portStr)
	q := u.Query()

	encryption := q.Get("encryption")
	if encryption == "" {
		encryption = "none"
	}

	outbound := map[string]interface{}{
		"protocol": "vless",
		"settings": map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{
					"address": host,
					"port":    port,
					"users": []interface{}{
						map[string]interface{}{
							"id":         uuid,
							"encryption": encryption,
							"flow":       q.Get("flow"),
							"level":      0,
						},
					},
				},
			},
		},
	}

	s.applyStreamSettings(u, outbound)
	return outbound, nil
}

func (s *ProxyBridgeService) parseTrojan(u *url.URL) (map[string]interface{}, error) {
	password := u.User.Username()
	host := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		portStr = "443"
	}
	port, _ := strconv.Atoi(portStr)

	outbound := map[string]interface{}{
		"protocol": "trojan",
		"settings": map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{
					"address":  host,
					"port":     port,
					"password": password,
					"level":    0,
				},
			},
		},
	}

	s.applyStreamSettings(u, outbound)
	return outbound, nil
}

func (s *ProxyBridgeService) parseVmess(u *url.URL) (map[string]interface{}, error) {
	raw := u.Host + u.Path
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid vmess base64: %w", err)
	}

	var v map[string]interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("invalid vmess json: %w", err)
	}

	outbound := map[string]interface{}{
		"protocol": "vmess",
		"settings": map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{
					"address": v["add"],
					"port":    v["port"],
					"users": []interface{}{
						map[string]interface{}{
							"id":       v["id"],
							"alterId":  0,
							"security": v["scy"],
							"level":    0,
						},
					},
				},
			},
		},
	}

	stream := map[string]interface{}{
		"network": v["net"],
		"security": v["tls"],
	}

	if v["tls"] == "tls" {
		stream["tlsSettings"] = map[string]interface{}{
			"serverName": v["sni"],
			"allowInsecure": false,
		}
	}

	network, _ := v["net"].(string)
	switch network {
	case "ws":
		stream["wsSettings"] = map[string]interface{}{
			"path": v["path"],
			"headers": map[string]interface{}{
				"Host": v["host"],
			},
		}
	case "grpc":
		stream["grpcSettings"] = map[string]interface{}{
			"serviceName": v["path"],
		}
	}

	outbound["streamSettings"] = stream
	return outbound, nil
}

func (s *ProxyBridgeService) applyStreamSettings(u *url.URL, outbound map[string]interface{}) {
	q := u.Query()
	network := q.Get("type")
	if network == "" {
		network = "tcp"
	}
	security := q.Get("security")

	stream := map[string]interface{}{
		"network":  network,
		"security": security,
	}

	switch security {
	case "tls":
		stream["tlsSettings"] = map[string]interface{}{
			"serverName":    q.Get("sni"),
			"allowInsecure": q.Get("insecure") == "1",
			"alpn":          strings.Split(q.Get("alpn"), ","),
			"fingerprint":   q.Get("fp"),
		}
	case "reality":
		spx := q.Get("spx")
		if spx == "" {
			spx = "/"
		}
		stream["realitySettings"] = map[string]interface{}{
			"serverName":   q.Get("sni"),
			"fingerprint":  q.Get("fp"),
			"publicKey":    q.Get("pbk"),
			"shortId":      q.Get("sid"),
			"spiderX":      spx,
		}
	}

	switch network {
	case "xhttp":
		extraStr := q.Get("extra")
		var extra interface{}
		if extraStr != "" {
			json.Unmarshal([]byte(extraStr), &extra)
		}
		stream["splithttpSettings"] = map[string]interface{}{
			"path":  q.Get("path"),
			"host":  q.Get("host"),
			"mode":  q.Get("mode"),
			"extra": extra,
		}
	case "ws":
		stream["wsSettings"] = map[string]interface{}{
			"path": q.Get("path"),
			"headers": map[string]interface{}{
				"Host": q.Get("host"),
			},
		}
	case "grpc":
		stream["grpcSettings"] = map[string]interface{}{
			"serviceName": q.Get("serviceName"),
			"multiMode":   q.Get("mode") == "multi",
		}
	case "tcp":
		if q.Get("headerType") == "http" {
			stream["tcpSettings"] = map[string]interface{}{
				"header": map[string]interface{}{
					"type": "http",
					"request": map[string]interface{}{
						"path": []string{q.Get("path")},
						"headers": map[string]interface{}{
							"Host": []string{q.Get("host")},
						},
					},
				},
			}
		}
	}

	outbound["streamSettings"] = stream
}
