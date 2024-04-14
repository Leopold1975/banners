package authservice

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Tags     []int  `json:"tag_ids"`    //nolint:tagliatelle
	Feature  int    `json:"feature_id"` //nolint:tagliatelle
	Token    string `json:"token"`
}
