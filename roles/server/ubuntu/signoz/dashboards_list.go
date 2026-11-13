package signoz

import (
	"context"
	"fmt"
)

// list reads every dashboard of SigNoz. SigNoz lets dashboards share a name, and its filter matches
// the name its UI shows, which can be changed there, so the step finds its own by the fixed name.
// The API gives 200 at most, and the total, by which the step reads the rest.
func (s Dashboards) list(ctx context.Context) ([]dashboard, error) {
	var all []dashboard
	for offset := 0; ; offset += 200 {
		var got struct {
			Data struct {
				Dashboards []dashboard `json:"dashboards"`
				Total      int         `json:"total"`
			} `json:"data"`
		}
		q := fmt.Sprintf("/api/v2/dashboards?sort=created_at&order=asc&limit=200&offset=%d", offset)
		if err := s.API.Get(ctx, q, &got); err != nil {
			return nil, err
		}
		all = append(all, got.Data.Dashboards...)
		if offset+200 >= got.Data.Total || len(got.Data.Dashboards) == 0 {
			return all, nil
		}
	}
}

func indexOf(ds []dashboard, name string) int {
	for i, d := range ds {
		if d.Name == name {
			return i
		}
	}
	return -1
}
