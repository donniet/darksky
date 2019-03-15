package darksky

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
	Latitude  float32     `json:"latitude"`
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
Flags give additional metadata from Darksky
*/
type Flags struct {
	Sources        []string `json:"sources"`
	NearestStation float32  `json:"nearest-station"`
	Units          string   `json:"units"`
}

type tempResponse struct {
	Latitude  float32          `json:"latitude"`
	Longitude float32          `json:"longitude"`
	Timezone  string           `json:"timezone"`
	Currently *json.RawMessage `json:"currently"`
	Minutely  *json.RawMessage `json:"minutely"`
	Hourly    *json.RawMessage `json:"hourly"`
	Daily     *json.RawMessage `json:"daily"`
	Flags     Flags            `json:"flags"`
	Offset    int              `json:"offset"`
}

func (r *Response) UnmarshalJSON(b []byte) error {
	tr := tempResponse{}

	if err := json.Unmarshal(b, &tr); err != nil {
		return err
	}

	r.Latitude = tr.Latitude
	r.Longitude = tr.Longitude
	r.Timezone = tr.Timezone
	r.Flags = tr.Flags
	r.Offset = tr.Offset

	ctx := context.TODO()

	switch r.Flags.Units {
	case "us":
		ctx = context.WithValue(ctx, "units/temperature", "fahrenheit")
	default:
		ctx = context.WithValue(ctx, "units/temperature", "celsius")
	}

	if tr.Currently == nil {
	} else if err := r.Currently.UnmarshalJSONWithContext(ctx, *tr.Currently); err != nil {
		return err
	}

	if tr.Minutely == nil {
	} else if err := r.Minutely.UnmarshalJSONWithContext(ctx, *tr.Minutely); err != nil {
		return err
	}

	if tr.Hourly == nil {
	} else if err := r.Hourly.UnmarshalJSONWithContext(ctx, *tr.Hourly); err != nil {
		return err
	}

	if tr.Daily == nil {
	} else if err := r.Daily.UnmarshalJSONWithContext(ctx, *tr.Daily); err != nil {
		return err
	}

	return nil
}

func (r Response) MarshalJSON() ([]byte, error) {
	ctx := context.TODO()

	switch r.Flags.Units {
	case "us":
		ctx = context.WithValue(ctx, "units/temperature", "fahrenheit")
	default:
		ctx = context.WithValue(ctx, "units/temperature", "celsius")
	}

	tr := tempResponse{
		Latitude:  r.Latitude,
		Longitude: r.Longitude,
		Timezone:  r.Timezone,
		Flags:     r.Flags,
		Offset:    r.Offset,
	}

	if b, err := r.Currently.MarshalJSONWithContext(ctx); err != nil {
		return nil, err
	} else {
		tr.Currently = (*json.RawMessage)(&b)
	}

	if b, err := r.Minutely.MarshalJSONWithContext(ctx); err != nil {
		return nil, err
	} else {
		tr.Minutely = (*json.RawMessage)(&b)
	}

	if b, err := r.Hourly.MarshalJSONWithContext(ctx); err != nil {
		return nil, err
	} else {
		tr.Hourly = (*json.RawMessage)(&b)
	}

	if b, err := r.Daily.MarshalJSONWithContext(ctx); err != nil {
		return nil, err
	} else {
		tr.Daily = (*json.RawMessage)(&b)
	}

	return json.Marshal(tr)
}

/*
Temperature has a default value of absolute zero
*/
type Temperature float64

/*
UnmarshalJSON unmarshals temperatures
*/
func (t *Temperature) UnmarshalJSON(b []byte) error {
	var temp float64

	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}

	// store temp as kelvin
	*t = FromCelsuis(temp)

	return nil
}

/*
UnmarshalJSONWithContext unmarshalls temperatures using the provided context
*/
func (t *Temperature) UnmarshalJSONWithContext(ctx context.Context, b []byte) error {
	u := "kelvin"

	units := ctx.Value("units/temperature")
	if units == nil {
	} else if u = units.(string); u == "" {
		u = "kelvin"
	}

	var temp float64

	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}

	switch u {
	case "celsius":
		*t = FromCelsuis(temp)
	case "kelvin":
		*t = Temperature(temp)
	case "fahrenheit":
		*t = FromFahrenheit(temp)
	default:
		return fmt.Errorf("unknown temperature unit: %s", u)
	}

	return nil
}

/*
MarshalJSONWithConrtext marshals temperatures using the provided context
*/
func (t *Temperature) MarshalJSONWithContext(ctx context.Context) ([]byte, error) {
	u := "kelvin"

	units := ctx.Value("units/temperature")
	if units == nil {
	} else if u = units.(string); u == "" {
		u = "kelvin"
	}

	log.Printf("temp: %f", t.Fahrenheit())

	switch u {
	case "kelvin":
		return json.Marshal(float32(*t))
	case "celsius":
		return json.Marshal(t.Celsius())
	case "fahrenheit":
		return json.Marshal(t.Fahrenheit())
	default:
		return []byte{}, fmt.Errorf("unknown temperature unit: %s", u)
	}
}

/*
MarshalJSON for temperatures as fahrenheit
*/
func (t Temperature) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Fahrenheit())
}

/*
Celsius converts the kelvin stored temperature to celsius
*/
func (t Temperature) Celsius() float64 {
	return float64(t) - 273.15
}

/*
String outputs the temperature as fahrenheit
*/
func (t Temperature) String() string {
	return fmt.Sprintf("%f", t.Fahrenheit())
}

/*
FromFahrenheit creates a temperature object from a fahrenheight float
*/
func FromFahrenheit(temp float64) Temperature {
	return Temperature((temp-32)*5./9. + 273.15)
}

/*
FromCelsuis creates a temperature object from a celsuis float
*/
func FromCelsuis(temp float64) Temperature {
	return Temperature(temp + 273.15)
}

/*
Fahrenheit converts the kelvin stored temperature to fahrenheit
*/
func (t Temperature) Fahrenheit() float64 {
	return (float64(t)-273.15)*9./5. + 32
}

/*
Data is a struct to hold a set of weather data
*/
type Data struct {
	Time                 time.Time   `json:"time"`
	Summary              string      `json:"summary,omitempty"`
	Icon                 string      `json:"icon"`
	NearestStormDistance float32     `json:"nearestStormDistance"`
	PrecipIntensity      float32     `json:"precipIntensity"`
	PrecipProbability    float32     `json:"precipProbability"`
	PrecipType           string      `json:"precipType,omitempty"`
	Temperature          Temperature `json:"temperature"`
	ApparentTemperature  Temperature `json:"apparentTemperature"`
	TemperatureLow       Temperature `json:"temperatureLow"`
	TemperatureHighTime  time.Time   `json:"temperatureHighTime"`
	TemperatureHigh      Temperature `json:"temperatureHigh"`
	TemperatureLowTime   time.Time   `json:"temperatureLowTime"`
	DewPoint             Temperature `json:"dewPoint"`
	Humidity             float32     `json:"humidity"`
	Pressure             float32     `json:"pressure"`
	WindSpeed            float32     `json:"windSpeed"`
	WindGust             float32     `json:"windGust"`
	WindBearing          float32     `json:"windBearing"`
	CloudCover           float32     `json:"cloudCover"`
	UVIndex              float32     `json:"uvIndex"`
	Visibility           float32     `json:"visibility"`
	Ozone                float32     `json:"ozone"`
}

/*
tempData is a struct that serves as a temperary holding place for data and unmarshalled bytes
*/
type tempData struct {
	Time                 *json.RawMessage `json:"time"`
	Summary              string           `json:"summary,omitempty"`
	Icon                 string           `json:"icon"`
	NearestStormDistance float32          `json:"nearestStormDistance"`
	PrecipIntensity      float32          `json:"precipIntensity"`
	PrecipProbability    float32          `json:"precipProbability"`
	PrecipType           string           `json:"precipType,omitempty"`
	Temperature          *json.RawMessage `json:"temperature"`
	ApparentTemperature  *json.RawMessage `json:"apparentTemperature"`
	TemperatureLow       *json.RawMessage `json:"temperatureLow,omitempty"`
	TemperatureHighTime  *json.RawMessage `json:"temperatureHighTime,omitempty"`
	TemperatureHigh      *json.RawMessage `json:"temperatureHigh,omitempty"`
	TemperatureLowTime   *json.RawMessage `json:"temperatureLowTime,omitempty"`
	DewPoint             *json.RawMessage `json:"dewPoint"`
	Humidity             float32          `json:"humidity"`
	Pressure             float32          `json:"pressure"`
	WindSpeed            float32          `json:"windSpeed"`
	WindGust             float32          `json:"windGust"`
	WindBearing          float32          `json:"windBearing"`
	CloudCover           float32          `json:"cloudCover"`
	UVIndex              float32          `json:"uvIndex"`
	Visibility           float32          `json:"visibility"`
	Ozone                float32          `json:"ozone"`
}

func (d *Data) UnmarshalJSONWithContext(ctx context.Context, b []byte) error {
	td := tempData{}

	if err := json.Unmarshal(b, &td); err != nil {
		return err
	}

	d.Summary = td.Summary
	d.Icon = td.Icon
	d.NearestStormDistance = td.NearestStormDistance
	d.PrecipIntensity = td.PrecipIntensity
	d.PrecipProbability = td.PrecipProbability
	d.PrecipType = td.PrecipType
	d.Humidity = td.Humidity
	d.Pressure = td.Pressure
	d.WindSpeed = td.WindSpeed
	d.WindGust = td.WindGust
	d.WindBearing = td.WindBearing
	d.CloudCover = td.CloudCover
	d.UVIndex = td.UVIndex
	d.Visibility = td.Visibility
	d.Ozone = td.Ozone

	var t int64

	if td.Time == nil {
	} else if err := json.Unmarshal(*td.Time, &t); err != nil {
		return err
	} else {
		d.Time = time.Unix(t, 0)
	}

	if td.TemperatureHighTime == nil {
	} else if err := json.Unmarshal(*td.TemperatureHighTime, &t); err != nil {
		return err
	} else {
		d.TemperatureHighTime = time.Unix(t, 0)
	}

	if td.TemperatureLowTime == nil {
	} else if err := json.Unmarshal(*td.TemperatureLowTime, &t); err != nil {
		return err
	} else {
		d.TemperatureLowTime = time.Unix(t, 0)
	}

	if td.Temperature == nil {
	} else if err := d.Temperature.UnmarshalJSONWithContext(ctx, *td.Temperature); err != nil {
		return err
	}

	if td.ApparentTemperature == nil {
	} else if err := d.ApparentTemperature.UnmarshalJSONWithContext(ctx, *td.ApparentTemperature); err != nil {
		return err
	}

	if td.DewPoint == nil {
	} else if err := d.DewPoint.UnmarshalJSONWithContext(ctx, *td.DewPoint); err != nil {
		return err
	}

	if td.TemperatureLow == nil {
	} else if err := d.TemperatureLow.UnmarshalJSONWithContext(ctx, *td.TemperatureLow); err != nil {
		return err
	}

	if td.TemperatureHigh == nil {
	} else if err := d.TemperatureHigh.UnmarshalJSONWithContext(ctx, *td.TemperatureHigh); err != nil {
		return err
	}

	return nil
}

func (d *Data) MarshalJSONWithContext(ctx context.Context) ([]byte, error) {
	td := tempData{
		Summary:              d.Summary,
		Icon:                 d.Icon,
		NearestStormDistance: d.NearestStormDistance,
		PrecipIntensity:      d.PrecipIntensity,
		PrecipProbability:    d.PrecipProbability,
		PrecipType:           d.PrecipType,
		Humidity:             d.Humidity,
		Pressure:             d.Pressure,
		WindSpeed:            d.WindSpeed,
		WindGust:             d.WindGust,
		WindBearing:          d.WindBearing,
		CloudCover:           d.CloudCover,
		UVIndex:              d.UVIndex,
		Visibility:           d.Visibility,
		Ozone:                d.Ozone,
	}

	if d.Time == (time.Time{}) {
	} else if b, err := json.Marshal(d.Time.Unix()); err != nil {
		return nil, err
	} else {
		td.Time = (*json.RawMessage)(&b)
	}

	if d.TemperatureHighTime == (time.Time{}) {
	} else if b, err := json.Marshal(d.TemperatureHighTime.Unix()); err != nil {
		return nil, err
	} else {
		td.TemperatureHighTime = (*json.RawMessage)(&b)
	}

	if d.TemperatureLowTime == (time.Time{}) {
	} else if b, err := json.Marshal(d.TemperatureLowTime.Unix()); err != nil {
		return nil, err
	} else {
		td.TemperatureLowTime = (*json.RawMessage)(&b)
	}

	if d.Temperature == (Temperature(0)) {
	} else if b, err := d.Temperature.MarshalJSONWithContext(ctx); err != nil {
		return nil, err
	} else {
		td.Temperature = (*json.RawMessage)(&b)
	}

	if d.ApparentTemperature == (Temperature(0)) {
	} else if b, err := d.ApparentTemperature.MarshalJSONWithContext(ctx); err != nil {
		return nil, err
	} else {
		td.ApparentTemperature = (*json.RawMessage)(&b)
	}

	if d.DewPoint == (Temperature(0)) {
	} else if b, err := d.DewPoint.MarshalJSONWithContext(ctx); err != nil {
		return nil, err
	} else {
		td.DewPoint = (*json.RawMessage)(&b)
	}

	if d.TemperatureHigh == (Temperature(0)) {
	} else if b, err := d.TemperatureHigh.MarshalJSONWithContext(ctx); err != nil {
		return nil, err
	} else {
		td.TemperatureHigh = (*json.RawMessage)(&b)
	}

	if d.TemperatureLow == (Temperature(0)) {
	} else if b, err := d.TemperatureLow.MarshalJSONWithContext(ctx); err != nil {
		return nil, err
	} else {
		td.TemperatureLow = (*json.RawMessage)(&b)
	}

	return json.Marshal(td)
}

/*
DataSummary wraps an array of Data elements along with an icon and summary
*/
type DataSummary struct {
	Summary string `json:"summary"`
	Icon    string `json:"icon"`
	Data    []Data `json:"data"`
}

type tempDataSummary struct {
	Summary string             `json:"summary"`
	Icon    string             `json:"icon"`
	Data    []*json.RawMessage `json:"data"`
}

func (ds *DataSummary) MarshalJSONWithContext(ctx context.Context) ([]byte, error) {
	td := tempDataSummary{
		Summary: ds.Summary,
		Icon:    ds.Icon,
	}

	for _, d := range ds.Data {
		if b, err := d.MarshalJSONWithContext(ctx); err != nil {
			return nil, err
		} else {
			td.Data = append(td.Data, (*json.RawMessage)(&b))
		}
	}

	return json.Marshal(td)
}

func (ds *DataSummary) UnmarshalJSONWithContext(ctx context.Context, b []byte) error {
	td := tempDataSummary{}

	if err := json.Unmarshal(b, &td); err != nil {
		return err
	}

	ds.Summary = td.Summary
	ds.Icon = td.Icon
	ds.Data = nil

	for _, b := range td.Data {
		d := Data{}

		if b == nil {
		} else if err := d.UnmarshalJSONWithContext(ctx, *b); err != nil {
			return err
		}

		ds.Data = append(ds.Data, d)
	}

	return nil
}
