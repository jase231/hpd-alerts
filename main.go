package main

import (
	"encoding/json"
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

func NewServer() (*Server, error) {
	envToken := os.Getenv("MAPS_TOKEN")
	if envToken == "" {
		return nil, fmt.Errorf("missing google maps API key")
	}

	return &Server{
		incidents: make(map[string]models.Incident),
		mapsToken: envToken,
		isRunning: false,
	}, nil
}

func (s *Server) scrapeLoop() {
	ticker := time.NewTicker(60 * time.Second)
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
	server, err := NewServer()
	if err != nil {
		log.Fatalln("couldn't spawn server, quitting:", err)
	}
	go server.scrapeLoop()

	http.HandleFunc("/getAlerts", server.getAlerts)
	http.HandleFunc("/toggleScraper", server.toggleScraper)

	log.Println("listening on :8080")
	http.ListenAndServe(":8080", nil)
}
