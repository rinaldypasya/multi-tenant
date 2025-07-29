package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"multi-tenant/internal/auth"
)

func (a *API) Router() http.Handler {
	// Public
	a.Routers.Post("/tenants", a.CreateTenant)
	a.Routers.Delete("/tenants/{id}", a.DeleteTenant)

	// Secured
	a.Routers.Group(func(r chi.Router) {
		r.Use(auth.JWTAuthMiddleware)

		r.Put("/tenants/{id}/config/concurrency", a.UpdateConcurrency)
		r.Get("/messages", a.ListMessages)
	})

	return a.Routers
}

// @Summary Create a tenant
// @Tags Tenants
// @Produce json
// @Success 200 {object} map[string]string
// @Router /tenants [post]
func (a *API) CreateTenant(w http.ResponseWriter, r *http.Request) {
	id := uuid.New()

	if err := a.TenantMgr.AddTenant(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := a.TenantMgr.SetWorkerCount(id.String(), a.Cfg.Workers); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("API: Created tenant %s", id)
	json.NewEncoder(w).Encode(map[string]string{"tenant_id": id.String()})
}

// @Summary Delete a tenant
// @Tags Tenants
// @Param id path string true "Tenant UUID"
// @Success 204
// @Router /tenants/{id} [delete]
func (a *API) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid tenant id", http.StatusBadRequest)
		return
	}

	_ = a.TenantMgr.RemoveTenant(id)

	if pool, ok := a.WorkerPool[id]; ok {
		pool.Stop()
		delete(a.WorkerPool, id)
	}

	log.Printf("API: Deleted tenant %s", id)
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Update worker pool concurrency
// @Tags Tenants
// @Security ApiKeyAuth
// @Param body body ConcurrencyConfig true "Concurrency config"
// @Success 204
// @Router /tenants/{id}/config/concurrency [put]
func (a *API) UpdateConcurrency(w http.ResponseWriter, r *http.Request) {
	tenantStr := auth.GetTenantID(r)
	id, err := uuid.Parse(tenantStr)
	if err != nil {
		http.Error(w, "unauthorized tenant", http.StatusUnauthorized)
		return
	}

	var body ConcurrencyConfig
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request body", http.StatusBadRequest)
		return
	}

	if err := a.TenantMgr.SetWorkerCount(id.String(), a.Cfg.Workers); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Error(w, "tenant not found", http.StatusNotFound)
}

// @Summary List messages by tenant
// @Tags Messages
// @Security ApiKeyAuth
// @Produce json
// @Param cursor query string false "Pagination cursor"
// @Success 200 {object} map[string]interface{}
// @Router /messages [get]
func (a *API) ListMessages(w http.ResponseWriter, r *http.Request) {
	tenantStr := auth.GetTenantID(r)
	tenantID, err := uuid.Parse(tenantStr)
	if err != nil {
		http.Error(w, "unauthorized tenant", http.StatusUnauthorized)
		return
	}

	cursorStr := r.URL.Query().Get("cursor")

	messages, nextCursor, err := a.Storage.ListMessagesPaginated(tenantID, cursorStr, 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"data":        messages,
		"next_cursor": nextCursor,
	}
	json.NewEncoder(w).Encode(resp)
}
