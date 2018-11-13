package messages

// SetCounterArg is the argument on room.setcounter handler
type SetCounterArg struct {
	Value float64
	Tag1  string
	Tag2  string
}

// SetGaugeArg is the argument on room.setgauge* handler
type SetGaugeArg struct {
	Value float64
	Tag   string
}

// SetSummaryArg is the argument on room.setsummary handler
type SetSummaryArg struct {
	Value float64
	Tag   string
}
