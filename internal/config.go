package internal

type AlphaVantage struct {
	BaseUrl string `mapstructure:"base_url"`
	Token   string `mapstructure:"token"`
}

type InfluxDB struct {
	Url    string `mapstructure:"url"`
	Token  string `mapstructure:"token"`
	Org    string `mapstructure:"org"`
	Bucket string `mapstructure:"bucket"`
}

type MarketData struct {
	Symbol   string `mapstructure:"symbol"`
	Interval string `mapstructure:"interval"`
	Start    string `mapstructure:"start"`
}

type Postgres struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"db"`
}

type Config struct {
	AlphaVantage AlphaVantage `mapstructure:"alpha_vantage"`
	InfluxDB     InfluxDB     `mapstructure:"influxdb"`
	Data         []MarketData `mapstructure:"data"`
	Postrgres    Postgres     `mapstructure:"postgres"`
}
