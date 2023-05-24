Builder
===

Pitaya offers a [`Builder`](../builder.go) object which can be utilized to define a sort of pitaya properties.

### PostBuildHooks

Post-build hooks can be used to execute additional actions automatically after the build process. It also allows you to interact with the built pitaya app.

A common use case is where it becomes necessary to perform configuration steps in both the pitaya builder and the pitaya app being built. In such cases, an effective approach is to internalize these configurations, enabling you to handle them collectively in a single operation or process. It simplifies the overall configuration process, reducing the need for separate and potentially repetitive steps.

```go
// main.go
cfg := config.NewDefaultBuilderConfig()
builder := pitaya.NewDefaultBuilder(isFrontEnd, "my-server-type", pitaya.Cluster, map[string]string{}, *cfg)

customModule := NewCustomModule(builder)
customModule.ConfigurePitaya(builder)

app := builder.Build()

// custom_object.go
type CustomObject struct {
	builder *pitaya.Builder
}

func NewCustomObject(builder *pitaya.Builder) *CustomObject {
	return &CustomObject{
		builder: builder,
	}
}

func (object *CustomObject) ConfigurePitaya() {
	object.builder.AddAcceptor(...)
	object.builder.AddPostBuildHook(func (app pitaya.App) {
		app.Register(...)
	})
}
```

In the above example the `ConfigurePitaya` method of the `CustomObject` is adding an `Acceptor` to the pitaya app being built, and also adding a post-build function which will register a handler `Component` that will expose endpoints to receive calls. 
