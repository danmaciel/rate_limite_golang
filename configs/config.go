package configs

import (
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type conf struct {
	WebServerPort        string `mapstructure:"WEB_SERVER_PORT"`
	MaxRequisitionsByIp  int    `mapstructure:"MAX_REQUISITIONS_BY_IP"`
	BlackListMinutesByIp int    `mapstructure:"BLACK_LIST_MINUTES_BY_IP"`
	RedisAddress         string `mapstructure:"REDIS_ADDR"`
	RedisPasswd          string `mapstructure:"REDIS_PASSWD"`
	RedisDBUsed          int    `mapstructure:"REDIS_DB"`
}

func LoadConfig(path string) (*conf, error) {
	var cfg *conf
	viper.SetConfigName("app_config")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	err = viper.Unmarshal(&cfg)
	if err != nil {
		panic(err)
	}

	ipLimitByOs, _ := strconv.Atoi(os.Getenv("MAX_REQUISITIONS_BY_IP"))
	if ipLimitByOs != 0 {
		cfg.MaxRequisitionsByIp = ipLimitByOs
	}

	return cfg, err
}
