package forms

type TextHeader struct {
	ContentType string
}

// Form post body
/*
	Named request body schema.
		Keeps nested indentation.
*/
type FormPostBody struct {
	Name   string `json:"name" validate:"required"`
	Alias  string
	Age    int       `json:"age" validate:"gte=1"`
	Active bool      `json:"active"`
	Scores []float64 `json:"score" validate:"required"`
}

type MultipartResponseBody struct {
	Name  string `json:"name"`
	Count int
}

type FrourioSpec struct {
	Get struct {
		Res struct {
			Status200 struct {
				Header TextHeader
				Body   string `validate:"required"`
			}
		}
	}
	Post struct {
		URLEncoded bool
		Body       FormPostBody
		Res        struct {
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
	Patch struct {
		Res struct {
			Status200 struct {
				Body []byte
			}
		}
	}
	Delete struct {
		Res struct {
			Status200 struct {
				FormData bool
				Body     MultipartResponseBody
			}
		}
	}
}
