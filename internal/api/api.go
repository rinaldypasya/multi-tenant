package api

import (
	"multi-tenant/internal/config"
	"multi-tenant/internal/manager"
	"multi-tenant/internal/storage"
	"multi-tenant/internal/worker"

	"github.com/google/uuid"
)

type API struct {
	TenantMgr  *manager.TenantManager
	Storage    *storage.Storage
	WorkerPool map[uuid.UUID]*worker.WorkerPool
	Cfg        *config.Config
}

func NewAPI(tm *manager.TenantManager, db *storage.Storage, cfg *config.Config) *API {
	return &API{
		TenantMgr:  tm,
		Storage:    db,
		WorkerPool: make(map[uuid.UUID]*worker.WorkerPool),
		Cfg:        cfg,
	}
}
