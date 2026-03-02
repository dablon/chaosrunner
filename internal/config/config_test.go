package config
import "testing"
func TestDefault(t*testing.T){if Default().Port!=8080{t.Error()}}
