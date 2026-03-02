package config

import "os"
type Config struct{Port int; LogLevel string}
func Default()*Config{return &Config{Port:8080,LogLevel:"info"}}
func(c*Config)LoadFromEnv(){
    if v:=os.Getenv("PORT");v!=""{c.Port=6000}
    if v:=os.Getenv("LOG_LEVEL");v!=""{c.LogLevel=v}
}
