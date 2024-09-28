package trend

import (
	"fmt"
	"testing"

	"github.com/kurosann/aqt-sdk/stockinds/utils"
)

// RUN
// go test -v ./trend -run TestDema
func TestDema(t *testing.T) {
	t.Parallel()
	list := utils.GetTestKline()

	stock := NewDefaultDema(list)

	var dataList = stock.GetData()

	fmt.Printf("-- %s --\n", stock.Name)
	for i := len(dataList) - 1; i > 0; i-- {
		if i < len(dataList)-5 {
			break
		}
		var v = dataList[i]
		fmt.Printf("\t[%d]Time:%s\tDema%d:%f\n", i, v.Time.Format("2006-01-02 15:04:05"), stock.Period, v.Value)
	}

}