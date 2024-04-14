package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/Leopold1975/banners_control/internal/banners/api/oapi"
	"github.com/Leopold1975/banners_control/internal/banners/domain/models"
	repo "github.com/Leopold1975/banners_control/internal/banners/repository/bannerrepo"
	"github.com/Leopold1975/banners_control/internal/banners/services/authservice"
	"github.com/Leopold1975/banners_control/internal/banners/services/bannerservice"
	"github.com/Leopold1975/banners_control/internal/pkg/config"
	"github.com/Leopold1975/banners_control/pkg/logger"
)

type Server struct {
	serv          *http.Server
	bannerService BannerService
	authService   AuthService
}

type BannerService interface {
	GetBanner(context.Context, bannerservice.GetBannerRequest) ([]models.Banner, error)
	CreateBanner(context.Context, models.Banner) (int, error)
	DeleteBanner(context.Context, int) error
	UpdateBanner(context.Context, models.Banner) error
	Shutdown(context.Context) error
}

type AuthService interface {
	CreateUser(context.Context, authservice.CreateUserRequest) (string, error)
	Auth(string) (bool, error)
	Login(context.Context, string, string) (string, error)
}

func New(cfg config.Server, bs BannerService, authService AuthService, lg logger.Logger) *Server {
	var s Server
	h := oapi.HandlerWithOptions(&s, oapi.ChiServerOptions{ //nolint:exhaustruct
		BaseURL:     "/v1",
		Middlewares: []oapi.MiddlewareFunc{loggingMiddleware(lg)},
	})
	serv := &http.Server{ //nolint:exhaustruct
		Addr:         cfg.Addr,
		Handler:      h,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	s.serv = serv
	s.bannerService = bs
	s.authService = authService

	return &s
}

func (s Server) Start(ctx context.Context) error {
	errCh := make(chan error)

	go func() {
		if err := s.serv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			close(errCh)
		}
	}()

	select {
	case <-ctx.Done():
		ctxS, cancel := context.WithTimeout(context.Background(), time.Second*5) //nolint:gomnd
		defer cancel()

		if err := s.Shutdown(ctxS); err != nil { //nolint:contextcheck
			return fmt.Errorf("context error: %w server error %w", ctxS.Err(), err)
		}

		if !errors.Is(ctx.Err(), context.Canceled) {
			return fmt.Errorf("context cancelled error: %w", ctx.Err())
		}

		return nil
	case err := <-errCh:
		return fmt.Errorf("listen and serve error: %w", err)
	}
}

func (s Server) Shutdown(ctx context.Context) error {
	ctxS, cancel := context.WithTimeout(ctx, s.serv.IdleTimeout)
	defer cancel()

	if err := s.serv.Shutdown(ctxS); err != nil {
		return fmt.Errorf("shutdown server error: %w", err)
	}

	return nil
}

// Получение всех баннеров c фильтрацией по фиче и/или тегу
// (GET /banner).
func (s Server) GetBanner(w http.ResponseWriter, r *http.Request, params oapi.GetBannerParams) {
	w.Header().Add("Content-Type", "application/json")

	if params.Token == nil {
		handleError(w, fmt.Errorf("admin token required"), http.StatusUnauthorized) //nolint:perfsprint

		return
	}

	isAdmin, err := s.authService.Auth(*params.Token)
	if err != nil {
		handleError(w, fmt.Errorf("authorization error: %w", err), http.StatusUnauthorized)

		return
	}

	if !isAdmin {
		w.WriteHeader(http.StatusForbidden)

		return
	}

	var req bannerservice.GetBannerRequest
	if params.FeatureId == nil {
		req.FeatureID = -1
	} else {
		req.FeatureID = *params.FeatureId
	}

	if params.TagId != nil {
		req.Tags = []int{*params.TagId}
	}

	if params.Offset != nil {
		req.Offset = *params.Offset
	}

	if params.Limit != nil {
		req.Limit = *params.Limit
	}

	req.IsAdmin = isAdmin

	banners, err := s.bannerService.GetBanner(r.Context(), req)
	if err != nil {
		handleError(w, fmt.Errorf("get banner error: %w", err), http.StatusInternalServerError)

		return
	}

	enc := json.NewEncoder(w)

	err = enc.Encode(banners)
	if err != nil {
		handleError(w, fmt.Errorf("encode error: %w", err), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
}

// Создание нового баннера
// (POST /banner).
func (s Server) PostBanner(w http.ResponseWriter, r *http.Request, params oapi.PostBannerParams) { //nolint:cyclop
	w.Header().Add("Content-Type", "application/json")

	if params.Token == nil {
		handleError(w, fmt.Errorf("admin token required"), http.StatusUnauthorized) //nolint:perfsprint

		return
	}

	isAdmin, err := s.authService.Auth(*params.Token)
	if err != nil {
		handleError(w, fmt.Errorf("authorization error: %w", err), http.StatusUnauthorized)

		return
	}

	if !isAdmin {
		w.WriteHeader(http.StatusForbidden)

		return
	}

	var b oapi.PostBannerJSONBody

	dec := json.NewDecoder(r.Body)

	err = dec.Decode(&b)
	if err != nil {
		handleError(w, fmt.Errorf("decode error: %w", err), http.StatusBadRequest)

		return
	}

	var bn models.Banner

	if b.Content != nil {
		bn.Content = *b.Content
	}

	if b.FeatureId != nil {
		bn.FeatureID = *b.FeatureId
	}

	if b.IsActive != nil {
		bn.Active = *b.IsActive
	}

	if b.TagIds != nil {
		bn.Tags = *b.TagIds
	}

	id, err := s.bannerService.CreateBanner(r.Context(), bn)
	if err != nil {
		handleError(w, fmt.Errorf("create banner error: %w", err), http.StatusInternalServerError)

		return
	}

	resp := CreateBannerResponse{id}

	bts, err := json.Marshal(resp)
	if err != nil {
		handleError(w, fmt.Errorf("encode error: %w", err), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(bts) //nolint:errcheck
}

// Удаление баннера по идентификатору
// (DELETE /banner/{id}).
func (s Server) DeleteBannerId(w http.ResponseWriter, r *http.Request, id int, //nolint:revive,stylecheck
	params oapi.DeleteBannerIdParams,
) {
	w.Header().Add("Content-Type", "application/json")

	if params.Token == nil {
		handleError(w, fmt.Errorf("admin token required"), http.StatusUnauthorized) //nolint:perfsprint

		return
	}

	isAdmin, err := s.authService.Auth(*params.Token)
	if err != nil {
		handleError(w, fmt.Errorf("authorization error: %w", err), http.StatusUnauthorized)

		return
	}

	if !isAdmin {
		w.WriteHeader(http.StatusForbidden)

		return
	}

	if err := s.bannerService.DeleteBanner(r.Context(), id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		handleError(w, fmt.Errorf("delete banner error: %w", err), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Обновление содержимого баннера
// (PATCH /banner/{id}).
func (s Server) PatchBannerId(w http.ResponseWriter, r *http.Request, id int, //nolint:revive,cyclop,stylecheck
	params oapi.PatchBannerIdParams,
) {
	w.Header().Add("Content-Type", "application/json")

	if params.Token == nil {
		handleError(w, fmt.Errorf("admin token required"), http.StatusUnauthorized) //nolint:perfsprint

		return
	}

	isAdmin, err := s.authService.Auth(*params.Token)
	if err != nil {
		handleError(w, fmt.Errorf("authorization error: %w", err), http.StatusUnauthorized)

		return
	}

	if !isAdmin {
		w.WriteHeader(http.StatusForbidden)

		return
	}

	var b oapi.PatchBannerIdJSONBody

	dec := json.NewDecoder(r.Body)

	err = dec.Decode(&b)
	if err != nil {
		handleError(w, fmt.Errorf("decode error: %w", err), http.StatusBadRequest)

		return
	}

	var bn models.Banner

	if b.Content != nil {
		bn.Content = *b.Content
	}

	if b.FeatureId != nil {
		bn.FeatureID = *b.FeatureId
	}

	if b.IsActive != nil {
		bn.Active = *b.IsActive
	}

	if b.TagIds != nil {
		bn.Tags = *b.TagIds
	}

	bn.ID = int64(id)

	if err := s.bannerService.UpdateBanner(r.Context(), bn); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		handleError(w, fmt.Errorf("update banner error: %w", err), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
}

// Получение баннера для пользователя
// (GET /user_banner).
func (s Server) GetUserBanner(w http.ResponseWriter, r *http.Request, params oapi.GetUserBannerParams) { //nolint:revive
	w.Header().Add("Content-Type", "application/json")

	if params.Token == nil {
		handleError(w, fmt.Errorf("token required"), http.StatusUnauthorized) //nolint:perfsprint

		return
	}

	isAdmin, err := s.authService.Auth(*params.Token)
	if err != nil {
		handleError(w, fmt.Errorf("authorization error: %w", err), http.StatusUnauthorized)

		return
	}

	var req bannerservice.GetBannerRequest
	req.FeatureID = params.FeatureId
	req.Tags = []int{params.TagId}
	req.IsAdmin = isAdmin

	if params.UseLastRevision != nil {
		req.UseLastRevision = *params.UseLastRevision
	}

	banners, err := s.bannerService.GetBanner(r.Context(), req)
	if err != nil {
		handleError(w, fmt.Errorf("get banner error: %w", err), http.StatusInternalServerError)

		return
	}

	if len(banners) == 0 {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	i := rand.Intn(len(banners)) //nolint:gosec
	content := banners[i].Content

	enc := json.NewEncoder(w)

	err = enc.Encode(content)
	if err != nil {
		handleError(w, fmt.Errorf("encode error: %w", err), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
}

// Аутентификация пользователя
// (POST /auth).
func (s Server) PostAuth(w http.ResponseWriter, r *http.Request) {
	var b oapi.PostAuthJSONBody

	dec := json.NewDecoder(r.Body)

	err := dec.Decode(&b)
	if err != nil {
		handleError(w, fmt.Errorf("decode error: %w", err), http.StatusBadRequest)

		return
	}

	var username, password string

	if b.Password == nil || b.Username == nil {
		handleError(w, fmt.Errorf("not enought parameters to auth user"), http.StatusBadRequest) //nolint:perfsprint

		return
	}

	username = *b.Username
	password = *b.Password

	token, err := s.authService.Login(r.Context(), username, password)
	if err != nil {
		handleError(w, fmt.Errorf("login error: %w", err), http.StatusUnauthorized)

		return
	}

	resp := AuthUserResponse{Token: token}

	enc := json.NewEncoder(w)

	err = enc.Encode(resp)
	if err != nil {
		handleError(w, fmt.Errorf("encode error: %w", err), http.StatusInternalServerError)

		return
	}
}

// Создание пользователя
// (POST /user).
func (s Server) PostUser(w http.ResponseWriter, r *http.Request, params oapi.PostUserParams) {
	var b oapi.PostUserJSONBody

	dec := json.NewDecoder(r.Body)

	err := dec.Decode(&b)
	if err != nil {
		handleError(w, fmt.Errorf("decode error: %w", err), http.StatusBadRequest)

		return
	}

	if b.Password == nil || b.Username == nil || b.FeatureId == nil || b.TagIds == nil || b.Role == nil {
		handleError(w, fmt.Errorf("not enought parameters to call create user"), http.StatusBadRequest) //nolint:perfsprint

		return
	}

	var req authservice.CreateUserRequest

	req.Username = *b.Username
	req.Password = *b.Password
	req.Role = *b.Role
	req.Tags = *b.TagIds
	req.Feature = *b.FeatureId

	if params.Token != nil {
		req.Token = *params.Token
	}

	token, err := s.authService.CreateUser(r.Context(), req)
	if err != nil {
		handleError(w, fmt.Errorf("login error: %w", err), http.StatusUnauthorized)

		return
	}

	resp := CreateUserResponse{Token: token}

	bts, err := json.Marshal(resp)
	if err != nil {
		handleError(w, fmt.Errorf("encode error: %w", err), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(bts) //nolint:errcheck
}

func (s Server) GetDocs(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./docs/index.html")
}

func handleError(w http.ResponseWriter, err error, code int) {
	w.WriteHeader(code)

	e := Error{err.Error()}

	w.Write(e.ToJSON()) //nolint:errcheck
}
