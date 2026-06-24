package handler

import (
	"net/http"

	"github.com/hequan2017/new-api-proxy/internal/provider/ark"
)

// Embeddings POST /v1/embeddings —— 透传方舟。
func (d *Deps) Embeddings(w http.ResponseWriter, r *http.Request) {
	d.relayToArk(w, r, ark.PathEmbeddings)
}
