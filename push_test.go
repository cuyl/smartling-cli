package main

import (
	"bytes"
	"fmt"
	smartling "github.com/Smartling/api-sdk-go"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

type request struct {
	response string
	code     int
}

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func mockHttpClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestPushStopUnauthorized(t *testing.T) {
	args := getArgs("README.md README.md")

	httpClient := getMockHttpClient([]request{{"{}", 401}})

	mockGlobber(args)
	defer func() {
		globFilesLocally = globFilesLocallyFunc
	}()

	client := getClient(httpClient)

	err := doFilesPush(&client, getConfig(), args)

	assert.EqualError(
		t,
		err,
		"ERROR: unable to upload file \"README.md\"\n"+
			"└─ failed to upload original file: unable to authenticate: "+
			"authentication parameters are invalid\n\n"+
			"Check, that you have enough permissions to upload file to the specified project")
}

func TestPushContinueFakeError(t *testing.T) {
	args := getArgs("README.md README.md")

	responses := []request{{`{
    "response": {
        "code": "SUCCESS",
        "data": {
			"accessToken": "accessToken",
			"refreshToken": "refreshToken"
        }
    }
}`, 200},
		{`{
    "response": {
        "code": "SUCCESS",
        "data": {
        }
    }
}`, 401},
		{`{
    "response": {
        "code": "SUCCESS",
        "data": {
        }
    }
}`, 200},
	}

	httpClient := getMockHttpClient(responses)

	mockGlobber(args)
	defer func() {
		globFilesLocally = globFilesLocallyFunc
	}()

	client := getClient(httpClient)

	err := doFilesPush(&client, getConfig(), args)
	assert.EqualError(
		t,
		err,
		"ERROR: failed to upload 1 files\n\nfailed to upload files README.md")
}

func TestPushStopApiError(t *testing.T) {
	args := getArgs("README.md README.md")

	responses := []request{{`{
    "response": {
        "code": "SUCCESS",
        "data": {
			"accessToken": "accessToken",
			"refreshToken": "refreshToken"
        }
    }
}`, 200},
		{`{
    "response": {
        "code": "MAINTENANCE_MODE_ERROR",
        "data": {
			"accessToken": "accessToken",
			"refreshToken": "refreshToken"
        }
    }
}`, 500},
		{`{
    "response": {
        "code": "SUCCESS",
        "data": {
			"accessToken": "accessToken",
			"refreshToken": "refreshToken"
        }
    }
}`, 200},
	}

	httpClient := getMockHttpClient(responses)

	mockGlobber(args)
	defer func() {
		globFilesLocally = globFilesLocallyFunc
	}()

	client := getClient(httpClient)

	err := doFilesPush(&client, getConfig(), args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ERROR: unable to upload file \"README.md\"\n"+
		"└─ failed to upload original file: API call returned unexpected HTTP code: 500")
}

func getMockHttpClient(responses []request) *http.Client {
	responseCount := 0
	return mockHttpClient(func(req *http.Request) *http.Response {
		var response string
		var statusCode int
		header := make(http.Header)
		header.Add("Content-Type", "application/json")
		if responseCount >= len(responses) {
			response = responses[len(responses)-1].response
			statusCode = responses[len(responses)-1].code
		} else {
			response = responses[responseCount].response
			statusCode = responses[responseCount].code
			responseCount++
		}
		return &http.Response{
			StatusCode: statusCode,
			Body:       ioutil.NopCloser(bytes.NewBufferString(response)),
			Header:     header,
		}
	})
}

func getArgs(file string) map[string]interface{} {
	args := make(map[string]interface{})
	args["--authorize"] = false
	args["--directory"] = ""
	args["<file>"] = file
	return args
}

func getConfig() Config {
	fileConfig := make(map[string]FileConfig)
	fileConfig["default"] = FileConfig{
		Push: struct {
			Type       string            `yaml:"type,omitempty"`
			Directives map[string]string `yaml:"directives,omitempty,flow"`
		}{Type: "md"},
	}
	return Config{
		UserID:    "test",
		Secret:    "test",
		ProjectID: "test",
		Files:     fileConfig,
	}
}

func getClient(httpClient *http.Client) smartling.Client {
	client := smartling.NewClient("test", "test")
	client.HTTP = httpClient

	return *client
}

func mockGlobber(args map[string]interface{}) {
	globFilesLocally = func(
		directory string,
		base string,
		mask string,
	) ([]string, error) {
		return strings.Split(fmt.Sprintf("%s", args["<file>"]), " "), nil
	}
}
