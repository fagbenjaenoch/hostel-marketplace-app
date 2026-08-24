package dto

import "encoding/json"

type SearchResult struct {
	EntityID   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
	Entity     string `json:"entity"`
	Slug       string `json:"slug"`
	Address    string `json:"address"`
}

type SearchEvent struct {
	EventType    string
	EventPayload SearchEventPayload
}

type SearchEventPayload struct {
	Name string `json:"name"`
}

func (s SearchEvent) Type() string {
	return s.EventType
}

func (s SearchEvent) Payload() []byte {
	payload, _ := json.Marshal(s.EventPayload)
	return payload
}
