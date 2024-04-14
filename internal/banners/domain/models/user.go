package models

type User struct {
	ID           int    `json:"user_id"` //nolint:tagliatelle
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"` //nolint:tagliatelle
	Role         string `json:"role"`
	Feature      int    `json:"feature_id"` //nolint:tagliatelle
	Tags         []int  `json:"tag_ids"`    //nolint:tagliatelle
}
