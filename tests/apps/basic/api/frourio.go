package api

type FrourioSpec struct {
	Get struct {
		Query struct {
			Search *string `json:"search"`
			Limit  *int    `json:"limit"`
		}
		Res struct {
			Status200 struct {
				body string `validate:"required"`
			}
		}
	}
}
