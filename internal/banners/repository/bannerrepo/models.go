package bannerrepo

import "errors"

var ErrNotFound = errors.New("banner not found")

type GetBannerRequest struct {
	FeatureID  int
	Tags       []int
	Offset     int
	Limit      int
	OnlyActive bool
}
