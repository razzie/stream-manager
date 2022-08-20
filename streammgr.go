package main

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	generic_sync "github.com/SaveTheRbtz/generic-sync-map-go"
	"github.com/razzie/beepboop"
	"github.com/razzie/stream-manager/template"
)

var (
	ErrNotFound = fmt.Errorf("not found")
)

type StreamView struct {
	Name          string
	Source        string
	StartPosition string
	Audio         int
	Subtitle      int
	Status        string
	Actions       []string
}

func NewStreamView(stream *Stream) *StreamView {
	view := &StreamView{
		Name:          stream.Name,
		Source:        stream.Source,
		StartPosition: stream.StartPosition,
		Audio:         stream.Audio,
		Subtitle:      stream.Subtitle,
		Status:        stream.Status(),
		Actions:       []string{"start", "stop", "delete"},
	}
	return view
}

type StreamManager struct {
	target  string
	streams generic_sync.MapOf[string, *Stream]
}

func NewStreamManager(target string) *StreamManager {
	return &StreamManager{
		target: target,
	}
}

func (sm *StreamManager) Launch(name, source, startpos string, audio, subs int) error {
	stream, err := NewStream(name, source, sm.target, startpos, audio, subs)
	if err != nil {
		return err
	}
	if _, loaded := sm.streams.LoadOrStore(name, stream); loaded {
		return fmt.Errorf("stream name already exists")
	}
	return nil
}

func (sm *StreamManager) Streams() (results []*StreamView) {
	sm.streams.Range(func(name string, stream *Stream) bool {
		results = append(results, NewStreamView(stream))
		return true
	})
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})
	return
}

func (sm *StreamManager) Start(name string) error {
	if stream, ok := sm.streams.Load(name); ok {
		return stream.Start()
	}
	return ErrNotFound
}

func (sm *StreamManager) Stop(name string) error {
	if stream, ok := sm.streams.Load(name); ok {
		return stream.Close()
	}
	return ErrNotFound
}

func (sm *StreamManager) Delete(name string) error {
	if stream, ok := sm.streams.Load(name); ok {
		sm.streams.Delete(name)
		return stream.Close()
	}
	return ErrNotFound
}

func (sm *StreamManager) Pages() []*beepboop.Page {
	return []*beepboop.Page{
		{
			Path:            "/",
			ContentTemplate: template.Streams,
			Handler:         sm.handleStreams,
		},
		{
			Path:            "/launch",
			ContentTemplate: template.Launch,
			Handler:         sm.handleLaunch,
		},
		{
			Path:            "/probe",
			ContentTemplate: template.Probe,
			Handler:         sm.handleProbe,
		},
		{
			Path:    "/start/",
			Handler: sm.handleStart,
		},
		{
			Path:    "/stop/",
			Handler: sm.handleStop,
		},
		{
			Path:    "/delete/",
			Handler: sm.handleDelete,
		},
	}
}

func (sm *StreamManager) handleStreams(r *beepboop.PageRequest) *beepboop.View {
	return r.Respond(sm.Streams())
}

func (sm *StreamManager) handleLaunch(r *beepboop.PageRequest) *beepboop.View {
	req := r.Request
	if req.Method == "POST" {
		req.ParseForm()
		name := req.FormValue("name")
		source := req.FormValue("source")
		startpos := req.FormValue("startpos")
		audio, _ := strconv.ParseInt(req.FormValue("audio"), 10, 64)
		subtitle, _ := strconv.ParseInt(req.FormValue("subtitle"), 10, 64)
		if err := sm.Launch(name, source, startpos, int(audio), int(subtitle)); err != nil {
			return r.ErrorView(err.Error(), http.StatusBadRequest)
		}
		return r.RedirectView("/")
	}
	return nil
}

func (sm *StreamManager) handleProbe(r *beepboop.PageRequest) *beepboop.View {
	req := r.Request
	if req.Method == "POST" {
		req.ParseForm()
		source := req.FormValue("source")
		result, err := Probe(r.Context.Context, source)
		if err != nil {
			return r.ErrorView(err.Error(), http.StatusBadRequest)
		}
		return r.HandlerView(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(result)
		})
	}
	return nil
}

func (sm *StreamManager) handleStart(r *beepboop.PageRequest) *beepboop.View {
	name := r.RelPath
	if err := sm.Start(name); err != nil {
		return handleError(r, err)
	}
	return r.RedirectView("/")
}

func (sm *StreamManager) handleStop(r *beepboop.PageRequest) *beepboop.View {
	name := r.RelPath
	if err := sm.Stop(name); err != nil {
		return handleError(r, err)
	}
	return r.RedirectView("/")
}

func (sm *StreamManager) handleDelete(r *beepboop.PageRequest) *beepboop.View {
	name := r.RelPath
	if err := sm.Delete(name); err != nil {
		return handleError(r, err)
	}
	return r.RedirectView("/")
}

func handleError(r *beepboop.PageRequest, err error) *beepboop.View {
	if errors.Is(err, ErrNotFound) {
		return r.ErrorView(err.Error(), http.StatusNotFound)
	}
	return r.ErrorView(err.Error(), http.StatusInternalServerError)
}
