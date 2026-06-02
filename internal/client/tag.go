package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// UserTag represents a monitor tag returned by the public tags API.
type UserTag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// TagListResponse represents one page of user tags.
type TagListResponse struct {
	Data         []UserTag `json:"data"`
	NextCursorID *int64    `json:"nextCursorId"`
}

// ListTags lists user tags. If cursorID is nil, the first page is returned.
func (c *Client) ListTags(ctx context.Context, cursorID *int64) (*TagListResponse, error) {
	path := "/tags"
	if cursorID != nil {
		path += "?cursor=" + strconv.FormatInt(*cursorID, 10)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var tags TagListResponse
	if err := json.Unmarshal(resp, &tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags response: %w", err)
	}
	return &tags, nil
}

// ListAllTags follows pagination and returns all tags visible to the API key.
func (c *Client) ListAllTags(ctx context.Context) ([]UserTag, error) {
	var out []UserTag
	var cursorID *int64
	seenCursors := make(map[int64]struct{})

	const maxPages = 1000
	for page := 0; page < maxPages; page++ {
		resp, err := c.ListTags(ctx, cursorID)
		if err != nil {
			return nil, err
		}

		out = append(out, resp.Data...)
		if resp.NextCursorID == nil {
			return out, nil
		}
		if _, seen := seenCursors[*resp.NextCursorID]; seen {
			return nil, fmt.Errorf("tags pagination cursor repeated (%d)", *resp.NextCursorID)
		}
		seenCursors[*resp.NextCursorID] = struct{}{}
		cursorID = resp.NextCursorID
	}

	return nil, fmt.Errorf("tags pagination exceeded %d pages", maxPages)
}
