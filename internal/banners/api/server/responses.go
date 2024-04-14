package server

type CreateBannerResponse struct {
	BannerID int `json:"banner_id"` //nolint:tagliatelle
}

type GetUserBannerResponse struct {
	Banner map[string]interface{}
}

type AuthUserResponse struct {
	Token string `json:"token"`
}

type CreateUserResponse struct {
	Token string `json:"token"`
}
