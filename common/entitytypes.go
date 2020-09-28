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

// EntityID type
type EntityID string

// IsNil returns if EntityID is nil
func (id EntityID) IsNil() bool {
	return id == ""
}

// GenEntityID generates a new EntityID
func GenEntityID() EntityID {
	return EntityID(uuid.NewV1().String())
}

// MustEntityID assures a string to be EntityID
// func MustEntityID(id string) EntityID {
// 	if len(id) != 16 {
// 		logger.Panicf("%s of len %d is not a valid entity ID (len=%d)", id, len(id), entityIDLength)
// 	}
// 	return EntityID(id)
// }

// ClientID type
type ClientID string

// GenClientID generates a new Client ID
func GenClientID() ClientID {
	return ClientID(uuid.NewV1().String())
}

// IsNil returns if ClientID is nil
func (id ClientID) IsNil() bool {
	return id == ""
}

// clientIDLength is the length of Client IDs
const clientIDLength = 16

// GenRoadID 新建道路ID
func GenRoadID() EntityID {
	return EntityID(uuid.NewV1().String())
}

// GenBuildingID 新建建筑ID
func GenBuildingID() EntityID {
	return EntityID(uuid.NewV1().String())
}

// LandID 地块的实体ID
func LandID(unitIndex int) EntityID {
	// return utils.uuid.NewV1().String()
	// TODO 地块id 跟地块的单元索引是相关的
	return EntityID(fmt.Sprintf("%d", unitIndex))
}

// // 用来缓存 数字到string 和 string到数字
// var i2s map[int]string
// var s2i map[string]int

// func init() {
// 	const cacheLength = 1000000
// }
