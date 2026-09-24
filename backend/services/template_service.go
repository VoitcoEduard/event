package services

import (
	"context"
	"time"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/utils"
)

type TemplateService struct {
	Repo *database.TemplateRepository
}

func NewTemplateService() *TemplateService {
	return &TemplateService{Repo: database.NewTemplateRepository()}
}

func (s *TemplateService) Create(ctx context.Context, template *models.Template) error {
	template.ID = utils.NewID()
	template.CreatedAt = time.Now()
	return s.Repo.Create(ctx, template)
}

func (s *TemplateService) List(ctx context.Context) ([]models.Template, error) {
	return s.Repo.List(ctx)
}

func (s *TemplateService) Delete(ctx context.Context, id string) error {
	return s.Repo.Delete(ctx, id)
}

func (s *TemplateService) Update(ctx context.Context, id string, template *models.Template) error {
	return s.Repo.Update(ctx, id, template)
}
