package scraper

import (
	"fmt"
	"log"

	"github.com/gocolly/colly/v2"
	"github.com/jase231/hpd-alerts/models"
)

func RemoveDuplicates(oldIncidents map[string]models.Incident, newIncidents map[string]models.Incident) {
	for id := range newIncidents {
		if _, ok := oldIncidents[id]; ok {
			delete(newIncidents, id)
		}
	}
}

func RemoveStaleIncidents(oldIncidents map[string]models.Incident, newIncidents map[string]models.Incident) {
	for id := range oldIncidents {
		if _, ok := newIncidents[id]; !ok {
			delete(oldIncidents, id)
		}
	}
}

func Scrape() map[string]models.Incident {
	c := colly.NewCollector()
	incidents := make(map[string]models.Incident)

	c.OnHTML("table#dgCalls", func(e *colly.HTMLElement) {
		log.Println("Scrape Request")
		e.ForEach("tr", func(_ int, el *colly.HTMLElement) {
			// ignore table header
			if el.ChildText("td:nth-child(1)") == "" {
				return
			} else {
				id := el.ChildText("td:nth-child(1)")

				incidents[id] = models.Incident{
					ID:         id,
					Block:      el.ChildText("td:nth-child(2)"),
					Received:   el.ChildText("td:nth-child(3)"),
					Type:       el.ChildText("td:nth-child(4)"),
					CallStatus: el.ChildText("td:nth-child(5)"),
					Distr:      el.ChildText("td:nth-child(6)"),
				}
			}
		})
	})
	c.UserAgent = "Bot"
	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request to county failed with response: ", r, "\nError:", err)
	})
	c.Visit("https://activecalls.henrico.us/")

	return incidents
}
