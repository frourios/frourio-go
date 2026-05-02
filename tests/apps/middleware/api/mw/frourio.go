package mw

type FrourioSpec struct {
	Middleware struct {
		All struct {
			Context struct {
				UserID  *string `json:"userId"`
				TraceID string  `json:"traceId" validate:"required"`
			}
		}
	}
	Get struct {
		Res struct {
			Status200 struct {
				Body GetResponseBody
			}
		}
	}
}

type GetResponseBody struct {
	UserID  *string `json:"userId,omitempty"`
	TraceID string  `json:"traceId"`
}
