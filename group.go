package groups

import (
	"fmt"
	"strings"
)

//Group ...
type Group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Admins      []Member
	// Members     []Member
	// Invitations []Invitation
}

//Validate group contents
func (g *Group) Validate() error {
	g.Name = strings.TrimSpace(g.Name)
	if g.Name == "" {
		return fmt.Errorf("missing name")
	}
	return nil
}

// //Member ...
// type Member struct{}

// //Invitation ...
// type Invitation struct{}
