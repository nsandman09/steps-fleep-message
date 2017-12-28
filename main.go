package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"bytes"

	"strings"

	"github.com/bitrise-io/go-utils/colorstring"
)

// ConfigsModel ...
type ConfigsModel struct {
	// Fleep Inputs
	WebhookURL          string
	FromUsername        string
	FromUsernameOnError string
	Message             string
	MessageOnError      string
	// Other Inputs
	IsDebugMode bool
	// Other configs
	IsBuildFailed bool
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		WebhookURL:          os.Getenv("webhook_url"),
		FromUsername:        os.Getenv("from_username"),
		FromUsernameOnError: os.Getenv("from_username_on_error"),
		Message:             os.Getenv("message"),
		MessageOnError:      os.Getenv("message_on_error"),
		//
		IsDebugMode: (os.Getenv("is_debug_mode") == "yes"),
		//
		IsBuildFailed: (os.Getenv("STEPLIB_BUILD_STATUS") != "0"),
	}
}

func (configs ConfigsModel) print() {
	fmt.Println("")
	fmt.Println(colorstring.Blue("Fleep configs:"))
	fmt.Println(" - WebhookURL:", configs.WebhookURL)
	fmt.Println(" - FromUsername:", configs.FromUsername)
	fmt.Println(" - FromUsernameOnError:", configs.FromUsernameOnError)
	fmt.Println(" - Message:", configs.Message)
	fmt.Println(" - MessageOnError:", configs.MessageOnError)
	fmt.Println("")
	fmt.Println(colorstring.Blue("Other configs:"))
	fmt.Println(" - IsDebugMode:", configs.IsDebugMode)
	fmt.Println(" - IsBuildFailed:", configs.IsBuildFailed)
	fmt.Println("")
}

func (configs ConfigsModel) validate() error {
	// required
	if configs.WebhookURL == "" {
		return errors.New("No Webhook URL parameter specified")
	}
	if configs.Message == "" {
		return errors.New("No Message parameter specified")
	}

	return nil
}

type RequestParams struct {
	// - required
	Text string `json:"message"`
	Username  *string `json:"user"`
}

// ensureNewlineEscapeChar replaces the "\" + "n" char sequences with the "\n" newline char
func ensureNewlineEscapeChar(s string) string {
	return strings.Replace(s, "\\"+"n", "\n", -1)
}

// CreatePayloadParam ...
func CreatePayloadParam(configs ConfigsModel) ([]byte, error) {
	// - required
	msgText := configs.Message
	if configs.IsBuildFailed {
		if configs.MessageOnError == "" {
			fmt.Println(colorstring.Yellow(" (i) Build failed but no message_on_error defined, using default."))
		} else {
			msgText = configs.MessageOnError
		}
	}
	msgText = ensureNewlineEscapeChar(msgText)
	// - optional attachment params
	reqParams := RequestParams{
		Text: msgText,
	}

	// - optional
	reqUsername := configs.FromUsername
	if reqUsername != "" {
		reqParams.Username = &reqUsername
	}
	if configs.IsBuildFailed {
		if configs.FromUsernameOnError == "" {
			fmt.Println(colorstring.Yellow(" (i) Build failed but no from_username_on_error defined, using default."))
		} else {
			reqParams.Username = &configs.FromUsernameOnError
		}
	}

	if configs.IsDebugMode {
		fmt.Printf("Parameters: %#v\n", reqParams)
	}

	// JSON serialize the request params
	reqParamsJSONBytes, err := json.Marshal(reqParams)
	if err != nil {
		return []byte{}, nil
	}

	return reqParamsJSONBytes, nil
}

func main() {
	configs := createConfigsModelFromEnvs()
	configs.print()
	if err := configs.validate(); err != nil {
		fmt.Println()
		fmt.Println(colorstring.Red("Issue with input:"), err)
		fmt.Println()
		os.Exit(1)
	}

	//
	// request URL
	requestURL := configs.WebhookURL

	//
	// request parameters
	reqParamsJSONString, err := CreatePayloadParam(configs)
	if err != nil {
		fmt.Println(colorstring.Red("Failed to create JSON payload:"), err)
		os.Exit(1)
	}
	if configs.IsDebugMode {
		fmt.Println()
		fmt.Println("JSON payload: ", reqParamsJSONString)
	}

	//
	// send request
	resp, err := http.Post(requestURL, "application/json", bytes.NewBuffer(reqParamsJSONString))
	if err != nil {
		fmt.Println(colorstring.Red("Failed to send the request:"), err)
		os.Exit(1)
	}

	//
	// process the response
	body, err := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	resp.Body.Close()

	if resp.StatusCode != 200 || bodyStr != "ok" {
		fmt.Println()
		fmt.Println(colorstring.Red("Request failed"))
		fmt.Println("Response from Fleep: ", bodyStr)
		fmt.Println()
		os.Exit(1)
	}

	if configs.IsDebugMode {
		fmt.Println()
		fmt.Println("Response from Fleep: ", bodyStr)
	}
	fmt.Println()
	fmt.Println(colorstring.Green("Fleep message successfully sent! 🚀"))
	fmt.Println()
	os.Exit(0)
}
