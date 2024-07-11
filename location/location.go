package location

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jase231/hpd-alerts/models"
	"googlemaps.github.io/maps"
)

type NominatimResponse struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

func PopulateLocation(incidentsPtr map[string]models.Incident, mapsToken string) error {
	for key := range incidentsPtr {
		incident := (incidentsPtr)[key]

		if mapsToken == "nominatim" {
			if err := nominatimGeocode(&incident); err != nil {
				return err
			}
		} else {
			if err := googleGeocode(&incident, mapsToken); err != nil {
				return err
			}

		}

		(incidentsPtr)[key] = incident // set updated incident back to original map
	}

	return nil
}

func googleGeocode(incidentPtr *models.Incident, mapsToken string) error {
	client, err := maps.NewClient(maps.WithAPIKey(mapsToken))
	if err != nil {
		return fmt.Errorf("error creating Google Maps client: %v", err)
	}

	request := &maps.GeocodingRequest{
		Address: incidentPtr.Block + ", Henrico County, VA", // google maps is extremely forgiving for interpreting input here, so we can just use county's format directly
	}

	resp, err := client.Geocode(context.Background(), request)
	if err != nil {
		return fmt.Errorf("error requesting geocode from google: %v", err)
	}

	log.Println("Google Request Made")
	incidentPtr.Location.Lat = resp[0].Geometry.Location.Lat
	incidentPtr.Location.Lng = resp[0].Geometry.Location.Lng
	return nil
}

func nominatimGeocode(incidentPtr *models.Incident) error {
	address := removeBlock(incidentPtr.Block) // nominatim fails with "Block" in the address

	if isIntersection(address) {
		incidentPtr.Intersection = true
		address = removeIntersection(address)
	}

	baseURL := "https://nominatim.openstreetmap.org/search"

	params := url.Values{}
	params.Add("street", address)
	params.Add("county", "Henrico County")
	params.Add("format", "json")
	params.Add("limit", "1")

	url := baseURL + "?" + params.Encode()

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// User-Agent per Nominatim usage policy
	req.Header.Set("User-Agent", "HPD-Alerts/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("nominatim API request failed with status code: %d", resp.StatusCode)
	}
	if resp.ContentLength == 2 {
		log.Printf("no nominatim results found for address: %s", address) // usually this is because the provided address is an intersection which nominatim doesn't handle well
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	var nominatimResp []NominatimResponse
	err = json.Unmarshal(body, &nominatimResp)
	if err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	lat := nominatimResp[0].Lat
	lon := nominatimResp[0].Lon

	incidentPtr.Location.Lat, _ = strconv.ParseFloat(lat, 64)
	incidentPtr.Location.Lng, _ = strconv.ParseFloat(lon, 64)

	time.Sleep(1 * time.Second) // ensure we don't abuse nominatim API
	return nil
}

func removeBlock(address string) string {
	sanitized := strings.Replace(address, "Block ", "", -1) // whitespace behind block to avoid double whitespace
	sanitized = strings.TrimSpace(sanitized)
	return sanitized
}

func isIntersection(address string) bool {
	return strings.Contains(address, "/")
}

// removes one of the intersecting roads in order to at least get an approximate geocode from nominatim
func removeIntersection(address string) string {
	return strings.Split(address, "/")[0]
}
