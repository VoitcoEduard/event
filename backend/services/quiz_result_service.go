package services

import (
	"context"
	"time"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
)

type QuizResultService struct {
	Results     *database.QuizResultRepository
	Teams       *database.TeamRepository
	TeamResults *database.TeamResultRepository
}

func NewQuizResultService() *QuizResultService {
	return &QuizResultService{
		Results: database.NewQuizResultRepository(), Teams: database.NewTeamRepository(),
		TeamResults: database.NewTeamResultRepository(),
	}
}

func (s *QuizResultService) SaveQuiz(ctx context.Context, session *models.QuizSession) error {
	eventID := session.EventID
	if eventID == "" {
		event, err := s.Results.EventForModule(ctx, session.ModuleID)
		if err != nil {
			return err
		}
		eventID = event.ID
	}
	players := make([]models.PlayerResult, 0, len(session.Players))
	for _, p := range session.Players {
		players = append(players, models.PlayerResult{ID: p.ID, Nickname: p.Nickname, Score: p.Score})
	}
	signals := make([]models.SignalResult, 0, len(session.PlayerSignals))
	for _, signal := range session.PlayerSignals {
		signals = append(signals, models.SignalResult{
			PlayerID: signal.PlayerID, QuestionIndex: signal.QuestionIndex,
			AnswerValue: signal.AnswerValue, ScoreEarned: signal.ScoreEarned,
			TimeSpent: signal.TimeSpent, Guesses: signal.Guesses,
		})
	}
	return s.Results.Create(ctx, &models.QuizResult{
		EventID: eventID, ModuleID: session.ModuleID, FinishedAt: time.Now(),
		Players: players, PlayerSignals: signals,
	})
}

func (s *QuizResultService) SaveTeams(ctx context.Context, eventID, moduleID, moduleName string) (*models.TeamResult, error) {
	teams, err := s.Teams.GetByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	participants, err := s.Results.ParticipantsForEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	teamInfos := make([]models.TeamInfo, 0, len(teams))
	for _, team := range teams {
		members := make([]models.TeamMember, 0)
		for _, participant := range participants {
			if participant.TeamID == team.ID {
				members = append(members, models.TeamMember{ID: participant.ID, Nickname: participant.DisplayName})
			}
		}
		teamInfos = append(teamInfos, models.TeamInfo{
			ID: team.ID, Name: team.Name, Color: team.Color, Members: members,
		})
	}
	result := &models.TeamResult{
		EventID: eventID, ModuleID: moduleID, ModuleName: moduleName, Teams: teamInfos,
	}
	if err := s.TeamResults.Create(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}
