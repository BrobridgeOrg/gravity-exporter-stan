module github.com/BrobridgeOrg/gravity-exporter-stan

go 1.13

require (
	github.com/BrobridgeOrg/gravity-api v0.0.0-20200808191818-646e409ed0b8
	github.com/BrobridgeOrg/gravity-exporter-nats v0.0.0-20200808204317-03f51c4b68f3
	github.com/nats-io/nats-streaming-server v0.18.0 // indirect
	github.com/nats-io/nats.go v1.10.0
	github.com/nats-io/stan.go v0.7.0
	github.com/sirupsen/logrus v1.6.0
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/viper v1.7.1
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859
	google.golang.org/grpc v1.31.0
)
