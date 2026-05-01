package secure

type FrourioSpec struct {
	Middleware struct {
		All struct {
			Context struct {
				UserID  string `json:"userId" validate:"required"`
				TraceID string `json:"traceId" validate:"required"`
			}
		}
	}
	Get struct {
		Res struct {
			Status200 struct {
				Body string `json:"body" validate:"required"`
			}
		}
	}
}
