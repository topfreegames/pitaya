package common

import (
	"fmt"

	"github.com/tutumagi/gecs"
	"go.mongodb.org/mongo-driver/bson"
)

// 构造 entity-component 映射关系的表结构
// 参考 http://t-machine.org/index.php/2009/10/26/entity-systems-are-the-future-of-mmos-part-5/

// 首先注册实体，那么在创建实体时会通过元数据组件组成对应的数据

// MetaEntity 实体的元数据类型，每一个 name 表示一种类型的实体
type MetaEntity struct {
	Name        string `bson:"name"`  // 唯一的名字
	Label       string `bson:"label"` // for debug
	Description string `bson:"description"`
}

// MetaEntityComponents 实体映射组件的结构，每一个实体会映射一个或者多个组件类型
type MetaEntityComponents struct {
	EntityName  string `bson:"entity_name"` // the name mapper MetaEntity's name
	ComponentID string `bson:"component_id"`
}

// MetaComponent 组件的元数据结构
//	每个组件会有一个唯一ID，会有名字
//	TableName 是将不同组件类型的数据拆分到不同的表条目中去
type MetaComponent struct {
	ID          string `bson:"id"`   // 唯一id
	Name        string `bson:"name"` // 唯一 name
	Description string `bson:"description"`
	TableName   string `bson:"table_name"` // table-name 指向了该组件数组在哪个表里面

	RID        gecs.ComponentID `bson:"-"` // runtime id
	Persistent bool             `bson:"-"` // persistent or not
}

// Entity 实际的实体
type Entity struct {
	ID string `bson:"id" json:"id"` // 实体 id，唯一 id
	// 因为前端不使用 ecs，这里 Label 用来表示该 entity 是什么类型的 entity
	// 在 entityType 中 typeName 赋值到该 Label 上
	Label string `bson:"label" json:"label"`

	RID gecs.EntityID `bson:"-" json:"rid"` // runtime id
}

func (e *Entity) String() string {
	return fmt.Sprintf("<Entity>(ID:%v Label:%v RID:%v)", e.ID, e.Label, e.RID)
}

// EntityComponent 实际的实体含有的组件 ID 和组件数据 ID, 组件 ID 和组件数据 ID 多对多关系
type EntityComponent struct {
	EntityID    string `bson:"entity_id"`
	ComponentID string `bson:"component_id"`
	DataID      string `bson:"data_id"`
}

// ComponentReadData 由用户自定义，实际的组件数据
//	读取的时候用这个，方便反射到注册的组件数据类型
type ComponentReadData struct {
	DataID string   `bson:"data_id"`
	Data   bson.Raw `bson:"data"` // 使用 []byte 方便转换为 bson.Raw， 方便转换为实际注册的组件类型 参考：https://stackoverflow.com/questions/46762446/how-do-i-unmarshal-the-bson-from-mongo-of-a-nested-interface-with-mgo
}

// ComponentWriteData 存储的时候用这个，方便数据库查询时可读
type ComponentWriteData struct {
	DataID string      `bson:"data_id"`
	Data   interface{} `bson:"data"`
}

// ComponentDataPair componentID & ComponentDataID & Data
type ComponentDataPair struct {
	Data        interface{} // component data
	ComponentID string      // component id
	DataID      string      // component data id
}
