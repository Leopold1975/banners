package models

import (
	"time"
)

type Banner struct {
	FeatureID int                    `json:"feature_id"` //nolint:tagliatelle
	Tags      []int                  `json:"tag_ids"`    //nolint:tagliatelle
	Active    bool                   `json:"is_active"`  //nolint:tagliatelle
	UpdatedAt time.Time              `json:"updated_at"` //nolint:tagliatelle
	ID        int64                  `json:"banner_id"`  //nolint:tagliatelle
	CreatedAt time.Time              `json:"created_at"` //nolint:tagliatelle
	Content   map[string]interface{} `json:"content"`
}
