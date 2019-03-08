package darksky

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

const (
	defaultURLFormat = "https://api.darksky.net/forecast/%s/%f,%f?exclude=minutely&units=us"
	defaultTimeout   = 30 * time.Second
)

/*
Service houses the data to call the Darksky API
*/
type Service struct {
	URLFormat string
	Key       string
	Timeout   time.Duration
}

/*
NewService constructs a service from an API key
*/
func NewService(key string) *Service {
	return &Service{
		URLFormat: defaultURLFormat,
		Key:       key,
		Timeout:   defaultTimeout,
	}
}

/*
Get gets a response from darksky
*/
func (s *Service) Get(lat, long float32) (Response, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   s.Timeout,
				KeepAlive: s.Timeout,
			}).Dial,
			TLSHandshakeTimeout:   s.Timeout,
			ResponseHeaderTimeout: s.Timeout,
			ExpectContinueTimeout: s.Timeout,
		},
	}

	ret := Response{}

	if res, err := client.Get(fmt.Sprintf(s.URLFormat, s.Key, lat, long)); err != nil {
		return ret, err
	} else if res.StatusCode/100 != 2 {
		return ret, fmt.Errorf("invalid statuscode from darksky: %d", res.StatusCode)
	} else if b, err := ioutil.ReadAll(res.Body); err != nil {
		return ret, err
	} else if err := json.Unmarshal(b, &ret); err != nil {
		return ret, err
	}

	return ret, nil
}

/*
Response is the root level of the response from Darksky
*/
type Response struct {
	Lattitude float32     `json:"latitude"`
	Longitude float32     `json:"longitude"`
	Timezone  string      `json:"timezone"`
	Currently Data        `json:"currently"`
	Minutely  DataSummary `json:"minutely"`
	Hourly    DataSummary `json:"hourly"`
	Daily     DataSummary `json:"daily"`
	Flags     Flags       `json:"flags"`
	Offset    int         `json:"offset"`
}

/*
Time is a time.Time which Marshals and Unmarshals to Unix seconds
*/
type Time time.Time

/*
UnmarshalJSON unmarshals darksky.Time from unix seconds
*/
func (t *Time) UnmarshalJSON(b []byte) error {
	var sec int64

	if err := json.Unmarshal(b, &sec); err != nil {
		return err
	}

	*t = Time(time.Unix(sec, 0))
	return nil
}

/*
MarshalJSON marshals darksky.Time to unix seconds
*/
func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Unix())
}

/*
Data is a struct to hold a set of weather data
*/
type Data struct {
	Time                 Time    `json:"time"`
	Summary              string  `json:"summary,omitempty"`
	Icon                 string  `json:"icon"`
	NearestStormDistance float32 `json:"nearestStormDistance"`
	PrecipIntensity      float32 `json:"precipIntensity"`
	PrecipProbability    float32 `json:"precipProbability"`
	PrecipType           string  `json:"precipType,omitempty"`
	Temperature          float32 `json:"temperature"`
	TemperatureHigh      float32 `json:"temperatureHigh"`
	TemperatureHighTime  Time    `json:"temperatureHighTime"`
	TemperatureLow       float32 `json:"temperatureLow"`
	TemperatureLowTime   Time    `json:"temperatureLowTime"`
	ApparentTemperature  float32 `json:"apparentTemperature"`
	DewPoint             float32 `json:"dewPoint"`
	Humidity             float32 `json:"humidity"`
	Pressure             float32 `json:"pressure"`
	WindSpeed            float32 `json:"windSpeed"`
	WindGust             float32 `json:"windGust"`
	WindBearing          float32 `json:"windBearing"`
	CloudCover           float32 `json:"cloudCover"`
	UVIndex              float32 `json:"uvIndex"`
	Visibility           float32 `json:"visibility"`
	Ozone                float32 `json:"ozone"`
}

/*
DataSummary wraps an array of Data elements along with an icon and summary
*/
type DataSummary struct {
	Summary string `json:"summary"`
	Icon    string `json:"icon"`
	Data    []Data `json:"data"`
}

/*
Flags give additional metadata from Darksky
*/
type Flags struct {
	Sources        []string `json:"sources"`
	NearestStation float32  `json:"nearest-station"`
	Units          string   `json:"units"`
}
