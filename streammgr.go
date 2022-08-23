package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	generic_sync "github.com/SaveTheRbtz/generic-sync-map-go"
	"github.com/go-redis/redis/v8"
	"github.com/razzie/beepboop"
	"github.com/razzie/stream-manager/template"
)

var (
	ErrNotFound = fmt.Errorf("not found")
)

type StreamEntry struct {
	Name            string        `json:"name"`
	Source          string        `json:"source"`
	StartPosition   time.Duration `json:"startpos"`
	VideoChannel    int           `json:"video"`
	AudioChannel    int           `json:"audio"`
	SubtitleChannel int           `json:"subtitle"`
}

type StreamView struct {
	StreamEntry
	Status  string
	Actions []string
}

func NewStreamView(stream *Stream) *StreamView {
	view := &StreamView{
		StreamEntry: StreamEntry{
			Name:            stream.Name,
			Source:          stream.Source,
			StartPosition:   stream.StartPosition,
			VideoChannel:    stream.VideoChannel,
			AudioChannel:    stream.AudioChannel,
			SubtitleChannel: stream.SubtitleChannel,
		},
		Status:  stream.Status(),
		Actions: []string{"start", "stop", "delete"},
	}
	if len(view.Source) > 128 {
		view.Source = "..." + view.Source[len(view.Source)-100:]
	}
	return view
}

type StreamManager struct {
	target  string
	streams generic_sync.MapOf[string, *Stream]
	db      *redis.Client
}

func NewStreamManager(target string, opt *redis.Options) *StreamManager {
	sm := &StreamManager{
		target: target,
	}
	if opt != nil {
		sm.db = redis.NewClient(opt)
		sm.loadStreamsFromDB()
	}
	return sm
}

func (sm *StreamManager) loadStreamsFromDB() {
	streamNames, err := sm.db.Keys(context.Background(), "*").Result()
	if err != nil {
		log.Println("db error:", err)
	}
	for _, name := range streamNames {
		var entry StreamEntry
		entryStr, err := sm.db.Get(context.Background(), name).Result()
		if err != nil {
			log.Println("db error:", err)
			continue
		}
		if err := json.Unmarshal([]byte(entryStr), &entry); err != nil {
			log.Println("json error:", err)
			continue
		}
		if err := sm.launchInternal(&entry); err != nil {
			log.Println("error while adding stream to list:", err)
		}
	}
}

func (sm *StreamManager) launchInternal(entry *StreamEntry) error {
	stream, err := NewStream(
		entry.Name,
		entry.Source,
		sm.target,
		entry.StartPosition,
		entry.VideoChannel,
		entry.AudioChannel,
		entry.SubtitleChannel)
	if err != nil {
		return err
	}
	if _, loaded := sm.streams.LoadOrStore(entry.Name, stream); loaded {
		return fmt.Errorf("stream name already exists")
	}
	return nil
}

func (sm *StreamManager) Launch(entry *StreamEntry) error {
	if err := sm.launchInternal(entry); err != nil {
		return err
	}
	if sm.db != nil {
		entryStr, err := json.Marshal(entry)
		if err != nil {
			log.Println("json error:", err)
			return nil
		}
		if err := sm.db.Set(context.Background(), entry.Name, string(entryStr), 0).Err(); err != nil {
			log.Println("error while saving stream to db:", err)
		}
	}
	return nil
}

func (sm *StreamManager) Stream(name string) *StreamView {
	stream, _ := sm.streams.Load(name)
	if stream != nil {
		return NewStreamView(stream)
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
		if sm.db != nil {
			if err := sm.db.Del(context.Background(), name).Err(); err != nil {
				log.Println("error while deleting stream from db:", err)
			}
		}
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
		toInt := func(str string) int {
			val, _ := strconv.ParseInt(str, 10, 32)
			return int(val)
		}
		req.ParseForm()
		var entry StreamEntry
		entry.Name = req.FormValue("name")
		entry.Source = req.FormValue("source")
		entry.StartPosition, _ = time.ParseDuration(req.FormValue("startpos"))
		entry.VideoChannel = toInt(req.FormValue("video"))
		entry.AudioChannel = toInt(req.FormValue("audio"))
		entry.SubtitleChannel = toInt(req.FormValue("subtitle"))
		if err := sm.Launch(&entry); err != nil {
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
