package main

import (
	"flag"

	"./core"
	"./log"

	_ "./proxy/socks"
	_ "./proxy/masker"
	_ "./proxy/identical"
)

var (
	configFile	string
	logLevel	string
)

func init() {
	flag.StringVar(&configFile, "config_file", "server_config.json", "Node config file.")
	flag.StringVar(&logLevel, "log_level", "info", "Level of log info to be printed to console, available value: debug, info, warning, error.")
}

func main() {
	flag.Parse()

	switch logLevel {
	case "debug":
		log.SetCurLogLevel(log.DebugLevel)
	case "warning":
		log.SetCurLogLevel(log.WarningLevel)
	case "error":
		log.SetCurLogLevel(log.ErrorLevel)
	default:
		log.SetCurLogLevel(log.InfoLevel)
	}

	config, err := core.LoadConfig(configFile)
	if err != nil {
		panic(log.Error("Err in loading config: %v.", err))
	}
	log.Info("Succeed loading config.")

	node, err := core.NewNode(config)
	if err != nil {
		panic(log.Error("Err in creating a new node: %v.", err))
	}
	log.Info("Succceed creating the node.")
	
	err = node.Start()
	if err != nil {
		panic(log.Error("Err in starting node: %v.", err))
	}
	log.Info("Node starting...")

	exit := make(chan bool)
	<-exit
}