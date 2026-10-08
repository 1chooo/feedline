package model

// AdminAnalyticsSummary is deliberately shaped for dashboard cards and chart
// series, while keeping every number traceable to a persisted table.
type AdminAnalyticsSummary struct {
	From        string                  `json:"from"`
	To          string                  `json:"to"`
	Users       AdminUserMetrics        `json:"users"`
	Content     AdminContentMetrics     `json:"content"`
	Advertising AdminAdvertisingMetrics `json:"advertising"`
	Billing     AdminBillingMetrics     `json:"billing"`
	Retention   AdminRetentionMetrics   `json:"retention"`
	Daily       []AdminAnalyticsDaily   `json:"daily"`
}

type AdminUserMetrics struct {
	Total            int64 `json:"total"`
	ActiveInRange    int64 `json:"activeInRange"`
	NewRegistrations int64 `json:"newRegistrations"`
	DAU              int64 `json:"dau"`
	WAU              int64 `json:"wau"`
	MAU              int64 `json:"mau"`
}

type AdminContentMetrics struct {
	PostsCreated       int64 `json:"postsCreated"`
	SocialInteractions int64 `json:"socialInteractions"`
}

type AdminAdvertisingMetrics struct {
	Advertisers        int64   `json:"advertisers"`
	Companies          int64   `json:"companies"`
	PayingCompanies    int64   `json:"payingCompanies"`
	ActiveCampaigns    int64   `json:"activeCampaigns"`
	ScheduledCampaigns int64   `json:"scheduledCampaigns"`
	CompletedCampaigns int64   `json:"completedCampaigns"`
	CanceledCampaigns  int64   `json:"canceledCampaigns"`
	PausedCampaigns    int64   `json:"pausedCampaigns"`
	Impressions        int64   `json:"impressions"`
	Clicks             int64   `json:"clicks"`
	EngagementRate     float64 `json:"engagementRate"`
}

type AdminBillingMetrics struct {
	RevenueCents       int64 `json:"revenueCents"`
	Purchases          int64 `json:"purchases"`
	CreditsPurchased   int64 `json:"creditsPurchased"`
	CreditsUsed        int64 `json:"creditsUsed"`
	CouponRedemptions  int64 `json:"couponRedemptions"`
	PromotionalCredits int64 `json:"promotionalCredits"`
}

type AdminRetentionMetrics struct {
	Day1Percent  float64 `json:"day1Percent"`
	Day7Percent  float64 `json:"day7Percent"`
	Day30Percent float64 `json:"day30Percent"`
}

type AdminAnalyticsDaily struct {
	Date         string `json:"date"`
	NewUsers     int64  `json:"newUsers"`
	ActiveUsers  int64  `json:"activeUsers"`
	PostsCreated int64  `json:"postsCreated"`
	RevenueCents int64  `json:"revenueCents"`
	Purchases    int64  `json:"purchases"`
	Impressions  int64  `json:"impressions"`
	Clicks       int64  `json:"clicks"`
}

// MarketingSummary deliberately exposes only aggregate, non-identifying figures
// that are safe for the public advertising landing page.
type MarketingSummary struct {
	MonthlyActiveUsers int64 `json:"monthlyActiveUsers"`
	ActiveAdvertisers  int64 `json:"activeAdvertisers"`
	ActiveCampaigns    int64 `json:"activeCampaigns"`
	Impressions30d     int64 `json:"impressions30d"`
	Posts30d           int64 `json:"posts30d"`
}
