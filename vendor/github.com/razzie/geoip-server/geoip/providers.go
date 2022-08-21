package geoip

import (
	"strings"
)

// Providers is a map of known providers
var Providers = map[string]*Provider{
	"freegeoip.app": &Provider{
		Name:        "freegeoip.app",
		TemplateURL: "https://freegeoip.app/json/%[1]s",
		Mappings: Location{
			CountryCode: "country_code",
			Country:     "country_name",
			RegionCode:  "region_code",
			Region:      "region_name",
			City:        "city",
			TimeZone:    "time_zone",
		},
	},
	"db-ip.com": &Provider{
		Name:        "db-ip.com",
		TemplateURL: "http://api.db-ip.com/v2/free/%[1]s",
		Mappings: Location{
			CountryCode: "countryCode",
			Country:     "countryName",
			RegionCode:  "stateProvCode",
			Region:      "stateProv",
			City:        "city",
		},
	},
	"keycdn.com": &Provider{
		Name:        "keycdn.com",
		TemplateURL: "https://tools.keycdn.com/geo.json?host=%[1]s",
		Mappings: Location{
			CountryCode: "data.geo.country_code",
			Country:     "data.geo.country_name",
			RegionCode:  "data.geo.region_code",
			Region:      "data.geo.region_name",
			City:        "data.geo.city",
			TimeZone:    "data.geo.time_zone",
			ISP:         "data.geo.isp",
		},
	},
	"ip-api.com": &Provider{
		Name:        "ip-api.com",
		TemplateURL: "http://ip-api.com/json/%[1]s",
		Mappings: Location{
			CountryCode: "countryCode",
			Country:     "country",
			RegionCode:  "region",
			Region:      "regionName",
			City:        "city",
			TimeZone:    "timezone",
			ISP:         "isp",
		},
	},
	"ipinfo.io": &Provider{
		Name:        "ipinfo.io",
		TemplateURL: "http://ipinfo.io/%[1]s?token=%[2]s",
		Mappings: Location{
			Country:    "country",
			RegionCode: "region",
			City:       "city",
			TimeZone:   "timezone",
			ISP:        "org",
		},
	},
}

// GetClients creates new clients from the comma separated list of provider names
// Parameters are supported too, eg: "ipinfo.io xxtokenxx, ip-api.com, freegeoip.app"
func GetClients(providerList string) (clients []Client) {
	for _, provider := range strings.Split(providerList, ",") {
		params := strings.Fields(provider)
		if len(params) == 0 {
			continue
		}
		p, ok := Providers[params[0]]
		if !ok {
			continue
		}
		client := p.NewClient(params[1:]...)
		clients = append(clients, client)
	}
	return
}
