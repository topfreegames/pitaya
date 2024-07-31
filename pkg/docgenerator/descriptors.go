package docgenerator

import (
	"strings"

	"github.com/topfreegames/pitaya/v3/pkg/constants"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// ProtoDescriptors returns the descriptor for a given message name or .proto file
func ProtoDescriptors(protoName string) ([]byte, error) {
	if strings.HasSuffix(protoName, ".proto") {
		descriptor, _ := protoregistry.GlobalFiles.FindFileByPath(protoName)
		if descriptor == nil {
			return nil, constants.ErrProtodescriptor
		}

		desc, err := proto.Marshal(protodesc.ToFileDescriptorProto(descriptor))
		if err != nil {
			return nil, constants.ErrProtodescriptor
		}
		return desc, nil
	}

	if strings.HasPrefix(protoName, "types.") {
		protoName = strings.Replace(protoName, "types.", "google.protobuf.", 1)
	}

	protoReflectTypePointer, _ := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(protoName))
	if protoReflectTypePointer == nil {
		return nil, constants.ErrProtodescriptor
	}

	protoReflectType := protoReflectTypePointer.Descriptor()

	protoDescriptor, _ := proto.Marshal(protodesc.ToDescriptorProto(protoReflectType))

	return protoDescriptor, nil
}
