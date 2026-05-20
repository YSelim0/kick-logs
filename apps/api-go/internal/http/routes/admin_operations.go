package routes

import "net/http"

func RegisterAdminOperationRoutes(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("GET /admin/operations/summary", func(response http.ResponseWriter, request *http.Request) {
		operationsSummary(response, request, deps)
	})
	mux.HandleFunc("GET /admin/operations/failed-events", func(response http.ResponseWriter, request *http.Request) {
		listFailedEvents(response, request, deps)
	})
	mux.HandleFunc("POST /admin/operations/failed-events/retry", func(response http.ResponseWriter, request *http.Request) {
		retryFailedEvents(response, request, deps)
	})
	mux.HandleFunc("POST /admin/operations/failed-events/clear", func(response http.ResponseWriter, request *http.Request) {
		clearFailedEvents(response, request, deps)
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

func listFailedEvents(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.Operations == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	events, err := deps.Operations.ListFailedEvents(request.Context(), 50)
	if err != nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	writeJSON(response, http.StatusOK, failedRawEventsResponse(events))
}

func retryFailedEvents(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.Operations == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	affected, err := deps.Operations.RetryFailedEvents(request.Context())
	if err != nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"affected": affected, "message": "Failed events queued for retry."})
}

func clearFailedEvents(response http.ResponseWriter, request *http.Request, deps Dependencies) {
	if _, ok := requireAdmin(response, request, deps.Auth, deps.Config); !ok {
		return
	}
	if deps.Operations == nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	affected, err := deps.Operations.ClearFailedEvents(request.Context())
	if err != nil {
		writeError(response, http.StatusInternalServerError, "Internal server error.")
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"affected": affected, "message": "Failed events cleared."})
}
