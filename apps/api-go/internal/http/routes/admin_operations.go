package routes

import "net/http"

func RegisterAdminOperationRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /admin/operations/summary", func(response http.ResponseWriter, request *http.Request) {
		operationsSummary(response, request, deps)
	})
}

func operationsSummary(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.Operations == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}

	summary, err := deps.Operations.Summary(request.Context())
	if err != nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	writeJSON(response, http.StatusOK, operationsSummaryResponse(summary))
}
