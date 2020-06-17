package groups

import (
	"fmt"
	"regexp"
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
	if !ValidName(g.Name) {
		return fmt.Errorf("invalid name \"%s\"", g.Name)
	}
	return nil
}

const namePattern = `[a-zA-Z0-9]([a-zA-Z0-9 _\.\-][a-zA-Z0-9])*`

var nameRegex = regexp.MustCompile(`^` + namePattern + `$`)

//ValidName ...
func ValidName(n string) bool {
	if nameRegex.MatchString(n) {
		return true
	}
	return false
}

// //Member ...
// type Member struct{}

// //Invitation ...
// type Invitation struct{}
