package apis

import (
	"google.golang.org/appengine"
	"net/http"
	"strconv"
	"strings"
)

type Device struct {
	UserAgent      string
	AcceptLanguage string
	IP             string
	Country        string
	Region         string
	City           string
	CityLatLng     appengine.GeoPoint
}

func GetDevice(r *http.Request) Device {
	var lat, lng float64
	latlng := strings.Split(r.Header.Get("X-AppEngine-CityLatLong"), ",")
	if len(latlng) == 2 {
		lat, _ = strconv.ParseFloat(latlng[0], 64)
		lng, _ = strconv.ParseFloat(latlng[1], 64)
	}
	return Device{
		IP:             r.RemoteAddr,
		UserAgent:      r.UserAgent(),
		AcceptLanguage: r.Header.Get("accept-language"),
		City:           r.Header.Get("X-AppEngine-City"),
		Country:        r.Header.Get("X-AppEngine-Country"),
		Region:         r.Header.Get("X-AppEngine-Region"),
		CityLatLng:     appengine.GeoPoint{Lat: lat, Lng: lng},
	}
}
