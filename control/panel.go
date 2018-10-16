package control

import (
	"fmt"
	"github.com/go-chassis/go-chassis/control"
	"github.com/go-chassis/go-chassis/core/config"
)

var panelPlugin = make(map[string]func(options Options) control.Panel)

//DefaultPanel get fetch config
var DefaultPanelEgress control.Panel

//InstallPlugin install implementation
func InstallPlugin(name string, f func(options Options) control.Panel) {
	panelPlugin[name] = f
}

//Options is options
type Options struct {
	Address string
}

//Init initialize DefaultPanel
func Init() error {
	fmt.Println("Raj: Enterned init of control and value of infra is %v", config.GlobalDefinition.Panel.Infra)
	infra := config.GlobalDefinition.Panel.Infra
	if infra == "" || infra == "archaius" {
		infra = "egressarchaius"
	} else if infra == "pilot" {
		infra = "egresspilot"
	}
	fmt.Println("Raj: the value inside panel is %v ", panelPlugin)
	f, ok := panelPlugin[infra]
	if !ok {
		return fmt.Errorf("do not support [%s] panel", infra)
	}

	DefaultPanelEgress = f(Options{
		Address: config.GlobalDefinition.Panel.Settings["address"],
	})
	return nil
}
