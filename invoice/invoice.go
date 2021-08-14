package invoice

import (
	"bytes"
	"time"

	"github.com/klauspost/lctime"
)

type Invoice struct {
	CurrentDate time.Time
	TextTitle   string
	PaidTo      string
	BillTo      string
	ServiceDesc string
	ValuePaid   string
	ValueAdds   string
	AddsDesc    string
	City        string
	Locale      string
}

func (i Invoice) GetFullDocName() string {
	return i.TextTitle + " - " + i.CurrentDate.Format("20060102")
}

func (i Invoice) GetSignText() string {
	signText := lineFeed(6) + "______________________________________________________" + lineFeed(1) + i.PaidTo
	return signText
}

func (i Invoice) GetLocation() string {
	lctime.SetLocale(i.Locale)
	extDate := ", " + i.CurrentDate.Format("02") + " de " + lctime.Strftime("%B", i.CurrentDate) + " de " + i.CurrentDate.Format("2006") + "."
	return lineFeed(4) + i.City + extDate
}

func (i Invoice) GetDisclaimerText() string {
	disclaimerText := lineFeed(5) + "Eu, " + i.PaidTo + ", recebi da  Sr(a). " + i.BillTo + " , a importância de R$ " + i.ValuePaid + ", referente " + i.ServiceDesc + "no dia presente e R$ " + i.ValueAdds + ", referente aos custos de " + i.AddsDesc + " no dia presente."
	return disclaimerText
}

func lineFeed(a int) string {
	var b bytes.Buffer
	for i := 0; i < a; i++ {
		b.WriteString("\n")
	}
	return b.String()
}
