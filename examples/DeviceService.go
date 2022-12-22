package main

import (
	"context"
	"fmt"
	"github.com/jfsmig/onvif/networking"
	"log"
	"net/http"

	"github.com/jfsmig/onvif/device"
	"github.com/jfsmig/onvif/xsd/onvif"
)

const (
	login    = "admin"
	password = "Supervisor"
)

func main() {
	ctx := context.Background()

	//Getting a camera instance
	dev, err := networking.NewClient(networking.ClientParams{
		Xaddr:      "192.168.13.14:80",
		Username:   login,
		Password:   password,
		HttpClient: new(http.Client),
	})
	if err != nil {
		panic(err)
	}

	//Preparing commands
	systemDateAndTyme := device.GetSystemDateAndTime{}
	getCapabilities := device.GetCapabilities{Category: "All"}
	createUser := device.CreateUsers{
		User: []onvif.User{
			{Username: "TestUser", Password: "TestPassword", UserLevel: "User"},
		},
	}

	//Commands execution
	systemDateAndTimeResponse, err := device.Call_GetSystemDateAndTime(ctx, dev, systemDateAndTyme)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(systemDateAndTimeResponse)
	}
	getCapabilitiesResponse, err := device.Call_GetCapabilities(ctx, dev, getCapabilities)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(getCapabilitiesResponse)
	}

	createUserResponse, err := device.Call_CreateUsers(ctx, dev, createUser)
	if err != nil {
		log.Println(err)
	} else {
		// You could use https://github.com/jfsmig/onvif/gosoap for pretty printing response
		fmt.Println(createUserResponse)
	}

}
