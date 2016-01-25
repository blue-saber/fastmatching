package fastmatching

type IFastMatching interface {
	RegistData(keyword string, value int32) bool
	RetrieveData(keyword string) []int32
	Clear()
}
