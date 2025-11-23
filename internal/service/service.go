package service

import (
	"avito-test-assignment-backend/api"
	"avito-test-assignment-backend/internal/models"
	"avito-test-assignment-backend/pkg/response"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
    return &Service{store: store}
}

type Store interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)

	InsertTeamTx(ctx context.Context, tx *sql.Tx, teamName string) (int64, error)
	UpsertUsersTx(ctx context.Context, tx *sql.Tx, teamName string, users []any, placeholders []string) error
    GetTeam(ctx context.Context, teamName string) (*models.Team, error)

    SetIsActive(ctx context.Context, userID string, isActive bool) (*models.User, error)
    GetReview(ctx context.Context, userID string) (*[]models.PullRequestShort, error)

    PullRequestCreate(ctx context.Context, tx *sql.Tx, pr *api.PRCreateRequest) error
    AddPRReviewers(ctx context.Context, tx *sql.Tx, prID string, authorID string) ([]string, error)
    MergePullRequest(ctx context.Context, prID string) (*models.PullRequestShort, *time.Time, []string, error)
    ReassignPRReviewers(ctx context.Context, prID string, oldReviewerID string) (*models.PullRequestShort, []string, error)
}

func (s *Service) AddTeamService(ctx context.Context, t models.Team) error {
	const op = "service.AddTeamService"
	
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
        return fmt.Errorf("%s: begin tx: %w", op, err)
    }

	defer func() {
		if p := recover(); p != nil {
            _ = tx.Rollback()
            panic(p)
		}
	}()

	rows, err := s.store.InsertTeamTx(ctx, tx, t.TeamName)
    if rows == 0 {
        _ = tx.Rollback()
        return fmt.Errorf("%s: %w", op, response.ErrTeamExists) 
    }
    if err != nil {
        _ = tx.Rollback()
        return fmt.Errorf("%s: insert team: %w", op, err)
    }

    placeholders := make([]string, 0, len(t.Members))
	users := make([]interface{}, 0, len(t.Members)*4)
	idx := 1

	for i := range t.Members {
		placeholders = append(
			placeholders, 
			fmt.Sprintf("($%d,$%d,$%d,$%d)", 
			idx, idx+1, idx+2, idx+3))
		users = append(
			users, 
			t.Members[i].UserID, 
			t.Members[i].Username, 
			t.TeamName, 
			t.Members[i].IsActive)
		idx += 4
	}
	
    usrErr := s.store.UpsertUsersTx(ctx, tx, t.TeamName, users, placeholders)
    if usrErr != nil {
        _ = tx.Rollback()
        return fmt.Errorf("%s: upsert users: %w", op, err)
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("%s: commit: %w", op, err)
    }

    return nil
}

func (s *Service) GetTeamService(ctx context.Context, teamName string) (*api.Team, error) {
    const op = "service.GetTeamService"

    var dtoTeam api.Team

    team, err := s.store.GetTeam(ctx, teamName)

    if errors.Is(err, response.ErrNotFound) {
        return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
    }

    if err != nil {
        return nil, fmt.Errorf("%s: get team: %w", op, err)
    }

    for _, member := range team.Members {
        dtoTeam.Members = append(dtoTeam.Members, api.TeamMember{
            UserID:   member.UserID,
            Username: member.Username,
            IsActive: member.IsActive,
        })
    }

    dtoTeam.TeamName = team.TeamName
    
    return &dtoTeam, nil
}

func (s *Service) SetIsActiveService(ctx context.Context, userID string, isActive bool) (*api.UserDto, error) {
    const op = "service.SetIsActiveService"

    user, err := s.store.SetIsActive(ctx, userID, isActive)

    if err != nil {
        if errors.Is(err, response.ErrNotFound) {
            return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
        }

        return nil, fmt.Errorf("%s: %w", op, err)
    }

    userDto := &api.UserDto{
        UserID: user.UserID,
        Username: user.Username,
        TeamName: user.TeamName,
        IsActive: user.IsActive,
    }

    return userDto, nil
}

func (s *Service) CreatePullRequestService(ctx context.Context, pr *api.PRCreateRequest) (*api.PullRequest, error) {
    const op = "service.CreatePullRequestService"

    tx, err := s.store.BeginTx(ctx)
	if err != nil {
        return nil, fmt.Errorf("%s: begin tx: %w", op, err)
    }

	defer func() {
		if p := recover(); p != nil {
            _ = tx.Rollback()
            panic(p)
		}
	}()

    prErr := s.store.PullRequestCreate(ctx, tx, pr)
    if prErr != nil {
        if errors.Is(prErr, response.ErrPRExists) {
            _ = tx.Rollback()
            return nil, fmt.Errorf("%s: %w", op, response.ErrPRExists)
        }
        if errors.Is(prErr, response.ErrNotFound) {
            _ = tx.Rollback()
            return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
        }

        _ = tx.Rollback()
        return nil, fmt.Errorf("%s: %w", op, prErr)
    }

    reviewers, revErr := s.store.AddPRReviewers(ctx, tx, pr.PullRequestID, pr.AuthorID)
    if revErr != nil{
        _ = tx.Rollback()
        return nil, fmt.Errorf("%s: %w", op, revErr)
    }

    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("%s: %w", op, err)
    }

    dtoPR := &api.PullRequest{
        PullRequestID:   pr.PullRequestID,
        PullRequestName: pr.PullRequestName,
        AuthorID:        pr.AuthorID,
        Status:          string(models.PR_OPEN),
        Reviewers:       reviewers,
    }

    return dtoPR, nil
}

func (s *Service) MergePRService(ctx context.Context, prID string) (*api.PullRequest, error){
    const op = "service.MergePRService"

    pr, mergedAt, reviewers, err := s.store.MergePullRequest(ctx, prID)

    if err != nil {
        if errors.Is(err, response.ErrNotFound) {
            return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
        }

        return nil, fmt.Errorf("%s: %w", op, err)
    }

    dtoPR := &api.PullRequest{
        PullRequestID: prID,
        PullRequestName: pr.PullRequestName,
        AuthorID: pr.AuthorID,
        Status: string(pr.Status),
        Reviewers: reviewers,
        MergedAt: mergedAt,
    }

    return dtoPR, nil
}

func (s *Service) ReassignPRReviewersService(ctx context.Context, prID string, oldReviewerID string) (*api.PullRequest, error){
    const op = "service.ReassignPRReviewersService"

    pr, reviewers, err := s.store.ReassignPRReviewers(ctx, prID, oldReviewerID)

    if err != nil {
        if errors.Is(err, response.ErrNotFound) {
            return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
        }
        if errors.Is(err, response.ErrPRMerged) {
            return nil, fmt.Errorf("%s: %w", op, response.ErrPRMerged)
        }
        if errors.Is(err, response.ErrNoCandidate) {
            return nil, fmt.Errorf("%s: %w", op, response.ErrNoCandidate)
        }
        if errors.Is(err, response.ErrNotAssigned) {
            return nil, fmt.Errorf("%s: %w", op, response.ErrNotAssigned)
        }

        return nil, fmt.Errorf("%s: %w", op, err)
    }

    dtoPR := &api.PullRequest{
        PullRequestID: pr.PullRequestID,
        PullRequestName: pr.PullRequestName,
        AuthorID: pr.AuthorID,
        Status: string(pr.Status),
        Reviewers: reviewers,
    }

    return dtoPR, nil
}

func (s *Service) GetReviewService(ctx context.Context, userID string) (*[]api.PullRequestShortDto, error){
    const op = "service.GetReviewService"
    
    var prDto api.PullRequestShortDto
    var reviewsDto []api.PullRequestShortDto

    reviews, err := s.store.GetReview(ctx, userID)
    if err != nil {
        if errors.Is(err, response.ErrNotFound) {
            return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
        }

        return nil, fmt.Errorf("%s: %w", op, err)
    }

    if len(*reviews) == 0{
        return &reviewsDto, nil
    }

    for _, pr := range *reviews{
        prDto = api.PullRequestShortDto{
            PullRequestID: pr.PullRequestID,
            PullRequestName: pr.PullRequestName,
            AuthorID: pr.AuthorID,
            Status: string(pr.Status),
        }

        reviewsDto = append(reviewsDto, prDto)
    }

    return &reviewsDto, nil
}