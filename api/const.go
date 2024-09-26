package api

type RegionInfo struct {
	URLMini    string
	URLSignin  string
	URLRefresh string
	URLBase    string
	APIKey     string
	AppID      string
	AppSecret  string
}

var regionInfo = map[string]RegionInfo{
	"world": {
		URLMini:    "https://ayla-sso.owletdata.com/mini/",
		URLSignin:  "https://user-field-1a2039d9.aylanetworks.com/api/v1/token_sign_in",
		URLRefresh: "https://user-field-1a2039d9.aylanetworks.com/users/refresh_token.json",
		URLBase:    "https://ads-field-1a2039d9.aylanetworks.com/apiv1",
		APIKey:     "AIzaSyCsDZ8kWxQuLJAMVnmEhEkayH1TSxKXfGA",
		AppID:      "sso-prod-3g-id",
		AppSecret:  "sso-prod-UEjtnPCtFfjdwIwxqnC0OipxRFU",
	},
	"europe": {
		URLMini:    "https://ayla-sso.eu.owletdata.com/mini/",
		URLSignin:  "https://user-field-eu-1a2039d9.aylanetworks.com/api/v1/token_sign_in",
		URLRefresh: "https://user-field-eu-1a2039d9.aylanetworks.com/users/refresh_token.json",
		URLBase:    "https://ads-field-eu-1a2039d9.aylanetworks.com/apiv1",
		APIKey:     "AIzaSyDm6EhV70wudwN3iOSq3vTjtsdGjdFLuuM",
		AppID:      "OwletCare-Android-EU-fw-id",
		AppSecret:  "OwletCare-Android-EU-JKupMPBoj_Npce_9a95Pc8Qo0Mw",
	},
}
