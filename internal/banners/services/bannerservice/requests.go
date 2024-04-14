package bannerservice

type GetBannerRequest struct {
	FeatureID       int
	Tags            []int
	Offset          int
	Limit           int
	IsAdmin         bool
	UseLastRevision bool
}
