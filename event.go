package main

type Event struct {
	Date                Date                  `json:"date"`
	Name                string                `json:"name"`
	OrganisersNatIDList []string              `json:"organisers"` //todo: add phone nr to profile for lookup
	Groups              map[string]EventGroup `json:"groups"`     //enter into one of these at the event
}

type EventGroup struct {
	Cost Amount `json:"cost"`
}
