package masker

import (
	"net"
	"io/ioutil"
	"encoding/json"

	"../../log"
	"../../network"
	"../../account"
)

func loadCallerConfig(configFile string) (config []nextNodeConfig, err error) {
	rawData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return
	}

	err = json.Unmarshal(rawData, &config)
	return
}

type nextNodeConfig struct {
	Address		string			`json:"address"`
	Port		uint16			`json:"port"`
	UserList	[]userConfig	`json:"users"`
}

type userConfig struct {
	Id string `json:"id"`
}

func (config nextNodeConfig) toNextNode() (nextNode, bool) {
	ip := net.ParseIP(config.Address)
	if ip == nil {
		panic(log.Error("Unable to parse ip: %v", config.Address))
	}
	
	var err error
	var addr network.Address
	addr, err = network.NewIPv4Address(ip, config.Port)
	if err != nil {
		addr, err = network.NewIPv6Address(ip, config.Port)
	}
	if err != nil {
		panic(log.Error("Illegal ip: %v", config.Address))
	}

	users := make([]account.User, 0, len(config.UserList))
	for _, tmpUserConfig := range config.UserList {
		if tmpUser, ok := tmpUserConfig.toUser(); ok {
			users = append(users, tmpUser)
		}
	}
	// if a node has no user can access, discard it
	if len(users) == 0 {
		return nextNode{}, false
	}

	return nextNode{
		destination:	network.NewTCPDestination(addr),
		userList:		users,
	}, true
}

func (config userConfig) toUser() (account.User, bool) {
	userID, err := account.NewID(config.Id)
	return account.User{
		Id: userID,
	}, (err == nil)
}

func loadListenerConfig(configFile string) (config listenerConfig, err error) {
	rawData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return
	}

	err = json.Unmarshal(rawData, &config)
	return
}

type listenerConfig struct {
	UserList	[]userConfig	`json:"users"`
}