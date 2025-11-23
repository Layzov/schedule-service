package postgres

import (
	"avito-test-assignment-backend/api"
	"avito-test-assignment-backend/internal/models"
	"avito-test-assignment-backend/pkg/response"
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.postgres.New"

    db, err := sql.Open("postgres", storagePath)
    if err != nil {
        return nil, fmt.Errorf("%s: %w", op, err)
    }

    return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	if s.db == nil || s == nil {
		return nil
	}

	return s.db.Close()
}

// #### team/add ####

func (s *Storage) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, nil)
}

func (s *Storage) InsertTeamTx(ctx context.Context, tx *sql.Tx, teamName string) (int64, error) {
	const op = "storage.postgres.InsertTeamTx"

	res, err := tx.ExecContext(ctx, `INSERT INTO teams (team_name) VALUES ($1) ON CONFLICT DO NOTHING`, teamName)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	n, err := res.RowsAffected()

	if err != nil {
		return 0, fmt.Errorf("%s %w", op, err)
	}

	return n, nil
}

func (s *Storage) UpsertUsersTx(ctx context.Context, tx *sql.Tx, teamName string, users []any, placeholders []string) error {
	const op = "storage.postgres.UpsertUsersTx"

	if len(users) == 0 {
		return nil
	}

	query := fmt.Sprintf(`
		INSERT INTO users (user_id, username, team_name, is_active)
		VALUES %s
		ON CONFLICT (user_id)
		DO UPDATE
		SET team_name = EXCLUDED.team_name,
			username = EXCLUDED.username,
			is_active = EXCLUDED.is_active;
		`, 
		strings.Join(placeholders, ","),
	)

	_, err := tx.ExecContext(ctx, query, users...)
	if err != nil {
		return fmt.Errorf("%s exec: %w", op, err)
	}

	return nil
}

// #### team/get ####

func (s *Storage) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	const op = "storage.postgres.GetTeamByName"

	var team models.Team
	var user models.User

	err := s.db.QueryRowContext(ctx, `SELECT team_name FROM teams WHERE team_name=$1`, teamName).Scan(&team.TeamName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := s.db.QueryContext(ctx, `SELECT user_id, username, is_active FROM users WHERE team_name=$1`, teamName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&user.UserID, &user.Username, &user.IsActive)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		user.TeamName = teamName

		team.Members = append(team.Members, user)
	}
	
	return &team, nil
}

// #### user/set_is_active ####

func (s *Storage) SetIsActive(ctx context.Context, userID string, isActive bool) (*models.User, error) {
	const op = "storage.postgres.SetIsActive"

	var user models.User
	var isActiveDB bool

	err := s.db.QueryRowContext(ctx, `SELECT is_active FROM users WHERE user_id=$1`, userID).Scan(&isActiveDB)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, response.ErrNotFound
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if isActiveDB != isActive {
		_, err := s.db.ExecContext(ctx, `UPDATE users SET is_active=$1 WHERE user_id=$2`, isActive, userID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	err = s.db.QueryRowContext(ctx, 
	`SELECT username, team_name, is_active 
	FROM users WHERE user_id=$1`,userID).
	Scan(
		&user.Username, 
		&user.TeamName, 
		&user.IsActive,
	)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	user.UserID = userID

	return &user, nil

}

func (s *Storage) PullRequestCreate(ctx context.Context, tx *sql.Tx, pr *api.PRCreateRequest) error {
	const op = "storage.postgres.PullRequestCreate"

	_, err := tx.ExecContext(ctx,
		`INSERT INTO pull_requests 
		(pull_request_id, pull_request_name, author_id, pr_status) 
		VALUES ($1, $2, $3, $4)`,
		pr.PullRequestID,
		pr.PullRequestName,
		pr.AuthorID,
		string(models.PR_OPEN),
	)
	if err != nil {
		sqlErr, ok := err.(*pq.Error)
		if ok && sqlErr.Code == "23505" { 
			return fmt.Errorf("%s: %w", op, response.ErrPRExists)
		}
		if ok && sqlErr.Code == "23503" {
			return fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) AddPRReviewers(ctx context.Context, tx *sql.Tx, prID string, authorID string) ([]string, error) {
	const op = "storage.postgres.AddPRReviewers"

	var teamName string
	var userID string
	var members []string
	var isActive bool

	err := tx.QueryRowContext(ctx, `SELECT team_name FROM users WHERE user_id=$1`, authorID).Scan(&teamName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := tx.QueryContext(ctx, `SELECT user_id, is_active FROM users WHERE team_name=$1`, teamName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&userID, &isActive)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		if userID != authorID {
			if isActive{
				members = append(members, userID)
			}
		}
	}

	switch {
	case len(members) == 0:
		return nil, nil
	case len(members) == 1:
		
		_, err := tx.ExecContext(ctx,
			`INSERT INTO pr_reviewers
			(pull_request_id, reviewer_id) 
			VALUES ($1, $2) 
			ON CONFLICT DO NOTHING`,
			prID,
			members[0],
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		return members, nil
	default:
		i := rand.Intn(len(members))
		j := rand.Intn(len(members) - 1)
		if j == i {
			j++
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO pr_reviewers
			(pull_request_id, reviewer_id) 
			VALUES 
			($1, $2),
			($1, $3)
			ON CONFLICT DO NOTHING`,
			prID,
			members[i],
			members[j],
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		return []string{members[i], members[j]}, nil
	}
}

func (s *Storage) MergePullRequest(ctx context.Context, prID string) (*models.PullRequestShort, *time.Time, []string, error) {
	const op = "storage.postgres.MergePullRequest"

	var pr models.PullRequestShort
	var mergedAt time.Time
	var reviewer string
	var reviewers []string


	err := s.db.QueryRowContext(ctx, `SELECT pr_status FROM pull_requests WHERE pull_request_id=$1`, prID).Scan(&pr.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}

		return nil, nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	if pr.Status != models.PR_MERGED {
		_, err := s.db.ExecContext(ctx, `UPDATE pull_requests SET pr_status=$1 
		WHERE pull_request_id=$2`, string(models.PR_MERGED), prID)

		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", op, err)
		}
	}

	prErr := s.db.QueryRowContext(ctx, `SELECT pull_request_name, author_id, pr_status, merged_at 
	FROM pull_requests WHERE pull_request_id=$1`, prID).
	Scan(
		&pr.PullRequestName,
		&pr.AuthorID,
		&pr.Status,
		&mergedAt,
	)

	if prErr != nil {
		return nil, nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	pr.PullRequestID = prID

	rows, revErr := s.db.QueryContext(ctx, `SELECT reviewer_id FROM pr_reviewers WHERE pull_request_id=$1`, prID)
	if revErr != nil {
		return nil, nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&reviewer)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", op, err)
		}

		reviewers = append(reviewers, reviewer)
	}

	return &pr, &mergedAt, reviewers, nil
}

func (s *Storage) ReassignPRReviewers(ctx context.Context, prID string, oldReviewerID string) (*models.PullRequestShort, []string, error) {
	const op = "storage.postgres.ReassignPRReviewers"

	var pr models.PullRequestShort
	var revID string
	var oldReviewerID2 string
	var reviewers []string

	err := s.db.QueryRowContext(ctx, `SELECT pull_request_name, author_id, pr_status 
	FROM pull_requests WHERE pull_request_id=$1`, prID).
		Scan(
			&pr.PullRequestName,
			&pr.AuthorID,
			&pr.Status,
		)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}
	if pr.Status == models.PR_MERGED {
		return nil, nil, fmt.Errorf("%s: %w", op, response.ErrPRMerged)
	}

	pr.PullRequestID = prID

	rows, err := s.db.QueryContext(ctx, `SELECT reviewer_id FROM pr_reviewers WHERE pull_request_id=$1`, prID)
	if err != nil{
		if err == sql.ErrNoRows{
			return nil, nil, fmt.Errorf("%s: %w", op, response.ErrNoCandidate)
		}
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&revID)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", op, err)
		}

		reviewers = append(reviewers, revID)
	}

	switch {
	case len(reviewers) == 1 && reviewers[0] == oldReviewerID :
		return nil, nil, fmt.Errorf("%s: %w", op, response.ErrNoCandidate)
	case len(reviewers) == 1 && reviewers[0] != oldReviewerID:
		return nil, nil, fmt.Errorf("%s: %w", op, response.ErrNotAssigned)
	case len(reviewers) == 2 && (reviewers[0] != oldReviewerID && reviewers[1] != oldReviewerID):
		return nil, nil, fmt.Errorf("%s: %w", op, response.ErrNotAssigned)
	default:
		for _, reviewer := range reviewers{
			if reviewer != oldReviewerID{
				oldReviewerID2 = reviewer
				break
			}
		}
	}

	var teamName string
	err = s.db.QueryRowContext(ctx, `SELECT team_name FROM users WHERE user_id=$1`, pr.AuthorID).Scan(&teamName)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	var newReviewerID string
	err = s.db.QueryRowContext(ctx, `
		SELECT user_id FROM users
		WHERE team_name=$1 
		AND is_active=TRUE 
		AND user_id != $2 
		AND user_id != $3
		ORDER BY random()
		LIMIT 1`, teamName, 
		pr.AuthorID, 
		oldReviewerID).
		Scan(&newReviewerID)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return &pr, nil, fmt.Errorf("%s: %w", op, response.ErrNoCandidate)
		}
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = s.db.ExecContext(ctx, `UPDATE pr_reviewers SET reviewer_id=$1 WHERE pull_request_id=$2 AND reviewer_id=$3`, newReviewerID, prID, oldReviewerID)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	return &pr, []string{newReviewerID, oldReviewerID2}, nil
}

func (s *Storage) GetReview(ctx context.Context, userID string) (*[]models.PullRequestShort, error){
	const op = "storage.postgres.GetReview"

	var prID string
	var pullRequests []models.PullRequestShort
	var pr models.PullRequestShort

	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM users WHERE user_id=$1`, userID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := s.db.QueryContext(ctx, `SELECT pull_request_id FROM pr_reviewers WHERE reviewer_id=$1`, userID)
	if err != nil {
		if err == sql.ErrNoRows{
			return &pullRequests, nil
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&prID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		
		prErr := s.db.QueryRowContext(ctx, 
			`SELECT pull_request_name, 
			author_id, pr_status 
			FROM pull_requests 
			WHERE pull_request_id=$1`, prID).
			Scan(
				&pr.PullRequestName,
				&pr.AuthorID,
				&pr.Status,
			)
		if prErr != nil{
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		pr.PullRequestID = prID

		pullRequests = append(pullRequests, pr)
	}

	return &pullRequests, nil
}