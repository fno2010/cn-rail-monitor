package api

import (
	"encoding/json"
	"testing"
)

func TestConvertToStationCode(t *testing.T) {
	EnsureStationCodesLoaded()

	tests := []struct {
		input    string
		expected string
	}{
		{"北京西", "BXP"},
		{"北京南", "VNP"},
		{"上海", "SHH"},
		{"广州", "GZQ"},
		{"深圳北", "IOQ"},
		{"信阳", "XUN"},
		{"BJP", "BJP"},
		{"SHH", "SHH"},
		{"beijing", "BJP"},
		{"xianyang", "XYY"},
		{"xy", "XYK"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertToStationCode(tt.input)
			if result != tt.expected {
				t.Errorf("convertToStationCode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseStationNames(t *testing.T) {
	jsContent := `station_names ='@bji|北京|BJP|beijing|bj|0|@sha|上海|SHH|shanghai|sh|1|'`

	codes := parseStationNames(jsContent)

	if len(codes) != 2 {
		t.Fatalf("expected 2 codes, got %d", len(codes))
	}

	if codes[0].Code != "BJP" {
		t.Errorf("first code.Code = %q, want %q", codes[0].Code, "BJP")
	}
	if codes[0].Name != "北京" {
		t.Errorf("first code.Name = %q, want %q", codes[0].Name, "北京")
	}

	if codes[1].Code != "SHH" {
		t.Errorf("second code.Code = %q, want %q", codes[1].Code, "SHH")
	}
	if codes[1].Name != "上海" {
		t.Errorf("second code.Name = %q, want %q", codes[1].Name, "上海")
	}
}

func TestGetStationCodeByName(t *testing.T) {
	EnsureStationCodesLoaded()

	tests := []struct {
		input    string
		expected string
	}{
		{"北京西", "BXP"},
		{"北京南", "VNP"},
		{"上海", "SHH"},
		{"广州", "GZQ"},
		{"深圳北", "IOQ"},
		{"信阳", "XUN"},
		{"beijing", "BJP"},
		{"shanghai", "SHH"},
		{"xianyang", "XYY"},
		{"notexist", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := GetStationCodeByName(tt.input)
			if result != tt.expected {
				t.Errorf("GetStationCodeByName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseAvailable(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"有", 9999},
		{"充足", 9999},
		{"10", 10},
		{"0", 0},
		{"", 0},
		{"--", 0},
		{"无", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseAvailable(tt.input)
			if result != tt.expected {
				t.Errorf("parseAvailable(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParsePrice(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"553", 55.3},
		{"0", 0},
		{"", 0},
		{"--", 0},
		{"无", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parsePrice(tt.input)
			if result != tt.expected {
				t.Errorf("parsePrice(%q) = %f, want %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseSeatsFromFields(t *testing.T) {
	fields := []string{
		"", "", "", "G531", "BJP", "SHH", "", "", "08:00", "11:30", "03:30",
		"", "", "", "10", "20", "30", "40", "50", "60", "70", "80", "90",
		"100", "110", "", "", "", "", "", "", "", "553.0", "0", "0",
		"0", "0", "0", "0", "0", "0", "0", "0", "0",
	}

	seats := parseSeatsFromFields(fields, nil)

	if len(seats) == 0 {
		t.Fatal("expected seats to be parsed, got empty")
	}

	found := false
	for _, seat := range seats {
		if seat.code == "1" {
			found = true
			if seat.available != 20 {
				t.Errorf("硬座 available = %d, want 20", seat.available)
			}
		}
	}
	if !found {
		t.Error("expected to find 硬座 (code 1) in seats")
	}
}

func TestParseSeatsFromFieldsEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		fields      []string
		expectSeats bool
	}{
		{
			name:        "empty fields",
			fields:      []string{},
			expectSeats: false,
		},
		{
			name:        "insufficient fields",
			fields:      []string{"a", "b"},
			expectSeats: false,
		},
		{
			name:        "all sold out",
			fields:      []string{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "0", "0", "0", "0", "0", "0", "0", "0", "0"},
			expectSeats: true,
		},
		{
			name:        "all available",
			fields:      []string{"", "", "", "", "", "", "", "", "", "", "", "", "", "", "有", "有", "有", "有", "有", "充足", "充足", "充足", "充足"},
			expectSeats: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seats := parseSeatsFromFields(tt.fields, nil)
			if tt.expectSeats && len(seats) == 0 {
				t.Errorf("expected seats but got none")
			}
			if !tt.expectSeats && len(seats) > 0 {
				t.Errorf("expected no seats but got %d", len(seats))
			}
		})
	}
}

func TestGetMapString(t *testing.T) {
	m := map[string]interface{}{
		"BJP": "北京",
		"SHH": "上海",
	}

	tests := []struct {
		key      string
		fallback string
		expected string
	}{
		{"BJP", "未知", "北京"},
		{"SHH", "未知", "上海"},
		{"GZQ", "广州", "广州"},
		{"", "空", "空"},
	}

	for _, tt := range tests {
		result := getMapString(m, tt.key, tt.fallback)
		if result != tt.expected {
			t.Errorf("getMapString(%q, %q) = %q, want %q", tt.key, tt.fallback, result, tt.expected)
		}
	}
}

func TestSeatTypeMap(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"9", "swz"},
		{"P", "tz"},
		{"M", "zy"},
		{"O", "ze"},
		{"6", "gr"},
		{"4", "rw"},
		{"3", "yw"},
		{"2", "rz"},
		{"1", "yz"},
		{"W", "wz"},
		{"WZ", "wz"},
		{"H", "qt"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			info, ok := seatTypeMap[tt.code]
			if !ok {
				t.Errorf("seatTypeMap missing code %q", tt.code)
				return
			}
			if info.code != tt.expected {
				t.Errorf("seatTypeMap[%q].code = %q, want %q", tt.code, info.code, tt.expected)
			}
		})
	}
}

func TestTicketInfoJSONSerialization(t *testing.T) {
	ticket := TicketInfo{
		TrainInfo: TrainInfo{
			TrainNo:       "G531",
			TrainType:     "G",
			FromStation:   "北京南",
			ToStation:     "上海虹桥",
			DepartureTime: "08:00",
			ArrivalTime:   "11:30",
			Duration:      "03:30",
		},
		Date:         "2026-02-20",
		SeatType:     "二等座",
		SeatTypeCode: "O",
		Price:        553.0,
		Available:    100,
		Status:       "available",
	}

	data, err := json.Marshal(ticket)
	if err != nil {
		t.Fatalf("failed to marshal TicketInfo: %v", err)
	}

	var unmarshalled TicketInfo
	if err := json.Unmarshal(data, &unmarshalled); err != nil {
		t.Fatalf("failed to unmarshal TicketInfo: %v", err)
	}

	if unmarshalled.TrainNo != ticket.TrainNo {
		t.Errorf("TrainNo = %q, want %q", unmarshalled.TrainNo, ticket.TrainNo)
	}
	if unmarshalled.Available != ticket.Available {
		t.Errorf("Available = %d, want %d", unmarshalled.Available, ticket.Available)
	}
	if unmarshalled.Price != ticket.Price {
		t.Errorf("Price = %f, want %f", unmarshalled.Price, ticket.Price)
	}
}

func TestClientInitialization(t *testing.T) {
	client := NewClient(true)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}
	if client.enablePrice != true {
		t.Errorf("enablePrice = %v, want true", client.enablePrice)
	}

	clientNoPrice := NewClient(false)
	if clientNoPrice.enablePrice != false {
		t.Errorf("enablePrice = %v, want false", clientNoPrice.enablePrice)
	}
}

func TestTrainInfoFields(t *testing.T) {
	info := TrainInfo{
		TrainNo:       "G531",
		TrainType:     "G",
		FromStation:   "北京南",
		ToStation:     "上海虹桥",
		DepartureTime: "08:00",
		ArrivalTime:   "11:30",
		Duration:      "03:30",
	}

	if info.TrainNo != "G531" {
		t.Errorf("TrainNo = %q, want G531", info.TrainNo)
	}
	if info.TrainType != "G" {
		t.Errorf("TrainType = %q, want G", info.TrainType)
	}
}

func TestPriceStruct(t *testing.T) {
	price := Price{
		SeatName:     "商务座",
		Short:        "swz",
		SeatTypeCode: "9",
		Num:          "10",
		Price:        1748.0,
	}

	if price.SeatName != "商务座" {
		t.Errorf("SeatName = %q, want 商务座", price.SeatName)
	}
	if price.Price != 1748.0 {
		t.Errorf("Price = %f, want 1748.0", price.Price)
	}
}

func TestLoadStationCodes(t *testing.T) {
	EnsureStationCodesLoaded()

	stationCacheMutex.RLock()
	defer stationCacheMutex.RUnlock()

	if stationCache == nil {
		t.Fatal("stationCache is nil after EnsureStationCodesLoaded")
	}
	if len(stationCache.Codes) == 0 {
		t.Error("stationCache.Codes is empty")
	}
}

func TestRefreshStationCodes(t *testing.T) {
	err := RefreshStationCodes()
	if err != nil {
		t.Logf("RefreshStationCodes error (expected if no network): %v", err)
	}
}
