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
	mongoTotpCollectionName := os.Getenv("MONGO_TOTP_COLLECTION")
	// |========================================== END ==============================================|
	// |===================================== Start Mongo Conn ======================================|
	ctx := context.Background()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+mongoUser+":"+mongoPass+"@"+mongoHost))
	if err != nil {
		panic("Erro ao conectar ao MongoDB")
	}
	defer func(mongoClient *mongo.Client, ctx context.Context) {
		err := mongoClient.Disconnect(ctx)
		if err != nil {
			panic("Erro ao fechar a Conexão do MongoDB")
		}
	}(mongoClient, ctx)
	collection := mongoClient.Database(mongoDatabase).Collection(mongoTotpCollectionName)
	// |========================================== END ==============================================|
	// |=================================== Start Postgres Conn =====================================|
	ldapConn, err := ldap.DialURL("ldap://" + ldapHost)
	if err != nil {
		panic("Erro ao conectar ao LDAP")
	}
	defer func(ldapConn *ldap.Conn) {
		err := ldapConn.Close()
		if err != nil {
			panic("Erro ao fechar a Conexão do LDAP")
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
		SecretSource: radius.StaticSecretSource([]byte(os.Getenv("SECRET"))),
	}
	log.Print("Starting server on :1812")
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
