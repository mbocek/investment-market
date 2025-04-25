package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/mbocek/investment-market/db"
	"github.com/mbocek/investment-market/internal/market"
	"github.com/mbocek/investment-market/internal/market/symbol"
	"github.com/rotisserie/eris"
	"os"
	"strconv"
	"strings"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/mbocek/investment-market/internal"
	alphaVantage "github.com/mbocek/investment-market/internal/alpha-vantage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func callerMarshalFuncWithShortFileName(_ uintptr, file string, line int) string {
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}
	file = short
	return file + ":" + strconv.Itoa(line)
}

func configureLogger() {
	zerolog.CallerMarshalFunc = callerMarshalFuncWithShortFileName
	log.Logger = log.With().Caller().Logger()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(os.Stdout)
}

func readConfigFile() internal.Config {
	viper.SetConfigName("market")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/app/config") // EKS path
	viper.AddConfigPath("config")      // local path
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read config file (probably doesn't exists)")
	}
	var config internal.Config
	if errUnmarshal := viper.Unmarshal(&config); errUnmarshal != nil {
		log.Fatal().Err(errUnmarshal).Msg("failed to unmarshal config file")
	}

	log.Debug().Interface("Configuration", config).Msg("")
	return config
}

func getDatabaseURI(c internal.Postgres) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", c.User, c.Password, c.Host, c.Port, c.Password)
}

func migrateDatabase(c string) error {
	migrations, err := iofs.New(db.Migrations, "migrations")
	if err != nil {
		return eris.Wrap(err, "cannot open migrations")
	}

	m, err := migrate.NewWithSourceInstance("iofs", migrations, c)
	if err != nil {
		return eris.Wrap(err, "cannot open source migrations")
	}

	defer func(m *migrate.Migrate) {
		errClose, _ := m.Close()
		if errClose != nil {
			log.Fatal().Err(err).Msg("cannot close migration")
		}
	}(m)

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return eris.Wrap(err, "cannot migrate")
	}

	return nil
}

func main() {
	ctx := context.Background()
	// add zero log support
	configureLogger()

	// add viper configuration
	c := readConfigFile()
	log.Info().Interface("config", c).Msg("loaded configuration")

	// migrations
	err := migrateDatabase(getDatabaseURI(c.Postrgres))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to migrate database")
	}

	// db connection
	conn, err := pgx.Connect(ctx, getDatabaseURI(c.Postrgres))
	if err != nil {
		log.Fatal().Err(err).Msg("unable to connect to database")
	}
	defer conn.Close(context.Background())

	client := alphaVantage.NewClient(c.AlphaVantage)

	symbolRepo := symbol.NewRepository(conn)
	symbolService := symbol.NewService(ctx, client, symbolRepo)
	influxDB := market.NewInflux(ctx, c.InfluxDB)

	for _, m := range c.Data {
		errLoad := symbolService.LoadSymbolTradingData(m, influxDB)
		if errLoad == nil {
			log.Info().Str("symbol", m.Symbol).Msg("loaded data")
		} else {
			log.Error().Err(errLoad).Msg("failed to load data")
		}
	}
}
