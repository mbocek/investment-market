package market

import (
	"context"
	influxdb2 "github.com/influxdata/influxdb-client-go"
	"github.com/influxdata/influxdb-client-go/api"
	"github.com/mbocek/investment-market/internal"
	"github.com/rs/zerolog/log"
)

type Influx struct {
	client influxdb2.Client
	write  api.WriteAPIBlocking
	ctx    context.Context
}

func NewInflux(ctx context.Context, c internal.InfluxDB) *Influx {
	client := influxdb2.NewClient(c.Url, c.Token)
	return &Influx{
		ctx:    ctx,
		client: client,
		write:  client.WriteAPIBlocking(c.Org, c.Bucket),
	}
}

func (i *Influx) Process(data InstrumentData) error {
	log.Debug().Interface("data", data).Msg("Processing data")

	p := influxdb2.NewPoint(data.Symbol,
		nil,
		map[string]interface{}{"open": data.Open, "high": data.High, "low": data.Low, "close": data.Close, "volume": data.Volume},
		data.Timestamp)

	// Write point immediately
	err := i.write.WritePoint(i.ctx, p)
	if err != nil {
		return err
	}

	return nil
}

func (i *Influx) Close() {
	i.client.Close()
}
