package models

type DataSource int

const (
	None DataSource = iota
	Amazon
	Newegg
	BestBuy
)

func (d DataSource) String() string {
	switch d {
	case Amazon:
		return "Amazon"
	case Newegg:
		return "Newegg"
	case BestBuy:
		return "BestBuy"
	default:
		return "None"
	}
}
