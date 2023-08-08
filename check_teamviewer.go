package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type Device struct {
	RemoteControlID string `json:"remotecontrol_id"`
	DeviceID        string `json:"device_id"`
	Alias           string `json:"alias"`
	GroupID         string `json:"groupid"`
	OnlineState     string `json:"online_state"`
	AssignedTo      bool   `json:"assigned_to"`
	TeamViewerID    int    `json:"teamviewer_id"`
}

type APIResponse struct {
	Devices []Device `json:"devices"`
}

var (
	apiKey       string
	host         string
	teamviewerid string
	stateCode    int
	stateText    string
)

const (
	teamViewerAPIURL = "https://webapi.teamviewer.com/api/v1/devices"
)

func main() {
	flag.StringVar(&apiKey, "apikey", "", "TeamViewer API key")
	flag.StringVar(&host, "host", "", "Host to test")
	flag.StringVar(&teamviewerid, "teamviewerid", "", "ID Teamviewer of Host to test")
	flag.Parse()

	if apiKey == "" || (host == "" && teamviewerid == "" ) {
		fmt.Println("Please provide the TeamViewer API key and host to test or teamviewer id")
		flag.Usage()
		os.Exit(1)
	}

	req, err := http.NewRequest("GET", teamViewerAPIURL, strings.NewReader(""))
	if err != nil {
		fmt.Println("error while creating request :", err)
		return
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Send HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("CRITICAL - Failed to connect to TeamViewer API: %s\n", err.Error())
		os.Exit(2)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("CRITICAL - Failed to read TeamViewer API response: %s\n", err.Error())
		os.Exit(2)
	}

	devices, err := ParseAPIResponse(string(body))
	var currentDevice Device

	if host != "" {
		currentDevice, err = FindDeviceWithPropertyValue(devices, host, TestHostname)
	} else {
		currentDevice, err = FindDeviceWithPropertyValue(devices, teamviewerid, TestID)
	}

	if err != nil {
		fmt.Printf("CRITICAL - Device not found: %s\n", host)
		os.Exit(2)
	}

	if currentDevice.OnlineState == "Online" {
		stateCode = 0
		stateText = "OK"
	} else {
		stateCode = 2
		stateText = "CRITICAL"
	}

	fmt.Printf("%s - TeamViewer %s: %s\n", stateText, currentDevice.OnlineState, currentDevice.Alias)
	os.Exit(stateCode)
}

func ParseAPIResponse(responseJSON string) ([]Device, error) {
	var apiResponse APIResponse
	err := json.Unmarshal([]byte(responseJSON), &apiResponse)
	if err != nil {
		return nil, err
	}

	return apiResponse.Devices, nil
}

func FindDeviceWithPropertyValue(devices []Device, propertyValue string, TestFunc func(Device, string) bool) (Device, error) {
	for _, device := range devices {
		if TestFunc(device, propertyValue) {
			return device, nil
		}
	}
	return devices[0], errors.New("Device not found")
}

func TestID(device Device, id string) bool {
	return ("r" +id) == device.RemoteControlID
}

func TestHostname(device Device, hostname string) bool {
	match, _ := regexp.MatchString("\\d+_"+hostname, device.Alias)
	return match
}
