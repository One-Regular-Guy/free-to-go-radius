package main

import (
	"context"
	"github.com/go-ldap/ldap/v3"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"layeh.com/radius"
	"log"
	"os"
)

func main() {
	// Load environment variables from .env file if Exists
	err := godotenv.Load()
	if err != nil {
		log.Print("Error loading .env file")
	}
	// |=================================== Environment Variables ===================================|
	ldapHost := os.Getenv("LDAP_HOST")
	ldapBase := os.Getenv("LDAP_BASE")
	mongoUser := os.Getenv("MONGO_USER")
	mongoPass := os.Getenv("MONGO_PASS")
	mongoHost := os.Getenv("MONGO_HOST")
	mongoDatabase := os.Getenv("MONGO_DB")
	secret := os.Getenv("SECRET")
	mongoTotpCollectionName := os.Getenv("MONGO_TOTP_COLLECTION")
	if ldapHost == "" || ldapBase == "" || mongoUser == "" || mongoPass == "" || mongoHost == "" || mongoDatabase == "" || secret == "" {
		log.Fatal("Missing any of required environment variables: LDAP_HOST, LDAP_BASE, MONGO_USER, MONGO_PASS, MONGO_HOST, MONGO_DB, SECRET, MONGO_TOTP_COLLECTION")
	}
	// |========================================== END ==============================================|
	// |===================================== Start Mongo Conn ======================================|
	ctx := context.Background()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+mongoUser+":"+mongoPass+"@"+mongoHost))
	if err != nil {
		panic("Error when connecting MongoDB")
	}
	defer func(mongoClient *mongo.Client, ctx context.Context) {
		err := mongoClient.Disconnect(ctx)
		if err != nil {
			panic("Error when disconnecting MongoDB")
		}
	}(mongoClient, ctx)
	collection := mongoClient.Database(mongoDatabase).Collection(mongoTotpCollectionName)
	// |========================================== END ==============================================|
	// |=================================== Start Postgres Conn =====================================|
	ldapConn, err := ldap.DialURL("ldap://" + ldapHost)
	if err != nil {
		panic("Error when connecting to LDAP")
	}
	defer func(ldapConn *ldap.Conn) {
		err := ldapConn.Close()
		if err != nil {
			panic("Error when disconnecting LDAP")
		}
	}(ldapConn)
	// |========================================== END ==============================================|
	// Initialize the service instance
	ServiceInstance = NewService(ldapBase, ldapConn, collection)
	// Create the handler function
	handler := Handler()
	// Create the Radius server with the handler and secret
	server := radius.PacketServer{
		Handler:      radius.HandlerFunc(handler),
		SecretSource: radius.StaticSecretSource([]byte(secret)),
	}
	log.Print("Starting server on :1812")
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
