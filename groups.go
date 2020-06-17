package groups

//IGroups is a collection of Groups
type IGroups interface {
	List(filter map[string]interface{}, sizeLimit int, orderBy []string) []Group
	New(g Group) (Group, error)
	Get(id string) (*Group, error)
	Upd(g Group) error
	Del(id string) error
}
