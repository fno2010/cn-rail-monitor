package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	stationNamesURL  = "https://kyfw.12306.cn/otn/resources/js/framework/station_name.js"
	stationCacheFile = "station_codes.json"
)

type StationCode struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Pinyin    string `json:"pinyin"`
	ShortName string `json:"short_name"`
}

type StationCodes struct {
	Codes   []StationCode `json:"codes"`
	Updated time.Time     `json:"updated"`
}

var (
	stationCache       *StationCodes
	stationCacheMutex  sync.RWMutex
	stationCodesLoaded bool
	customCachePath    string
)

func SetStationCachePath(path string) {
	customCachePath = path
}

func GetStationCodesPath() string {
	if customCachePath != "" {
		return customCachePath
	}
	execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil || execDir == "" {
		execDir, _ = os.Getwd()
	}
	return filepath.Join(execDir, stationCacheFile)
}

func EnsureStationCodesLoaded() {
	if stationCodesLoaded {
		return
	}
	LoadStationCodes()
}

func init() {
	LoadStationCodes()
}

func LoadStationCodes() error {
	stationCacheMutex.Lock()
	defer stationCacheMutex.Unlock()

	if stationCache != nil {
		stationCodesLoaded = true
		return nil
	}

	cachePath := GetStationCodesPath()

	if data, err := os.ReadFile(cachePath); err == nil {
		var codes StationCodes
		if err := json.Unmarshal(data, &codes); err == nil {
			stationCache = &codes
			fmt.Printf("Loaded %d station codes from cache (updated: %s)\n",
				len(codes.Codes), codes.Updated.Format("2006-01-02 15:04:05"))
			return nil
		}
	}

	if err := fetchAndCacheStationCodes(); err != nil {
		fmt.Printf("Failed to fetch station codes, using fallback: %v\n", err)
		loadFallbackStationCodes()
	}
	return nil
}

func fetchAndCacheStationCodes() error {
	req, err := http.NewRequest("GET", stationNamesURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch station codes: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	stationCodes := parseStationNames(string(body))

	stationCache = &StationCodes{
		Codes:   stationCodes,
		Updated: time.Now(),
	}

	cachePath := GetStationCodesPath()
	data, err := json.MarshalIndent(stationCache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal station codes: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		fmt.Printf("Warning: failed to write cache file: %v\n", err)
	}

	fmt.Printf("Fetched and cached %d station codes\n", len(stationCodes))
	return nil
}

func parseStationNames(jsContent string) []StationCode {
	var codes []StationCode

	prefix := "station_names ='"
	suffix := "'"
	start := strings.Index(jsContent, prefix)
	if start == -1 {
		return codes
	}
	start += len(prefix)
	end := strings.Index(jsContent[start:], suffix)
	if end == -1 {
		return codes
	}
	content := jsContent[start : start+end]

	entries := strings.Split(content, "@")
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, "|")
		// Format: shortCode|name|pinyin|shortName
		// Example: bji|北京|BJP|beijing|bj
		if len(parts) >= 3 {
			code := StationCode{
				Code:      parts[2], // 3-letter station code like "BJP"
				Name:      parts[1], // Chinese name like "北京"
				Pinyin:    parts[3],
				ShortName: parts[4],
			}
			codes = append(codes, code)
		}
	}

	return codes
}

func loadFallbackStationCodes() {
	stationCache = &StationCodes{
		Codes: []StationCode{
			{Code: "BJP", Name: "北京", Pinyin: "beijing", ShortName: "bj"},
			{Code: "SHH", Name: "上海", Pinyin: "shanghai", ShortName: "sh"},
			{Code: "GZQ", Name: "广州", Pinyin: "guangzhou", ShortName: "gz"},
			{Code: "SZP", Name: "深圳", Pinyin: "shenzhen", ShortName: "sz"},
			{Code: "HZH", Name: "杭州", Pinyin: "hangzhou", ShortName: "hz"},
			{Code: "CDW", Name: "成都", Pinyin: "chengdu", ShortName: "cd"},
			{Code: "WHH", Name: "武汉", Pinyin: "wuhan", ShortName: "wh"},
			{Code: "XAY", Name: "西安", Pinyin: "xian", ShortName: "xa"},
			{Code: "NJH", Name: "南京", Pinyin: "nanjing", ShortName: "nj"},
			{Code: "TJP", Name: "天津", Pinyin: "tianjin", ShortName: "tj"},
			{Code: "CQW", Name: "重庆", Pinyin: "chongqing", ShortName: "cq"},
			{Code: "XUN", Name: "信阳", Pinyin: "xinyang", ShortName: "xy"},
		},
		Updated: time.Now(),
	}
}

func GetStationCodeByName(name string) string {
	stationCacheMutex.RLock()
	defer stationCacheMutex.RUnlock()

	if stationCache == nil {
		return ""
	}

	// If input looks like a station code (all uppercase, 3 letters), validate it exists
	if len(name) == 3 && name == strings.ToUpper(name) && isAllLetters(name) {
		for _, code := range stationCache.Codes {
			if code.Code == name {
				return code.Code
			}
		}
		// If it's a valid-looking code but not in our list, return it anyway
		// (12306 may accept codes we don't have cached)
		return name
	}

	for _, code := range stationCache.Codes {
		// Exact match (name, pinyin, or short name)
		if code.Name == name || strings.EqualFold(code.Pinyin, name) || strings.EqualFold(code.ShortName, name) {
			return code.Code
		}
	}

	// Partial match (station name contains search term)
	for _, code := range stationCache.Codes {
		if strings.Contains(code.Name, name) || strings.Contains(name, code.Name) {
			return code.Code
		}
	}

	return ""
}

// isAllLetters checks if string contains only ASCII letters
func isAllLetters(s string) bool {
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

func RefreshStationCodes() error {
	return fetchAndCacheStationCodes()
}
