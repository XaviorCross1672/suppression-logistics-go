package main

import (
	"context"
	"strings"
)

type SuppressionLog struct{ addresses map[string]bool }

func NewSuppressionLog() *SuppressionLog { return &SuppressionLog{addresses: map[string]bool{}} }
func (s *SuppressionLog) Add(address string) {
	s.addresses[strings.ToLower(strings.TrimSpace(address))] = true
}
func (s *SuppressionLog) Contains(address string) bool {
	return s.addresses[strings.ToLower(strings.TrimSpace(address))]
}

func IsHardBounce(events []EmailEvent) bool {
	for _, event := range events {
		value := strings.ToLower(event.Type + " " + event.Event)
		if strings.Contains(value, "hard") && strings.Contains(value, "bounce") {
			return true
		}
	}
	return false
}

func SendDeliveryUpdate(ctx context.Context, client *Client, log *SuppressionLog, to, shipmentID string) (string, bool, error) {
	if log.Contains(to) {
		return "", false, nil
	}
	result, err := client.send(ctx, to, "Shipment update "+shipmentID, "Shipment "+shipmentID+" is ready for dispatch.")
	if err != nil {
		return "", false, err
	}
	return result.MessageID, true, nil
}

func RecordBounce(ctx context.Context, client *Client, log *SuppressionLog, messageID string) error {
	if _, err := client.get(ctx, messageID); err != nil {
		return err
	}
	events, err := client.events(ctx, messageID)
	if err != nil {
		return err
	}
	for _, event := range events {
		address := event.To
		if address == "" {
			address = event.Email
		}
		if address != "" && IsHardBounce([]EmailEvent{event}) {
			log.Add(address)
		}
	}
	return nil
}
