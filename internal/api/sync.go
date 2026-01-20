package api

import (
	"encoding/json"
	"time"

	"github.com/nzaccagnino/go-notes/internal/db"
)

type SyncResult struct {
	Uploaded   int
	Downloaded int
	Deleted    int
	Errors     []error
}

func Sync(database *db.DB, client *Client, lastSync int64) (*SyncResult, error) {
	result := &SyncResult{}

	// 1. Upload pending local changes
	pending, err := database.GetPendingNotes()
	if err != nil {
		return nil, err
	}

	for _, note := range pending {
		if note.Deleted {
			// Delete from server
			if note.ServerID != "" {
				if err := client.DeleteNote(note.ServerID); err != nil {
					result.Errors = append(result.Errors, err)
					continue
				}
				// Permanently delete local
				database.PermanentlyDeleteSynced(note.ID)
				result.Deleted++
			} else {
				// Never synced, just delete locally
				database.PermanentlyDeleteSynced(note.ID)
				result.Deleted++
			}
			continue
		}

		// Upload to server
		tagsJSON, _ := json.Marshal(note.Tags)
		req := UpsertNoteRequest{
			ID:        note.ServerID,
			Title:     note.Title,
			Content:   note.Content,
			Tags:      string(tagsJSON),
			CreatedAt: note.CreatedAt.Unix(),
			UpdatedAt: note.UpdatedAt.Unix(),
		}

		resp, err := client.UpsertNote(req)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		// Mark as synced with server ID
		database.SetNoteSynced(note.ID, resp.ID)
		result.Uploaded++
	}

	// 2. Download changes from server since last sync
	serverNotes, err := client.SyncNotes(lastSync)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result, nil
	}

	for _, sn := range serverNotes {
		err := database.UpsertFromServer(
			sn.ID,
			sn.Title,
			sn.Content,
			sn.Tags,
			time.Unix(sn.CreatedAt, 0),
			time.Unix(sn.UpdatedAt, 0),
		)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}
		result.Downloaded++
	}

	return result, nil
}
