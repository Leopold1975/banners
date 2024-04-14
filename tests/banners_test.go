package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Leopold1975/banners_control/internal/banners/api/oapi"
	"github.com/Leopold1975/banners_control/internal/banners/api/server"
	"github.com/Leopold1975/banners_control/internal/banners/app"
	"github.com/Leopold1975/banners_control/internal/banners/domain/models"
	"github.com/Leopold1975/banners_control/internal/pkg/config"

	"github.com/stretchr/testify/suite"
)

type BannerSuite struct {
	suite.Suite
	app    app.BannersApp
	cancel context.CancelFunc
	client *oapi.Client
}

var (
	defaultUserUsername = "default_user"
	defaultUserPassword = "qwerty"
	adminUsername       = "Admin"
	adminPassword       = "1234"
)

var banners = []models.Banner{
	{
		FeatureID: 5,
		Tags:      []int{1, 2, 3, 4, 5},
		Active:    true,
		Content: map[string]interface{}{
			"title": "some title",
			"text":  "some text",
			"url":   "some url",
		},
	},
	{
		FeatureID: 5,
		Tags:      []int{2, 6},
		Active:    true,
		Content: map[string]interface{}{
			"title": "another title",
			"text":  "another text",
			"url":   "another url",
		},
	},
	{
		FeatureID: 3,
		Tags:      []int{2, 6},
		Active:    true,
		Content: map[string]interface{}{
			"title": "simple title",
			"text":  "simple text",
			"url":   "simple url",
		},
	},
}

func (bs *BannerSuite) SetupSuite() {
	cmd := exec.Command("docker", "compose", "-f", "./test-compose.yaml", "up", "--build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		bs.T().Fatalf("cannot start docker compose error: %v", err)
	}

	cfg, err := config.New("config_test.yaml")
	if err != nil {
		bs.T().Fatalf("cannot get config error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())

	a, err := app.New(ctx, cfg)
	if err != nil {
		cancel()
		bs.T().Fatalf("cannot get app error: %v", err)
	}

	client, err := oapi.NewClient("http://" + cfg.Server.Addr + "/v1")
	if err != nil {
		cancel()
		bs.T().Fatalf("cannot get app error: %v", err)
	}

	bs.app = a
	bs.cancel = cancel
	bs.client = client

	go a.Run(ctx)
	time.Sleep(time.Second * 2) // Время для запуска сервера и баз данных.
}

func (bs *BannerSuite) TearDownSuite() {
	bs.cancel()

	cmd := exec.Command("docker", "compose", "-f", "./test-compose.yaml", "down", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		bs.T().Fatalf("cannot down docker conatainers error: %v", err)
	}
}

func (bs *BannerSuite) TestGetBanner() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Админ проходит аутентификацию
	req := oapi.PostAuthJSONRequestBody(
		oapi.PostAuthJSONBody{
			Username: &adminUsername,
			Password: &adminPassword,
		},
	)

	resp, err := bs.client.PostAuth(ctx, req)
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)

	var respToken server.AuthUserResponse
	err = dec.Decode(&respToken)
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)

	// Админ создает баннеры
	for i, b := range banners {
		resp, err := bs.client.PostBanner(ctx, &oapi.PostBannerParams{Token: &respToken.Token}, oapi.PostBannerJSONRequestBody(
			oapi.PostBannerJSONBody{
				Content:   &b.Content,
				FeatureId: &b.FeatureID,
				TagIds:    &b.Tags,
				IsActive:  &b.Active,
			},
		))
		bs.Require().NoError(err, "expected %v	actual %v", nil, err)
		bs.Require().Equal(http.StatusCreated, resp.StatusCode)

		var r server.CreateBannerResponse
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&r)
		bs.Require().NoError(err, "expected %v	actual %v", nil, err)
		bs.Require().Equal(i+1, r.BannerID)
		resp.Body.Close()
	}

	// Пользователь проходит аутентификацию
	req = oapi.PostAuthJSONRequestBody(
		oapi.PostAuthJSONBody{
			Username: &defaultUserUsername,
			Password: &defaultUserPassword,
		},
	)

	resp, err = bs.client.PostAuth(ctx, req)
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	dec = json.NewDecoder(resp.Body)

	err = dec.Decode(&respToken)
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)

	// Пользователь получает один из актуальных для него баннеров
	resp, err = bs.client.GetUserBanner(ctx, &oapi.GetUserBannerParams{
		Token:     &respToken.Token,
		FeatureId: 5,
		TagId:     2,
	})
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	userBannerResp := server.GetUserBannerResponse{
		Banner: make(map[string]interface{}),
	}

	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&userBannerResp.Banner)
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)

	bs.Require().True(banners[0].Content["title"] == userBannerResp.Banner["title"] || banners[1].Content["title"] == userBannerResp.Banner["title"])
}

func (bs *BannerSuite) TestOtherScenarios() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Админ проходит аутентификацию
	req := oapi.PostAuthJSONRequestBody(
		oapi.PostAuthJSONBody{
			Username: &adminUsername,
			Password: &adminPassword,
		},
	)

	resp, err := bs.client.PostAuth(ctx, req)
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)

	dec := json.NewDecoder(resp.Body)

	var respToken server.AuthUserResponse
	err = dec.Decode(&respToken)
	resp.Body.Close()
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	adminToken := respToken.Token

	// Пользователь проходит аутентификацию
	req = oapi.PostAuthJSONRequestBody(
		oapi.PostAuthJSONBody{
			Username: &defaultUserUsername,
			Password: &defaultUserPassword,
		},
	)

	resp, err = bs.client.PostAuth(ctx, req)
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)

	dec = json.NewDecoder(resp.Body)

	err = dec.Decode(&respToken)
	resp.Body.Close()

	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	userToken := respToken.Token

	bs.Require().NotEqual(adminToken, userToken)

	// Пользователь запрашивает конкретный баннер
	resp, err = bs.client.GetUserBanner(ctx, &oapi.GetUserBannerParams{
		Token:     &userToken,
		FeatureId: 3,
		TagId:     2,
	})
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)

	userBannerResp := server.GetUserBannerResponse{
		Banner: make(map[string]interface{}),
	}

	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&userBannerResp.Banner)
	resp.Body.Close()

	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(banners[2].Content["title"], userBannerResp.Banner["title"])

	// Админ получает список баннеров
	resp, err = bs.client.GetBanner(ctx, &oapi.GetBannerParams{
		Token: &adminToken,
	})
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)

	var bannersFromDB []models.Banner

	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&bannersFromDB)
	resp.Body.Close()

	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(3, len(bannersFromDB))

	// Админ изменяет баннер
	resp, err = bs.client.PatchBannerId(ctx, 3, &oapi.PatchBannerIdParams{
		Token: &adminToken,
	}, oapi.PatchBannerIdJSONRequestBody(
		oapi.PatchBannerIdJSONBody{
			Content: &map[string]interface{}{
				"title": "not simple title",
				"text":  "not simple text",
				"url":   "not simple url",
			},
			FeatureId: &banners[2].FeatureID,
			TagIds:    &banners[2].Tags,
			IsActive:  &banners[2].Active,
		},
	))
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Пользователь запрашивает измененный баннер, получает необновленное значение из кэша
	resp, err = bs.client.GetUserBanner(ctx, &oapi.GetUserBannerParams{
		Token:     &userToken,
		FeatureId: 3,
		TagId:     2,
	})
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)

	userBannerResp = server.GetUserBannerResponse{
		Banner: make(map[string]interface{}),
	}

	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&userBannerResp.Banner)
	resp.Body.Close()

	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(banners[2].Content["title"], userBannerResp.Banner["title"])

	// Пользователь передает флаг useLastRevision, получает актуальный баннер
	trueV := true
	resp, err = bs.client.GetUserBanner(ctx, &oapi.GetUserBannerParams{
		Token:           &userToken,
		FeatureId:       3,
		TagId:           2,
		UseLastRevision: &trueV,
	})
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)

	userBannerResp = server.GetUserBannerResponse{
		Banner: make(map[string]interface{}),
	}

	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&userBannerResp.Banner)
	resp.Body.Close()

	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal("not simple title", userBannerResp.Banner["title"])

	// Админ удаляет баннер
	resp, err = bs.client.DeleteBannerId(ctx, 3, &oapi.DeleteBannerIdParams{
		Token: &adminToken,
	})
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Пользователь запрашивает удаленный баннер, не получает данных
	resp, err = bs.client.GetUserBanner(ctx, &oapi.GetUserBannerParams{
		Token:     &userToken,
		FeatureId: 3,
		TagId:     2,
	})
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusNotFound, resp.StatusCode)

	// Пользователь запрашивает конкретный баннер
	resp, err = bs.client.GetUserBanner(ctx, &oapi.GetUserBannerParams{
		Token:     &userToken,
		FeatureId: 5,
		TagId:     6,
	})
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)

	userBannerResp = server.GetUserBannerResponse{
		Banner: make(map[string]interface{}),
	}

	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&userBannerResp.Banner)
	resp.Body.Close()

	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(banners[1].Content["title"], userBannerResp.Banner["title"])

	// Админ делает баннер неактивным
	inactive := false
	resp, err = bs.client.PatchBannerId(ctx, 2, &oapi.PatchBannerIdParams{
		Token: &adminToken,
	}, oapi.PatchBannerIdJSONRequestBody(
		oapi.PatchBannerIdJSONBody{
			Content: &map[string]interface{}{
				"title": "another title",
				"text":  "another text",
				"url":   "another url",
			},
			FeatureId: &banners[1].FeatureID,
			TagIds:    &banners[1].Tags,
			IsActive:  &inactive,
		},
	))
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Пользователь получает неактивный баннер из кэша
	resp, err = bs.client.GetUserBanner(ctx, &oapi.GetUserBannerParams{
		Token:     &userToken,
		FeatureId: 5,
		TagId:     6,
	})
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusOK, resp.StatusCode)

	time.Sleep(time.Second * 2)

	// Пользователь не получает неактивный баннер из кэша
	resp, err = bs.client.GetUserBanner(ctx, &oapi.GetUserBannerParams{
		Token:     &userToken,
		FeatureId: 5,
		TagId:     6,
	})
	bs.Require().NoError(err, "expected %v	actual %v", nil, err)
	bs.Require().Equal(http.StatusNotFound, resp.StatusCode)
}

func TestBannerServiceSuite(t *testing.T) {
	suite.Run(t, new(BannerSuite))
}
