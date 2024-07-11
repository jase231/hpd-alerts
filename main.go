package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jase231/hpd-alerts/location"
	"github.com/jase231/hpd-alerts/models"
	"github.com/jase231/hpd-alerts/scraper"
)

type Error struct {
	Error string `json:"error"`
}

type Server struct {
	incidents map[string]models.Incident
	mu        sync.Mutex
	isRunning bool
	mapsToken string
}

func NewServer(envToken string) (*Server, error) {

	return &Server{
		incidents: make(map[string]models.Incident),
		mapsToken: envToken,
		isRunning: true,
	}, nil
}

func (s *Server) scrapeLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		if !s.isRunning {
			continue
		}

		newIncidents := scraper.Scrape(s.incidents)
		scraper.RemoveStaleIncidents(s.incidents, newIncidents)
		scraper.RemoveDuplicates(s.incidents, newIncidents)
		if err := location.PopulateLocation(newIncidents, s.mapsToken); err != nil {
			log.Fatalln("can't get locations, quitting...", err)
		}

		s.mu.Lock()
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(s.incidents)
}

func (s *Server) toggleScraper(w http.ResponseWriter, r *http.Request) {
	s.isRunning = !s.isRunning
	w.WriteHeader(http.StatusOK)
}

func (s *Server) writeError(w http.ResponseWriter, err error, status int) {
	log.Println("error:", status, err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Error{err.Error()})
}

func main() {
	// --provider flag for geocode provider
	var provider, envToken string
	flag.StringVar(&provider, "provider", "", "geocode provider to use (nominatim or google)")
	flag.Parse()

	if provider == "nominatim" {
		envToken = "nominatim"
	} else if provider == "google" {
		envToken = os.Getenv("MAPS_TOKEN")
	} else {
		log.Fatalln("invalid provider, quitting...")
	}

	server, err := NewServer(envToken)
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

	http.HandleFunc("/getAlerts", server.getAlerts)
	// this function may be useful for eliminating wasted requests to the google maps API when no clients are connected
	// however, this is not properly implemented yet in the frontend
	// http.HandleFunc("/toggleScraper", server.toggleScraper)

	log.Println("listening on :8080")
	http.ListenAndServe(":8080", nil)
}
