package users

type FrourioSpec struct {
	Get struct {
		Query struct {
			Limit *int `json:"limit"`
		}
		Res struct {
			Status200 struct {
				Body []string `json:"body"`
			}
		}
	}
	Post struct {
		Body struct {
			Name string `json:"name" validate:"required"`
			Age  *int   `json:"age"`
		}
		Res struct {
			Status201 struct {
				Body string `json:"body" validate:"required"`
			}
			Status400 struct {
				Body string `json:"body" validate:"required"`
			}
		}
	}
}
