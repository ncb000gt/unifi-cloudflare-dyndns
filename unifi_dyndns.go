package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath("./")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	gatewayIP := getGatewayIP()
	zone := getZoneInfo()
	dnsResults := getDNSRecords(zone)
	updateDNSWithIP(zone, dnsResults, gatewayIP)
}

func getHttpClient() http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	return *client
}

func getGatewayIP() string {
	fmt.Println("Get Gateway IP")

	unifiUsername := viper.Get("unifi.username")
	unifiPassword := viper.Get("unifi.password")
	unifiHost := viper.Get("unifi.host")

	client := getHttpClient()
	user := map[string]string{"username": unifiUsername.(string), "password": unifiPassword.(string)}
	loginData, _ := json.Marshal(user)
	loginResponse, err := client.Post("https://"+unifiHost.(string)+"/api/login", "application/json", bytes.NewBuffer(loginData))
	if err != nil {
		panic(err)
	}

	_cookies := loginResponse.Cookies()

	var cookies []string
	for i := 0; i < len(_cookies); i++ {
		cookie := _cookies[i]
		cookies = append(cookies, cookie.String())
	}

	healthRequest, _ := http.NewRequest("GET", "https://"+unifiHost.(string)+"/api/s/default/stat/health", nil)
	healthRequest.Header.Add("Cookie", strings.Join(cookies, ";"))

	healthResponse, err := client.Do(healthRequest)
	if err != nil {
		panic(err)
	}

	defer healthResponse.Body.Close()
	body, err := ioutil.ReadAll(healthResponse.Body)
	if err != nil {
		panic(err)
	}

	re := regexp.MustCompile("wan_ip\" : \"(\\d{1,3}.\\d{1,3}.\\d{1,3}.\\d{1,3})\"")
	gatewayIP := re.FindStringSubmatch(string(body))[1]

	return gatewayIP
}

func getZoneInfo() ZoneRecord {
	fmt.Println("Get Zone Records")

	cfZoneName := viper.Get("cloudflare.zoneName")

	zoneBody := getCloudFlareAPI("/client/v4/zones")

	var zoneResults ZoneResults
	if err := json.Unmarshal(zoneBody, &zoneResults); err != nil {
		panic(err)
	}

	for i := 0; i < len(zoneResults.Results); i++ {
		zone := zoneResults.Results[i]

		if zone.Name == cfZoneName {
			return zone
		}
	}

	return ZoneRecord{}
}

func getDNSRecords(zone ZoneRecord) DNSResults {
	fmt.Println("Get DNS Records")

	dnsBody := getCloudFlareAPI("/client/v4/zones/" + zone.ID + "/dns_records")

	var dnsResults DNSResults
	if err := json.Unmarshal(dnsBody, &dnsResults); err != nil {
		panic(err)
	}

	return dnsResults
}

func getCloudFlareAPI(path string) []byte {
	cfAuthEmail := viper.Get("cloudflare.authEmail")
	cfAuthKey := viper.Get("cloudflare.authKey")

	apiRequest, _ := http.NewRequest("GET", "https://api.cloudflare.com"+path, nil)
	apiRequest.Header.Add("X-Auth-Email", cfAuthEmail.(string))
	apiRequest.Header.Add("X-Auth-Key", cfAuthKey.(string))
	apiRequest.Header.Add("Content-Type", "application/json")

	client := getHttpClient()

	apiResponse, err := client.Do(apiRequest)
	if err != nil {
		panic(err)
	}

	defer apiResponse.Body.Close()
	apiBody, err := ioutil.ReadAll(apiResponse.Body)
	if err != nil {
		panic(err)
	}

	return apiBody
}

func updateDNSWithIP(zone ZoneRecord, dnsResults DNSResults, ip string) {
	fmt.Println("Updating Record")

	cfAuthEmail := viper.Get("cloudflare.authEmail")
	cfAuthKey := viper.Get("cloudflare.authKey")
	cfDNSName := viper.Get("cloudflare.dnsName")

	for i := 0; i < len(dnsResults.Results); i++ {
		result := dnsResults.Results[i]

		if result.Name == cfDNSName {
			dnsContent := DNSRecord{
				Content: ip,
				Type:    "A",
				Name:    cfDNSName.(string),
			}
			dnsData, _ := json.Marshal(dnsContent)
			dnsUpdateRequest, _ := http.NewRequest("PUT", "https://api.cloudflare.com/client/v4/zones/"+zone.ID+"/dns_records/"+result.ID, bytes.NewBuffer(dnsData))
			dnsUpdateRequest.Header.Add("X-Auth-Email", cfAuthEmail.(string))
			dnsUpdateRequest.Header.Add("X-Auth-Key", cfAuthKey.(string))
			dnsUpdateRequest.Header.Add("Content-Type", "application/json")

			client := getHttpClient()
			dnsUpdateResponse, err := client.Do(dnsUpdateRequest)
			if err != nil {
				panic(err)
			}

			fmt.Println(result.Name, ":=>", dnsUpdateResponse.Status)
		}
	}
}

type DNSResults struct {
	Results    []DNSRecord            `json:"result"`
	ResultInfo map[string]interface{} `json:"result_info"`
	Success    bool                   `json:"success"`
	Errors     []string               `json:"errors"`
	Messages   []string               `json:"messages"`
}

type DNSRecord struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

type ZoneResults struct {
	Results    []ZoneRecord           `json:"result"`
	ResultInfo map[string]interface{} `json:"result_info"`
	Success    bool                   `json:"success"`
	Errors     []string               `json:"errors"`
	Messages   []string               `json:"messages"`
}

type ZoneRecord struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}
