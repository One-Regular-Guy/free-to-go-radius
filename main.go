package main

import (
	"context"
	"github.com/go-ldap/ldap/v3"
	"github.com/joho/godotenv"
	"github.com/pquerna/otp/totp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"log"
	"os"
)

func main() {
	godotenv.Load()
	ctx := context.Background()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+os.Getenv("MONGO_USER")+":"+os.Getenv("MONGO_PASS")+"@"+os.Getenv("MONGO_HOST")))
	if err != nil {
		log.Fatal("Erro ao conectar ao MongoDB:", err)
	}
	defer mongoClient.Disconnect(ctx)

	collection := mongoClient.Database(os.Getenv("MONGO_DB")).Collection(os.Getenv("MONGO_TOTP_COLLECTION"))
	ldapHost := os.Getenv("LDAP_HOST")
	ldapBase := os.Getenv("LDAP_BASE")

	handler := func(w radius.ResponseWriter, r *radius.Request) {
		username := rfc2865.UserName_GetString(r.Packet)
		fullPassword := rfc2865.UserPassword_GetString(r.Packet)
		if len(fullPassword) < 6 {
			w.Write(r.Response(radius.CodeAccessReject))
			return
		}
		otp := fullPassword[len(fullPassword)-6:]
		password := fullPassword[:len(fullPassword)-6]

		// Cria nova conexão LDAP por requisição
		ldapConn, err := ldap.DialURL("ldap://" + ldapHost)
		if err != nil {
			log.Printf("Erro ao conectar ao LDAP: %v", err)
			w.Write(r.Response(radius.CodeAccessReject))
			return
		}
		defer ldapConn.Close()

		err = ldapConn.Bind("cn="+username+",ou=users,"+ldapBase, password)
		code := radius.CodeAccessAccept
		if err != nil {
			log.Printf("LDAP auth failed for user %s with pass %s : %v", username, password, err)
			code = radius.CodeAccessReject
		} else {
			log.Printf("LDAP auth succeeded for user: %s", username)
		}

		// Validação TOTP
		if code == radius.CodeAccessAccept {
			var result bson.M
			err := collection.FindOne(context.Background(), bson.D{{"_id", username}}).Decode(&result)
			if err != nil {
				log.Printf("Erro ao buscar TOTP do usuário %s: %v", username, err)
				code = radius.CodeAccessReject
			} else {
				totpSecret, ok := result["totp_secret"].(string)
				if !ok || !totp.Validate(otp, totpSecret) {
					log.Printf("TOTP inválido para o usuário %s", username)
					code = radius.CodeAccessReject
				} else {
					log.Printf("TOTP validado com sucesso para o usuário %s", username)
				}
			}
		}

		log.Printf("Writing %v to %v", code, r.RemoteAddr)
		w.Write(r.Response(code))
	}

	server := radius.PacketServer{
		Handler:      radius.HandlerFunc(handler),
		SecretSource: radius.StaticSecretSource([]byte(`secret`)),
	}
	log.Printf("Starting server on :1812")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
