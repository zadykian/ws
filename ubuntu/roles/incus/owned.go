package incus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/ubuntu/values"
)

// Owned mends the set owned4 of the table of sources to hold each guest's addresses and port, and
// takes out an element that no guest holds. The table's file fills the set as it loads, so this
// finds a hand edit. A table that is not loaded fails the run, as the unit of the table comes first.
type Owned struct {
	M      effect.Machine
	Guests []values.Guest
}

// element is a member of owned4 as nft prints and takes it: the address, then the port in quotes.
func element(address, port string) string { return address + ` . "` + port + `"` }

// held returns the elements that owned4 holds, as element writes them.
func (o Owned) held(ctx context.Context) ([]string, error) {
	out, err := o.M.Run(ctx, "nft", "-j", "list", "set", "bridge", "ws_sources", "owned4")
	if err != nil {
		return nil, err
	}
	var doc struct {
		Nftables []struct {
			Set *struct {
				Elem []struct {
					Concat []string `json:"concat"`
				} `json:"elem"`
			} `json:"set"`
		} `json:"nftables"`
	}
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&doc); err != nil {
		return nil, fmt.Errorf("nft list set owned4: its output is no JSON of a set of address and port: %w", err)
	}
	var held []string
	for _, item := range doc.Nftables {
		if item.Set == nil {
			continue
		}
		for _, e := range item.Set.Elem {
			if len(e.Concat) != 2 {
				return nil, fmt.Errorf("nft list set owned4: an element of %d parts, not an address and a port", len(e.Concat))
			}
			held = append(held, element(e.Concat[0], e.Concat[1]))
		}
	}
	return held, nil
}

// Plan plans one change to add the missing elements and one to delete the others.
func (o Owned) Plan(ctx context.Context) ([]resource.Change, error) {
	held, err := o.held(ctx)
	if err != nil {
		return nil, err
	}
	var want, add []string
	for _, g := range o.Guests {
		for _, src := range g.Sources() {
			e := element(src.Address, src.Port)
			want = append(want, e)
			if !slices.Contains(held, e) {
				add = append(add, e)
			}
		}
	}
	drop := slices.DeleteFunc(slices.Clone(held), func(e string) bool { return slices.Contains(want, e) })
	var changes []resource.Change
	if len(add) > 0 {
		changes = append(changes, setElements{o.M, "add", add})
	}
	if len(drop) > 0 {
		changes = append(changes, setElements{o.M, "delete", drop})
	}
	return changes, nil
}

// setElements adds elements to owned4, or deletes them.
type setElements struct {
	m        effect.Machine
	verb     string
	elements []string
}

func (c setElements) String() string {
	return fmt.Sprintf("%s the guests' sources of the table ws_sources: %s", c.verb, strings.Join(c.elements, ", "))
}

func (c setElements) Apply(ctx context.Context) error {
	script := fmt.Sprintf("%s element bridge ws_sources owned4 { %s }\n", c.verb, strings.Join(c.elements, ", "))
	_, err := c.m.RunInput(ctx, []byte(script), "nft", "-f", "-")
	return err
}
