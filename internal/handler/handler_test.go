package handler
import "testing"
func TestProcess(t*testing.T){if New().Process()!="ok"{t.Error()}}
func TestHealth(t*testing.T){if New().Health()["status"]!="healthy"{t.Error()}}
