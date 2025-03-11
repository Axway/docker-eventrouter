package report

import (
	"encoding/json"
	"strconv"
)

type CftMetrics struct {
	TransferCount int `json:"transfercount"`
	Mainframe     int `json:"mainframe"`
}

func NewCftMetrics(transferCount int, mainframe int) CftMetrics {
	m := CftMetrics{TransferCount: transferCount, Mainframe: mainframe}
	return m
}

func (m *CftMetrics) ToJson() string {
	out := `{`
	out += `"transfercount": ` + strconv.Itoa(m.TransferCount) + `,`
	out += `"mainframe": ` + strconv.Itoa(m.Mainframe) + ``
	out += `}`

	return out
}

func (m *CftMetrics) ToPlatformJson() string {
	out := `{`
	out += `"CFT.Transfers": ` + strconv.Itoa(m.TransferCount) + `,`
	out += `"CFT.MainframeTransfers": ` + strconv.Itoa(m.Mainframe) + ``
	out += `}`

	return out
}

func (m *CftMetrics) FromJson(jsonin string) {
	json.Unmarshal([]byte(jsonin), m)
}
