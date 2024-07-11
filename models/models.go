package models

// each individual incident is of this type
type Incident struct {
	ID           string
	Block        string
	Location     Coordinate
	Intersection bool
	Received     string
	Type         string
	CallStatus   string
	Distr        string
	// Message    string
}

// coordinate
type Coordinate struct {
	Lat float64
	Lng float64
}
