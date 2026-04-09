package main

import (
	"fmt"
	"log"
	"time"

	"clash-subscription-manager/handlers"
	"clash-subscription-manager/models"
)

func main() {
	dataFile := "./data/subscriptions.json"

	// Example 1: List all subscriptions (initially empty)
	fmt.Println("=== Listing Subscriptions ===")
	subs, err := handlers.ListSubscriptions(dataFile)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Found %d subscriptions\n", len(subs))

	// Example 2: Add a new subscription
	fmt.Println("\n=== Adding Subscription ===")
	newSub, err := handlers.AddSubscription(models.Subscription{
		Name:      "Example Subscription",
		URL:       "https://example.com/config",
		Type:      "v2ray",
		UpdatedAt: time.Now(),
		Status:    "active",
		NodeCount: 15,
	}, dataFile)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Added subscription with ID: %s\n", newSub.ID)

	// Example 3: Get specific subscription
	fmt.Println("\n=== Getting Subscription ===")
	retrieved, err := handlers.GetSubscription(newSub.ID, dataFile)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Retrieved: %s (%s)\n", retrieved.Name, retrieved.Type)

	// Example 4: List all subscriptions again
	fmt.Println("\n=== Listing All Subscriptions ===")
	subs, err = handlers.ListSubscriptions(dataFile)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	for _, sub := range subs {
		fmt.Printf("- %s: %s\n", sub.ID, sub.Name)
	}

	// Example 5: Delete subscription
	fmt.Println("\n=== Deleting Subscription ===")
	err = handlers.DeleteSubscription(newSub.ID, dataFile)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Deleted subscription %s\n", newSub.ID)

	// Example 6: Verify deletion
	fmt.Println("\n=== Final Count ===")
	subs, err = handlers.ListSubscriptions(dataFile)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Total subscriptions: %d\n", len(subs))
}
