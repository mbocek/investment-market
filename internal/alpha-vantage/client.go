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
	// Create an HTTP client to download CSV stock data and process it
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/query?function=TIME_SERIES_INTRADAY&symbol=%s&interval=%s&month=%s&outputsize=full&apikey=%s&datatype=csv", c.baseUrl, symbol, interval, month, c.token), nil)
	if err != nil {
		return eris.Wrap(err, "failed to create request")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return eris.Wrap(err, "failed to make HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return eris.Wrap(err, "received non-OK HTTP status")
	}

	// Check if the response is JSON
	contentType := resp.Header.Get("Content-Type")
	if contentType == "application/json" || contentType == "application/json; charset=utf-8" {
		// Try to parse the body as JSON
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return eris.Wrap(err, "failed to read response body")
		}

		var jsonBody interface{}
		if json.Unmarshal(bodyBytes, &jsonBody) == nil {
			log.Error().Interface("jsonBody", jsonBody).Msg("response returned JSON instead of expected CSV")
			return eris.New("response returned JSON instead of expected CSV")
		}
	}

	// Create a new reader for further processing since the body has been read
	reader := csv.NewReader(resp.Body)

	// Typically, the first record in CSV is a header. Let's read it first.
	_, err = reader.Read()
	if err != nil {
		return eris.Wrap(err, "failed to read CSV header")
	}

	// Iterate through CSV records one at a time
	processed := 0
	for {
		record, errRead := reader.Read()
		if errRead == io.EOF {
			break
		}
		if errRead != nil {
			return eris.Wrap(errRead, "error reading CSV record")
		}

		// Process each CSV record
		// Customize this according to your market.Processing implementation
		timestamp, errConvertTime := c.convertToTime(record[0])
		if errConvertTime != nil {
			return eris.Wrap(errConvertTime, "failed to convert time")
		}

		open, errOpen := c.convertToFloat(record[1])
		if errOpen != nil {
			return eris.Wrap(errOpen, "failed to convert open")
		}

		high, errHigh := c.convertToFloat(record[2])
		if errHigh != nil {
			return eris.Wrap(errHigh, "failed to convert high")
		}

		low, errLow := c.convertToFloat(record[3])
		if errLow != nil {
			return eris.Wrap(errLow, "failed to convert low")
		}

		cls, errClose := c.convertToFloat(record[4])
		if errClose != nil {
			return eris.Wrap(errClose, "failed to convert close")
		}

		volume, errVolume := c.convertToInt(record[5])
		if errVolume != nil {
			return eris.Wrap(errVolume, "failed to convert volume")
		}

		if errProcess := p.Process(market.InstrumentData{
			Symbol:    symbol,
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     cls,
			Volume:    volume,
		}); errProcess != nil {
			return eris.Wrap(errProcess, "failed to process record")
		}
		processed++
	}
	log.Info().Int("processed", processed).Str("symbol", symbol).Str("month", month).Msg("processed records")

	return nil
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
