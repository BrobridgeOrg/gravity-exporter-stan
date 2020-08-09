package exporter

import (
	"golang.org/x/net/context"

	pb "github.com/BrobridgeOrg/gravity-api/service/exporter"
	app "github.com/BrobridgeOrg/gravity-exporter-stan/pkg/app"
)

type Service struct {
	app app.App
}

func NewService(a app.App) *Service {

	service := &Service{
		app: a,
	}

	return service
}

func (service *Service) SendEvent(ctx context.Context, in *pb.SendEventRequest) (*pb.SendEventReply, error) {

	conn := service.app.GetEventBus().GetConnection()

	err := conn.Publish(in.Channel, in.Payload)
	if err != nil {
		return &pb.SendEventReply{
			Success: false,
		}, err
	}

	return &pb.SendEventReply{
		Success: true,
	}, nil
}
