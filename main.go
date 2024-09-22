package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"bytes"
	"io/ioutil"
)

func check(res *http.Response) {
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		log.Fatalf("Error: %d - %s", res.StatusCode, res.Status)
	}
}

func main() {
	token := os.Getenv("ODIDO_TOKEN")
	if token == "" {
		log.Fatal("ODIDO_TOKEN environment variable is not set")
	}

	thresholdStr := os.Getenv("ODIDO_THRESHOLD")
	threshold := 1500
	if thresholdStr != "" {
		parsed, err := strconv.Atoi(thresholdStr)
		if err == nil {
			threshold = parsed
		}
	}

	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://capi.t-mobile.nl/account/current?resourcelabel=LinkedSubscriptions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "T-Mobile 5.3.28 (Android 10; 10)")
	req.Header.Set("Accept", "application/json")

	// First request
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	check(res)

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	var dict map[string]interface{}
	if err := json.Unmarshal(body, &dict); err != nil {
		log.Fatal(err)
	}

	// Get resources URL
	resourcesUrl := dict["Resources"].([]interface{})[0].(map[string]interface{})["Url"].(string)

	// Second request
	req, _ = http.NewRequest("GET", resourcesUrl, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	check(res)

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	var dict2 map[string]interface{}
	if err := json.Unmarshal(body, &dict2); err != nil {
		log.Fatal(err)
	}

	subscriptionUrl := dict2["subscriptions"].([]interface{})[0].(map[string]interface{})["SubscriptionURL"].(string)

	// Third request
	req, _ = http.NewRequest("GET", subscriptionUrl+"/roamingbundles", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	check(res)

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	var dict3 map[string]interface{}
	if err := json.Unmarshal(body, &dict3); err != nil {
		log.Fatal(err)
	}

	totalRemaining := 0
	for _, bundle := range dict3["Bundles"].([]interface{}) {
		b := bundle.(map[string]interface{})
		if b["ZoneColor"] == "NL" {
			remaining := b["Remaining"].(map[string]interface{})["Value"].(float64)
			totalRemaining += int(remaining)
		}
	}

	fmt.Printf("Threshold: %d\n", threshold)

	if totalRemaining/1024 < threshold {
		data := map[string]interface{}{
			"Bundles": []map[string]string{
				{"BuyingCode": "A0DAY01"},
			},
		}
		jsonData, _ := json.Marshal(data)
		req, _ = http.NewRequest("POST", subscriptionUrl+"/roamingbundles", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()

		check(res)

		fmt.Println("2000MB added")
	} else {
		fmt.Printf("Remaining: %d MB, no need to update\n", totalRemaining/1024)
	}
}
