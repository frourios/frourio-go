package api

type FrourioSpec struct {
	// List root
	/*
		List root endpoint.
			Indented detail is preserved.
	*/
	Get struct {
		Query struct {
			// Search term
			Search *string `json:"search"`
			/*
				Maximum number of items.
					Indented limit detail.
			*/
			Limit   *int `json:"limit"`
			RawName *string
			Active  *bool     `json:"active"`
			Scores  []float64 `json:"score"`
		}
		Res struct {
			// Successful root response
			Status200 struct {
				body string `validate:"required"`
			}
		}
	}
}
