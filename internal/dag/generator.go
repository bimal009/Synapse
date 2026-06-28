package dag

type Dag interface{}
type dag struct{}

func NewDag() Dag {
	return dag{}
}
