package database

import (
	"context"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TemplateRepository struct {
	collection *mongo.Collection
}

func NewTemplateRepository() *TemplateRepository {
	return &TemplateRepository{collection: db.GetCollection("templates")}
}

func (r *TemplateRepository) Create(ctx context.Context, template *models.Template) error {
	_, err := r.collection.InsertOne(ctx, template)
	return err
}

func (r *TemplateRepository) List(ctx context.Context) ([]models.Template, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	templates := make([]models.Template, 0)
	err = cursor.All(ctx, &templates)
	return templates, err
}

func (r *TemplateRepository) Delete(ctx context.Context, id string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *TemplateRepository) Update(ctx context.Context, id string, template *models.Template) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"name": template.Name, "description": template.Description,
		"data": template.Data, "moduleType": template.ModuleType,
	}})
	return err
}
