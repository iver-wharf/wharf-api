package main

import (
	"io"
	"math"
	"net"

	v5 "github.com/iver-wharf/wharf-api/v5/api/wharfapi/v5"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type grpcWharfServer struct {
	v5.UnimplementedBuildsServer
	db *gorm.DB
}

func serveGRPC(listener net.Listener, db *gorm.DB) {
	grpcServer := grpc.NewServer()
	grpcWharf := &grpcWharfServer{db: db}
	v5.RegisterBuildsServer(grpcServer, grpcWharf)
	grpcServer.Serve(listener)
}

func (s *grpcWharfServer) CreateLogStream(stream v5.Builds_CreateLogStreamServer) error {
	var logsInserted uint64
	for {
		line, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		} else if line == nil {
			log.Warn().Message("Received nil message, skipping.")
			continue
		}
		if err := line.Timestamp.CheckValid(); err != nil {
			log.Warn().WithError(err).
				Message("Received invalid log timestamp, skipping.")
			continue
		}
		if line.BuildID == 0 {
			log.Warn().Message("Received log with build ID: 0, skipping.")
			continue
		}
		if line.BuildID > math.MaxUint {
			log.Warn().WithUint64("buildId", line.BuildID).
				Message("Received too big log build ID, skipping.")
			return status.Errorf(codes.InvalidArgument,
				"received build ID is too big: %d (build ID) > %d (max)",
				line.BuildID, uint(math.MaxUint))
		}
		createdLog, err := saveLog(s.db.WithContext(stream.Context()),
			uint(line.BuildID),
			line.Message,
			line.Timestamp.AsTime(),
		)
		if err != nil {
			return status.Errorf(codes.Internal, "insert logs: %v", err)
		}
		log.Debug().WithUint("logId", createdLog.LogID).
			Message("Inserted log into database.")
		build(createdLog.BuildID).Submit(response.Log{
			LogID:     createdLog.LogID,
			BuildID:   createdLog.BuildID,
			Message:   createdLog.Message,
			Timestamp: createdLog.Timestamp,
		})
		logsInserted++
	}
	return stream.SendAndClose(&v5.CreateLogStreamResponse{
		LinesInserted: logsInserted,
	})
}
