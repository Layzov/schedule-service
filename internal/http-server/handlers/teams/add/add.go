package add

import (
	"avito-test-assignment-backend/api"
	"avito-test-assignment-backend/internal/models"
	"avito-test-assignment-backend/pkg/response"
	"avito-test-assignment-backend/pkg/sl"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)


type TeamAdder interface {
	AddTeamService(ctx context.Context, t models.Team) error
}

type Request struct {
	api.Team
}

type Response struct {
	response.Response
	api.Team `json:"team"`
}

func New(log *slog.Logger, teamAdder TeamAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.teams.add.add.New"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("Failed to decode request body", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST),"failed to decode request"))

			return
		}

		log.Info("Request body decoded", slog.Any("request", req))

		if req.TeamName == ""{
			log.Error("team_name is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST),"team_name is empty"))

			return
		}

		mappedTeam, idErr := Mapper(req.Team)

		if errors.Is(idErr, response.ErrInvalidId) {
			log.Error("User(-s) ID has a wrong type", sl.Err(idErr))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST),"invalid user_id"))

			return
		}
		
		err := teamAdder.AddTeamService(r.Context(), *mappedTeam)

		if errors.Is(err, response.ErrTeamExists) {
			log.Error("team_name already exists", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.TEAM_EXISTS),"team_name already exists"))

			return
		}
		
		if err != nil {
			log.Error("Failed to add team", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST),"failed to add team"))

			return
		}

		log.Info("Team added", slog.Any("team", req.Team))
	
		team := req.Team
		w.WriteHeader(http.StatusCreated)	
		responseOK(w, r, &team)
	}	

}

func responseOK(w http.ResponseWriter, r *http.Request, team *api.Team) {
	render.JSON(w, r, Response{
		Team: *team,
	})
}

func Mapper(t api.Team) (*models.Team, error) {
	const op = "handlers.teams.post.add.Maper"
	users := make([]models.User, 0, len(t.Members))
	fmt.Println(len(t.Members))
	reqRegex := regexp.MustCompile(`^u[1-9][0-9]*$`)

	for _, member := range t.Members {
		
		if member.UserID == "" {
            return nil, fmt.Errorf("%s: %w", op, response.ErrInvalidId)
        }
        
        if !reqRegex.MatchString(member.UserID) {
            return nil, fmt.Errorf("%s: %w", op, response.ErrInvalidId)
        }
        
        users = append(users, models.User{
            UserID:   member.UserID,
            Username: member.Username,
            TeamName: t.TeamName,
            IsActive: member.IsActive,
        })
	}

	return &models.Team{
		TeamName: t.TeamName,
		Members: users,
	}, nil
}