// Command server is the BroadcastPlanner backend entrypoint.
//
// Milestone 1 focuses on the ported domain + export core (see internal/domain and
// internal/exporter, covered by tests). This entrypoint currently exposes a
// health endpoint and a demo TELE export so the Docker image builds and runs; the
// full REST/WS API lands in Milestone 2.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/broadcastplanner/backend/internal/domain"
	"github.com/broadcastplanner/backend/internal/exporter"
)

func main() {
	addr := os.Getenv("BP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "milestone": "1-core"})
	})

	// Demo: POST a playlist JSON, receive a TELE CSV. Illustrates the export core;
	// proper resource-oriented endpoints follow in M2.
	mux.HandleFunc("POST /api/export/tele", handleExportTele)

	log.Printf("BroadcastPlanner backend listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

type teleExportRequest struct {
	FrameRate string                       `json:"frameRate"`
	Preset    string                       `json:"preset"`
	Playlist  []domain.PlaylistItem        `json:"playlist"`
	Markers   map[string][]domain.AdMarker `json:"markers"`
}

func handleExportTele(w http.ResponseWriter, r *http.Request) {
	var req teleExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	fr := domain.FrameRateFromRaw(req.FrameRate)
	preset := exporter.Preset(req.Preset)
	if preset == "" {
		preset = exporter.PresetTeleUTF8
	}
	data := exporter.TeleCSV(req.Playlist, req.Markers, fr, preset)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="playlist_tele.csv"`)
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
