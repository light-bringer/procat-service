package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	pb "github.com/light-bringer/procat-service/proto/product/v1"
)

// EventsHandler handles HTTP requests for events.
type EventsHandler struct {
	productService pb.ProductServiceClient
}

// NewEventsHandler creates a new HTTP events handler.
func NewEventsHandler(productService pb.ProductServiceClient) *EventsHandler {
	return &EventsHandler{
		productService: productService,
	}
}

// Event represents a domain event in the HTTP response.
type Event struct {
	EventID     string  `json:"event_id"`
	EventType   string  `json:"event_type"`
	AggregateID string  `json:"aggregate_id"`
	Payload     string  `json:"payload"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	ProcessedAt *string `json:"processed_at,omitempty"`
}

// ListEventsResponse represents the HTTP response for listing events.
type ListEventsResponse struct {
	Events     []Event `json:"events"`
	TotalCount int64   `json:"total_count"`
}

// ServeHTTP handles GET /api/v1/events requests.
func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	req := &pb.ListEventsRequest{
		Limit: 100, // Default limit
	}

	if eventType := query.Get("event_type"); eventType != "" {
		req.EventType = &eventType
	}

	if aggregateID := query.Get("aggregate_id"); aggregateID != "" {
		req.AggregateId = &aggregateID
	}

	if status := query.Get("status"); status != "" {
		req.Status = &status
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			req.Limit = int32(limit)
		}
	}

	// Call gRPC service
	resp, err := h.productService.ListEvents(context.Background(), req)
	if err != nil {
		http.Error(w, "Failed to fetch events: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert proto events to HTTP response
	events := make([]Event, 0, len(resp.Events))
	for _, protoEvent := range resp.Events {
		event := Event{
			EventID:     protoEvent.EventId,
			EventType:   protoEvent.EventType,
			AggregateID: protoEvent.AggregateId,
			Payload:     protoEvent.Payload,
			Status:      protoEvent.Status,
			CreatedAt:   protoEvent.CreatedAt.AsTime().Format("2006-01-02T15:04:05Z07:00"),
		}
		if protoEvent.ProcessedAt != nil {
			processedAt := protoEvent.ProcessedAt.AsTime().Format("2006-01-02T15:04:05Z07:00")
			event.ProcessedAt = &processedAt
		}
		events = append(events, event)
	}

	response := ListEventsResponse{
		Events:     events,
		TotalCount: resp.TotalCount,
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
