package report

type CycleIDObj struct {
	CycleID     string
	Monitor     string
	EventDate   string
	IsMainframe bool
}
type UsageObj struct {
	DateTime string
	Monitor  string
	Metrics  string
}

func SQLInsertOpenCycleID() string {
	return "INSERT INTO " + cycleIDTable + " (cycleid, monitor, eventdate, ismainframe) VALUES "
}
func SQLInsertOpenUsage() string {
	return "INSERT INTO " + usageTable + " (datetime, monitor, metrics) VALUES "
}
func SQLInsertSeparator() string {
	return ","
}
func SQLInsertClose() string {
	return ";"
}

func NewCycleIDObj(cycleId string, monitor string, eventDate string, isMainframe int) CycleIDObj {
	obj := CycleIDObj{CycleID: cycleId, Monitor: monitor, EventDate: eventDate, IsMainframe: isMainframe == 1}
	return obj
}
func (obj *CycleIDObj) Key() string {
	return obj.Monitor + obj.CycleID
}
func (obj *CycleIDObj) SQLInsertCmd() string {
	return SQLInsertOpenCycleID() + obj.SQLInsertValues() + SQLInsertClose()
}
func (obj *CycleIDObj) SQLInsertValues() string {
	mainframe := "0"
	if obj.IsMainframe {
		mainframe = "1"
	}
	return "('" + obj.CycleID + "','" + obj.Monitor + "','" + obj.EventDate + "','" + mainframe + "')"
}
func (obj *CycleIDObj) SQLUpdateCmd() string {
	mainframe := "0"
	if obj.IsMainframe {
		mainframe = "1"
	}
	return "UPDATE " + cycleIDTable + " SET ismainframe='" + mainframe + "' " +
		"WHERE cycleid='" + obj.CycleID + "' AND monitor='" + obj.Monitor + "';"
}

func NewUsageObj(dateTime string, monitor string, metrics string) UsageObj {
	obj := UsageObj{DateTime: dateTime, Monitor: monitor, Metrics: metrics}
	return obj
}
func (obj *UsageObj) Key() string {
	return obj.Monitor + obj.DateTime
}
func (obj *UsageObj) SQLInsertCmd() string {
	return SQLInsertOpenUsage() + obj.SQLInsertValues() + SQLInsertClose()
}
func (obj *UsageObj) SQLInsertValues() string {
	return "('" + obj.DateTime + "','" + obj.Monitor + "','" + obj.Metrics + "')"
}
func (obj *UsageObj) SQLUpdateCmd() string {
	return "UPDATE " + usageTable + " SET metrics = '" + obj.Metrics +
		"' WHERE datetime='" + obj.DateTime + "' AND monitor='" + obj.Monitor + "';"

}
