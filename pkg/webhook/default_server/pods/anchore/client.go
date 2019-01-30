package anchore

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
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

	errNotFound = "response from Anchore: 404"

	log = logrus.New()
)

func init() {

	log.SetFormatter(&logrus.JSONFormatter{})

	yamlFile, err := ioutil.ReadFile(anchoreConfigFile)
	if err != nil {
		log.Errorf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &anchoreConfig)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
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
		log.Fatal(err)
	}
	req.SetBasicAuth(username, password)
	log.Infof("Sending request to %s, with params %s", fullURL, bodyParams)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("failed to complete request to Anchore: %v", err)
	}

	bodyText, err := ioutil.ReadAll(resp.Body)

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

	count := 0

	// wait for image analyzing
	for err != nil && err.Error() == errNotFound && count < 3 {
		body, err = anchoreRequest(path, nil, "GET")
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
		count++
	}

	// @todo: there is 30 sec limit for admission controller to reponse to the k8s api-server,
	// @todo: when the limit is removed, should be able to wait till the image scanned completed
	// @todo: a temp solution is return true when the image is not scanned for the first time
	if err != nil && err.Error() == errNotFound && count == 3 {
		// first time scanned image, return true
		log.Warnf("image %s with tag %s has not been scanned.", digest, tag)
		return true
	}

	if err != nil {
		log.Error(err)
		return false
	}

	ret := string(body)
	ret = strings.Replace(ret, "\n", "", -1)
	ret = strings.Replace(ret, "\t", "", -1)

	log.Infof("Anchore Response Body: %s", ret)

	var result []map[string]map[string][]SHAResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Error(err)
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
		log.Error(err)
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
	for err != nil && count < 5 {
		digest, err = getImageDigest(image)
		time.Sleep(time.Second)
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
	log.Infof("Added image to Anchore Engine: %s", image)
	return nil
}

func CheckImage(image string) bool {
	digest, err := waitForImageLoaded(image)
	if err != nil {
		log.Error(err)
		return false
	}
	return getStatus(digest, image)
}
