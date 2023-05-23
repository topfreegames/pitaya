Handshake Validators
=====

Pitaya allows to defined Handshake Validators.<br />

The primary purpose of these validators is to perform validation checks on the data transmitted by the client. The validators play a crucial role in verifying the integrity and reliability of the client's input before establishing a connection.

In addition to data validation, handshake validators can also execute other custom logic to assess the client's compliance with the server-defined requirements. This additional logic may involve evaluating factors such as authenticating credentials, permissions, or any other criteria necessary to determine the client's eligibility to access the server.

### Adding handshake validators

To ensure the effective utilization of these validators, they should be added to the `SessionPool` component. As a result, each newly created session within the `SessionPool` will automatically incorporate the designated validators.

Once the handshake process is initiated, the validators will be invoked to execute their validation routines.

```go
cfg := config.NewDefaultBuilderConfig()
builder := pitaya.NewDefaultBuilder(isFrontEnd, "my-server-type", pitaya.Cluster, map[string]string{}, *cfg)
builder.SessionPool.AddHandshakeValidator("MyCustomValidator", func (data *session.HandshakeData) error {
	if data.Sys.Version != "1.0.0" {
		return errors.New("Unknown client version")
	}

	return nil
})
```

As a result of the validation process, if an error is encountered, the server will transmit a message to client within the code 400. This code emulates the widely recognized HTTP Bad Request status code, indicating that the client's request could not be fulfilled due to invalid data. Otherwise, if the validation process succeeds, the server will dispatch a message to client containing a code 200, mirroring the HTTP Ok status code.<br />
**Is important to mention that, when there are many validator functions, the validation will stop as soon it encounters the first error.**
