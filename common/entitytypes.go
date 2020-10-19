package common

import (
	"fmt"

	uuid "github.com/satori/go.uuid"
)

// entityIDLength is the length of Entity IDs
const entityIDLength = 16

// GenDataID gen data ID
func GenDataID() string {
	return string(GenEntityID())
}

// GenEntityID generates a new EntityID
func GenEntityID() string {
	return uuid.NewV1().String()
}

// MustEntityID assures a string to be EntityID
// func MustEntityID(id string) EntityID {
// 	if len(id) != 16 {
// 		logger.Panicf("%s of len %d is not a valid entity ID (len=%d)", id, len(id), entityIDLength)
// 	}
// 	return EntityID(id)
// }

// clientIDLength is the length of Client IDs
const clientIDLength = 16

// LandID 地块的实体ID
func LandID(unitIndex int) string {
	// return utils.uuid.NewV1().String()
	// TODO 地块id 跟地块的单元索引是相关的
	return fmt.Sprintf("%d", unitIndex)
}

// // 用来缓存 数字到string 和 string到数字
// var i2s map[int]string
// var s2i map[string]int

// func init() {
// 	const cacheLength = 1000000
// }
