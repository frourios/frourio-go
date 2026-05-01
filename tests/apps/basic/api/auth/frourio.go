package auth

type FrourioSpec struct {
	Middleware struct {
		All struct {
			Context struct {
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
