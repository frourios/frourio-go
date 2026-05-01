package forms

type FrourioSpec struct {
	Post struct {
		URLEncoded bool
		Body       struct {
			Name   string    `json:"name" validate:"required"`
			Age    int       `json:"age" validate:"gte=1"`
			Active bool      `json:"active"`
			Scores []float64 `json:"score" validate:"required"`
		}
		Res struct {
			Status201 struct {
				Body string `json:"body" validate:"required"`
			}
		}
	}
	Put struct {
		FormData bool
		Body     struct {
			Title string `json:"title" validate:"required"`
			Count uint8  `json:"count" validate:"gte=1"`
		}
		Res struct {
			Status200 struct {
				Body string `json:"body" validate:"required"`
			}
		}
	}
}
