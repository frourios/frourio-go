package api

type FrourioSpec struct {
	Get struct {
		Query struct {
			Search  *string `json:"search"`
			Limit   *int    `json:"limit"`
			RawName *string
			Active  *bool     `json:"active"`
			Scores  []float64 `json:"score"`
		}
		Res struct {
			Status200 struct {
				body string `validate:"required"`
			}
		}
	}
}
