package handler

import (
	"context"

	"github.com/webitel/storage/store"
)

type EventDomainCreated struct {
	DomainId int `json:"domain_id"`
}

const ClusterEventDomainCreate string = "domains.create"

type EventDomainCreatedHandler struct {
	filePolicyStore store.FilePoliciesStore
}

func NewEventDomainCreatedHandler(filePolicyStore store.FilePoliciesStore) *EventDomainCreatedHandler {
	return &EventDomainCreatedHandler{
		filePolicyStore: filePolicyStore,
	}
}

func (h *EventDomainCreatedHandler) Event() string {
	return ClusterEventDomainCreate
}

func (h *EventDomainCreatedHandler) Handle(ctx context.Context, event EventDomainCreated) error {
	if err := h.filePolicyStore.CreateDefaultPolicies(ctx, int64(event.DomainId)); err != nil {
		return err
	}
	return nil
}
