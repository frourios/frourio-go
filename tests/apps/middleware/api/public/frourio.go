package public

type PublicResponseBody struct {
	Message string `json:"message" validate:"required"`
}

type FrourioSpec struct {
	Get struct {
		Res struct {
			Status200 struct {
				Body PublicResponseBody
			}
		}
	}
}
