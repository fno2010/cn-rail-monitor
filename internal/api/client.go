package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	APIBase  = "https://kyfw.12306.cn"
	InitURL  = APIBase + "/otn/leftTicket/init"
	QueryURL = APIBase + "/otn/leftTicket/query"
)

var (
	seatTypeMap = map[string]struct {
		code string
		name string
	}{
		"9":  {"swz", "商务座"},
		"P":  {"tz", "特等座"},
		"M":  {"zy", "一等座"},
		"D":  {"zy", "优选一等座"},
		"O":  {"ze", "二等座"},
		"S":  {"ze", "二等包座"},
		"6":  {"gr", "高级软卧"},
		"A":  {"gr", "高级动卧"},
		"4":  {"rw", "软卧"},
		"I":  {"rw", "一等卧"},
		"F":  {"rw", "动卧"},
		"3":  {"yw", "硬卧"},
		"J":  {"yw", "二等卧"},
		"2":  {"rz", "软座"},
		"1":  {"yz", "硬座"},
		"W":  {"wz", "无座"},
		"WZ": {"wz", "无座"},
		"H":  {"qt", "其他"},
	}
)

type Client struct {
	httpClient  *http.Client
	cookies     string
	mu          sync.RWMutex
	enablePrice bool
}

type QueryResult struct {
	HTTPStatus int                    `json:"httpstatus"`
	Data       map[string]interface{} `json:"data"`
}

type TrainInfo struct {
	TrainNo       string `json:"train_no"`
	TrainType     string `json:"train_type_code"`
	FromStation   string `json:"from_station_name"`
	ToStation     string `json:"to_station_name"`
	DepartureTime string `json:"start_time"`
	ArrivalTime   string `json:"arrive_time"`
	Duration      string `json:"lishi"`
}

type TicketInfo struct {
	TrainInfo
	Date         string  `json:"date"`
	SeatType     string  `json:"seat_type"`
	SeatTypeCode string  `json:"seat_type_code"`
	Price        float64 `json:"price"`
	Available    int     `json:"available"`
	Status       string  `json:"status"`
}

type Price struct {
	SeatName     string  `json:"seat_name"`
	Short        string  `json:"short"`
	SeatTypeCode string  `json:"seat_type_code"`
	Num          string  `json:"num"`
	Price        float64 `json:"price"`
	Discount     *int    `json:"discount"`
}

func NewClient(enablePrice bool) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &Client{
		httpClient:  client,
		enablePrice: enablePrice,
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (c *Client) refreshCookie() error {
	req, err := http.NewRequest("GET", InitURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get cookie: %w", err)
	}
	defer resp.Body.Close()

	cookies := ""
	for _, h := range resp.Header["Set-Cookie"] {
		parts := strings.Split(h, ";")
		if len(parts) > 0 && strings.Contains(parts[0], "=") {
			if cookies != "" {
				cookies += "; "
			}
			cookies += parts[0]
		}
	}

	if cookies == "" {
		return fmt.Errorf("no cookie returned")
	}

	c.mu.Lock()
	c.cookies = cookies
	c.mu.Unlock()

	log.Printf("Cookie refreshed: %s", cookies[:minInt(60, len(cookies))])
	return nil
}

func (c *Client) QueryTickets(fromStation, toStation, date string) ([]TicketInfo, error) {
	if err := c.refreshCookie(); err != nil {
		return nil, fmt.Errorf("cookie refresh failed: %w", err)
	}

	fromCode := convertToStationCode(fromStation)
	toCode := convertToStationCode(toStation)

	c.mu.RLock()
	cookies := c.cookies
	c.mu.RUnlock()

	log.Printf("Using cookies: %s", cookies)

	req, err := http.NewRequest("GET", QueryURL+"?"+
		"leftTicketDTO.train_date="+date+
		"&leftTicketDTO.from_station="+fromCode+
		"&leftTicketDTO.to_station="+toCode+
		"&purpose_codes=ADULT", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Cookie", cookies)
	req.Header.Set("Referer", "https://www.12306.cn/")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var initialResp map[string]interface{}
	if err := json.Unmarshal(body, &initialResp); err == nil {
		if cUrl, ok := initialResp["c_url"].(string); ok && cUrl != "" {
			log.Printf("Redirecting to: %s", cUrl)
			req, err = http.NewRequest("GET", APIBase+"/otn/"+cUrl+"?"+
				"leftTicketDTO.train_date="+date+
				"&leftTicketDTO.from_station="+fromCode+
				"&leftTicketDTO.to_station="+toCode+
				"&purpose_codes=ADULT", nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create redirect request: %w", err)
			}
			req.Header.Set("Cookie", cookies)
			req.Header.Set("Referer", InitURL)
			req.Header.Set("Accept", "application/json")
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

			resp, err = c.httpClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("redirect request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read redirect response: %w", err)
			}
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result QueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.HTTPStatus != 200 {
		return nil, fmt.Errorf("API returned non-200 status: %d", result.HTTPStatus)
	}

	tickets := c.parseTicketData(result.Data, fromStation, toStation, date)
	log.Printf("Queried tickets for %s -> %s on %s: %d trains found", fromStation, toStation, date, len(tickets))

	return tickets, nil
}

func (c *Client) parseTicketData(data map[string]interface{}, fromStation, toStation, date string) []TicketInfo {
	var tickets []TicketInfo

	resultRaw, ok := data["result"]
	if !ok {
		return tickets
	}

	resultArr, ok := resultRaw.([]interface{})
	if !ok {
		return tickets
	}

	for _, item := range resultArr {
		itemStr, ok := item.(string)
		if !ok {
			continue
		}

		unescaped, err := url.QueryUnescape(itemStr)
		if err != nil {
			continue
		}

		parts := strings.Split(unescaped, "|")
		if len(parts) < 25 {
			continue
		}

		fromTelecode := parts[4]
		toTelecode := parts[5]

		ticketMap, _ := data["map"].(map[string]interface{})
		fromName := getMapString(ticketMap, fromTelecode, fromStation)
		toName := getMapString(ticketMap, toTelecode, toStation)

		startTime := parts[8]
		arriveTime := parts[9]
		duration := parts[10]

		seats := parseSeatsFromFields(parts, nil)

		if len(seats) == 0 {
			continue
		}

		for _, seat := range seats {
			// Use parts[3] for human-readable train number (e.g., "G531")
			// parts[2] is the internal 12306 ID
			seatPrice := 0.0
			if seat.price > 0 {
				seatPrice = seat.price
			}
			tickets = append(tickets, TicketInfo{
				TrainInfo: TrainInfo{
					TrainNo:       parts[3],
					TrainType:     string(parts[3][0]),
					FromStation:   fromName,
					ToStation:     toName,
					DepartureTime: startTime,
					ArrivalTime:   arriveTime,
					Duration:      duration,
				},
				Date:         date,
				SeatType:     seat.name,
				SeatTypeCode: seat.code,
				Price:        seatPrice,
				Available:    seat.available,
				Status:       seat.status,
			})
		}
	}

	return tickets
}

func getMapString(m map[string]interface{}, key, fallback string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return fallback
}

type seatInfo struct {
	code      string
	name      string
	price     float64
	available int
	status    string
}

func parseSeatsFromFields(fields []string, prices []Price) []seatInfo {
	var seats []seatInfo

	numFields := map[string]int{
		"swz_num": 23, // 商务座
		"tz_num":  24, // 特等座
		"zy_num":  22, // 一等座
		"ze_num":  21, // 二等座
		"gr_num":  18, // 高级软卧
		"rw_num":  19, // 软卧
		"yw_num":  17, // 硬卧
		"rz_num":  16, // 软座
		"yz_num":  15, // 硬座
		"wz_num":  14, // 无座
	}

	// Price fields are at different indices in the new API format
	priceFields := map[string]int{
		"swz_price": 28,
		"tz_price":  29,
		"zy_price":  27,
		"ze_price":  26,
		"gr_price":  35,
		"rw_price":  36,
		"yw_price":  37,
		"rz_price":  38,
		"yz_price":  39,
		"wz_price":  40,
	}

	availableSeats := map[string]string{}
	priceSeats := map[string]string{}
	for name, idx := range numFields {
		if idx < len(fields) {
			availableSeats[name] = fields[idx]
		}
	}
	for name, idx := range priceFields {
		if idx < len(fields) {
			priceSeats[name] = fields[idx]
		}
	}

	seatTypeNames := map[string]string{
		"swz_num": "商务座",
		"tz_num":  "特等座",
		"zy_num":  "一等座",
		"ze_num":  "二等座",
		"gr_num":  "高级软卧",
		"rw_num":  "软卧",
		"yw_num":  "硬卧",
		"rz_num":  "软座",
		"yz_num":  "硬座",
		"wz_num":  "无座",
	}

	seatTypeCodes := map[string]string{
		"swz_num": "9",
		"tz_num":  "P",
		"zy_num":  "M",
		"ze_num":  "O",
		"gr_num":  "6",
		"rw_num":  "4",
		"yw_num":  "3",
		"rz_num":  "2",
		"yz_num":  "1",
		"wz_num":  "W",
	}

	for name, availStr := range availableSeats {
		available := parseAvailable(availStr)
		status := formatStatus(availStr)

		priceStr := priceSeats[name+"_price"]
		priceVal := parsePrice(priceStr)

		seats = append(seats, seatInfo{
			code:      seatTypeCodes[name],
			name:      seatTypeNames[name],
			price:     priceVal,
			available: available,
			status:    status,
		})
	}

	return seats
}

func parseAvailable(s string) int {
	s = strings.TrimSpace(s)
	if s == "" || s == "--" || s == "无" {
		return 0
	}
	if s == "有" || s == "充足" {
		return 9999
	}
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func formatStatus(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "--" || s == "无" {
		return "无票"
	}
	if s == "有" || s == "充足" {
		return "有票"
	}
	return "剩余" + s + "张"
}

func parsePrice(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "--" || s == "无" {
		return 0
	}
	// Price format: multiply by 10 to get actual price (e.g., "553" = 55.3 yuan)
	var n int
	fmt.Sscanf(s, "%d", &n)
	return float64(n) / 10
}

func extractPrices(ypInfo, seatDiscountInfo string, fields []string) []Price {
	var prices []Price
	priceStrLen := 10
	discountStrLen := 5

	discounts := make(map[string]int)
	for i := 0; i < len(seatDiscountInfo)/discountStrLen; i++ {
		discountStr := seatDiscountInfo[i*discountStrLen : (i+1)*discountStrLen]
		if len(discountStr) >= 2 {
			var n int
			fmt.Sscanf(discountStr[1:], "%d", &n)
			discounts[string(discountStr[0])] = n
		}
	}

	for i := 0; i < len(ypInfo)/priceStrLen; i++ {
		priceStr := ypInfo[i*priceStrLen : (i+1)*priceStrLen]
		if len(priceStr) < 6 {
			continue
		}

		seatTypeCode := priceStr[0:1]
		var priceVal int
		fmt.Sscanf(priceStr[1:6], "%d", &priceVal)
		price := float64(priceVal) / 10

		seatType, ok := seatTypeMap[seatTypeCode]
		if !ok {
			seatType = struct {
				code string
				name string
			}{"qt", "其他"}
		}

		discount, hasDiscount := discounts[seatTypeCode]

		p := Price{
			SeatName:     seatType.name,
			Short:        seatType.code,
			SeatTypeCode: seatTypeCode,
			Num:          "",
			Price:        price,
		}
		if hasDiscount {
			p.Discount = &discount
		}

		prices = append(prices, p)
	}

	return prices
}

func convertToStationCode(station string) string {
	stationCodes := map[string]string{
		"BJP": "北京",
		"BJ":  "北京",
		"SHH": "上海",
		"SH":  "上海",
		"HZH": "杭州",
		"HZ":  "杭州",
		"GZQ": "广州",
		"GZH": "广州",
		"GZ":  "广州",
		"SZP": "深圳",
		"SZH": "深圳",
		"SZ":  "深圳",
		"CDW": "成都",
		"CD":  "成都",
		"WHH": "武汉",
		"WH":  "武汉",
		"XAY": "西安",
		"XA":  "西安",
		"NJH": "南京",
		"NJ":  "南京",
		"TJP": "天津",
		"TJ":  "天津",
		"CQW": "重庆",
		"CQ":  "重庆",
		"XYY": "信阳",
	}

	if len(station) >= 2 && station == strings.ToUpper(station) {
		return station
	}

	for code, name := range stationCodes {
		if strings.Contains(name, station) || strings.Contains(station, name) {
			return code
		}
	}

	return station
}

func (c *Client) QueryTicketsWithMockData(fromStation, toStation, date string) []TicketInfo {
	trains := []string{"G1", "G2", "D1", "K1"}
	seatTypes := []struct {
		Code string
		Name string
	}{
		{"M", "一等座"},
		{"O", "二等座"},
		{"WZ", "无座"},
		{"YZ", "硬座"},
		{"SR", "商务座"},
	}

	tickets := make([]TicketInfo, 0)

	for _, trainNo := range trains {
		for _, seat := range seatTypes {
			tickets = append(tickets, TicketInfo{
				TrainInfo: TrainInfo{
					TrainNo:       trainNo,
					TrainType:     string(trainNo[0]),
					FromStation:   fromStation,
					ToStation:     toStation,
					DepartureTime: "08:00",
					ArrivalTime:   "12:00",
					Duration:      "04:00",
				},
				Date:         date,
				SeatType:     seat.Name,
				SeatTypeCode: seat.Code,
				Price:        500.0,
				Available:    100,
				Status:       "有票",
			})
		}
	}

	return tickets
}
