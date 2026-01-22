package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Device struct {
	Device struct {
		ProductName        string   `json:"product_name"`
		Model              string   `json:"model"`
		DSN                string   `json:"dsn"`
		OEMModel           string   `json:"oem_model"`
		SWVersion          string   `json:"sw_version"`
		TemplateID         int      `json:"template_id"`
		MAC                string   `json:"mac"`
		UniqueHardwareID   *string  `json:"unique_hardware_id"` // Could be nil
		HWSig              string   `json:"hwsig"`
		LanIP              string   `json:"lan_ip"`
		ConnectedAt        string   `json:"connected_at"`
		Key                int      `json:"key"`
		LanEnabled         bool     `json:"lan_enabled"`
		ConnectionPriority []string `json:"connection_priority"`
		HasProperties      bool     `json:"has_properties"`
		ProductClass       *string  `json:"product_class"` // Could be nil
		ConnectionStatus   string   `json:"connection_status"`
		Lat                string   `json:"lat"`
		Lng                string   `json:"lng"`
		Locality           string   `json:"locality"`
		DeviceType         string   `json:"device_type"`
		Dealer             *string  `json:"dealer"` // Could be nil
		ManufModel         string   `json:"manuf_model"`
	} `json:"device"`
}

type Property struct {
	Type             string      `json:"type"`
	Name             string      `json:"name"`
	BaseType         string      `json:"base_type"`
	ReadOnly         bool        `json:"read_only"`
	Direction        string      `json:"direction"`
	Scope            string      `json:"scope"`
	DataUpdatedAt    string      `json:"data_updated_at"`
	Key              int         `json:"key"`
	DeviceKey        int         `json:"device_key"`
	ProductName      string      `json:"product_name"`
	TrackOnlyChanges bool        `json:"track_only_changes"`
	DisplayName      string      `json:"display_name"`
	HostSwVersion    bool        `json:"host_sw_version"`
	TimeSeries       bool        `json:"time_series"`
	Derived          bool        `json:"derived"`
	AppType          *string     `json:"app_type"`
	Recipe           *string     `json:"recipe"`
	Value            interface{} `json:"value"`
	GeneratedFrom    *string     `json:"generated_from"`
	GeneratedAt      *string     `json:"generated_at"`
	DeniedRoles      []string    `json:"denied_roles"`
	AckEnabled       bool        `json:"ack_enabled"`
	RetentionDays    int         `json:"retention_days"`
	AckStatus        interface{} `json:"ack_status"`
	AckMessage       interface{} `json:"ack_message"`
	AckedAt          interface{} `json:"acked_at"`
}

type PropertyMap struct {
	Property Property `json:"property"`
}

type TokenInfo struct {
	APIToken string
	Expiry   time.Time
	Refresh  string
}

type OwletAPI struct {
	region     string
	user       string
	password   string
	authToken  string
	expiry     time.Time
	refresh    string
	httpClient *http.Client
	headers    map[string]string
	devices    map[string]interface{}
}

func tokenInfoEqual(t1, t2 TokenInfo) bool {
	return t1.APIToken == t2.APIToken &&
		t1.Expiry.Equal(t2.Expiry) &&
		t1.Refresh == t2.Refresh
}

func NewOwletAPI(region, user, password string) (*OwletAPI, error) {
	if region != "europe" && region != "world" {
		return nil, NewOwletAuthenticationError("Supplied region not valid")
	}

	return &OwletAPI{
		region:     region,
		user:       user,
		password:   password,
		httpClient: &http.Client{},
		headers:    make(map[string]string),
		devices:    make(map[string]interface{}),
	}, nil
}

func (api *OwletAPI) Authenticate() error {
	if api.authToken == "" && api.refresh == "" {
		if api.user == "" || api.password == "" {
			return NewOwletAuthenticationError("Username or password not supplied")
		}
		if err := api.passwordVerification(); err != nil {
			return err
		}
	}

	if api.authToken == "" {
		log.Println("Auth token empty; refreshing")
		return api.refreshAuthentication()
	}

	if time.Now().After(api.expiry) {
		log.Println("Auth token expired; refreshing")
		return api.refreshAuthentication()
	}

	api.headers["Authorization"] = "auth_token " + api.authToken
	return api.validateAuthentication()
}

func (api *OwletAPI) passwordVerification() error {
	log.Println("Verifying password.")

	apiKey := regionInfo[api.region].APIKey
	urlStr := fmt.Sprintf("https://www.googleapis.com/identitytoolkit/v3/relyingparty/verifyPassword?key=%s", apiKey)

	data := url.Values{}
	data.Set("email", api.user)
	data.Set("password", api.password)
	data.Set("returnSecureToken", "true")
	encodedData := data.Encode()

	req, err := http.NewRequest("POST", urlStr, strings.NewReader(encodedData))
	if err != nil {
		return NewOwletError(fmt.Sprintf("Error creating request: %v", err))
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Android-Package", "com.owletcare.owletcare")
	req.Header.Set("X-Android-Cert", "2A3BC26DB0B8B0792DBE28E6FFDC2598F9B12B74")

	resp, err := api.httpClient.Do(req)

	if err != nil {
		return NewOwletConnectionError(fmt.Sprintf("Error sending request: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewOwletError(fmt.Sprintf("Error reading response: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return NewOwletError(fmt.Sprintf("Error parsing error response: %v", err))
		}

		switch errorResp.Error.Message {
		case "INVALID_PASSWORD":
			return NewOwletPasswordError("Incorrect Password")
		case "INVALID_EMAIL":
			return NewOwletEmailError("Invalid email")
		case "EMAIL_NOT_FOUND":
			return NewOwletEmailError("Email address not found")
		case "INVALID_LOGIN_CREDENTIALS":
			return NewOwletCredentialsError("Invalid login credentials")
		case "TOO_MANY_ATTEMPTS_TRY_LATER":
			return NewOwletAuthenticationError("Too many incorrect attempts")
		default:
			return NewOwletAuthenticationError(fmt.Sprintf("Generic identitytoolkit error: %s", errorResp.Error.Message))
		}
	}

	var respData struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.Unmarshal(body, &respData); err != nil {
		return NewOwletError(fmt.Sprintf("Error parsing response: %v", err))
	}

	api.refresh = respData.RefreshToken
	return nil
}

func (api *OwletAPI) refreshAuthentication() error {
	log.Println("Refreshing authentication tokens.")

	if api.refresh == "" {
		return NewOwletAuthenticationError("No refresh token supplied")
	}

	apiKey := regionInfo[api.region].APIKey
	urlStr := fmt.Sprintf("https://securetoken.googleapis.com/v1/token?key=%s", apiKey)

	data := map[string]interface{}{
		"grant_type":    "refresh_token",
		"refresh_token": api.refresh,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return NewOwletError(fmt.Sprintf("Error marshaling JSON: %v", err))
	}

	maxRetries := 5
	retryDelay := time.Second * 2

	var req *http.Request
	for i := 0; i <= maxRetries; i++ {
		req, err = http.NewRequest("POST", urlStr, bytes.NewBuffer(jsonData))
		if err == nil {
			break
		}

		if i < maxRetries {
			log.Printf("Attempt %d failed to create request: %v", i+1, err)
			time.Sleep(retryDelay)
			retryDelay *= 2
		}

		if i == maxRetries {
			return NewOwletError(fmt.Sprintf("Error creating request after %d attempts: %v", maxRetries, err))
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Android-Package", "com.owletcare.owletcare")
	req.Header.Set("X-Android-Cert", "2A3BC26DB0B8B0792DBE28E6FFDC2598F9B12B74")

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return NewOwletConnectionError(fmt.Sprintf("Error sending request: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewOwletError(fmt.Sprintf("Error reading response: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		return NewOwletAuthenticationError("Refresh token not valid")
	}

	var respData struct {
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &respData); err != nil {
		return NewOwletError(fmt.Sprintf("Error parsing response: %v", err))
	}

	api.refresh = respData.RefreshToken

	miniToken, err := api.getMiniToken(respData.IDToken)
	if err != nil {
		return err
	}

	return api.tokenSignIn(miniToken)
}

func (api *OwletAPI) getMiniToken(idToken string) (string, error) {
	log.Println("Getting mini authentication token.")

	url := regionInfo[api.region].URLMini

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", NewOwletError(fmt.Sprintf("Error creating request: %v", err))
	}

	req.Header.Set("Authorization", idToken)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return "", NewOwletConnectionError(fmt.Sprintf("Error sending request: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", NewOwletAuthenticationError("Invalid id token")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", NewOwletError(fmt.Sprintf("Error reading response: %v", err))
	}

	var respData struct {
		MiniToken string `json:"mini_token"`
	}
	if err := json.Unmarshal(body, &respData); err != nil {
		return "", NewOwletError(fmt.Sprintf("Error parsing response: %v", err))
	}

	return respData.MiniToken, nil
}

func (api *OwletAPI) tokenSignIn(miniToken string) error {
	log.Println("Getting authentication tokens.")

	url := regionInfo[api.region].URLSignin

	data := map[string]string{
		"app_id":     regionInfo[api.region].AppID,
		"app_secret": regionInfo[api.region].AppSecret,
		"provider":   "owl_id",
		"token":      miniToken,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return NewOwletError(fmt.Sprintf("Error marshaling data: %v", err))
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return NewOwletError(fmt.Sprintf("Error creating request: %v", err))
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return NewOwletConnectionError(fmt.Sprintf("Error sending request: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewOwletError(fmt.Sprintf("Error reading response: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return NewOwletAuthenticationError("Invalid mini token")
		case http.StatusNotFound:
			return NewOwletAuthenticationError("404 error - contact Ayla")
		default:
			return NewOwletAuthenticationError("Generic error - contact Ayla")
		}
	}

	var respData struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &respData); err != nil {
		return NewOwletError(fmt.Sprintf("Error parsing response: %v", err))
	}

	api.authToken = respData.AccessToken
	api.expiry = time.Now().Add(time.Duration(respData.ExpiresIn-60) * time.Second)

	api.headers["Authorization"] = "auth_token " + api.authToken

	return nil
}

func (api *OwletAPI) validateAuthentication() error {
	log.Println("Validating authentication token.")

	url := regionInfo[api.region].URLBase
	req, err := http.NewRequest("GET", url+"/devices.json", nil)
	if err != nil {
		return err
	}

	for key, value := range api.headers {
		req.Header.Set(key, value)
	}

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		log.Println("Invalid token; refreshing...")
		api.authToken = ""
		err = api.Authenticate()
		if err != nil {
			return err
		}
	}

	return nil
}

func (api *OwletAPI) Tokens() TokenInfo {
	return TokenInfo{
		APIToken: api.authToken,
		Expiry:   api.expiry,
		Refresh:  api.refresh,
	}
}

func (api *OwletAPI) GetDevices() (map[string]interface{}, error) {
	log.Println("Getting devices.")

	tempTokens := api.Tokens()
	maxRetries := 5
	retryDelay := time.Second * 2

	var devices []byte
	var err error

	for i := 0; i <= maxRetries; i++ {
		devices, err = api.request("GET", "/devices.json", nil, nil)
		if err == nil {
			break
		}

		if i < maxRetries {
			log.Printf("Attempt %d failed, retrying in %v: %v", i+1, retryDelay, err)
			time.Sleep(retryDelay)
			retryDelay *= 2
		}

		if i == maxRetries {
			return nil, NewOwletError(fmt.Sprintf("Error creating request after %d attempts: %v", maxRetries, err))
		}
	}

	if err != nil {
		return nil, err
	}

	var deviceList []Device
	err = json.Unmarshal(devices, &deviceList)
	if err != nil {
		return nil, err
	}

	if len(deviceList) < 1 {
		return nil, errors.New("No devices found")
	}

	result := map[string]interface{}{
		"response": deviceList,
		"tokens":   api.CheckTokens(tempTokens),
	}

	return result, nil
}

func (api *OwletAPI) Activate(deviceSerial string) error {
	data := map[string]interface{}{
		"datapoint": map[string]interface{}{
			"metadata": map[string]interface{}{},
			"value":    1,
		},
	}

	log.Printf("Activating device %s.", deviceSerial)
	_, err := api.request("POST", fmt.Sprintf("/dsns/%s/properties/APP_ACTIVE/datapoints.json", deviceSerial), data, nil)
	return err
}

func (api *OwletAPI) GetProperties(deviceSerial string) (map[string]interface{}, error) {
	tempTokens := api.Tokens()

	// Activate device
	err := api.Activate(deviceSerial)
	if err != nil {
		return nil, err
	}

	response, err := api.request("GET", fmt.Sprintf("/dsns/%s/properties.json", deviceSerial), nil, nil)
	if err != nil {
		return nil, err
	}

	var propertyList []PropertyMap
	err = json.Unmarshal(response, &propertyList)
	if err != nil {
		return nil, err
	}

	properties := make(map[string]Property)
	for _, prop := range propertyList {
		properties[prop.Property.Name] = prop.Property
	}

	result := map[string]interface{}{
		"response": properties,
		"tokens":   api.CheckTokens(tempTokens),
	}

	return result, nil
}

func (api *OwletAPI) CheckTokens(tempTokens TokenInfo) *TokenInfo {
	currentTokens := api.Tokens()

	if !tokenInfoEqual(tempTokens, currentTokens) {
		return &currentTokens
	}

	return nil
}

func (api *OwletAPI) request(method, url string, data interface{}, additionalHeaders map[string]string) (json.RawMessage, error) {
	maxRetries := 10
	baseDelay := time.Second * 5

	var resp *http.Response
	var body []byte
	var err error

	for i := 0; i <= maxRetries; i++ {
		log.Printf("Requesting %s, attempt %d", url, i+1)
		if i > 0 {
			delay := time.Duration(i) * baseDelay
			time.Sleep(delay)
		}

		err = api.validateAuthentication()
		if err != nil {
			return nil, err
		}

		baseUrl := regionInfo[api.region].URLBase

		var req *http.Request
		if data != nil {
			jsonData, err := json.Marshal(data)
			if err != nil {
				return nil, err
			}
			req, err = http.NewRequest(method, baseUrl+url, bytes.NewBuffer(jsonData))
		} else {
			req, err = http.NewRequest(method, baseUrl+url, nil)
		}
		if err != nil {
			return nil, err
		}

		for key, value := range api.headers {
			req.Header.Set(key, value)
		}

		for key, value := range additionalHeaders {
			req.Header.Set(key, value)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err = api.httpClient.Do(req)
		if err != nil {
			if i == maxRetries {
				return nil, fmt.Errorf("Max retries reached: %w", err)
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			if i == maxRetries {
				return nil, fmt.Errorf("Max retries reached - error sending request: %d", resp.StatusCode)
			}
			continue
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			if i == maxRetries {
				return nil, fmt.Errorf("max retries reached: %w", err)
			}
			continue
		}

		return body, nil
	}
	return nil, fmt.Errorf("Unexpected error: all retries failed")
}
