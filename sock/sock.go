package sock

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"github.com/edfincham/owlet-sock-scraper/api"
)

type Vitals struct {
	Ox   int    `json:"ox"`   // Oxygen level
	Hr   int    `json:"hr"`   // Heart rate
	Mv   int    `json:"mv"`   // Movement
	Sc   int    `json:"sc"`   // Skin conductance?
	St   int    `json:"st"`   // Skin temperature?
	Bso  int    `json:"bso"`  // Base station online?
	Bat  int    `json:"bat"`  // Battery level
	Btt  int    `json:"btt"`  // Battery time?
	Chg  int    `json:"chg"`  // Charging?
	Aps  int    `json:"aps"`  // Active power source?
	Alrt int    `json:"alrt"` // Alert
	Ota  int    `json:"ota"`  // Over-the-air update?
	Srf  int    `json:"srf"`  // Signal radio frequency?
	Rsi  int    `json:"rsi"`  // Received signal intensity?
	Sb   int    `json:"sb"`   // Sleep behavior?
	Ss   int    `json:"ss"`   // Sleep state?
	Mvb  int    `json:"mvb"`  // Movement behavior?
	Mst  int64  `json:"mst"`  // Timestamp (Unix time)
	Oxta int    `json:"oxta"` // Oxygen trend average?
	Onm  int    `json:"onm"`  // On movement?
	Bsb  int    `json:"bsb"`  // Base station behavior?
	Mrs  int    `json:"mrs"`  // Movement recognition state?
	Hw   string `json:"hw"`   // Hardware version
}

type Sock struct {
	api              *api.OwletAPI
	name             string
	model            string
	Serial           string
	oemModel         string
	swVersion        string
	mac              string
	lanIP            string
	connectionStatus string
	deviceType       string
	manufModel       string
	version          *int
	revision         *int
	rawProperties    map[string]api.Property
	properties       Vitals
}

func NewSock(owletApi *api.OwletAPI, data api.Device) *Sock {
	return &Sock{
		api:              owletApi,
		name:             data.Device.ProductName,
		model:            data.Device.Model,
		Serial:           data.Device.DSN,
		oemModel:         data.Device.OEMModel,
		swVersion:        data.Device.SWVersion,
		mac:              data.Device.MAC,
		lanIP:            data.Device.LanIP,
		connectionStatus: data.Device.ConnectionStatus,
		deviceType:       data.Device.DeviceType,
		manufModel:       data.Device.ManufModel,
		rawProperties:    map[string]api.Property{},
		properties:       Vitals{},
	}
}

func (s *Sock) GetProperty(property string) interface{} {
	v := reflect.ValueOf(s.properties)
	if v.Kind() == reflect.Struct {
		field := v.FieldByName(property)
		if field.IsValid() {
			return field.Interface()
		}
	}
	return nil
}

func (s *Sock) checkVersion() {
	version := 0
	if _, ok := s.rawProperties["REAL_TIME_VITALS"]; ok {
		version = 3
	} else if _, ok := s.rawProperties["CHARGE_STATUS"]; ok {
		version = 2
	}
	s.version = &version
}

func (s *Sock) checkRevision() error {
	if oemSockVersion, ok := s.rawProperties["oem_sock_version"]; ok {
		if value, ok := oemSockVersion.Value.(string); ok {
			var revisionJSON map[string]interface{}
			err := json.Unmarshal([]byte(value), &revisionJSON)
			if err != nil {
				return err
			}
			if rev, ok := revisionJSON["rev"].(float64); ok {
				revision := int(rev)
				s.revision = &revision
			}
		}
	}
	return nil
}

func (s *Sock) UpdateVitals() (*Vitals, error) {
	log.Printf("Updating vitals for device %s", s.Serial)
	properties, err := s.api.GetProperties(s.Serial)
	if err != nil {
		return nil, err
	}

	s.rawProperties = properties["response"].(map[string]api.Property)
	if s.version == nil {
		s.checkVersion()
	}
	if s.revision == nil && s.version != nil && *s.version == 3 {
		err := s.checkRevision()
		if err != nil {
			return nil, err
		}
	}

	if s.version == nil {
		return nil, fmt.Errorf("Version is nil")
	}

	if *s.version == 3 {
		vitalsRaw, ok := s.rawProperties["REAL_TIME_VITALS"]
		if !ok {
			return nil, fmt.Errorf("Could not retrieve vitalsRaw")
		}

		valueStr, ok := vitalsRaw.Value.(string)
		if !ok {
			return nil, fmt.Errorf("Unexpected type for REAL_TIME_VITALS value: %T", vitalsRaw.Value)
		}

		var vitals Vitals
		err := json.Unmarshal([]byte(valueStr), &vitals)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshalling string into RealTimeVitals: %w", err)
		}

		return &vitals, nil
	}

	return nil, fmt.Errorf("Unsupported version: %d", *s.version)
}
