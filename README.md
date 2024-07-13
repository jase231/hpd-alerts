# Henrico County PD Alerts

This tool scrapes [Henrico County's Active Police Calls](https://activecalls.henrico.us) and displays them on a Semantic + LeafletJS front-end.

# Features:
- Access to the underlying scraped data using the /getAlerts endpoint
- Configurable scraping intervals
- Supports multiple geocoding providers
  - Google Maps API for accurate, [free-ish](https://cloud.google.com/free?hl=en) geocoding
  - OpenStreetMap's Nominatim for decently-accurate, free geocoding
 
# A note on Nominatim
- The tool is currently configured to use the Nominatim instance hosted at [geocoding.ai](https://nominatim.geocoding.ai/search.html)
- Nominatim [does not geocode intersections](https://github.com/osm-search/Nominatim/issues/123). Sometimes incidents on the county's portal will use an intersection as their location. When running in Nominatim mode, the tool will provide an approximate location for the incident by geocoding the first address of the intersection (e.g. "Lauderdale Dr / Church Rd" will be geocoded as "Lauderdale Dr". This isn't very accurate, but is better then outright failure with incidents rendering [somewhere in the Atlantic](https://www.google.com/maps/place/0%C2%B000'00.0%22N+0%C2%B000'00.0%22E).

# Usage
- To build, simply `go build` in the project's root directory
- To run, make sure to specify the geocoding provider and the scraping interval (60 second default), e.g.:
  - `./hpd-alerts --provider nominatim --interval 180`
  - `./hpd-alerts --provider google --interval 120`
- If you intend to run in Google Maps mode, make sure to provide your own API key by setting the `MAPS_TOKEN` environment variable:
 `export MAPS_TOKEN=xyz`

# Thank you to
-  Efron Licht for his blog series [Backend from the Beginning](https://eblog.fly.dev/backendbasics.html)
-  Paul Fournel for [this very cool encoded audio alert](https://stackoverflow.com/a/23395136)
