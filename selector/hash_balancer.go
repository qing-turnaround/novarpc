package selector

type hashBalancer struct {
}

func (h hashBalancer) Balance(s string, nodes []*Node) *Node {
	panic("implement me")
}
