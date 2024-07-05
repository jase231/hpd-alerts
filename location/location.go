package location

import (
	"context"
	"fmt"

	"github.com/jase231/hpd-alerts/models"
	"googlemaps.github.io/maps"
)

func getCoordinates(incidentPtr *models.Incident, mapsToken string) error {
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

	fmt.Println("DEBUG: Google Request Made")
	incidentPtr.Location.Lat = resp[0].Geometry.Location.Lat
	incidentPtr.Location.Lng = resp[0].Geometry.Location.Lng
	return nil
}

func PopulateLocation(incidentsPtr map[string]models.Incident, mapsToken string) error {
	for key := range incidentsPtr {
		incident := (incidentsPtr)[key]

		if err := getCoordinates(&incident, mapsToken); err != nil {
			return err
		}

		(incidentsPtr)[key] = incident // set updated incident back to original map
	}

	return nil
}
