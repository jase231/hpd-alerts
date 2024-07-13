package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/jase231/hpd-alerts/location"
	"github.com/jase231/hpd-alerts/models"
	"github.com/jase231/hpd-alerts/scraper"
)

type Error struct {
	Error string `json:"error"`
}

type Resp struct {
	Provider bool `json:"nominatim"`
}

type Server struct {
	incidents      map[string]models.Incident
	mu             sync.Mutex
	isRunning      bool
	nominatim      bool
	mapsToken      string
	scrapeInterval int
}

func NewServer(envToken string, scrapeInterval int) (*Server, error) {
	nominatim := false

	if envToken == "" {
		return nil, fmt.Errorf("missing Google Maps API token")
	} else if envToken == "nominatim" {
		nominatim = true
		log.Printf("Starting Server using Nominatim. Polling county every %d seconds.\n", scrapeInterval)
	} else {
		log.Printf("Starting Server using Google Maps. Polling county %d seconds.\n", scrapeInterval)
	}

	return &Server{
		incidents:      make(map[string]models.Incident),
		mapsToken:      envToken,
		isRunning:      true,
		nominatim:      nominatim,
		scrapeInterval: scrapeInterval,
	}, nil
}

func (s *Server) scrapeLoop() {
	ticker := time.NewTicker(time.Duration(s.scrapeInterval) * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		if !s.isRunning {
			continue
		}

		newIncidents := scraper.Scrape()
		scraper.RemoveStaleIncidents(s.incidents, newIncidents)
		s.mu.Lock()
		scraper.RemoveDuplicates(s.incidents, newIncidents)
		if err := location.PopulateLocation(newIncidents, s.mapsToken); err != nil {
			log.Fatalln("can't get locations, quitting...", err)
		}

		for k, v := range newIncidents {
			s.incidents[k] = v
		}
		s.mu.Unlock()

	}
}

func (s *Server) getAlerts(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		scraperError := fmt.Errorf("scraper is not running")
		s.writeError(w, scraperError, 403)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	json.NewEncoder(w).Encode(s.incidents)
}

func (s *Server) toggleScraper(w http.ResponseWriter) {
	s.isRunning = !s.isRunning
	w.WriteHeader(http.StatusOK)
}

func (s *Server) writeError(w http.ResponseWriter, err error, status int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Println("error:", status, err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Error{err.Error()})
}

func (s *Server) writeNominatimJSON(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Resp{s.nominatim})
}

func main() {
	// --provider flag for geocode provider
	// --interval flag for scrape interval
	var provider, envToken, scrapeIntervalString string
	flag.StringVar(&provider, "provider", "", "geocode provider to use (nominatim or google)")
	flag.StringVar(&scrapeIntervalString, "interval", "60", "interval in seconds to scrape for new incidents")
	flag.Parse()

	if provider == "nominatim" {
		envToken = "nominatim"
	} else if provider == "google" {
		envToken = os.Getenv("MAPS_TOKEN")
	} else {
		log.Fatalln("invalid provider, quitting...")
	}

	scrapeInterval, err := strconv.Atoi(scrapeIntervalString)
	if err != nil || scrapeInterval < 10 { // 10 seconds to avoid spamming the county
		log.Fatalln("invalid interval, quitting...")
	}

	server, err := NewServer(envToken, scrapeInterval)
	if err != nil {
		log.Fatalln("couldn't spawn server, quitting:", err)
	}
	go server.scrapeLoop()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/", "/index.html":
			http.ServeFile(w, r, "static/index.html")
		case "/henrico.geojson":
			http.ServeFile(w, r, "static/henrico.geojson")
		case "/script.js":
			http.ServeFile(w, r, "static/script.js")
		case "/style.css":
			http.ServeFile(w, r, "static/style.css")
		default:
			http.NotFound(w, r)
		}
	})

	http.HandleFunc("/getProvider", server.writeNominatimJSON)
	http.HandleFunc("/getAlerts", server.getAlerts)
	// this function may be useful for eliminating wasted requests to the google maps API when no clients are connected
	// however, this is not properly implemented yet in the frontend
	// http.HandleFunc("/toggleScraper", server.toggleScraper)

	log.Println("listening on :8080")
	http.ListenAndServe(":8080", nil)
}
