package main


import (
	"context"
	"log"

	firebase "firebase.google.com/go"
)
 
func main() {
    // Use the application default credentials
	ctx := context.Background()
	conf := &firebase.Config{ProjectID: "ruirui-synerex-simulation"}
	app, err := firebase.NewApp(ctx, conf)
	if err != nil {
	log.Fatalln(err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
	log.Fatalln(err)
	}
	
	defer client.Close()

	_, _, err = client.Collection("users").Add(ctx, map[string]interface{}{
        "first": "Ada",
        "last":  "Lovelace",
        "born":  1815,
	})
	if err != nil {
			log.Fatalf("Failed adding alovelace: %v", err)
	}
}