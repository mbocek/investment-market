package alpha_vantage

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/mbocek/investment-market/internal"
	"github.com/mbocek/investment-market/internal/market"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"strconv"
	"time"
)

const SymbolDateLayout = "2006-01"

type Client struct {
	baseUrl string
	token   string
}

func NewClient(c internal.AlphaVantage) *Client {
	return &Client{
		baseUrl: c.BaseUrl,
		token:   c.Token,
	}
}

func (c *Client) ProcessStockPerMonth(symbol, interval, month string, p market.Processing) error {
	httpClient := &http.Client{}
	url := fmt.Sprintf("%s/query?function=TIME_SERIES_INTRADAY&symbol=%s&interval=%s&month=%s&outputsize=full&apikey=%s&datatype=csv", c.baseUrl, symbol, interval, month, c.token)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return eris.Wrap(err, "failed to create request")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return eris.Wrap(err, "failed to make HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return eris.Wrap(fmt.Errorf("unexpected HTTP status: %s", resp.Status), "received non-OK HTTP status")
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && (contentType == "application/json" || contentType == "application/json; charset=utf-8") {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return eris.Wrap(readErr, "failed to read response body")
		}
		var asJson interface{}
		if json.Unmarshal(bodyBytes, &asJson) == nil {
			log.Error().Interface("jsonBody", asJson).Msg("response returned JSON instead of expected CSV")
			return eris.New("response returned JSON instead of expected CSV")
		}
		// Not valid JSON either, return an error with first 200 chars to help debugging
		return eris.Errorf("response returned neither CSV nor parseable JSON: %s", string(bodyBytes[:min(len(bodyBytes), 200)]))
	}

	reader := csv.NewReader(resp.Body)
	if _, err = reader.Read(); err != nil {
		return eris.Wrap(err, "failed to read CSV header")
	}

	var processed int
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return eris.Wrap(err, "error reading CSV record")
		}

		data, err := c.parseCSVRecord(symbol, record)
		if err != nil {
			return eris.Wrap(err, "cannot parse CSV record")
		}

		if processErr := p.Process(data); processErr != nil {
			return eris.Wrap(processErr, "failed to process record")
		}
		processed++
	}
	log.Info().Int("processed", processed).Str("symbol", symbol).Str("month", month).Msg("processed records")
	return nil
}

// parseCSVRecord parses a single record into market.InstrumentData, or returns a wrapped error with context.
func (c *Client) parseCSVRecord(symbol string, record []string) (market.InstrumentData, error) {
	if len(record) < 6 {
		return market.InstrumentData{}, eris.New("CSV record: not enough fields")
	}
	timestamp, err := c.convertToTime(record[0])
	if err != nil {
		return market.InstrumentData{}, eris.Wrap(err, "failed to convert time")
	}
	open, err := c.convertToFloat(record[1])
	if err != nil {
		return market.InstrumentData{}, eris.Wrap(err, "failed to convert open")
	}
	high, err := c.convertToFloat(record[2])
	if err != nil {
		return market.InstrumentData{}, eris.Wrap(err, "failed to convert high")
	}
	low, err := c.convertToFloat(record[3])
	if err != nil {
		return market.InstrumentData{}, eris.Wrap(err, "failed to convert low")
	}
	cls, err := c.convertToFloat(record[4])
	if err != nil {
		return market.InstrumentData{}, eris.Wrap(err, "failed to convert close")
	}
	volume, err := c.convertToInt(record[5])
	if err != nil {
		return market.InstrumentData{}, eris.Wrap(err, "failed to convert volume")
	}
	return market.InstrumentData{
		Symbol:    symbol,
		Timestamp: timestamp,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     cls,
		Volume:    volume,
	}, nil
}

// min helper to avoid panic when body on error is short
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *Client) convertToTime(s string) (time.Time, error) {
	layout := "2006-01-02 15:04:05"

	t, err := time.Parse(layout, s)
	if err != nil {
		return time.Time{}, eris.Wrap(err, "failed to parse time")
	}
	return t, nil
}

func (c *Client) convertToFloat(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return .0, eris.Wrap(err, "failed to parse float")
	}
	return v, nil
}

func (c *Client) convertToInt(s string) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, eris.Wrap(err, "failed to parse int")
	}

	return v, nil
}
