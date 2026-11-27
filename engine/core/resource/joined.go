package resource

import "context"

// Joined is one resource of several parts, for a unit whose kind takes one: their changes in the
// parts' order.
type Joined []Resource

// Plan joins the parts' changes. A part's error stops it, as in Group.
func (j Joined) Plan(ctx context.Context) ([]Change, error) {
	var all []Change
	for _, part := range j {
		cs, err := part.Plan(ctx)
		if err != nil {
			return nil, err
		}
		all = append(all, cs...)
	}
	return all, nil
}
