package entity

type Request struct {
	ActorID    string      `json:"actor_id"`
	PatientID  string      `json:"patient_id"`
	Operations []Operation `json:"operations"`
}

type Operation struct {
	Operation  string `json:"operation"`
	Collection string `json:"collection"`
	Filter     string `json:"filter"`
	Set        string `json:"set"`
	Id         string `json:"id"`
}
