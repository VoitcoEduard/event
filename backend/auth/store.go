package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const sessionLifetime = 12 * time.Hour
const loginLifetime = 10 * time.Minute

var ErrNoSession = errors.New("no valid session")

type User struct {
	ID        string    `bson:"_id" json:"id"`
	Issuer    string    `bson:"issuer" json:"-"`
	Subject   string    `bson:"subject" json:"-"`
	Email     string    `bson:"email" json:"email"`
	Name      string    `bson:"name" json:"name"`
	Role      string    `bson:"role" json:"role"`
	Status    string    `bson:"status" json:"status"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

type Session struct {
	Hash      string    `bson:"_id"`
	UserID    string    `bson:"userId"`
	CSRF      string    `bson:"csrf"`
	ExpiresAt time.Time `bson:"expiresAt"`
}

type LoginTransaction struct {
	Hash      string    `bson:"_id"`
	Browser   string    `bson:"browser"`
	Nonce     string    `bson:"nonce"`
	Verifier  string    `bson:"verifier"`
	ExpiresAt time.Time `bson:"expiresAt"`
}

type Store struct{ DB *mongo.Database }

func (s Store) Users() *mongo.Collection    { return s.DB.Collection("auth_users") }
func (s Store) Sessions() *mongo.Collection { return s.DB.Collection("auth_sessions") }
func (s Store) Logins() *mongo.Collection   { return s.DB.Collection("auth_logins") }
func (s Store) Audit() *mongo.Collection    { return s.DB.Collection("auth_audit") }

func (s Store) EnsureIndexes(ctx context.Context) error {
	_, err := s.Users().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "issuer", Value: 1}, {Key: "subject", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}
	for _, collection := range []*mongo.Collection{s.Sessions(), s.Logins()} {
		_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (s Store) UpsertGoogleUser(ctx context.Context, issuer, subject, email, name string) (User, error) {
	filter := bson.M{"issuer": issuer, "subject": subject}
	now := time.Now().UTC()
	update := bson.M{
		"$setOnInsert": bson.M{"_id": uuid.NewString(), "role": "admin", "status": "pending", "createdAt": now},
		"$set":         bson.M{"email": email, "name": name},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var user User
	err := s.Users().FindOneAndUpdate(ctx, filter, update, opts).Decode(&user)
	return user, err
}

func (s Store) EnsureDevUser(ctx context.Context) (User, error) {
	filter := bson.M{"issuer": "development", "subject": "owner"}
	update := bson.M{"$setOnInsert": bson.M{"_id": "development-owner", "issuer": "development", "subject": "owner", "createdAt": time.Now().UTC()}, "$set": bson.M{"email": "", "name": "Development owner", "role": "owner", "status": "active"}}
	var user User
	err := s.Users().FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&user)
	return user, err
}

func (s Store) NewSession(ctx context.Context, userID string) (string, Session, error) {
	token, err := randomToken()
	if err != nil {
		return "", Session{}, err
	}
	csrf, err := randomToken()
	if err != nil {
		return "", Session{}, err
	}
	session := Session{Hash: digest(token), UserID: userID, CSRF: csrf, ExpiresAt: time.Now().Add(sessionLifetime).UTC()}
	_, err = s.Sessions().InsertOne(ctx, session)
	return token, session, err
}

func (s Store) Resolve(ctx context.Context, token string) (Session, User, error) {
	if token == "" {
		return Session{}, User{}, ErrNoSession
	}
	var session Session
	err := s.Sessions().FindOne(ctx, bson.M{"_id": digest(token), "expiresAt": bson.M{"$gt": time.Now().UTC()}}).Decode(&session)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Session{}, User{}, ErrNoSession
	}
	if err != nil {
		return Session{}, User{}, err
	}
	var user User
	err = s.Users().FindOne(ctx, bson.M{"_id": session.UserID}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Session{}, User{}, ErrNoSession
	}
	return session, user, err
}

func (s Store) Revoke(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	_, err := s.Sessions().DeleteOne(ctx, bson.M{"_id": digest(token)})
	return err
}

func (s Store) AuditAction(ctx context.Context, actor, target, action string) error {
	_, err := s.Audit().InsertOne(ctx, bson.M{"actor": actor, "target": target, "action": action, "at": time.Now().UTC()})
	return err
}
