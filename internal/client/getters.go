package client

import "github.com/google/uuid"

func (w WorkflowIndexResultResponseSchema) GetID() uuid.UUID {
	return w.Id
}
func (w WorkflowIndexResultResponseSchema) GetName() string {
	return w.Name
}

func (b BundleIndexResultResponseSchema) GetID() uuid.UUID {
	return b.Id
}
func (b BundleIndexResultResponseSchema) GetName() string {
	return b.Name
}

func (i IntegrationBaseResponseSchema) GetID() uuid.UUID {
	return i.Id
}
func (i IntegrationBaseResponseSchema) GetName() string {
	return i.Name
}
