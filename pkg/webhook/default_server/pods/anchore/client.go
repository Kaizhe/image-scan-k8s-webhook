package anchore

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"
)

var (
	transCfg = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore expired SSL certificates
	}

	client = &http.Client{
		Transport: transCfg,
	}

	anchoreConfig AnchoreConfig

	anchoreConfigFile = "/tmp/sysdig-token/config.yaml"
)

func init() {
	yamlFile, err := ioutil.ReadFile(anchoreConfigFile)
	if err != nil {
		glog.Errorf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &anchoreConfig)
	if err != nil {
		glog.Fatalf("Unmarshal: %v", err)
	}
}

func anchoreRequest(path string, bodyParams map[string]string, method string) ([]byte, error) {
	username := anchoreConfig.Token
	password := ""
	anchoreEngineURL := anchoreConfig.EndpointURL
	fullURL := anchoreEngineURL + path

	bodyParamJson, err := json.Marshal(bodyParams)
	req, err := http.NewRequest(method, fullURL, bytes.NewBuffer(bodyParamJson))
	if err != nil {
		glog.Fatal(err)
	}
	req.SetBasicAuth(username, password)
	glog.Infof("Sending request to %s, with params %s", fullURL, bodyParams)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("failed to complete request to Anchore: %v", err)
	}

	bodyText, err := ioutil.ReadAll(resp.Body)
	//	glog.Info("Anchore Response Body: " + string(bodyText))
	if err != nil {
		return nil, fmt.Errorf("failed to complete request to Anchore: %v", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("response from Anchore: %d", resp.StatusCode)
	}
	return bodyText, nil
}

func getStatus(digest string, tag string) bool {
	path := fmt.Sprintf("/images/%s/check?tag=%s&history=false&detail=false", digest, tag)
	body, err := anchoreRequest(path, nil, "GET")
	if err != nil {
		glog.Error(err)
		return false
	}
	fmt.Println("Anchore Response Body: " + string(body))
	var result []map[string]map[string][]SHAResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		glog.Error(err)
		return false
	}

	foundStatus := findResult(result)

	// Is this the easiest way to get this info?
	return strings.ToLower(foundStatus) == "pass"
}

func findResult(parsed_result []map[string]map[string][]SHAResult) string {
	//Looks thru a parsed result for the status value, assumes this result is for a single image

	digest := reflect.ValueOf(parsed_result[0]).MapKeys()[0].String()
	tag := reflect.ValueOf(parsed_result[0][digest]).MapKeys()[0].String()
	return parsed_result[0][digest][tag][0].Status
}

func getImage(imageRef string) (Image, error) {
	// Tag or repo??
	params := map[string]string{
		"tag":     imageRef,
		"history": "true",
	}

	body, err := anchoreRequest("/images", params, "GET")
	if err != nil {
		glog.Error(err)
		return Image{}, err
	}

	var images []Image
	err = json.Unmarshal(body, &images)

	if err != nil {
		return Image{}, fmt.Errorf("failed to unmarshal JSON from response: %v", err)
	}

	return images[0], nil
}
func getImageDigest(imageRef string) (string, error) {
	image, err := getImage(imageRef)
	if err != nil {
		return "", err
	}
	return image.ImageDigest, nil
}

func waitForImageLoaded(image string) (digest string, err error) {
	err = addImage(image)
	if err != nil {
		return
	}
	count := 0
	digest, err = getImageDigest(image)
	for err != nil && count < 30 {
		digest, err = getImageDigest(image)
		time.Sleep(time.Second * 5)
		count++
	}
	return
}

func addImage(image string) error {
	params := map[string]string{"tag": image}
	_, err := anchoreRequest("/images", params, "POST")
	if err != nil {
		return err
	}
	glog.Infof("Added image to Anchore Engine: %s", image)
	return nil
}

func CheckImage(image string) bool {
	digest, err := waitForImageLoaded(image)
	if err != nil {
		glog.Error(err)
		return false
	}
	return getStatus(digest, image)
}
