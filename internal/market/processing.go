package market

import "time"

type InstrumentData struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    int
}

type Processing interface {
	Process(data InstrumentData) error
}
