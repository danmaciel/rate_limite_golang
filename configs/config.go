package configs

import (
 "errors"
 "os"

 "github.com/spf13/viper"
)

type conf struct {
 WebServerPort string `mapstructure:"WEB_SERVER_PORT"`
 MaxRequisitionsByIp int `mapstructure:"MAX_REQUISITIONS_BY_IP"`
 BlackListMinutesByIp int `mapstructure:"BLACK_LIST_MINUTES_BY_IP"`
 BlackListMinutesByToken int `mapstructure:"BLACK_LIST_MINUTES_BY_TOKEN"`
 RedisAddress string `mapstructure:"REDIS_ADDR"`
 RedisPasswd string `mapstructure:"REDIS_PASSWD"`
 RedisDBUsed int `mapstructure:"REDIS_DB"`
}

func LoadConfig(path string) (*conf, error) {
 v := viper.New()
 v.SetConfigFile(path + "/.env")
 v.SetConfigType("env")
 v.AutomaticEnv()

 defaults := map[string]interface{}{
  "WEB_SERVER_PORT": ":8080",
  "MAX_REQUISITIONS_BY_IP": 5,
  "BLACK_LIST_MINUTES_BY_IP": 2,
  "BLACK_LIST_MINUTES_BY_TOKEN": 2,
  "REDIS_ADDR": "redis:6379",
  "REDIS_PASSWD": "",
  "REDIS_DB": 0,
 }
 for key, val := range defaults {
  v.SetDefault(key, val)
  if err := v.BindEnv(key); err != nil { return nil, err }
 }
 if err := v.ReadInConfig(); err != nil {
  var notFound viper.ConfigFileNotFoundError
  // An optional .env is allowed. All other errors must be reported.
  if !errors.As(err, &notFound) && !os.IsNotExist(err) {
   // SetConfigFile produces *os.PathError for a missing file.
   // The default values and environment variables still work without it.
   return nil, err
  }
 }
 var cfg conf
 if err := v.Unmarshal(&cfg); err != nil { return nil, err }
 return &cfg, nil
}
