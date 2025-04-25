package symbol

import (
	"context"
	"github.com/mbocek/investment-market/internal"
	alphaVantage "github.com/mbocek/investment-market/internal/alpha-vantage"
	"github.com/mbocek/investment-market/internal/market"
	"github.com/rotisserie/eris"
	"time"
)

type Service struct {
	alphaClient *alphaVantage.Client
	repo        *Repository
	ctx         context.Context
}

func NewService(ctx context.Context, c *alphaVantage.Client, r *Repository) *Service {
	return &Service{
		ctx:         ctx,
		alphaClient: c,
		repo:        r,
	}
}

func (s *Service) LoadSymbolTradingData(m internal.MarketData, influxDB *market.Influx) error {
	currentMonth := s.getCurrentMonth()
	for {
		month, errMonth := s.getMonth(m.Symbol, m.Start)
		if errMonth != nil {
			return errMonth
		}

		errProcess := s.alphaClient.ProcessStockPerMonth(m.Symbol, m.Interval, *month, influxDB)
		if errProcess != nil {
			return eris.Wrap(errProcess, "failed to process stock")
		}

		timestamp, err := s.createTimestamp(*month)
		if err != nil {
			return err
		}

		errUpdate := s.repo.UpdateSymbol(s.ctx, m.Symbol, timestamp)
		if errUpdate != nil {
			return errUpdate
		}

		// stop processing
		if currentMonth == *month {
			break
		}
	}
	return nil
}

func (s *Service) getCurrentMonth() string {
	t := time.Now()
	return t.Format(alphaVantage.SymbolDateLayout)
}

func (s *Service) getMonth(symbol string, start string) (*string, error) {
	progress, err := s.repo.FindSymbolProgress(s.ctx, symbol)
	if err != nil {
		return nil, eris.Wrap(err, "failed to find progress")
	}

	// if progress is nil, we need to start from the beginning
	if progress == nil {
		return &start, nil
	}

	// get current date with reset day to first day of month without time
	now := time.Now()
	t := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	// if progress is not nil, we need to start from the last update + one month if is not hgher than current month
	if progress.LastUpdate.Before(t) {
		month := progress.LastUpdate.AddDate(0, 1, 0).Format(alphaVantage.SymbolDateLayout)
		return &month, nil
	}
	month := time.Now().AddDate(0, 1, 0).Format(alphaVantage.SymbolDateLayout)

	return &month, nil
}

func (s *Service) createTimestamp(month string) (time.Time, error) {
	return time.Parse(alphaVantage.SymbolDateLayout, month)
}
