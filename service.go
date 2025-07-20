package main

import (
	"context"
	"github.com/go-ldap/ldap/v3"
	"github.com/pquerna/otp/totp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"layeh.com/radius"
	"log"
)

type Service interface {
	Pool() *ldap.Conn
	Base() string
	ValidLdapCredencials(username, password string) (bool, string, radius.Code)
	ValidTotp(otp, username string) bool
}
type service struct {
	pool *ldap.Conn
	base string
	db   *mongo.Collection
}

func (s *service) Pool() *ldap.Conn {
	return s.pool
}
func (s *service) Base() string {
	return s.base
}
func (s *service) DB() *mongo.Collection {
	return s.db
}
func NewService(pool *ldap.Conn, base string, collection *mongo.Collection) Service {
	return &service{
		pool: pool,
		base: base,
		db:   collection,
	}
}

func (s *service) ValidLdapCredencials(username, password string) (bool, string, radius.Code) {
	otp := password[len(password)-6:]
	password = password[:len(password)-6]
	err := s.Pool().Bind("cn="+username+","+s.Base(), password)
	if err != nil {
		log.Println("LDAP auth failed:", err)
		return false, "", radius.CodeAccessReject
	}
	log.Println("LDAP auth succeeded for user:", username)
	return true, otp, radius.CodeAccessAccept
}
func (s *service) ValidTotp(otp, username string) bool {
	// Implement OTP validation logic here
	// This is a placeholder implementation
	log.Println("Validating OTP for user:", username)
	var result bson.M
	err := s.DB().FindOne(context.Background(), bson.D{{"_id", username}}).Decode(&result)
	if err != nil {
		log.Println("Error fetching TOTP secret from database:", err)
		return false
	}
	totpSecret, ok := result["totp_secret"].(string)
	if ok == false {
		log.Println("TOTP secret not found for user:", username)
		return false
	}
	return totp.Validate(otp, totpSecret)
}

var ServiceInstance Service
