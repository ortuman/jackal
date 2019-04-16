package pubsubmodel

type Option struct {
	Name  string
	Value string
}

type Node struct {
	Host    string
	Name    string
	Options []Option
}
