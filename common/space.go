package common

// SpaceID 空间ID，属于框架定义的类型
type SpaceID int

// MasterSpaceID 世界空间ID
const MasterSpaceID SpaceID = 1

const InvalidSpaceID SpaceID = -1

func (s SpaceID) Valid() bool {
	return s >= MasterSpaceID
}
