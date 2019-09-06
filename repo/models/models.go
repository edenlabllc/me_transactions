package models

import "time"

type ModelAuditLog struct {
	Collection string      `bson:"collection,omitempty"`
	ActorID    string      `bson:"actor_id,omitempty"`
	PatientID  string      `bson:"patient_id,omitempty"`
	Params     interface{} `bson:"params,omitempty"`
	Filter     interface{} `bson:"first_name,omitempty"`
	Type       string      `bson:"type,omitempty"`
	InsertedAt time.Time   `bson:"inserted_at,omitempty"`
}
