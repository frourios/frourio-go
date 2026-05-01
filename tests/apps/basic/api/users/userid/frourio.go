package userid

type FrourioSpec struct {
	Get struct {
		Param int `validate:"required"`
		Res   struct {
			Status200 struct {
				Body string `json:"body" validate:"required"`
			}
			Status404 struct {
				Body string `json:"body" validate:"required"`
			}
		}
	}
}
