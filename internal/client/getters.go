package client

import "github.com/google/uuid"

func (w WorkflowIndexResultResponseSchema) GetID() uuid.UUID {
	return w.Id
}
func (w WorkflowIndexResultResponseSchema) GetName() string {
	return w.Name
}
